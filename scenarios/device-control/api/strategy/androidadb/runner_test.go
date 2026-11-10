package androidadb

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBoundedCommandRejectsOversizedOutputAndCancellation(t *testing.T) {
	if os.Getenv("DEVICE_CONTROL_BOUNDED_HELPER") == "output" {
		_, _ = os.Stdout.Write(bytes.Repeat([]byte("x"), 1000))
		return
	}
	if os.Getenv("DEVICE_CONTROL_BOUNDED_HELPER") == "block" {
		select {}
	}
	t.Setenv("DEVICE_CONTROL_BOUNDED_HELPER", "output")
	_, err := (commandRunner{}).RunBounded(context.Background(), os.Args[0], 100, "-test.run=^TestBoundedCommandRejectsOversizedOutputAndCancellation$")
	require.ErrorContains(t, err, "exceeded configured limit")
	t.Setenv("DEVICE_CONTROL_BOUNDED_HELPER", "block")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = (commandRunner{}).RunBounded(ctx, os.Args[0], 100, "-test.run=^TestBoundedCommandRejectsOversizedOutputAndCancellation$")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestCommandRunnerRunReportsContextCancellation(t *testing.T) {
	if os.Getenv("DEVICE_CONTROL_COMMAND_RUNNER_BLOCK") == "1" {
		select {}
	}

	t.Setenv("DEVICE_CONTROL_COMMAND_RUNNER_BLOCK", "1")
	tests := []struct {
		name    string
		context func() (context.Context, func(), error)
	}{
		{
			name: "deadline",
			context: func() (context.Context, func(), error) {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				return ctx, cancel, context.DeadlineExceeded
			},
		},
		{
			name: "cancel",
			context: func() (context.Context, func(), error) {
				ctx, cancel := context.WithCancel(context.Background())
				timer := time.AfterFunc(100*time.Millisecond, cancel)
				return ctx, func() { timer.Stop(); cancel() }, context.Canceled
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel, want := tt.context()
			defer cancel()
			_, err := (commandRunner{}).Run(ctx, os.Args[0], "-test.run=^TestCommandRunnerRunReportsContextCancellation$")
			require.ErrorIs(t, err, want)
		})
	}
}
