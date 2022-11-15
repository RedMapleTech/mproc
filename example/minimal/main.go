package main

import (
	"context"
	"log"
	"time"

	"github.com/redmapletech/mproc"
	"github.com/redmapletech/mproc/example/common"
)

type MinimalExample struct{}

func main() {
	w := MinimalExample{}
	if err := mproc.Run(&w); err != nil {
		log.Fatalln(err)
	}
}

func (w *MinimalExample) Run(ctx context.Context) error {
	log.Println("Running...")
	err := common.SleepCtx(ctx, 5*time.Second)
	log.Println("Run complete")
	return err
}
