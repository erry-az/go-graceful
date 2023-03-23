# Go Graceful

Graceful is a Go package that helps to handle graceful shutdowns for applications. It provides a simple API to register shutdown functions that will be called when the application receives an operating system signal.

## Installation

To use the graceful package, you first need to install Go and set up a Go development environment. Once you have Go installed, you can install the package using the following command:

```shell
go get github.com/erry-az/go-graceful
```
## Example

This example is how we can implement graceful to watch http server serve and call http server shutdown when got OS Signal.
```go
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
```

## Usage
To use Graceful, first create a new instance of `Graceful`:
```go
g := graceful.New()
```
By default, `Graceful` will listen for the following OS signals:

- `os.Interrupt`
- `syscall.SIGINT`
- `syscall.SIGTERM`
- `syscall.SIGHUP`

To specify custom signals, you can use `NewContext`:
```go
var (
    ctx     = context.Background()
    signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}	
)

g := graceful.NewContext(ctx, signals...)
```

### RegisterProcess
`RegisterProcess` is used to register a function to run in the background during the application's runtime.
```go
g := graceful.New()

g.RegisterProcess(func() error {
    // do something in the background
})
```
### RegisterShutdownProcess
`RegisterShutdownProcess` is used to register a function to be called when the application receives a shutdown signal.
```go
g := graceful.New()

g.RegisterShutdownProcess(func(ctx context.Context) error {
    // do something during shutdown
})
```
### RegisterShutdownProcessWithTag
`RegisterShutdownProcessWithTag`is same like register shutdown process but is must define shutdown process tag to make it easier to identify in the logs.
```go
g := graceful.New()

g.RegisterShutdownProcessWithTag(func(ctx context.Context) error {
    // do something during shutdown
}, "shutdown process tag")
```
### Wait
Wait is used to start the application and wait for a shutdown signal. When a signal is received, the registered shutdown processes will be executed.

```go
g := graceful.New()

// Register processes and shutdown processes

if err := g.Wait(); err != nil {
    log.Fatal().Err(err).Msg("Application exited with an error")
}
```


## Options

Graceful provides several options to configure the behavior of the shutdown process:

### SetCancelOnError
`SetCancelOnError` is used to specify whether the application should be canceled immediately upon encountering an error during shutdown. The default value is `false`.
```go
g := graceful.New()
g.SetCancelOnError(true)

```
### SetMaxShutdownTime
`SetMaxShutdownTime` is used to set the maximum amount of time the shutdown process can take. If the shutdown process takes longer than the specified duration, the application will exit forcefully. The default value is 10 seconds.
```go
g := graceful.New()
g.SetMaxShutdownTime(30 * time.Second)
```
### SetMaxShutdownProcess
`SetMaxShutdownProcess` is used to set the maximum number of shutdown processes that can be executed concurrently. The default value is 5.
```go
g := graceful.New()
g.SetMaxShutdownProcess(10)
```