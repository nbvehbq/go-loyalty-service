package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nbvehbq/go-loyalty-service/internal/server"
)

func main() {
	cfg, err := server.NewConfig()

	if err != nil {
		log.Fatal(err, "Load config")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var db server.Repository

	server, err := server.NewServer(db, cfg)
	if err != nil {
		log.Fatal(err, "create server")
	}

	go func() {
		defer cancel()
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
		panic(err)
	}
}
