package main

import (
	"accrual-system/internal/server"
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-rfe/logging"
	"github.com/go-rfe/logging/log"
)

const (
	defaultServerAddr  = "localhost:8080"
	defaultDatabaseURI = "postgresql://postgres:mysecret@localhost/accrual?sslmode=disable"
	defaultLogLevel    = "INFO"
)

func GetEnvVar(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func init() {
	ReadFlags()
	if args.Execute() != nil {
		os.Exit(1)
	}
}

func main() {
	logging.Level(GetEnvVar("LOG_LEVEL", LogLevel))

	accrualServer := server.AccrualServer{
		RunAddress:  GetEnvVar("RUN_ADDRESS", ServerAddress),
		DatabaseURI: GetEnvVar("DATABASE_URI", DatabaseURI),
		Signal:      make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	go accrualServer.Run(ctx)

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	log.Debug().Msgf("caught %v", <-osSignal)

	cancel()
	log.Info().Msg("server is stopped")
}
