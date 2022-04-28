package repository

import (
	"accrual-system/internal/models"
	"context"
)

type Storage interface {
	CreateOrder(ctx context.Context, order models.Order) error
	GetOrder(ctx context.Context, orderID string) (*models.Order, error)
	CreateReward(ctx context.Context, reward models.Reward) error
	UpdateOrder(ctx context.Context) error
	Close() error
}
