package lifecycle

import (
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func openWaitTestStore(t *testing.T, home string) *scenarioruntime.SQLiteStore {
	t.Helper()
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("open runtime registry: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func seedRunningStartOperation(t *testing.T, store *scenarioruntime.SQLiteStore, scenarioName string, pid int) scenarioruntime.StartOperation {
	t.Helper()
	op, err := store.BeginStartOperation(context.Background(), scenarioruntime.StartOperation{
		Scenario:     scenarioName,
		Operation:    "start",
		InitiatorPID: &pid,
	})
	if err != nil {
		t.Fatalf("seed start operation: %v", err)
	}
	return op
}

func newWaitTestRunner(t *testing.T, alivePIDs func(int) bool) (*Runner, *fakeAwaitClock, string) {
	t.Helper()
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")
	clock := newFakeAwaitClock()
	runner, err := newRunnerWithDeps(root, home, io.Discard, io.Discard, lifecycleDeps{
		now:          clock.Now,
		sleep:        clock.Sleep,
		isPIDRunning: alivePIDs,
	})
	if err != nil {
		t.Fatalf("newRunnerWithDeps: %v", err)
	}
	return runner, clock, home
}

// TestWaitScenarioNoInFlightReturnsImmediately: no operation record and no
// running instance → immediate not_running verdict, zero sleeps.
func TestWaitScenarioNoInFlightReturnsImmediately(t *testing.T) {
	runner, clock, _ := newWaitTestRunner(t, func(int) bool { return true })
	outcome, err := runner.WaitScenario("alpha", WaitOptions{})
	if err != nil {
		t.Fatalf("WaitScenario: %v", err)
	}
	if outcome.Attached {
		t.Fatal("must not attach with no in-flight operation")
	}
	if outcome.Verdict != WaitVerdictNotRunning {
		t.Fatalf("verdict = %q, want %q", outcome.Verdict, WaitVerdictNotRunning)
	}
	if len(clock.sleeps) != 0 {
		t.Fatalf("expected immediate return, slept %v", clock.sleeps)
	}
}

// TestWaitScenarioAttachesUntilOwnerFinishes: a live in-flight record is
// awaited; when the owner marks it failed, the attacher returns that verdict
// with the owner's error.
func TestWaitScenarioAttachesUntilOwnerFinishes(t *testing.T) {
	runner, clock, home := newWaitTestRunner(t, func(int) bool { return true })
	store := openWaitTestStore(t, home)
	op := seedRunningStartOperation(t, store, "alpha", 99999)

	transitions := 0
	finishAfter := 3
	origSleep := clock.Sleep
	_ = origSleep
	sleeps := 0
	runner.deps.sleep = func(d time.Duration) {
		clock.Sleep(d)
		sleeps++
		if sleeps == finishAfter {
			op.Status = scenarioruntime.StartOperationStatusFailed
			op.Error = "develop exploded"
			now := clock.Now()
			op.FinishedAt = &now
			if _, err := store.UpdateStartOperation(context.Background(), op); err != nil {
				t.Errorf("finish op: %v", err)
			}
		}
	}

	outcome, err := runner.WaitScenario("alpha", WaitOptions{
		OnTransition: func(StartOperationView) { transitions++ },
	})
	if err != nil {
		t.Fatalf("WaitScenario: %v", err)
	}
	if !outcome.Attached {
		t.Fatal("expected attach to in-flight operation")
	}
	if outcome.Verdict != WaitVerdictFailed || outcome.Error != "develop exploded" {
		t.Fatalf("outcome = %+v, want failed/develop exploded", outcome)
	}
	if transitions == 0 {
		t.Fatal("expected at least one transition callback")
	}
}

// TestWaitScenarioTimeoutLeavesOwnerUnaffected: ceiling expiry returns the
// timeout verdict and the owner's record is still running.
func TestWaitScenarioTimeoutLeavesOwnerUnaffected(t *testing.T) {
	runner, _, home := newWaitTestRunner(t, func(int) bool { return true })
	store := openWaitTestStore(t, home)
	seedRunningStartOperation(t, store, "alpha", 99999)

	outcome, err := runner.WaitScenario("alpha", WaitOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("WaitScenario: %v", err)
	}
	if !outcome.TimedOut || outcome.Verdict != WaitVerdictTimeout {
		t.Fatalf("outcome = %+v, want timeout", outcome)
	}
	after, err := store.GetLatestStartOperation(context.Background(), "alpha", "")
	if err != nil {
		t.Fatalf("GetLatestStartOperation: %v", err)
	}
	if after.Status != scenarioruntime.StartOperationStatusRunning {
		t.Fatalf("owner record = %q after wait timeout, want running (unaffected)", after.Status)
	}
}

// TestWaitScenarioReportsAbandonedOwner: a running record with a dead
// initiator resolves immediately as abandoned.
func TestWaitScenarioReportsAbandonedOwner(t *testing.T) {
	runner, clock, home := newWaitTestRunner(t, func(int) bool { return false })
	store := openWaitTestStore(t, home)
	seedRunningStartOperation(t, store, "alpha", 99999)

	outcome, err := runner.WaitScenario("alpha", WaitOptions{})
	if err != nil {
		t.Fatalf("WaitScenario: %v", err)
	}
	if outcome.Verdict != WaitVerdictAbandoned {
		t.Fatalf("verdict = %q, want abandoned", outcome.Verdict)
	}
	if len(clock.sleeps) != 0 {
		t.Fatalf("dead-owner record must resolve immediately, slept %v", clock.sleeps)
	}
}

// TestStartBusyLockWithoutStartRecordStaysBusy: a busy lock whose holder is
// NOT a start (no operation record appears within the grace window) keeps
// ErrScenarioBusy — the true-conflict path (e.g. concurrent stop).
func TestStartBusyLockWithoutStartRecordStaysBusy(t *testing.T) {
	runner, _, _ := newWaitTestRunner(t, func(int) bool { return true })
	origFlock := lockFileFlockFn
	defer func() { lockFileFlockFn = origFlock }()
	lockFileFlockFn = func(fd int, how int) error {
		if how == syscall.LOCK_UN {
			return nil
		}
		return syscall.EWOULDBLOCK
	}

	_, err := runner.Start("alpha", StartOptions{})
	if !errors.Is(err, ErrScenarioBusy) {
		t.Fatalf("error = %v, want ErrScenarioBusy", err)
	}
}

// TestStartAttachesAndReturnsOwnersFailure: a busy lock with a live in-flight
// record attaches; the owner's failure becomes this caller's error instead of
// ErrScenarioBusy.
func TestStartAttachesAndReturnsOwnersFailure(t *testing.T) {
	runner, clock, home := newWaitTestRunner(t, func(int) bool { return true })
	store := openWaitTestStore(t, home)
	op := seedRunningStartOperation(t, store, "alpha", 99999)

	origFlock := lockFileFlockFn
	defer func() { lockFileFlockFn = origFlock }()
	lockFileFlockFn = func(fd int, how int) error {
		if how == syscall.LOCK_UN {
			return nil
		}
		return syscall.EWOULDBLOCK
	}

	sleeps := 0
	runner.deps.sleep = func(d time.Duration) {
		clock.Sleep(d)
		sleeps++
		if sleeps == 5 {
			op.Status = scenarioruntime.StartOperationStatusFailed
			op.Error = "owner failed"
			now := clock.Now()
			op.FinishedAt = &now
			if _, err := store.UpdateStartOperation(context.Background(), op); err != nil {
				t.Errorf("finish op: %v", err)
			}
		}
	}

	_, err := runner.Start("alpha", StartOptions{})
	if err == nil {
		t.Fatal("expected owner's failure to surface")
	}
	if errors.Is(err, ErrScenarioBusy) {
		t.Fatalf("attach must replace ErrScenarioBusy, got %v", err)
	}
	if !strings.Contains(err.Error(), "owner failed") {
		t.Fatalf("error = %v, want owner's failure detail", err)
	}
}

// TestStartTakesOverAbandonedInFlightStart: busy lock + dead-owner record →
// the attacher takes over: it retries the lock and runs the start itself.
func TestStartTakesOverAbandonedInFlightStart(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")
	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.deps.isPIDRunning = nil // real liveness; seeded pid below is dead
	t.Cleanup(func() { _ = runner.Stop("alpha", StopOptions{}) })

	store := openWaitTestStore(t, home)
	seedRunningStartOperation(t, store, "alpha", 999999) // dead pid

	origFlock := lockFileFlockFn
	defer func() { lockFileFlockFn = origFlock }()
	flockAttempts := 0
	lockFileFlockFn = func(fd int, how int) error {
		if how == syscall.LOCK_UN {
			return nil
		}
		flockAttempts++
		if flockAttempts == 1 {
			return syscall.EWOULDBLOCK // simulate the (now dead) owner's lock
		}
		return nil
	}

	result, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("takeover start failed: %v", err)
	}
	if result.Health != "healthy" {
		t.Fatalf("health = %q, want healthy after takeover", result.Health)
	}
	final := latestStartOperation(t, home, "alpha")
	if final.Status != scenarioruntime.StartOperationStatusSucceeded {
		t.Fatalf("final record = %q, want succeeded from the takeover start", final.Status)
	}
}
