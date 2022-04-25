package server

import (
	repository2 "accrual-system/internal/repository"
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
	Storage     repository2.Storage
	Signal      chan struct{}
	context     context.Context
	server      *http.Server
}

// initStorage Init database as a storage
func initStorage(s *AccrualServer) repository2.Storage {
	st, err := repository2.NewDBStorage(s.DatabaseURI)
	if err != nil {
		log.Error().Msgf("unable to init database: %s", err)
	}
	return st
}

// Run HTTP server main loop
func (s *AccrualServer) Run(ctx context.Context) {
	sCtx, sCancel := context.WithCancel(ctx)
	defer sCancel()

	s.context = sCtx
	storage := initStorage(s)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	_ = r.SetTrustedProxies([]string{"127.0.0.1"})
	r.HandleMethodNotAllowed = true
	r.RedirectTrailingSlash = false
	r.ForwardedByClientIP = true

	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(realip.RealIP())

	r.GET("/api/orders/:number", limitMiddleware(), getOrderHandler(storage))
	r.POST("/api/orders/", getOrderHandler(storage))
	r.POST("/api/orders", updateOrdersHandler(storage, s.Signal))
	r.POST("/api/goods", updateGoodsHandler(storage))

	worker := Worker{Signal: s.Signal}
	wCtx, wCancel := context.WithCancel(ctx)
	go worker.Run(wCtx, s.Storage)
	defer wCancel()

	err := r.Run(s.RunAddress)
	if err != nil {
		panic(err)
	}
}
