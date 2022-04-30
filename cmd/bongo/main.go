package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mazzegi/bongo/server"
	"github.com/mazzegi/log"
)

func main() {
	s, err := server.New("localhost:8080", "../../testsite/")
	if err != nil {
		panic(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()
	s.RunCtx(ctx)

	log.Infof("done")
}
