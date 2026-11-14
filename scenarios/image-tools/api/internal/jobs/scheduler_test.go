package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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
