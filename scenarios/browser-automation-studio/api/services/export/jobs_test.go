package export

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestExportRunnerOwnsAdmissionCancellationAndDrain(t *testing.T) {
	runner := NewJobRunner(1, time.Minute)
	ctx, cancel := context.WithCancel(context.WithValue(t.Context(), struct{}{}, "routed"))
	defer cancel()
	started := make(chan context.Context, 1)
	ended := make(chan struct{})
	id := uuid.New()
	if err := runner.Start(ctx, id, func(context.Context) error { return nil }, func(job context.Context) { started <- job; <-job.Done(); close(ended) }); err != nil {
		t.Fatal(err)
	}
	job := <-started
	cancel()
	if job.Err() != nil {
		t.Fatal("request cancellation owned export")
	}
	if job.Value(struct{}{}) != "routed" {
		t.Fatal("lost routing values")
	}
	admitted := false
	if err := runner.Start(t.Context(), uuid.New(), func(context.Context) error { admitted = true; return nil }, func(context.Context) {}); !errors.Is(err, ErrExportCapacity) || admitted {
		t.Fatal("over-admitted", err)
	}
	if !runner.Cancel(id) {
		t.Fatal("missing owned job")
	}
	select {
	case <-ended:
	case <-time.After(time.Second):
		t.Fatal("cancel did not reach job")
	}
	drain, cancelDrain := context.WithTimeout(t.Context(), time.Second)
	defer cancelDrain()
	if err := runner.Drain(drain); err != nil {
		t.Fatal(err)
	}
	if err := runner.Start(t.Context(), uuid.New(), func(context.Context) error { return nil }, func(context.Context) {}); !errors.Is(err, ErrExportCapacity) {
		t.Fatal("accepted while draining", err)
	}
	if err := runner.Drain(drain); err != nil {
		t.Fatal("drain not idempotent", err)
	}
}
func TestExportRunnerAdmissionFailureReleasesCapacity(t *testing.T) {
	runner := NewJobRunner(1, time.Minute)
	defer runner.Drain(t.Context())
	err := runner.Start(t.Context(), uuid.New(), func(context.Context) error { return errors.New("storage unavailable") }, func(context.Context) { t.Error("ran unpersisted job") })
	if err == nil {
		t.Fatal("lost admission failure")
	}
	if err := runner.Start(t.Context(), uuid.New(), func(context.Context) error { return nil }, func(context.Context) {}); err != nil {
		t.Fatal("leaked capacity", err)
	}
}

func TestExportRunnerDrainInterruptsAdmissionWithoutLocking(t *testing.T) {
	runner := NewJobRunner(1, time.Minute)
	entered := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- runner.Start(t.Context(), uuid.New(), func(ctx context.Context) error { close(entered); <-ctx.Done(); return ctx.Err() }, func(context.Context) { t.Error("ran failed admission") })
	}()
	<-entered
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := runner.Drain(ctx); err != nil {
		t.Fatal("admission blocked drain", err)
	}
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestExportRunnerDeadlineOwnsActualJobContext(t *testing.T) {
	runner := NewJobRunner(1, 5*time.Millisecond)
	ended := make(chan error, 1)
	if err := runner.Start(t.Context(), uuid.New(), func(context.Context) error { return nil }, func(ctx context.Context) { <-ctx.Done(); ended <- ctx.Err() }); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-ended:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("deadline did not reach actual work")
	}
	if err := runner.Drain(t.Context()); err != nil {
		t.Fatal(err)
	}
}
