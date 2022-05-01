package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/mazzegi/bongo/server"
	"github.com/mazzegi/log"
)

func main() {
	bind := flag.String("bind", "127.0.0.1:8080", "bind address")
	site := flag.String("site", "../../testsite/", "site to serve")
	flag.Parse()

	s, err := server.New(*bind, *site)
	if err != nil {
		log.Errorf("new-server bind=%q, site=%q: %v", *bind, *site, err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()
	s.RunCtx(ctx)

	log.Infof("done")
}
