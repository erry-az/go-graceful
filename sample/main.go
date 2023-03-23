package main

import (
	"context"
	"log"
	"net/http"

	"github.com/erry-az/go-graceful"
)

func main() {
	watcher := graceful.New()

	httpServer := &http.Server{Addr: ":8070"}

	watcher.RegisterProcess(func() error {
		log.Println("starting http server on :8070")

		if err := httpServer.ListenAndServe();
			err != nil && err != http.ErrServerClosed {
			log.Printf("got error from http server: %s", err)

			return err
		}

		return nil
	})

	watcher.RegisterShutdownProcessWithTag(func(ctx context.Context) error {
		log.Println("stopping http server on :8070")

		return httpServer.Shutdown(ctx)
	}, "http-server")

	if err := watcher.Wait(); err != nil {
		log.Printf("failed while gracefully shutdown on: %s", err)
	}
}
