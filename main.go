package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	r "github.com/aarrwnh/stg-backup-reader/reader"
)

var path = flag.String("p", ".", "path")

func main() {
	flag.Parse()

	data, count, err := r.LoadFiles(path)
	if err != nil {
		return
	}
	log.Printf("\033[30mloaded %d tabs\033[0m", count)

	ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(interrupt)

	app := r.NewApp(data, 10, cancel)

	go r.StartWebsocket(&app)
	go app.Start()

	select {
	case <-ctx.Done():
		log.Println("Exiting program")
	case sig := <-interrupt:
		log.Printf("Caught signal: %v\n", sig)
	}

	time.Sleep(time.Second * 1)
}
