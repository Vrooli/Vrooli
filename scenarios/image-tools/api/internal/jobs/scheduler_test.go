package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
)

func TestLifecycleRejectsTerminalResurrectionAndInvalidEdges(t *testing.T) {
	states := []State{StateQueued, StateAdmitting, StateRunning, StateCanceling, StateSucceeded, StateFailed, StateCanceled, "unknown"}
	for _, terminal := range []State{StateSucceeded, StateFailed, StateCanceled} {
		for _, next := range states {
			job := Job{State: terminal}
			if err := job.transition(next, time.Now(), ""); err == nil || job.State != terminal {
				t.Fatalf("terminal %s transitioned to %s", terminal, next)
			}
		}
	}
	job := Job{State: StateQueued}
	if err := job.transition(StateSucceeded, time.Now(), ""); err == nil {
		t.Fatal("queued work succeeded without admission/execution")
	}
	for _, state := range []State{StateAdmitting, StateQueued, StateAdmitting, StateRunning, StateCanceling, StateCanceled} {
		if err := job.transition(state, time.Now(), ""); err != nil {
			t.Fatal(err)
		}
	}
	if job.StartedAt == nil || job.FinishedAt == nil {
		t.Fatal("lifecycle did not record execution/terminal times")
	}
}

func TestSchedulerOrdersByPriorityAndPersistsIt(t *testing.T) {
	started := make(chan string, 4)
	gate := make(chan struct{})
	m := startManager(t, Config{CPUWorkers: 1, Runner: func(ctx context.Context, j Job, _ func(int, string)) (Result, error) {
		started <- j.Operation
		if j.Operation == "blocker" {
			select {
			case <-gate:
			case <-ctx.Done():
				return Result{}, ctx.Err()
			}
		}
		return Result{}, nil
	}})
	blocker, err := m.Submit(context.Background(), Spec{Operation: "blocker"})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	ids := []string{blocker.ID}
	for _, priority := range []Priority{PriorityBatch, PriorityService, PriorityInteractive} {
		job, err := m.Submit(context.Background(), Spec{Operation: string(priority), Lane: LaneGPU, Priority: priority})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, job.ID)
		stored, err := m.st.get(context.Background(), job.ID)
		if err != nil || stored.Priority != priority {
			t.Fatalf("priority not durable: %+v %v", stored, err)
		}
	}
	close(gate)
	for _, expected := range []string{"interactive", "service", "batch"} {
		select {
		case got := <-started:
			if got != expected {
				t.Fatalf("priority order: got %s, want %s", got, expected)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("scheduler did not dispatch")
		}
	}
	for _, id := range ids {
		if _, err := m.Wait(context.Background(), id); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSchedulerRequeuesCapacityRefusalAfterPartialCleanup(t *testing.T) {
	var attempts, releases, executions atomic.Int32
	m := startManager(t, Config{
		Admit: func(ctx context.Context, _ Job) (Admission, error) {
			admission := Admission{Context: ctx, Release: func(cleanup context.Context) error {
				if cleanup.Err() != nil {
					t.Error("canceled cleanup")
				}
				releases.Add(1)
				return nil
			}}
			if attempts.Add(1) == 1 {
				return admission, fmt.Errorf("%w: host busy", ErrCapacityBusy)
			}
			return admission, nil
		},
		Runner: func(context.Context, Job, func(int, string)) (Result, error) { executions.Add(1); return Result{}, nil },
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "model"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	final, err := m.Wait(ctx, job.ID)
	if err != nil || final.State != StateSucceeded || attempts.Load() != 2 || releases.Load() != 2 || executions.Load() != 1 {
		t.Fatalf("retry did not preserve ownership: %+v err=%v attempts=%d releases=%d executions=%d", final, err, attempts.Load(), releases.Load(), executions.Load())
	}
}

func TestCancelWaitsForEveryClaimAndIgnoresLateSuccess(t *testing.T) {
	started, releasing, allowRelease := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var claims atomic.Int32
	m := startManager(t, Config{
		Admit: func(ctx context.Context, _ Job) (Admission, error) {
			claims.Add(2)
			return Admission{Context: ctx, Release: func(cleanup context.Context) error {
				if cleanup.Err() != nil {
					t.Error("cleanup inherited cancellation")
				}
				close(releasing)
				<-allowRelease
				claims.Add(-2)
				return nil
			}}, nil
		},
		Runner: func(ctx context.Context, _ Job, emit func(int, string)) (Result, error) {
			close(started)
			<-ctx.Done()
			emit(100, "late success")
			return Result{Ref: "must-not-publish"}, nil
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "model"})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := m.Cancel(job.ID); err != nil {
		t.Fatal(err)
	}
	<-releasing
	current, _ := m.Get(context.Background(), job.ID)
	if current.State != StateCanceling || claims.Load() != 2 {
		t.Fatalf("cancel completed before release: %+v claims=%d", current, claims.Load())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	_, err = m.Wait(ctx, job.ID)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait returned before cleanup: %v", err)
	}
	close(allowRelease)
	final, err := m.Wait(context.Background(), job.ID)
	if err != nil || final.State != StateCanceled || final.ResultRef != "" || claims.Load() != 0 {
		t.Fatalf("bad canceled completion: %+v err=%v claims=%d", final, err, claims.Load())
	}
}

func TestRunnerPanicAndCleanupFailureCannotLeakSuccess(t *testing.T) {
	for _, panicRunner := range []bool{true, false} {
		t.Run(fmt.Sprint(panicRunner), func(t *testing.T) {
			var releases atomic.Int32
			m := startManager(t, Config{
				Admit: func(ctx context.Context, _ Job) (Admission, error) {
					return Admission{Context: ctx, Release: func(context.Context) error {
						releases.Add(1)
						if !panicRunner {
							return errors.New("release unavailable")
						}
						return nil
					}}, nil
				},
				Runner: func(context.Context, Job, func(int, string)) (Result, error) {
					if panicRunner {
						panic("backend panic")
					}
					return Result{Ref: "result"}, nil
				},
			})
			job, err := m.Submit(context.Background(), Spec{Operation: "model"})
			if err != nil {
				t.Fatal(err)
			}
			final, err := m.Wait(context.Background(), job.ID)
			if err != nil || final.State != StateFailed || final.ResultRef != "" || releases.Load() != 1 {
				t.Fatalf("cleanup/panic result: %+v %v releases=%d", final, err, releases.Load())
			}
			if !panicRunner && !strings.Contains(final.Error, "release unavailable") {
				t.Fatalf("cleanup failure hidden: %+v", final)
			}
		})
	}
}

func TestShutdownFinalizesQueuedAndRunningJobs(t *testing.T) {
	started := make(chan struct{})
	m := startManager(t, Config{CPUWorkers: 1, Runner: func(ctx context.Context, _ Job, _ func(int, string)) (Result, error) {
		close(started)
		<-ctx.Done()
		return Result{}, ctx.Err()
	}})
	var ids []string
	for i := 0; i < 10; i++ {
		job, err := m.Submit(context.Background(), Spec{Operation: "model"})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, job.ID)
		if i == 0 {
			<-started
		}
	}
	m.Close()
	for _, id := range ids {
		final, err := m.Wait(context.Background(), id)
		if err != nil || final.State != StateCanceled {
			t.Fatalf("shutdown stranded job: %+v %v", final, err)
		}
	}
}

// A capacity refusal must not become a fixed-rate admission loop: every
// attempt opens the shared broker and reserves disk before refusing. Retries
// back off exponentially, up to maxAdmissionRetry.
func TestSchedulerBacksOffRepeatedCapacityRefusals(t *testing.T) {
	clock := schedule.NewFake(time.Date(2026, 11, 12, 22, 22, 0, 0, time.UTC))
	attempts := make(chan struct{}, 16)
	m := startManager(t, Config{
		Clock: clock,
		Admit: func(ctx context.Context, _ Job) (Admission, error) {
			attempts <- struct{}{}
			return Admission{Context: ctx}, fmt.Errorf("%w: host CPU fully committed", ErrCapacityBusy)
		},
		Runner: func(context.Context, Job, func(int, string)) (Result, error) { return Result{}, nil },
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "model", Lane: LaneGPU, Priority: PriorityBatch})
	if err != nil {
		t.Fatal(err)
	}
	expectAttempt := func(want bool, step string) {
		t.Helper()
		select {
		case <-attempts:
			if !want {
				t.Fatalf("%s: admission retried before its backoff elapsed", step)
			}
		case <-time.After(300 * time.Millisecond):
			if want {
				t.Fatalf("%s: admission was not retried", step)
			}
		}
	}
	awaitQueued := func() {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for {
			current, _ := m.Get(context.Background(), job.ID)
			m.mu.Lock()
			pending := len(m.pending)
			m.mu.Unlock()
			if current.State == StateQueued && pending == 1 {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("refused job was not requeued: %+v", current)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	advance := func(d time.Duration) { clock.Advance(d); clock.Tick() }

	expectAttempt(true, "first admission")
	for _, backoff := range []time.Duration{admissionRetry, 2 * admissionRetry, 4 * admissionRetry} {
		awaitQueued()
		advance(backoff - time.Millisecond)
		expectAttempt(false, fmt.Sprintf("before %s", backoff))
		advance(time.Millisecond)
		expectAttempt(true, fmt.Sprintf("after %s", backoff))
	}
	awaitQueued()
	advance(24 * time.Hour)
	expectAttempt(true, "after a long wait")
	awaitQueued()
	advance(maxAdmissionRetry)
	expectAttempt(true, "backoff is capped")
}

// When a local job ends, its released capacity must reach queued local work
// immediately rather than after the queued job's backoff.
func TestSchedulerWakesQueuedLocalWorkWhenLocalJobFinishes(t *testing.T) {
	clock := schedule.NewFake(time.Date(2026, 11, 12, 22, 22, 0, 0, time.UTC))
	gate := make(chan struct{})
	var running atomic.Bool
	var refusals atomic.Int32
	m := startManager(t, Config{
		Clock:      clock,
		CPUWorkers: 2,
		Admit: func(ctx context.Context, j Job) (Admission, error) {
			if j.Operation == "second" && running.Load() {
				refusals.Add(1)
				return Admission{Context: ctx}, fmt.Errorf("%w: first job holds the CPU", ErrCapacityBusy)
			}
			return Admission{Context: ctx}, nil
		},
		Runner: func(ctx context.Context, j Job, _ func(int, string)) (Result, error) {
			if j.Operation == "first" {
				defer running.Store(false)
				select {
				case <-gate:
				case <-ctx.Done():
					return Result{}, ctx.Err()
				}
			}
			return Result{}, nil
		},
	})
	running.Store(true)
	first, err := m.Submit(context.Background(), Spec{Operation: "first", Lane: LaneGPU})
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.Submit(context.Background(), Spec{Operation: "second", Lane: LaneGPU})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for refusals.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("second job was never refused while the first ran")
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(gate)
	if _, err := m.Wait(context.Background(), first.ID); err != nil {
		t.Fatal(err)
	}
	// The fake clock never advances, so only the completion wake can admit it.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	final, err := m.Wait(ctx, second.ID)
	if err != nil || final.State != StateSucceeded {
		t.Fatalf("queued local job not admitted after capacity was freed: %+v err=%v", final, err)
	}
}

func TestRetryableReleaseSurvivesTransientContentionAndNeverDiscardsFinishedWork(t *testing.T) {
	oldBudget, oldBackoff := releaseRetryBudget, releaseRetryBackoff
	releaseRetryBudget, releaseRetryBackoff = 200*time.Millisecond, []time.Duration{10 * time.Millisecond}
	t.Cleanup(func() { releaseRetryBudget, releaseRetryBackoff = oldBudget, oldBackoff })
	cases := []struct {
		name       string
		retryable  bool
		failures   int32
		wantState  State
		wantResult string
		// wantRelease is the capacity_release meta a succeeded job must carry.
		wantRelease string
		leaks       bool
	}{
		{name: "transient lock contention clears", retryable: true, failures: 2, wantState: StateSucceeded, wantResult: "result"},
		{name: "contention outlasts the budget keeps the finished result", retryable: true, failures: 1 << 20, wantState: StateSucceeded, wantResult: "result", wantRelease: "unconfirmed"},
		{name: "non-idempotent release is attempted once and fails the job", retryable: false, failures: 1, wantState: StateFailed},
		{name: "release that leaks work fails even a retryable job", retryable: true, failures: 1 << 20, leaks: true, wantState: StateFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var attempts atomic.Int32
			m := startManager(t, Config{
				Admit: func(ctx context.Context, _ Job) (Admission, error) {
					return Admission{Context: ctx, ReleaseRetryable: tc.retryable, Release: func(context.Context) error {
						if attempts.Add(1) <= tc.failures {
							err := errors.New("lock capacity-admission: context deadline exceeded")
							if tc.leaks {
								err = fmt.Errorf("%w: %w", ErrReleaseLeaksWork, err)
							}
							return err
						}
						return nil
					}}, nil
				},
				Runner: func(context.Context, Job, func(int, string)) (Result, error) {
					return Result{Ref: "result"}, nil
				},
			})
			job, err := m.Submit(context.Background(), Spec{Operation: "model"})
			if err != nil {
				t.Fatal(err)
			}
			final, err := m.Wait(context.Background(), job.ID)
			if err != nil || final.State != tc.wantState || final.ResultRef != tc.wantResult {
				t.Fatalf("final = %+v err=%v attempts=%d", final, err, attempts.Load())
			}
			if !tc.retryable && attempts.Load() != 1 {
				t.Fatalf("non-retryable release attempted %d times", attempts.Load())
			}
			if got := final.Meta[MetaCapacityRelease]; got != tc.wantRelease {
				t.Fatalf("capacity_release meta = %q, want %q (meta %v)", got, tc.wantRelease, final.Meta)
			}
			if tc.wantState == StateFailed && !strings.Contains(final.Error, "resource cleanup failed") {
				t.Fatalf("an unrecoverable release must stay a visible cleanup failure: %+v", final)
			}
			if tc.wantRelease != "" && !strings.Contains(final.Meta[MetaCapacityReleaseWarning], "context deadline exceeded") {
				t.Fatalf("unconfirmed release must carry its cause in meta: %v", final.Meta)
			}
		})
	}
}

func TestReleaseFailureAfterFailedOrCanceledWorkStaysAVisibleCleanupFailure(t *testing.T) {
	oldBudget, oldBackoff := releaseRetryBudget, releaseRetryBackoff
	releaseRetryBudget, releaseRetryBackoff = 50*time.Millisecond, []time.Duration{5 * time.Millisecond}
	t.Cleanup(func() { releaseRetryBudget, releaseRetryBackoff = oldBudget, oldBackoff })
	m := startManager(t, Config{
		Admit: func(ctx context.Context, _ Job) (Admission, error) {
			return Admission{Context: ctx, ReleaseRetryable: true, Release: func(context.Context) error {
				return errors.New("lock capacity-admission: context deadline exceeded")
			}}, nil
		},
		Runner: func(context.Context, Job, func(int, string)) (Result, error) {
			return Result{}, errors.New("inference failed")
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "model"})
	if err != nil {
		t.Fatal(err)
	}
	final, err := m.Wait(context.Background(), job.ID)
	if err != nil || final.State != StateFailed {
		t.Fatalf("final = %+v err=%v", final, err)
	}
	for _, want := range []string{"inference failed", "resource cleanup failed"} {
		if !strings.Contains(final.Error, want) {
			t.Fatalf("failed job error %q must name %q", final.Error, want)
		}
	}
}

func TestAdmissionProgressIsVisibleWhileAdmitting(t *testing.T) {
	reported, proceed := make(chan struct{}), make(chan struct{})
	m := startManager(t, Config{
		Admit: func(ctx context.Context, _ Job) (Admission, error) {
			AdmissionProgress(ctx)(40, "acquiring weights for dncnn: downloading")
			close(reported)
			<-proceed
			return Admission{Context: ctx}, nil
		},
		Runner: func(context.Context, Job, func(int, string)) (Result, error) { return Result{}, nil },
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "model"})
	if err != nil {
		t.Fatal(err)
	}
	<-reported
	stored, err := m.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	close(proceed)
	if stored.State != StateAdmitting || stored.Progress != 40 || stored.Message != "acquiring weights for dncnn: downloading" {
		t.Fatalf("admitting job = state %s progress %d message %q", stored.State, stored.Progress, stored.Message)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if final, err := m.Wait(ctx, job.ID); err != nil || final.State != StateSucceeded {
		t.Fatalf("final=%+v err=%v", final, err)
	}
	AdmissionProgress(context.Background())(1, "no reporter is a no-op")
}

func TestOnlyTheJobsOwnTimeLimitIsReportedAsExecutionTimeLimit(t *testing.T) {
	cases := []struct {
		name      string
		timeout   int
		runner    func(context.Context) error
		wantError string
	}{
		{name: "own execution timeout", timeout: 1, runner: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }, wantError: "execution time limit exceeded"},
		{name: "inner capacity deadline keeps its cause", timeout: 0, runner: func(context.Context) error {
			return fmt.Errorf("renew disk lease: %w", context.DeadlineExceeded)
		}, wantError: "renew disk lease"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := startManager(t, Config{Runner: func(ctx context.Context, _ Job, _ func(int, string)) (Result, error) {
				return Result{}, tc.runner(ctx)
			}})
			job, err := m.Submit(context.Background(), Spec{Operation: "model", ExecutionTimeoutSeconds: tc.timeout})
			if err != nil {
				t.Fatal(err)
			}
			final, err := m.Wait(context.Background(), job.ID)
			if err != nil || final.State != StateFailed || !strings.Contains(final.Error, tc.wantError) {
				t.Fatalf("final = %+v err=%v, want failed with %q", final, err, tc.wantError)
			}
			if tc.timeout == 0 && strings.Contains(final.Error, "execution time limit") {
				t.Fatalf("inner deadline mislabelled as the job time limit: %q", final.Error)
			}
		})
	}
}
