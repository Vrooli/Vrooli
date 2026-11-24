package agent_manager

import (
	"context"
	"errors"
	"io"
	"log"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryProfileInitializationRecoversAfterTemporaryFailure(t *testing.T) {
	status := &ProfileStatus{}
	status.Set(errors.New("agent-manager unavailable at startup"))
	var calls atomic.Int32
	initialize := func(context.Context) error {
		if calls.Add(1) < 3 {
			return errors.New("temporary refusal")
		}
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		retryProfileInitialization(ctx, initialize, status, log.New(io.Discard, "", 0), 5*time.Millisecond, 50*time.Millisecond)
		close(done)
	}()
	deadline := time.After(time.Second)
	for {
		if ready, _ := status.ReadyStatus(); ready {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("profile did not recover after retries; calls=%d", calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("retry loop did not stop after successful reconciliation")
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("initialize calls = %d, want 3", got)
	}
}

func TestRetryProfileInitializationStopsWithLifecycle(t *testing.T) {
	status := &ProfileStatus{}
	status.Set(errors.New("initial failure"))
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		retryProfileInitialization(ctx, func(context.Context) error {
			calls.Add(1)
			return errors.New("still unavailable")
		}, status, log.New(io.Discard, "", 0), time.Minute, time.Second)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("retry loop did not stop when lifecycle ended")
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("initialize calls after cancellation = %d, want 0", got)
	}
	if ready, message := status.ReadyStatus(); ready || message == "" {
		t.Fatalf("status after canceled retry = ready:%v message:%q", ready, message)
	}
}
