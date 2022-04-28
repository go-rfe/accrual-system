package server

import (
	"accrual-system/internal/repository"
	"context"

	"github.com/go-rfe/logging/log"
)

type Worker struct {
	Signal <-chan struct{}
}

func (w *Worker) Run(ctx context.Context, s repository.Storage) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.Signal:
			go UpdateOrder(ctx, s)
		}
	}
}

func UpdateOrder(ctx context.Context, s repository.Storage) {
	if err := s.UpdateOrder(ctx); err != nil {
		log.Error().Err(err).Msg("failed to update registered order")
	}
}
