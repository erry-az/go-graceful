package graceful

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// Graceful struct to hold the provided options and dependencies
type Graceful struct {
	groupCtx, signalCtx context.Context
	signalCancel        context.CancelFunc
	group               *errgroup.Group
	shutdowns           []shutdown
	maxShutdownTime     time.Duration
	maxShutdownProcess  int
	cancelOnError       bool
	mutex               sync.Mutex
}

// New initiate graceful using context background.
func New(signals ...os.Signal) *Graceful {
	return NewWithContext(
		context.Background(),
		signals...)
}

// NewWithContext initiate graceful with context param.
// create signal waiting from os signal that will be triggered when some signal is called.
func NewWithContext(ctx context.Context, signals ...os.Signal) *Graceful {
	if len(signals) == 0 {
		signals = defaultSignals
	}

	var (
		signalCtx, signalCancel = signal.NotifyContext(ctx, signals...)
		group, groupCtx         = errgroup.WithContext(signalCtx)
	)

	return &Graceful{
		groupCtx:           groupCtx,
		signalCtx:          signalCtx,
		signalCancel:       signalCancel,
		group:              group,
		shutdowns:          make([]shutdown, 0),
		maxShutdownTime:    defaultMaxShutdownTime,
		maxShutdownProcess: defaultMaxShutdownProcess,
	}
}

// SetCancelOnError set cancel on error value.
func (g *Graceful) SetCancelOnError(value bool) {
	g.cancelOnError = value
}

// SetMaxShutdownTime set max shutdown time value.
func (g *Graceful) SetMaxShutdownTime(duration time.Duration) {
	if duration < 1 {
		g.maxShutdownTime = defaultMaxShutdownTime

		return
	}

	g.maxShutdownTime = duration
}

// SetMaxShutdownProcess set max shutdown process value.
func (g *Graceful) SetMaxShutdownProcess(max int) {
	if max < 1 {
		g.maxShutdownProcess = defaultMaxShutdownProcess

		return
	}

	g.maxShutdownProcess = max
}

// RegisterProcess register running process to background.
func (g *Graceful) RegisterProcess(process func() error) {
	if process == nil {
		return
	}

	g.group.Go(process)
}

// RegisterProcessWithContext register running process to background with context param.
// context is from signal and
func (g *Graceful) RegisterProcessWithContext(process func(ctx context.Context) error) {
	if process == nil {
		return
	}

	g.group.Go(func() error {
		return process(g.groupCtx)
	})
}

// RegisterShutdownProcess register shutdown process that will be called when got some os signal.
func (g *Graceful) RegisterShutdownProcess(process func(context.Context) error) string {
	return g.RegisterShutdownProcessWithTag(process, "")
}

// RegisterShutdownProcessWithTag register shutdown process using tag.
func (g *Graceful) RegisterShutdownProcessWithTag(process func(context.Context) error, tag string) string {
	if process == nil {
		return ""
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	shutdownProcess, id := newShutdown(tag, process)
	g.shutdowns = append(g.shutdowns, shutdownProcess)

	return id.String()
}

// shutdown handle all shutdown process with concurrency.
func (g *Graceful) shutdown() error {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), g.maxShutdownTime)
	defer shutdownCancel()

	shutdownGroup, shutdownGroupCtx := errgroup.WithContext(shutdownCtx)
	shutdownGroup.SetLimit(g.maxShutdownProcess)

	for _, s := range g.shutdowns {
		shutdownCopy := s

		shutdownGroup.Go(func() error {
			errChan := make(chan error)

			go func() {
				err := shutdownCopy.process(shutdownGroupCtx)
				errChan <- err
			}()

			select {
			case <-shutdownGroupCtx.Done():
				return shutdownGroupCtx.Err()
			case err := <-errChan:
				if err != nil {
					log.Error().Str(shutdownTag, shutdownCopy.tag).Err(err).Send()
				} else {
					log.Info().Str(shutdownTag, shutdownCopy.tag).Msg(shutdownSuccessMessage)
				}

				if g.cancelOnError {
					return err
				}
			}

			return nil
		})
	}

	return shutdownGroup.Wait()
}

// Wait waiting for os signal send and call shutdown process when got some signal.
func (g *Graceful) Wait() error {
	defer g.signalCancel()

	g.group.Go(func() error {
		<-g.groupCtx.Done()

		if len(g.shutdowns) > 0 {
			return g.shutdown()
		}

		return nil
	})

	return g.group.Wait()
}
