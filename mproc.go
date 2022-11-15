package mproc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ManagedProcess interface at a minimum ensures OS signals are caught
type ManagedProcess interface {
	Run(ctx context.Context) error
}

// Optional Run stage timeout
type ManagedProcessWithRunTimeout interface {
	GetRunTimeout() time.Duration
}

// Worker equivalent, as RunTimeout is not optional
type ManagedWorkerProcess interface {
	ManagedProcess
	ManagedProcessWithRunTimeout
}

// Optional OnSignal callback
type ManagedProcessWithOnSignal interface {
	OnSignal(signal os.Signal)
}

// Optional Init stage with timeout
type ManagedProcessWithInit interface {
	Init(ctx context.Context) error
	GetInitTimeout() time.Duration
}

// Optional Cleanup stage with timeout
type ManagedProcessWithCleanup interface {
	Cleanup(ctx context.Context) error
	GetCleanupTimeout() time.Duration
}

var (
	// Default signals to intercept
	signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}

	// Signal channel
	quit chan os.Signal = nil

	// Global lock (prevents simultaneous use in goroutines)
	gl = sync.Mutex{}
)

// Run manages single execution of a process
func Run(impl ManagedProcess) error {
	gl.Lock()
	defer gl.Unlock()

	// Main context to receive OS signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go catchSignals(cancel, impl)

	// Run init if configured
	if err := procInit(ctx, impl); err != nil {
		return err
	}

	// Create wrapped context with run timeout
	var runCtx context.Context
	if implWithTimeout, ok := impl.(ManagedProcessWithRunTimeout); ok {
		var cancelRun context.CancelFunc
		runCtx, cancelRun = context.WithTimeout(ctx, implWithTimeout.GetRunTimeout())
		defer cancelRun()
	} else {
		runCtx = ctx
	}

	// Run managed process
	if err := impl.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	// Run cleanup if configured
	if err := procCleanup(impl); err != nil {
		return err
	}
	return nil
}

// RunWorker manages looped execution of a process
func RunWorker(impl ManagedWorkerProcess) error {
	gl.Lock()
	defer gl.Unlock()

	// Main context to receive OS signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go catchSignals(cancel, impl)

	// Run init if configured
	if err := procInit(ctx, impl); err != nil {
		return err
	}

	var loopErr error = nil

LOOP: // Labelled loop to allow break inside select
	for {
		// Create inner loop context so that current loop completes on interrupt
		// (cancel not deferred as it is probably a memory leak in a loop, and is immediately called anyway)
		loopCtx, cancelLoop := context.WithTimeout(context.Background(), impl.GetRunTimeout())

		// Run managed process loop
		loopErr = impl.Run(loopCtx)
		cancelLoop() // Release inner loop context resources

		// Terminate loop if an error is encountered in the loop
		if loopErr != nil {
			break
		}

		// Break on outer context cancel
		select {
		case <-ctx.Done():
			break LOOP
		default: // Continue
		}
	}

	// Any error other than context canceled is fatal
	if loopErr != nil && !errors.Is(loopErr, context.Canceled) {
		return loopErr
	}

	// Run cleanup if configured
	if err := procCleanup(impl); err != nil {
		return err
	}
	return nil
}

// SetSignals allows the monitored signals to be changed before running
func SetSignals(sigs []os.Signal) {
	signals = sigs
}

// Init if implemented
func procInit(ctx context.Context, impl ManagedProcess) error {
	if implWithInit, ok := impl.(ManagedProcessWithInit); ok {
		// Create wrapped context with init timeout
		// First signal during init will cancel init and exit process
		initCtx, cancelInit := context.WithTimeout(ctx, implWithInit.GetInitTimeout())
		defer cancelInit()

		// Run init
		if err := implWithInit.Init(initCtx); err != nil {
			return fmt.Errorf("mproc: failed init - %w", err)
		}
	}
	return nil
}

// Cleanup if implemented
func procCleanup(impl ManagedProcess) error {
	if implWithCleanup, ok := impl.(ManagedProcessWithCleanup); ok {
		// Create fresh context with cleanup timeout
		// First signal during cleanup will be caught and ignored as the process will exit shortly
		// Further signals will have their default behaviour
		ctx, cancel := context.WithTimeout(context.Background(), implWithCleanup.GetCleanupTimeout())
		defer cancel()

		// Run cleanup
		if err := implWithCleanup.Cleanup(ctx); err != nil {
			return fmt.Errorf("mproc: failed cleanup - %w", err)
		}
	}
	return nil
}

// Shared code for watching OS signals, intended to be executed in a goroutine
func catchSignals(cancel context.CancelFunc, impl interface{}) {
	defer cancel()
	quit = make(chan os.Signal, 1)
	signal.Notify(quit, signals...)
	sig := <-quit
	signal.Stop(quit) // Allow user to terminate if stuck

	// Handle optional callback if specified
	if implWithOnSignal, ok := impl.(ManagedProcessWithOnSignal); ok {
		implWithOnSignal.OnSignal(sig)
	}
}
