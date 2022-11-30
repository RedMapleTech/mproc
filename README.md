# Golang Managed Process Wrapper (mproc)

[![Go Reference](https://pkg.go.dev/badge/github.com/redmapletech/mproc.svg)](https://pkg.go.dev/github.com/redmapletech/mproc)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/redmapletech/mproc)
[![Go Report Card](https://goreportcard.com/badge/github.com/redmapletech/mproc)](https://goreportcard.com/report/github.com/redmapletech/mproc)

This is a simple, dependency-free process wrapper to handle OS signals during process init, run (including looped), and cleanup stages.

Only one process can be run at a time using this wrapper.
It cannot be used inside parallel goroutines, instead the parallel goroutines should be executed inside the Run handler with the provided context.

## Prerequisites

```bash
go get github.com/redmapletech/mproc
```

## Usage

In order to use this package, you will need a struct that implements ManagedProcess as defined in [mproc.go](./mproc.go).
For basic processes this will involve moving any code in main() to the Run() function of this struct, and replacing your entry point with the following:

```go
type MyProcess struct{}

func main() {
	p := &MyProcess{} // Implements ManagedProcess
	if err := mproc.Run(p); err != nil {
		log.Fatalln(err)
	}
}

func (p *MyProcess) Run(ctx context.Context) error {
  // ...
}
```

See [examples](./example) for minimal, full, and looped versions.

### With http.Server

Golang's http.Server has a blocking ListenAndServe method, but can be gracefully shutdown within a context using its Shutdown method.

This works by pausing the Run method until the context is cancelled, whilst running ListenAndServe in its own goroutine.

```go
func (p *MyProcess) Run(ctx context.Context) error {
	// Run ListenAndServe in a goroutine (p.server is *http.Server)
	go func() {
		if err := p.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalln(err)
		}
	}()

	// Block on Run context
	<-ctx.Done()
	return ctx.Err()
}

func (p *MyProcess) Cleanup(ctx context.Context) error {
	// This will cause ListenAndServe to return
	return s.server.Shutdown(ctx)
}
```

## Timeout

The context provided to Run can also enforce a deadline if the ManagedProcessWithRunTimeout interface is implemented.
This requires that the GetRunTimeout function is present, and returns a duration.

If a managed process stops due to a timeout, the returned error will be context.DeadlineExceeded.

## Run Stage

This is the standard place for the bulk of the process work to be done.
A context is provided to cancel when a signal has been received, or the timeout has expired.

## Init Stage

This can be used to do any initialisation logic, again with a timeout.
Unlike the Run function, the timeout is not optional and the full ManagedProcessWithInit interface must be implemented to work correctly.

This is a good place to parse args, load config data (which can then influence the other timeouts), and do any connection setup which can also be stored in the struct.

## Cleanup Stage

Functionally identical to Init, however it runs after the Run function has completed if no error was encountered.
The only exception to this is context cancelled, which will happen if a signal is caught in which case Cleanup is still run for graceful termination.

This is a good place to close any connection pools which were stored in the process struct during Init.

## Looping Processes

The Run function can be looped by calling RunWorker() instead.
In this mode, the ManagedProcessWithRunTimeout interface is not optional.

The only difference here is that the Run() function context will only cancel due to the timeout, not in response to a signal.
Therefore the behaviour will be such that on a signal, the current loop will complete, and then Cleanup will be called if implemented.

Should the code inside the loop wish to signal completion, returning context.Cancelled instead of nil from Run will trigger Cleanup and exit.
