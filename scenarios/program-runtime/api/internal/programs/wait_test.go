package programs

import (
	"context"
	"strings"
	"testing"
	"time"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"program-runtime/internal/budgets"
)

// slowRunner blocks until released, so a test can hold a program in a
// non-terminal state and observe the wait rather than racing it.
type slowRunner struct{ release chan struct{} }

func (r *slowRunner) Execute(ctx context.Context, _ string, _ string, _ bool) (Result, error) {
	select {
	case <-r.release:
		return Result{Stdout: "done"}, nil
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

func newWaitService(runner Runner) *Service {
	return NewService(Options{
		Runner:          runner,
		ValidateSession: func(string) bool { return true },
	})
}

// TestWaitReturnsImmediatelyForATerminalProgram is the fast path: a caller that
// waits on finished work must not be made to sit through its deadline.
func TestWaitReturnsImmediatelyForATerminalProgram(t *testing.T) {
	service := newWaitService(fakeRunner{result: Result{Stdout: "ok"}})
	program, err := service.Submit(context.Background(), "sess", "print(1)", programsv1.Provenance_PROVENANCE_TEST, false)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	started := time.Now()
	waited, terminal, err := service.Wait(context.Background(), program.GetId(), 5*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if !terminal {
		t.Fatalf("a completed program must be terminal, got %s", waited.GetStatus())
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("wait on a terminal program took %s; it must not sit out its deadline", elapsed)
	}
}

// TestWaitWakesOnTheTerminalTransition is the property that lets the wait
// replace a polling loop: completion notifies the waiter rather than the waiter
// discovering it on a later tick.
func TestWaitWakesOnTheTerminalTransition(t *testing.T) {
	runner := &slowRunner{release: make(chan struct{})}
	service := newWaitService(runner)
	program, err := service.Submit(context.Background(), "sess", "print(1)", programsv1.Provenance_PROVENANCE_TEST, false, true)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	type outcome struct {
		program  *programsv1.Program
		terminal bool
		elapsed  time.Duration
	}
	results := make(chan outcome, 1)
	go func() {
		started := time.Now()
		waited, terminal, waitErr := service.Wait(context.Background(), program.GetId(), 30*time.Second)
		if waitErr != nil {
			results <- outcome{}
			return
		}
		results <- outcome{program: waited, terminal: terminal, elapsed: time.Since(started)}
	}()

	// Give the waiter time to register, then let the program finish.
	time.Sleep(50 * time.Millisecond)
	close(runner.release)

	select {
	case got := <-results:
		if !got.terminal {
			t.Fatalf("wait did not observe the terminal transition; status=%s", got.program.GetStatus())
		}
		if got.elapsed > 5*time.Second {
			t.Fatalf("wait took %s; it should wake on notification, not on a timer", got.elapsed)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("wait never returned after the program completed")
	}
}

// TestWaitReturnsNonTerminalAtItsDeadline states the contract that keeps a
// caller from busy-looping: exceeding the wait is a stated outcome carrying the
// current program, not an error.
func TestWaitReturnsNonTerminalAtItsDeadline(t *testing.T) {
	runner := &slowRunner{release: make(chan struct{})}
	defer close(runner.release)
	service := newWaitService(runner)
	program, err := service.Submit(context.Background(), "sess", "print(1)", programsv1.Provenance_PROVENANCE_TEST, false, true)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	waited, terminal, err := service.Wait(context.Background(), program.GetId(), 100*time.Millisecond)
	if err != nil {
		t.Fatalf("a deadline is not an error: %v", err)
	}
	if terminal {
		t.Fatal("program was reported terminal while still running")
	}
	if waited.GetId() != program.GetId() {
		t.Fatalf("wait returned the wrong program: %s", waited.GetId())
	}
}

func TestWaitReportsAnUnknownProgram(t *testing.T) {
	service := newWaitService(fakeRunner{result: Result{Stdout: "ok"}})
	if _, _, err := service.Wait(context.Background(), "prog_missing", time.Second); err == nil {
		t.Fatal("waiting on an unknown id must report not-found")
	}
}

// TestSyncExecutionBudgetMatchesTheLadder pins the duplicated constant to the
// budget authority. The value is declared in two packages to avoid a dependency
// edge; this test is what makes that duplication safe.
func TestSyncExecutionBudgetMatchesTheLadder(t *testing.T) {
	if SyncExecutionBudget != budgets.SyncSubmit {
		t.Fatalf("sync execution budget %s drifted from the ladder's %s", SyncExecutionBudget, budgets.SyncSubmit)
	}
}

// TestSyncDeadlineErrorNamesTheRemedy: the message is the entire user
// experience of hitting this bound, and the previous one (`unexpected EOF`)
// named neither the limit nor what to do.
func TestSyncDeadlineErrorNamesTheRemedy(t *testing.T) {
	err := &SyncDeadlineExceededError{Limit: 2 * time.Minute, ProgramID: "prog_1"}
	message := err.Error()
	for _, needle := range []string{"deadline_exceeded", "2m0s", "prog_1", "--async", "programs wait"} {
		if !strings.Contains(message, needle) {
			t.Fatalf("message %q does not mention %q", message, needle)
		}
	}
}

// TestSeveredConnectionsClassifyAsTransport is the B3 regression: these four
// shapes all reached the corpus as `kernel_runtime`, pointing readers at their
// own program instead of at the boundary that actually failed.
func TestSeveredConnectionsClassifyAsTransport(t *testing.T) {
	for _, detail := range []string{
		"RemoteDisconnected: Remote end closed connection without response",
		"unavailable: unexpected EOF",
		"connection reset by peer",
	} {
		shape, cause := failureShape(detail)
		if shape != "bridge_transport" {
			t.Fatalf("detail %q classified as %q, want bridge_transport", detail, shape)
		}
		if cause != programsv1.FailureCause_FAILURE_CAUSE_BRIDGE_TRANSPORT {
			t.Fatalf("detail %q carried cause %v, want BRIDGE_TRANSPORT", detail, cause)
		}
	}
}

func TestDeadlineDetailClassifiesAsDeadlineExceeded(t *testing.T) {
	shape, cause := failureShape("deadline_exceeded: synchronous submission exceeded 2m0s")
	if shape != "deadline_exceeded" || cause != programsv1.FailureCause_FAILURE_CAUSE_DEADLINE_EXCEEDED {
		t.Fatalf("got %q/%v, want deadline_exceeded", shape, cause)
	}
}

// TestNoFailureDetailClassifiesAsKernelRuntimeByDefault keeps the fallback
// honest: an unrecognised detail must not be silently mapped onto a specific
// cause it does not warrant.
func TestNoFailureDetailClassifiesAsKernelRuntimeByDefault(t *testing.T) {
	shape, _ := failureShape("ZeroDivisionError: division by zero")
	if shape != "kernel_runtime" {
		t.Fatalf("an ordinary program error classified as %q, want kernel_runtime", shape)
	}
}
