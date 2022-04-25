package repository

import (
	"accrual-system/internal/pkg/accrual"
	"context"
)

type Storage interface {
	CreateOrder(ctx context.Context, order accrual.Order) error
	GetOrder(ctx context.Context, orderID string) (*accrual.Order, error)
	CreateReward(ctx context.Context, reward accrual.Reward) error
	UpdateOrder(ctx context.Context) error
}
