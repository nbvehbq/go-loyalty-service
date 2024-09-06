package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nbvehbq/go-loyalty-service/internal/logger"
	"github.com/nbvehbq/go-loyalty-service/internal/server"
	"github.com/nbvehbq/go-loyalty-service/internal/session"
	"github.com/nbvehbq/go-loyalty-service/internal/storage/postgres"
)

func main() {
	cfg, err := server.NewConfig()

	if err != nil {
		log.Fatal(err, "Load config")
	}

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatal(err, "initialize logger")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := session.NewSessionStorage(ctx)
	db, err := postgres.NewStorage(ctx, cfg.DSN)
	if err != nil {
		log.Fatal(err, "connect to db")
	}

	server, err := server.NewServer(db, session, cfg)
	if err != nil {
		log.Fatal(err, "create server")
	}

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

		<-stop

		nctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		if err := server.Shutdown(nctx); err != nil {
			log.Fatal(err)
		}
	}()

	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
