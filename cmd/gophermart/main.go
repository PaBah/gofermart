package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/PaBah/gofermart/cmd/gophermart/server"
	"github.com/PaBah/gofermart/internal/accrual"
	"github.com/PaBah/gofermart/internal/config"
	"github.com/PaBah/gofermart/internal/logger"
	"github.com/PaBah/gofermart/internal/storage"
	"go.uber.org/zap"
)

func main() {
	options := &config.Options{}
	ParseFlags(options)

	if err := logger.Initialize(options.LogsLevel); err != nil {
		fmt.Printf("Logger can not be initialized %s", err)
		return
	}

	logger.Log().Info("Start server on", zap.String("address", options.RunAddress))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var store storage.Repository
	dbStore, err := storage.NewDBStorage(context.Background(), options.DatabaseURI)
	if err != nil {
		logger.Log().Error("Database error with start", zap.Error(err))
		return
	}
	store = &dbStore
	defer dbStore.Close()

	newServer := server.NewRouter(options, &store)
	scraper := accrual.NewOrdersAccrualClient(options, store)
	scraper.ScrapeOrders()

	go func() {
		err := http.ListenAndServe(options.RunAddress, newServer)

		if err != nil {
			logger.Log().Error("Server crashed with error: ", zap.Error(err))
		}
	}()

	<-ctx.Done()
	//accrual.Main()
}
