package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/erry-az/go-graceful"
	"github.com/rs/zerolog/log"
)

func main() {
	watcher := graceful.New()

	httpServer := &http.Server{Addr: ":8070"}

	watcher.RegisterProcess(func() error {
		log.Info().Msg("starting http server on :8070")

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("got error from http server: %s", err)

			return err
		}

		return nil
	})

	watcher.RegisterShutdownProcessWithTag(func(ctx context.Context) error {
		log.Info().Msg("stopping http server on :8070")
		return httpServer.Shutdown(ctx)
	}, "http-server")

	watcher.RegisterShutdownProcess(func(ctx context.Context) error {
		time.Sleep(20 * time.Second)
		return errors.New("err 2")
	})

	watcher.SetCancelOnError(true)
	watcher.SetMaxShutdownTime(1 * time.Second)

	if err := watcher.Wait(); err != nil {
		log.Error().Err(err).Msg("failed while gracefully shutdown")
	}
}
