package common

import (
	"context"
	"log"
	"time"
)

func SleepCtx(ctx context.Context, dur time.Duration) error {
	select {
	case <-ctx.Done():
		log.Println("Sleep interrupted")
	case <-time.After(dur):
		log.Printf("Slept for %s\n", dur)
	}
	return ctx.Err()
}
