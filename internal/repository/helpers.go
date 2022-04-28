package repository

import (
	"accrual-system/internal/models"
	"context"

	"github.com/go-rfe/logging/log"
)

// helperUpdateOrder updates provided order in database inside transaction
func (db *Database) helperUpdateOrder(ctx context.Context, order models.Order) error {
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
