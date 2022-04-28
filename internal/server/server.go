package server

import (
	"accrual-system/internal/repository"
	"accrual-system/internal/updater"
	"context"
	"net/http"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/go-rfe/logging/log"
	realip "github.com/thanhhh/gin-gonic-realip"
)

type AccrualServer struct {
	RunAddress  string
	DatabaseURI string
	Storage     repository.Storage
	Signal      chan struct{}
	ctx         context.Context
	server      *http.Server
}

// InitStorage Init database as a storage
func initStorage(s *AccrualServer) repository.Storage {
	log.Debug().Msg("initializing storage")
	storage, err := repository.NewDBStorage(s.DatabaseURI)
	if err != nil {
		log.Error().Msgf("unable to init database: %s", err)
	}
	return storage
}

// Run HTTP server main loop
func (s *AccrualServer) Run(ctx context.Context) {
	log.Debug().Msg("starting server")

	sCtx, sCancel := context.WithCancel(ctx)
	defer sCancel()

	s.Storage = initStorage(s)
	defer func(storage repository.Storage) {
		err := storage.Close()
		if err != nil {
			log.Error().Err(err).Msg("couldn't close database")
		}
	}(s.Storage)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	_ = r.SetTrustedProxies([]string{"127.0.0.1"})
	r.HandleMethodNotAllowed = true
	r.RedirectTrailingSlash = false

	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(realip.RealIP())

	r.GET("/api/orders/:number", limitMiddleware(), getOrderHandler(s.Storage))
	r.POST("/api/orders", updateOrdersHandler(s.Storage, s.Signal))
	r.POST("/api/goods", updateGoodsHandler(s.Storage))

	worker := updater.Worker{Signal: s.Signal}
	wCtx, wCancel := context.WithCancel(sCtx)
	go worker.Run(wCtx, s.Storage)
	defer wCancel()

	err := r.Run(s.RunAddress)
	if err != nil {
		panic(err)
	}
}
