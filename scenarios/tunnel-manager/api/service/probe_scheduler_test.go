package service

import (
	"context"
	"testing"
	"time"

	"tunnel-manager/domain"
)

// [REQ:PROBE-005] Periodic probe scheduling - starts and runs at interval
func TestProbeSchedulerRunsAtInterval(t *testing.T) {
	runner := &mockProbeRunner{
		runAllFn: func(ctx context.Context) ([]domain.ProbeResult, error) {
			return []domain.ProbeResult{
				{RouteID: 1, Subdomain: "sched-test", ProbeType: "internal", Status: "up"},
				{RouteID: 1, Subdomain: "sched-test", ProbeType: "external", Status: "up"},
			}, nil
		},
	}

	scheduler := NewProbeScheduler(runner, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler.Start(ctx)
	defer scheduler.Stop()

	if !scheduler.IsRunning() {
		t.Error("scheduler should be running after Start()")
	}

	// Wait for at least 2 cycles
	time.Sleep(350 * time.Millisecond)

	lastRun := scheduler.LastRun()
	if lastRun.IsZero() {
		t.Error("expected lastRun to be non-zero after running")
	}

	if scheduler.LastError() != nil {
		t.Errorf("unexpected scheduler error: %v", scheduler.LastError())
	}
}

// [REQ:PROBE-005] Scheduler stops cleanly
func TestProbeSchedulerStops(t *testing.T) {
	runner := &mockProbeRunner{
		runAllFn: func(ctx context.Context) ([]domain.ProbeResult, error) {
			return []domain.ProbeResult{}, nil
		},
	}

	scheduler := NewProbeScheduler(runner, 50*time.Millisecond)

	ctx := context.Background()
	scheduler.Start(ctx)

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	scheduler.Stop()

	if scheduler.IsRunning() {
		t.Error("scheduler should not be running after Stop()")
	}

	// Allow any in-flight runOnce to complete before taking the snapshot
	time.Sleep(50 * time.Millisecond)

	// Record calls before waiting
	callsBefore := runner.runAllCallCount()
	time.Sleep(150 * time.Millisecond)
	callsAfter := runner.runAllCallCount()

	if callsAfter > callsBefore {
		t.Error("probes continued running after Stop()")
	}
}

// [REQ:PROBE-005] Scheduler is idempotent (double start is no-op)
func TestProbeSchedulerDoubleStart(t *testing.T) {
	runner := &mockProbeRunner{
		runAllFn: func(ctx context.Context) ([]domain.ProbeResult, error) {
			return []domain.ProbeResult{}, nil
		},
	}

	scheduler := NewProbeScheduler(runner, 1*time.Second)

	ctx := context.Background()
	scheduler.Start(ctx)
	scheduler.Start(ctx) // second call should be no-op

	defer scheduler.Stop()

	if !scheduler.IsRunning() {
		t.Error("scheduler should be running")
	}
}
