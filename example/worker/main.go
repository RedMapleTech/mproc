package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redmapletech/mproc"
	"github.com/redmapletech/mproc/example/common"
)

type WorkerExample struct {
	counter int
}

func main() {
	w := WorkerExample{}

	if err := mproc.RunWorker(&w); err != nil {
		log.Fatalln(err)
	}
}

func (w *WorkerExample) Init(ctx context.Context) error {
	log.Println("Init for 1 second...")
	err := common.SleepCtx(ctx, 1*time.Second)
	log.Println("Init complete")
	return err
}

func (w *WorkerExample) Run(ctx context.Context) error {
	log.Println("Running for 5 seconds...")
	err := common.SleepCtx(ctx, 5*time.Second)
	log.Println("Run complete")
	w.counter++
	if err == nil && w.counter == 5 {
		return context.Canceled
	}
	return err
}

func (w *WorkerExample) Cleanup(ctx context.Context) error {
	log.Println("Cleanup for 1 second...")
	err := common.SleepCtx(ctx, 1*time.Second)
	log.Println("Cleanup complete")
	return err
}

func (w *WorkerExample) OnSignal(signal os.Signal)        { log.Printf("Caught %s signal\n", signal) }
func (w *WorkerExample) GetInitTimeout() time.Duration    { return 6 * time.Second }
func (w *WorkerExample) GetRunTimeout() time.Duration     { return 6 * time.Second }
func (w *WorkerExample) GetCleanupTimeout() time.Duration { return 6 * time.Second }
