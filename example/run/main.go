package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redmapletech/mproc"
	"github.com/redmapletech/mproc/example/common"
)

type RunExample struct{}

func main() {
	w := RunExample{}
	if err := mproc.Run(&w); err != nil {
		log.Fatalln(err)
	}
}

func (w *RunExample) Init(ctx context.Context) error {
	log.Println("Init for 1 second...")
	err := common.SleepCtx(ctx, 1*time.Second)
	log.Println("Init complete")
	return err
}

func (w *RunExample) Run(ctx context.Context) error {
	log.Println("Running for 5 seconds...")
	err := common.SleepCtx(ctx, 5*time.Second)
	log.Println("Run complete")
	return err
}

func (w *RunExample) Cleanup(ctx context.Context) error {
	log.Println("Cleanup for 1 second...")
	err := common.SleepCtx(ctx, 1*time.Second)
	log.Println("Cleanup complete")
	return err
}

func (w *RunExample) OnSignal(signal os.Signal)        { log.Printf("Caught %s signal\n", signal) }
func (w *RunExample) GetInitTimeout() time.Duration    { return 2 * time.Second }
func (w *RunExample) GetRunTimeout() time.Duration     { return 6 * time.Second }
func (w *RunExample) GetCleanupTimeout() time.Duration { return 2 * time.Second }
