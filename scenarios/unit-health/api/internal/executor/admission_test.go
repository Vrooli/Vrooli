package executor

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunAllUsesBoundedWorkersAndAdmission(t *testing.T) {
	var calls atomic.Int32
	runner := RunnerFunc(func(_ context.Context, command Command) Result {
		calls.Add(1)
		return Result{WorkspaceID: command.WorkspaceID, Name: command.Name, Status: StatusPassed}
	})
	commands := []Command{{WorkspaceID: "one", Resources: ResourceLimits{CPUWeight: 2}}, {WorkspaceID: "two", Resources: ResourceLimits{CPUWeight: 2}}}
	results := RunAllWithAdmission(context.Background(), runner, commands, 2, NewAdmission(2, 0))
	if calls.Load() != 2 || len(results) != 2 || results[0].Status != StatusPassed || results[1].Status != StatusPassed {
		t.Fatalf("calls=%d results=%+v", calls.Load(), results)
	}
	tooLarge := RunAllWithAdmission(context.Background(), runner, []Command{{Resources: ResourceLimits{CPUWeight: 3}}}, 1, NewAdmission(2, 0))
	if tooLarge[0].FailureReason == "" {
		t.Fatal("oversized command was admitted")
	}
	tooMuchMemory := RunAllWithAdmission(context.Background(), runner, []Command{{Resources: ResourceLimits{MemoryBytes: 3}}}, 1, NewAdmission(0, 2))
	if tooMuchMemory[0].FailureReason == "" {
		t.Fatal("oversized memory command was admitted")
	}
}

func TestAdmissionServesWaitersFIFO(t *testing.T) {
	a := NewAdmission(1, 0)
	first := Command{WorkspaceID: "first", Resources: ResourceLimits{CPUWeight: 1}}
	if err := a.Acquire(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	second := Command{WorkspaceID: "second", Resources: ResourceLimits{CPUWeight: 1}}
	third := Command{WorkspaceID: "third", Resources: ResourceLimits{CPUWeight: 1}}
	order := make(chan string, 2)
	go admissionAcquireAndRelease(t, a, second, order)
	waitForAdmissionWaiters(t, a, 1)
	go admissionAcquireAndRelease(t, a, third, order)
	waitForAdmissionWaiters(t, a, 2)
	a.Release(first)
	if got := <-order; got != "second" {
		t.Fatalf("first waiter served=%q, want second", got)
	}
	if got := <-order; got != "third" {
		t.Fatalf("second waiter served=%q, want third", got)
	}
}

func TestAdmissionCancellationRemovesWaiter(t *testing.T) {
	a := NewAdmission(1, 0)
	first := Command{WorkspaceID: "first", Resources: ResourceLimits{CPUWeight: 1}}
	if err := a.Acquire(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	cancelCtx, cancel := context.WithCancel(context.Background())
	canceled := make(chan error, 1)
	go func() { canceled <- a.Acquire(cancelCtx, Command{Resources: ResourceLimits{CPUWeight: 1}}) }()
	waitForAdmissionWaiters(t, a, 1)
	cancel()
	if err := <-canceled; err != context.Canceled {
		t.Fatalf("canceled acquire error=%v, want context canceled", err)
	}
	ready := make(chan error, 1)
	go func() { ready <- a.Acquire(context.Background(), Command{Resources: ResourceLimits{CPUWeight: 1}}) }()
	waitForAdmissionWaiters(t, a, 1)
	a.Release(first)
	if err := <-ready; err != nil {
		t.Fatalf("next waiter remained blocked after canceled head: %v", err)
	}
	// The successful waiter owns the reservation until it releases it.
	a.Release(Command{Resources: ResourceLimits{CPUWeight: 1}})
}

func admissionAcquireAndRelease(t *testing.T, a *Admission, command Command, order chan<- string) {
	t.Helper()
	if err := a.Acquire(context.Background(), command); err != nil {
		t.Errorf("acquire %s: %v", command.WorkspaceID, err)
		return
	}
	order <- command.WorkspaceID
	a.Release(command)
}

func waitForAdmissionWaiters(t *testing.T, a *Admission, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		got := len(a.waiters)
		a.mu.Unlock()
		if got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("admission waiters did not reach %d", want)
}

type RunnerFunc func(context.Context, Command) Result

func (f RunnerFunc) Run(ctx context.Context, command Command) Result { return f(ctx, command) }
