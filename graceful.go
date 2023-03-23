package graceful

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

const (
	// defaultMaxShutdownTime default value for max shutdown time.
	defaultMaxShutdownTime = 10 * time.Second
	// defaultMaxShutdownProcess default value for max shutdown process.
	defaultMaxShutdownProcess = 5
)

// defaultSignals default os signal that will be handled.
var defaultSignals = []os.Signal{os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}

// Graceful struct to hold the provided options and dependencies
type Graceful struct {
	groupCtx, signalCtx context.Context
	signalCancel        context.CancelFunc
	group               *errgroup.Group
	shutdownProcess     []func(ctx context.Context) error
	shutdownTags        []string
	maxShutdownTime     time.Duration
	maxShutdownProcess  int
	cancelOnError       bool
}

// New initiate graceful using context background.
func New() *Graceful {
	return NewContext(
		context.Background(),
		defaultSignals...)
}

// NewContext initiate graceful with context param.
// create signal waiting from os signal that will be triggered when some signal is called.
func NewContext(ctx context.Context, signals ...os.Signal) *Graceful {
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
		shutdownProcess:    make([]func(ctx context.Context) error, 0),
		shutdownTags:       make([]string, 0),
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

// RegisterShutdownProcess register shutdown process that will be called when got some os signal.
func (g *Graceful) RegisterShutdownProcess(process func(context.Context) error) {
	g.RegisterShutdownProcessWithTag(process, "")
}

// RegisterShutdownProcessWithTag register shutdown process using tag.
func (g *Graceful) RegisterShutdownProcessWithTag(process func(context.Context) error, tag string) {
	if process == nil {
		return
	}

	g.shutdownTags = append(g.shutdownTags, tag)
	g.shutdownProcess = append(g.shutdownProcess, process)
}

// createTag creating shutdown tag.
func (g *Graceful) createTag(i int) string {
	tag := g.shutdownTags[i]
	if tag == "" {
		tag = fmt.Sprintf("process %d", i)
	}

	return tag
}

// shutdown handle all shutdown process with concurrency.
func (g *Graceful) shutdown() error {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), g.maxShutdownTime)
	defer shutdownCancel()

	shutdownGroup, shutdownGroupCtx := errgroup.WithContext(shutdownCtx)
	shutdownGroup.SetLimit(g.maxShutdownProcess)

	for i, process := range g.shutdownProcess {
		var (
			iCopy       = i
			processCopy = process
		)

		shutdownGroup.Go(func() error {
			tag := g.createTag(iCopy)

			err := processCopy(shutdownGroupCtx)
			if err != nil {
				log.Error().Str("tag", tag).Err(err).Send()
			} else {
				log.Info().Str("tag", tag).Msg("shutdown success")
			}

			if g.cancelOnError {
				return err
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

		if len(g.shutdownProcess) > 0 {
			return g.shutdown()
		}

		return nil
	})

	return g.group.Wait()
}
