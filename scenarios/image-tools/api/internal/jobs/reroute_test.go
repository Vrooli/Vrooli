package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// busyLocalAdmit refuses local work while holdLocal is set, the way a saturated
// host broker does.
func busyLocalAdmit(holdLocal *atomic.Bool) Admitter {
	return func(ctx context.Context, j Job) (Admission, error) {
		if j.Lane != LaneNetwork && holdLocal.Load() {
			return Admission{Context: ctx}, fmt.Errorf("%w: host RAM is held by another owner", ErrCapacityBusy)
		}
		return Admission{Context: ctx}, nil
	}
}

func TestRerouteMovesAWaitingLocalJobToTheNetworkLaneAndRunsItExactlyOnce(t *testing.T) {
	var holdLocal atomic.Bool
	holdLocal.Store(true)
	var mu sync.Mutex
	var ran []string
	var offers atomic.Int32
	m := startManager(t, Config{
		Admit: busyLocalAdmit(&holdLocal),
		Reroute: func(_ context.Context, job Job) (*Reroute, error) {
			offers.Add(1)
			if string(job.Payload) != "local" {
				return nil, nil
			}
			return &Reroute{Payload: []byte("cloud"), Lane: LaneNetwork, EstimatedSeconds: 30, Message: "re-routed to cloud: waited past its budget"}, nil
		},
		Runner: func(_ context.Context, job Job, _ func(int, string)) (Result, error) {
			mu.Lock()
			ran = append(ran, fmt.Sprintf("%s/%s", job.Payload, job.Lane))
			mu.Unlock()
			return Result{Ref: "out/" + string(job.Payload)}, nil
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "edit_instruct", Lane: LaneGPU, Payload: []byte("local"), EstimatedSeconds: 180})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	final, err := m.Wait(ctx, job.ID)
	if err != nil || final.State != StateSucceeded || final.ResultRef != "out/cloud" || final.Lane != LaneNetwork || final.EstimatedSeconds != 30 {
		t.Fatalf("rerouted job = %+v err=%v", final, err)
	}
	// Local capacity frees afterwards: the job must not run again there.
	holdLocal.Store(false)
	time.Sleep(1500 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(ran) != 1 || ran[0] != "cloud/network" {
		t.Fatalf("job ran %v, want exactly one cloud run", ran)
	}
	stored, err := m.st.get(context.Background(), job.ID)
	if err != nil || string(stored.Payload) != "cloud" || stored.Lane != LaneNetwork || stored.EstimatedSeconds != 30 {
		t.Fatalf("the move was not persisted: %+v err=%v", stored, err)
	}
	if offers.Load() == 0 {
		t.Fatal("rerouter was never consulted")
	}
}

func TestRerouterIsNeverOfferedAJobThatAlreadyStarted(t *testing.T) {
	gate := make(chan struct{})
	started := make(chan struct{})
	var offered atomic.Bool
	var running atomic.Bool
	m := startManager(t, Config{
		Reroute: func(_ context.Context, job Job) (*Reroute, error) {
			if running.Load() {
				offered.Store(true)
			}
			return nil, nil
		},
		Runner: func(ctx context.Context, _ Job, _ func(int, string)) (Result, error) {
			running.Store(true)
			close(started)
			select {
			case <-gate:
			case <-ctx.Done():
			}
			return Result{}, nil
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "edit_instruct", Lane: LaneGPU, Payload: []byte("local")})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	time.Sleep(2500 * time.Millisecond)
	close(gate)
	if _, err := m.Wait(context.Background(), job.ID); err != nil {
		t.Fatal(err)
	}
	if offered.Load() {
		t.Fatal("a running job was offered for re-routing")
	}
}

func TestRerouteThatCannotBeAppliedOrIsDeclinedLeavesTheJobWaitingLocally(t *testing.T) {
	var holdLocal atomic.Bool
	holdLocal.Store(true)
	var calls atomic.Int32
	m := startManager(t, Config{
		Admit: busyLocalAdmit(&holdLocal),
		Reroute: func(context.Context, Job) (*Reroute, error) {
			calls.Add(1)
			return nil, errors.New("daily cloud cap reached")
		},
		Runner: func(_ context.Context, job Job, _ func(int, string)) (Result, error) {
			return Result{Ref: string(job.Payload)}, nil
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "edit_instruct", Lane: LaneGPU, Payload: []byte("local")})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	if got, _ := m.Get(context.Background(), job.ID); got.State.Terminal() || got.Lane != LaneGPU || string(got.Payload) != "local" {
		t.Fatalf("a declined reroute must leave the job queued locally: %+v", got)
	}
	// A declined offer is not re-evaluated against the ledger on every tick.
	if got := calls.Load(); got != 1 {
		t.Fatalf("rerouter consulted %d times in 2s after a decline; want it paced", got)
	}
	holdLocal.Store(false)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	final, err := m.Wait(ctx, job.ID)
	if err != nil || final.State != StateSucceeded || final.ResultRef != "local" {
		t.Fatalf("job should still run locally once capacity frees: %+v err=%v", final, err)
	}
}

func TestRerouteCancelsALongAdmissionAndMovesTheJobWithoutRunningItLocally(t *testing.T) {
	var mu sync.Mutex
	var ran []string
	var released atomic.Int32
	admitting := make(chan struct{}, 1)
	var inAdmission atomic.Bool
	m := startManager(t, Config{
		// Local admission never completes on its own (a model download, a held
		// host ledger); only cancellation ends it.
		Admit: func(ctx context.Context, j Job) (Admission, error) {
			if j.Lane == LaneNetwork {
				return Admission{Context: ctx}, nil
			}
			inAdmission.Store(true)
			admitting <- struct{}{}
			<-ctx.Done()
			return Admission{Context: ctx, Release: func(context.Context) error { released.Add(1); return nil }, ReleaseRetryable: true}, ctx.Err()
		},
		Reroute: func(_ context.Context, job Job) (*Reroute, error) {
			if string(job.Payload) != "local" || !inAdmission.Load() {
				return nil, nil
			}
			return &Reroute{Payload: []byte("cloud"), Lane: LaneNetwork, EstimatedSeconds: 30, Message: "re-routed to cloud: admission outlasted its budget"}, nil
		},
		Runner: func(_ context.Context, job Job, _ func(int, string)) (Result, error) {
			mu.Lock()
			ran = append(ran, fmt.Sprintf("%s/%s", job.Payload, job.Lane))
			mu.Unlock()
			return Result{Ref: "out/" + string(job.Payload)}, nil
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "edit_instruct", Lane: LaneGPU, Payload: []byte("local")})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-admitting:
	case <-time.After(5 * time.Second):
		t.Fatal("local admission never started")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	final, err := m.Wait(ctx, job.ID)
	if err != nil || final.State != StateSucceeded || final.ResultRef != "out/cloud" || final.Lane != LaneNetwork {
		t.Fatalf("job = %+v err=%v", final, err)
	}
	if released.Load() != 1 {
		t.Fatalf("the cancelled admission released its leases %d times, want 1", released.Load())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(ran) != 1 || ran[0] != "cloud/network" {
		t.Fatalf("ran %v, want exactly one cloud run and no local run", ran)
	}
	m.mu.Lock()
	local, network := m.localActive, m.networkActive
	m.mu.Unlock()
	if local != 0 || network != 0 {
		t.Fatalf("dispatch slots leaked: local=%d network=%d", local, network)
	}
}

func TestACancelDuringAdmissionStillWinsOverAReroute(t *testing.T) {
	admitting := make(chan struct{}, 1)
	m := startManager(t, Config{
		Admit: func(ctx context.Context, j Job) (Admission, error) {
			admitting <- struct{}{}
			<-ctx.Done()
			return Admission{Context: ctx}, ctx.Err()
		},
		Reroute: func(context.Context, Job) (*Reroute, error) { return nil, nil },
		Runner:  func(context.Context, Job, func(int, string)) (Result, error) { return Result{}, nil },
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "edit_instruct", Lane: LaneGPU, Payload: []byte("local")})
	if err != nil {
		t.Fatal(err)
	}
	<-admitting
	if err := m.Cancel(job.ID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if final, err := m.Wait(ctx, job.ID); err != nil || final.State != StateCanceled {
		t.Fatalf("job = %+v err=%v", final, err)
	}
}

// A route refused before any work returns the job to its previous route, and
// the record of the refusal is durable while the job still waits.
func TestRouteBackReturnsAJobWhoseRunnerRefusedItsRouteToTheQueueExactlyOnce(t *testing.T) {
	var mu sync.Mutex
	var ran []string
	var released atomic.Int32
	m := startManager(t, Config{
		Admit: func(ctx context.Context, j Job) (Admission, error) {
			return Admission{Context: ctx, Release: func(context.Context) error { released.Add(1); return nil }, ReleaseRetryable: true}, nil
		},
		Runner: func(_ context.Context, job Job, _ func(int, string)) (Result, error) {
			mu.Lock()
			ran = append(ran, fmt.Sprintf("%s/%s", job.Payload, job.Lane))
			mu.Unlock()
			if string(job.Payload) == "cloud" {
				return Result{}, &RouteBackError{
					Move:  Reroute{Payload: []byte("local"), Lane: LaneGPU, EstimatedSeconds: 45, Message: "cloud refused; back on the local route", Meta: map[string]string{"route_cloud_refused_at": "2026-11-21T12:00:00Z"}},
					Cause: errors.New("insufficient_credits"),
				}
			}
			return Result{Ref: "out/local", Meta: map[string]string{"route_reason": "cloud_unavailable_fell_back_local"}}, nil
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "edit_instruct", Lane: LaneNetwork, Payload: []byte("cloud"), EstimatedSeconds: 30})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	final, err := m.Wait(ctx, job.ID)
	if err != nil || final.State != StateSucceeded || final.ResultRef != "out/local" || final.Lane != LaneGPU || final.EstimatedSeconds != 45 {
		t.Fatalf("job = %+v err=%v", final, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(ran) != 2 || ran[0] != "cloud/network" || ran[1] != "local/gpu" {
		t.Fatalf("ran %v, want one cloud attempt then one local run", ran)
	}
	if released.Load() != 2 {
		t.Fatalf("each attempt must release its admission: %d", released.Load())
	}
	m.mu.Lock()
	local, network := m.localActive, m.networkActive
	m.mu.Unlock()
	if local != 0 || network != 0 {
		t.Fatalf("dispatch slots leaked: local=%d network=%d", local, network)
	}
	refusals, err := m.ListRouteRefusals(context.Background(), 10)
	if err != nil || len(refusals) != 1 || refusals[0].ID != job.ID {
		t.Fatalf("the refusal must be queryable from the job table: %+v err=%v", refusals, err)
	}
}

func TestRouteBackFromAdmissionNeverRunsTheRefusedRoute(t *testing.T) {
	var ran atomic.Int32
	m := startManager(t, Config{
		Admit: func(ctx context.Context, j Job) (Admission, error) {
			if string(j.Payload) == "cloud" {
				return Admission{Context: ctx}, &RouteBackError{Move: Reroute{Payload: []byte("local"), Lane: LaneGPU, Message: "cloud paused"}, Cause: errors.New("cooldown")}
			}
			return Admission{Context: ctx}, nil
		},
		Runner: func(_ context.Context, job Job, _ func(int, string)) (Result, error) {
			if string(job.Payload) == "cloud" {
				ran.Add(100)
			}
			ran.Add(1)
			return Result{Ref: "out/" + string(job.Payload)}, nil
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "edit_instruct", Lane: LaneNetwork, Payload: []byte("cloud")})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	final, err := m.Wait(ctx, job.ID)
	if err != nil || final.State != StateSucceeded || final.ResultRef != "out/local" || ran.Load() != 1 {
		t.Fatalf("job = %+v err=%v runs=%d", final, err, ran.Load())
	}
}

func TestACanceledJobIsNotReturnedToAnotherRoute(t *testing.T) {
	started := make(chan struct{})
	m := startManager(t, Config{
		Runner: func(ctx context.Context, _ Job, _ func(int, string)) (Result, error) {
			close(started)
			<-ctx.Done()
			return Result{}, &RouteBackError{Move: Reroute{Payload: []byte("local"), Lane: LaneGPU}, Cause: ctx.Err()}
		},
	})
	job, err := m.Submit(context.Background(), Spec{Operation: "edit_instruct", Lane: LaneNetwork, Payload: []byte("cloud")})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := m.Cancel(job.ID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if final, err := m.Wait(ctx, job.ID); err != nil || final.State != StateCanceled {
		t.Fatalf("job = %+v err=%v", final, err)
	}
}
