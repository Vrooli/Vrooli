package destinations

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPeriodicWorkerStopJoinsReconcileAndRejectsOverlappingRestart(t *testing.T) {
	worker := &periodicWorker{}
	firstStarted := make(chan struct{})
	firstCancelled := make(chan struct{})
	releaseFirst := make(chan struct{})
	restarted := make(chan struct{}, 1)
	var calls atomic.Int32
	reconcile := func(ctx context.Context) {
		switch calls.Add(1) {
		case 1:
			close(firstStarted)
			<-ctx.Done()
			close(firstCancelled)
			<-releaseFirst
		default:
			restarted <- struct{}{}
			<-ctx.Done()
		}
	}
	t.Cleanup(func() { _ = worker.stop(context.Background(), nil) })

	if err := worker.start(time.Hour, nil, reconcile); err != nil {
		t.Fatal(err)
	}
	waitWorkerSignal(t, firstStarted, "first reconcile start")
	stopped := make(chan error, 1)
	go func() { stopped <- worker.stop(context.Background(), nil) }()
	waitWorkerSignal(t, firstCancelled, "first reconcile cancellation")

	if err := worker.start(time.Hour, nil, reconcile); !errors.Is(err, errWorkerStopping) {
		close(releaseFirst)
		<-stopped
		t.Fatalf("Start during stop = %v, want %v", err, errWorkerStopping)
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := worker.stop(stopCtx, nil); !errors.Is(err, context.DeadlineExceeded) {
		close(releaseFirst)
		<-stopped
		t.Fatalf("second Stop during blocked reconcile = %v, want deadline while joining", err)
	}

	close(releaseFirst)
	if err := waitWorkerResult(t, stopped); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := worker.start(time.Hour, nil, reconcile); err != nil {
		t.Fatalf("restart after joined stop: %v", err)
	}
	waitWorkerSignal(t, restarted, "restarted reconcile")
	if err := worker.stop(context.Background(), nil); err != nil {
		t.Fatalf("stop restarted worker: %v", err)
	}
}

func TestPeriodicWorkerSerializesStopThroughPostStopDrain(t *testing.T) {
	worker := &periodicWorker{}
	drainStarted := make(chan struct{})
	releaseDrain := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseDrain) }) }
	var drains atomic.Int32
	afterStop := func(context.Context) error {
		if drains.Add(1) == 1 {
			close(drainStarted)
			<-releaseDrain
		}
		return nil
	}
	t.Cleanup(func() {
		release()
		_ = worker.stop(context.Background(), nil)
	})
	if err := worker.start(time.Hour, nil, func(context.Context) {}); err != nil {
		t.Fatal(err)
	}
	firstStop := make(chan error, 1)
	go func() { firstStop <- worker.stop(context.Background(), afterStop) }()
	waitWorkerSignal(t, drainStarted, "first post-stop drain")

	secondCtx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := worker.stop(secondCtx, afterStop); !errors.Is(err, context.DeadlineExceeded) {
		release()
		<-firstStop
		t.Fatalf("second Stop during drain = %v, want deadline while joining first Stop", err)
	}
	if got := drains.Load(); got != 1 {
		t.Fatalf("post-stop drain ran %d times while first Stop was active", got)
	}
	if err := worker.start(time.Hour, nil, func(context.Context) {}); !errors.Is(err, errWorkerStopping) {
		release()
		<-firstStop
		t.Fatalf("Start during post-stop drain = %v, want %v", err, errWorkerStopping)
	}

	release()
	if err := waitWorkerResult(t, firstStop); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := worker.start(time.Hour, nil, func(context.Context) {}); err != nil {
		t.Fatalf("Start after drain completes: %v", err)
	}
	if err := worker.stop(context.Background(), nil); err != nil {
		t.Fatalf("stop after drain completes: %v", err)
	}
}

func TestPeriodicWorkerStopFailureKeepsRestartClosedUntilStopRetry(t *testing.T) {
	worker := &periodicWorker{}
	var drainCalls atomic.Int32
	reconciled := make(chan struct{}, 1)
	reconcile := func(context.Context) {
		select {
		case reconciled <- struct{}{}:
		default:
		}
	}
	afterStop := func(context.Context) error {
		if drainCalls.Add(1) == 1 {
			return errors.New("migration drain is still pending")
		}
		return nil
	}
	t.Cleanup(func() { _ = worker.stop(context.Background(), nil) })
	if err := worker.start(time.Hour, nil, reconcile); err != nil {
		t.Fatal(err)
	}
	waitWorkerSignal(t, reconciled, "reconcile before stop")
	if err := worker.stop(context.Background(), afterStop); err == nil || err.Error() != "migration drain is still pending" {
		t.Fatalf("first Stop error = %v, want pending drain error", err)
	}
	if err := worker.start(time.Hour, nil, reconcile); !errors.Is(err, errWorkerStopping) {
		t.Fatalf("Start after incomplete Stop = %v, want %v", err, errWorkerStopping)
	}
	if err := worker.stop(context.Background(), afterStop); err != nil {
		t.Fatalf("retry Stop after drain failure: %v", err)
	}
	if got := drainCalls.Load(); got != 2 {
		t.Fatalf("post-stop drain attempts = %d, want first attempt and one retry", got)
	}
	if err := worker.start(time.Hour, nil, reconcile); err != nil {
		t.Fatalf("Start after completed retry: %v", err)
	}
	if err := worker.stop(context.Background(), nil); err != nil {
		t.Fatalf("stop restarted worker: %v", err)
	}
}

func TestBackupFailureWorkerStopWaitsForFreshMigrationDrain(t *testing.T) {
	flow := &BackupFailureFlow{}
	flow.migrationWG.Add(1)
	var releaseOnce sync.Once
	releaseMigration := func() { releaseOnce.Do(flow.migrationWG.Done) }
	worker := &BackupFailureWorker{flow: flow, interval: time.Hour, now: time.Now}
	t.Cleanup(func() {
		releaseMigration()
		_ = worker.Stop(context.Background())
	})
	if err := worker.Start(); err != nil {
		t.Fatal(err)
	}
	worker.lifecycle.mu.Lock()
	runDone := worker.lifecycle.done
	worker.lifecycle.mu.Unlock()

	stopCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	stopped := make(chan error, 1)
	go func() { stopped <- worker.Stop(stopCtx) }()
	waitWorkerSignal(t, runDone, "backup failure worker reconciliation stop")
	select {
	case err := <-stopped:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("BackupFailureWorker.Stop before migration drain = %v, want deadline", err)
		}
	case <-stopCtx.Done():
		if err := waitWorkerResult(t, stopped); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("BackupFailureWorker.Stop deadline result = %v", err)
		}
	}
	if err := worker.Start(); !errors.Is(err, errWorkerStopping) {
		t.Fatalf("BackupFailureWorker.Start with migration drain pending = %v, want %v", err, errWorkerStopping)
	}

	releaseMigration()
	if err := worker.Stop(context.Background()); err != nil {
		t.Fatalf("BackupFailureWorker.Stop after migration drain: %v", err)
	}
}

func waitWorkerSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func waitWorkerResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker stop")
		return nil
	}
}
