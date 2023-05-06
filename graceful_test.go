package graceful

import (
	"context"
	"errors"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGraceful_Run(t *testing.T) {
	graceful := New()

	graceful.SetMaxShutdownProcess(0)
	graceful.SetMaxShutdownTime(0)
	graceful.RegisterProcess(nil)
	graceful.RegisterProcessWithContext(nil)
	graceful.RegisterShutdownProcess(nil)

	graceful.SetMaxShutdownTime(1 * time.Second)
	graceful.SetMaxShutdownProcess(1)
	graceful.SetCancelOnError(false)

	mx := &sync.Mutex{}
	procs := make([]bool, 0)

	graceful.RegisterProcess(func() error {
		mx.Lock()
		defer mx.Unlock()
		procs = append(procs, true)
		return nil
	})

	graceful.RegisterProcessWithContext(func(ctx context.Context) error {
		mx.Lock()
		defer mx.Unlock()
		procs = append(procs, true)
		return nil
	})

	graceful.RegisterShutdownProcessWithTag(func(ctx context.Context) error {
		mx.Lock()
		defer mx.Unlock()
		procs = append(procs, true)
		time.Sleep(1 * time.Second)
		return nil
	}, "test app fiber")

	graceful.RegisterShutdownProcess(func(ctx context.Context) error {
		time.Sleep(1 * time.Second)
		mx.Lock()
		defer mx.Unlock()
		procs = append(procs, true)
		return errors.New("err")
	})

	go func() {
		sendSignal(syscall.SIGTERM)
	}()

	err := graceful.Wait()

	assert.Nil(t, err)
	assert.Len(t, procs, 4)
}

func TestGraceful_EmptyShutdown(t *testing.T) {
	graceful := NewWithContext(context.Background())

	graceful.RegisterProcess(func() error {
		return nil
	})

	go func() {
		sendSignal(syscall.SIGTERM)
	}()

	err := graceful.Wait()

	assert.Nil(t, err)
}

func TestGraceful_CancelOnError(t *testing.T) {
	graceful := New()
	graceful.SetCancelOnError(true)

	graceful.RegisterProcess(func() error {
		return nil
	})

	graceful.RegisterShutdownProcess(func(ctx context.Context) error {
		time.Sleep(1 * time.Second)
		return errors.New("err")
	})

	go func() {
		sendSignal(syscall.SIGTERM)
	}()

	err := graceful.Wait()

	assert.NotNil(t, err)
}

func sendSignal(sig os.Signal) {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		panic(err)
	}

	time.Sleep(100 * time.Millisecond)
	_ = p.Signal(sig)
	time.Sleep(10 * time.Millisecond) // give signal some time to propagate
}
