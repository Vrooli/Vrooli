package orchestration

import (
	"context"
	"testing"
	"time"
)

func TestFrictionPublishSchedulerSweepsOnStartAndStops(t *testing.T) {
	scheduler := NewFrictionPublishScheduler(&Orchestrator{}, time.Hour)
	scheduler.Start(context.Background())
	defer scheduler.Stop()
	deadline := time.After(2 * time.Second)
	for scheduler.Sweeps() == 0 {
		select {
		case <-deadline:
			t.Fatal("scheduler did not sweep at startup")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if !scheduler.LastPublishAt().IsZero() {
		t.Fatal("empty publisher should not report a publish")
	}
	scheduler.Start(context.Background())
}

func TestFrictionPublishSchedulerStartIsSafeForNil(t *testing.T) {
	var scheduler *FrictionPublishScheduler
	scheduler.Start(context.Background())
	scheduler.Stop()
}
