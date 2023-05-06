package graceful

import (
	"os"
	"syscall"
	"time"
)

const (
	// defaultMaxShutdownTime default value for max shutdown time.
	defaultMaxShutdownTime = 10 * time.Second
	// defaultMaxShutdownProcess default value for max shutdown process.
	defaultMaxShutdownProcess = 5
	// shutdownTag add process tag on shutdown process.
	shutdownTag = "graceful-shutdown-tag"
	// shutdownSuccessMessage default message when shutdown success.
	shutdownSuccessMessage = "shutdown success"
)

// defaultSignals default os signal that will be handled.
var defaultSignals = []os.Signal{os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}
