package repository

import (
	"accrual-system/internal/models"
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/shopspring/decimal"

	"github.com/go-rfe/logging/log"
)

const (
	driver       = "pgx"
	maxIdleConns = 10
)

type Database struct {
	conn  *sql.DB
	cache map[string]models.Reward
}

func NewDBStorage(uri string) (*Database, error) {
	log.Debug().Msg("creating database connection")

	conn, err := sql.Open(driver, uri)
	if err != nil {
		fmt.Println("db err: ", err)
	}

	db := Database{
		conn:  conn,
		cache: make(map[string]models.Reward),
	}
	db.conn.SetMaxIdleConns(maxIdleConns)

	_, err = db.conn.Exec("CREATE TABLE IF NOT EXISTS orders(" +
		"number BIGINT PRIMARY KEY, " +
		"status VARCHAR(50) DEFAULT 'REGISTERED', " +
		"accrual DECIMAL DEFAULT 0);")
	if err != nil {
		return nil, err
	}

	_, err = db.conn.Exec("CREATE TABLE IF NOT EXISTS goods(" +
		"id SERIAL PRIMARY KEY," +
		"order_number BIGINT REFERENCES orders(number)," +
		"description VARCHAR(250)," +
		"price DECIMAL);")
	if err != nil {
		return nil, err
	}

	_, err = db.conn.Exec("CREATE TABLE IF NOT EXISTS rewards(" +
		"id SERIAL PRIMARY KEY," +
		"match VARCHAR(250) UNIQUE," +
		"reward DECIMAL," +
		"reward_type VARCHAR(2));")
	if err != nil {
		return nil, err
	}

	rewardsRows, err := db.conn.Query("SELECT match,reward,reward_type FROM rewards")

	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Error().Err(err).Msgf("couldn't close rows")
		}
	}(rewardsRows)

	for rewardsRows.Next() {
		var reward models.Reward
		err = rewardsRows.Scan(&reward.Match, &reward.Reward, &reward.RewardType)
		if err != nil {
			return nil, err
		}

		db.cache[reward.Match] = reward
	}

	err = rewardsRows.Err()
	if err != nil {
		return nil, err
	}

	return &db, nil
}

func (db *Database) CreateOrder(ctx context.Context, order models.Order) error {
	var existingOrder string
	row := db.conn.QueryRowContext(ctx,
		"SELECT number FROM orders WHERE number = $1", order.Order)

	err := row.Scan(&existingOrder)
	if !errors.Is(err, nil) && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return models.ErrOrderExists
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmtInsertOrder, err := tx.Prepare("INSERT INTO orders (number) VALUES ($1)")
	if err != nil {
		return err
	}
	defer func(stmt *sql.Stmt) {
		if err := stmt.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close insert statement")
		}
	}(stmtInsertOrder)

	stmtInsertGood, err := tx.Prepare("INSERT INTO goods (order_number, description, price) VALUES ($1, $2, $3)")
	if err != nil {
		return err
	}
	defer func(stmt *sql.Stmt) {
		if err := stmt.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close insert statement")
		}
	}(stmtInsertGood)

	if _, err := stmtInsertOrder.Exec(order.Order); err != nil {
		if err := tx.Rollback(); err != nil {
			log.Error().Err(err).Msg("unable to rollback transaction")
		}

		return err
	}

	for _, good := range order.Goods {
		if _, err := stmtInsertGood.Exec(order.Order, good.Description, good.Price); err != nil {
			if err := tx.Rollback(); err != nil {
				log.Error().Err(err).Msg("unable to rollback transaction")
			}

			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *Database) GetOrder(ctx context.Context, orderID string) (*models.Order, error) {
	var sum decimal.Decimal
	order := models.Order{}

	row := db.conn.QueryRowContext(ctx,
		"SELECT number,status,accrual FROM orders WHERE number = $1", orderID)

	err := row.Scan(&order.Order, &order.Status, &sum)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, models.ErrOrderDoesntExist
	case !errors.Is(err, nil):
		return nil, err
	}

	_, ok := models.WithoutAccrualStatuses[order.Status]
	if ok {
		return &order, nil
	}
	order.Accrual = sum
	return &order, nil
}

// CreateReward creates pattern for goods
func (db *Database) CreateReward(ctx context.Context, reward models.Reward) error {
	var existingOrder string
	row := db.conn.QueryRowContext(ctx,
		"SELECT match FROM rewards WHERE match = $1", reward.Match)

	err := row.Scan(&existingOrder)
	if !errors.Is(err, nil) && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return models.ErrRewardExists
	}

	_, err = db.conn.ExecContext(ctx,
		"INSERT INTO rewards (match, reward, reward_type) VALUES ($1, $2, $3)",
		reward.Match, reward.Reward, reward.RewardType)

	if err != nil {
		return err
	}

	db.cache[reward.Match] = reward

	return nil
}

// UpdateOrder calculates rewards for the new one order
func (db *Database) UpdateOrder(ctx context.Context) error {
	order := models.Order{}

	row := db.conn.QueryRowContext(ctx,
		"SELECT number FROM orders WHERE status = $1 LIMIT 1", models.RegisteredStatus)

	err := row.Scan(&order.Order)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return models.ErrOrderDoesntExist
	case err != nil:
		return err
	}

	// Start processing order
	order.Status = models.ProcessingStatus
	if err := db.updateOrder(ctx, order); err != nil {
		return err
	}

	stmt, err := db.conn.Prepare(
		"SELECT price FROM goods WHERE order_number = $1 AND description LIKE '%'||$2||'%'")
	if err != nil {
		return err
	}
	for _, reward := range db.cache {
		goodsRows, err := stmt.QueryContext(ctx, order.Order, reward.Match)
		if err != nil {
			return err
		}

		for goodsRows.Next() {
			var price decimal.Decimal
			if err := goodsRows.Scan(&price); err != nil {
				return err
			}

			switch reward.RewardType {
			case models.PointsRewardType:
				order.Accrual = order.Accrual.Add(reward.Reward)
			case models.PercentsRewardType:
				order.Accrual = order.Accrual.Add(price.Mul(reward.Reward).Div(decimal.NewFromInt(models.OneHundredPercents)))
			}
		}

		if err := goodsRows.Err(); err != nil {
			return err
		}

		if err := goodsRows.Close(); err != nil {
			log.Error().Err(err).Msgf("couldn't close rows")
		}
	}

	// End processing order
	if order.Accrual.Equal(decimal.Zero) {
		order.Status = models.InvalidStatus
	} else {
		order.Status = models.ProcessedStatus
	}

	if err := db.updateOrder(ctx, order); err != nil {
		return err
	}

	return nil
}

// updateOrder updates provided order in database inside transaction
func (db *Database) updateOrder(ctx context.Context, order models.Order) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE orders SET status = $1, accrual = $2 WHERE number = $3",
		order.Status, order.Accrual, order.Order); err != nil {
		if err := tx.Rollback(); err != nil {
			log.Error().Err(err).Msg("unable to rollback transaction")
		}

		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// Close closes store database db
func (db *Database) Close() error {
	return db.conn.Close()
}
