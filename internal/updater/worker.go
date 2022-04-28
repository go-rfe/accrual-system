package updater

import (
	"accrual-system/internal/repository"
	"context"

	"github.com/go-rfe/logging/log"
)

type Worker struct {
	Signal <-chan struct{}
}

func (w *Worker) Run(ctx context.Context, s repository.Storage) {
	log.Info().Msg("worker is started")
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.Signal:
			go update(ctx, s)
		}
	}
}

func update(ctx context.Context, s repository.Storage) {
	log.Debug().Msg("worker is updating the order")
	if err := s.UpdateOrder(ctx); err != nil {
		log.Error().Msg("failed to update registered order")
	}
}
