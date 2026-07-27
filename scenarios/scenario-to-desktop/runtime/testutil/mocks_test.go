package testutil

import (
	"context"
	"testing"
	"time"
)

func TestMockClockAndCommandRunnerProvideDeterministicRuntimeControls(t *testing.T) {
	start := time.Date(2026, time.July, 27, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(start)
	clock.Sleep(time.Minute)
	if got := clock.Now(); !got.Equal(start.Add(time.Minute)) {
		t.Fatalf("Now() = %v", got)
	}
	if got := <-clock.After(time.Second); !got.Equal(start.Add(time.Minute + time.Second)) {
		t.Fatalf("After() = %v", got)
	}

	runner := NewMockCommandRunner()
	runner.SetOutput([]byte("ready"))
	if err := runner.Run(context.Background(), "desktop-runtime", []string{"status"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if output, err := runner.Output(context.Background(), "desktop-runtime", "status"); err != nil || string(output) != "ready" {
		t.Fatalf("Output() = %q, %v", output, err)
	}
	if len(runner.Commands()) != 2 {
		t.Fatalf("Commands() = %v, want two entries", runner.Commands())
	}
}
