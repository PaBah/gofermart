package main

import (
	"flag"
	"os"

	"github.com/PaBah/gofermart/internal/config"
)

func ParseFlags(options *config.Options) {
	var specified bool
	var runAddress, databaseURI, accrualSystemAddress, logsLevel string

	flag.StringVar(&options.RunAddress, "a", ":8081", "host:port on which server run")
	flag.StringVar(&options.DatabaseURI, "d", "host=localhost user=paulbahush dbname=gofermart password=", "database DSN address")
	flag.StringVar(&options.AccrualSystemAddress, "r", ":8080", "host:port on which accrual server run")
	flag.StringVar(&options.LogsLevel, "l", "info", "logs level")
	flag.Parse()

	runAddress, specified = os.LookupEnv("RUN_ADDRESS")
	if specified {
		options.RunAddress = runAddress
	}

	databaseURI, specified = os.LookupEnv("DATABASE_URI")
	if specified {
		options.DatabaseURI = databaseURI
	}

	accrualSystemAddress, specified = os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS")
	if specified {
		options.AccrualSystemAddress = accrualSystemAddress
	}

	logsLevel, specified = os.LookupEnv("LOG_LEVEL")
	if specified {
		options.LogsLevel = logsLevel
	}
}
