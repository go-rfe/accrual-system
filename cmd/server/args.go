package main

import (
	"github.com/spf13/cobra"
)

var (
	args = &cobra.Command{
		Use:     "./server -h",
		Example: "./server --address localhost:8080",
	}
	ServerAddress string
	DatabaseURI   string
	LogLevel      string
)

func ReadFlags() {
	args.Flags().StringVarP(&ServerAddress, "address", "a", defaultServerAddr,
		"server listening address like (ip|fqdn):port",
	)

	args.Flags().StringVarP(&DatabaseURI, "databaseURI", "d", defaultDatabaseURI,
		"database connection string",
	)

	args.Flags().StringVarP(&LogLevel, "logLevel", "l", defaultLogLevel,
		"log level",
	)
}
