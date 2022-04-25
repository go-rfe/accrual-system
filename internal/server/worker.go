package server

import (
	"accrual-system/internal/repository"
	"context"

	"github.com/go-rfe/logging/log"
)

type Worker struct {
	Signal <-chan struct{}
}

func (pw *Worker) Run(ctx context.Context, store repository.Storage) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-pw.Signal:
			go UpdateOrder(ctx, store)
		}
	}
}

func UpdateOrder(ctx context.Context, store repository.Storage) {
	if err := store.UpdateOrder(ctx); err != nil {
		log.Error().Err(err).Msg("failed to update registered order")
	}
}
