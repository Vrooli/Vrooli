package main

import (
	"context"
	"io"
	"testing"
)

type commandExecFunc func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error
type commandCaptureFunc func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error)

func stubPhaseCommandExecutor(t *testing.T, fn commandExecFunc) {
	t.Helper()
	prev := phaseCommandExecutor
	phaseCommandExecutor = func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
		return fn(ctx, dir, logWriter, name, args...)
	}
	t.Cleanup(func() {
		phaseCommandExecutor = prev
	})
}

func stubPhaseCommandCapture(t *testing.T, fn commandCaptureFunc) {
	t.Helper()
	prev := phaseCommandCapture
	phaseCommandCapture = func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
		return fn(ctx, dir, logWriter, name, args...)
	}
	t.Cleanup(func() {
		phaseCommandCapture = prev
	})
}
