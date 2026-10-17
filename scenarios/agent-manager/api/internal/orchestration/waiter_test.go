package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// TestProducerKeysMatchCliutilParkProducers pins the park.go invariant: the
// producer keys agents park with MUST match the keys these Waiters register,
// or a parked run's await-handle would never dispatch.
func TestProducerKeysMatchCliutilParkProducers(t *testing.T) {
	pairs := map[string]string{
		ProducerTestGenie:   cliutil.ParkProducerTestGenie,
		ProducerGCT:         cliutil.ParkProducerGCT,
		ProducerLifecycle:   cliutil.ParkProducerLifecycle,
		ProducerSupervision: cliutil.ParkProducerSupervision,
	}
	for waiter, park := range pairs {
		if waiter != park {
			t.Fatalf("waiter producer %q != cliutil park producer %q", waiter, park)
		}
	}
}

type fakeCohortWatchWaiter struct{ watch *domainpb.CohortWatch }

func (f fakeCohortWatchWaiter) WaitTerminal(context.Context, string) (*domainpb.CohortWatch, error) {
	return f.watch, nil
}

func TestSupervisionWaiterReturnsBoundedTerminalEvidence(t *testing.T) {
	evidence := make([]string, 25)
	for i := range evidence {
		evidence[i] = fmt.Sprintf("event-%d", i)
	}
	waiter := NewSupervisionWaiter(fakeCohortWatchWaiter{watch: &domainpb.CohortWatch{WatchId: "watch-1", Revision: 4, Status: domainpb.WatchStatus_WATCH_STATUS_TERMINAL, LastDecision: &domainpb.WatchDecision{Classification: "cohort_terminal", EvidenceIds: evidence}}})
	result, err := waiter.Wait(context.Background(), "watch-1")
	if err != nil || !strings.Contains(result, `"cohort_terminal"`) || strings.Contains(result, `event-24`) || len(result) > 4096 {
		t.Fatalf("result=%q err=%v", result, err)
	}
}

// fakeCommandRunner records the command it was asked to run and returns canned
// output/error so producer Waiters can be exercised without shelling binaries.
type fakeCommandRunner struct {
	name string
	args []string
	out  []byte
	err  error
}

func (f *fakeCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.name = name
	f.args = args
	return f.out, f.err
}

func TestTestGenieWaiter_ShellsBlockingWait(t *testing.T) {
	runner := &fakeCommandRunner{out: []byte(`{"status":"passed"}`)}
	w := NewTestGenieWaiter(runner)

	if w.Producer() != ProducerTestGenie {
		t.Fatalf("producer = %q", w.Producer())
	}

	got, err := w.Wait(context.Background(), "image-tools/20260625-run")
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if got != `{"status":"passed"}` {
		t.Fatalf("result = %q", got)
	}
	wantArgs := []string{"runs", "wait", "--json", "image-tools", "20260625-run"}
	if runner.name != "test-genie" || strings.Join(runner.args, " ") != strings.Join(wantArgs, " ") {
		t.Fatalf("command = %s %v", runner.name, runner.args)
	}
}

func TestGCTBaselineWaiter_ShellsBlockingDiff(t *testing.T) {
	runner := &fakeCommandRunner{out: []byte(`{"exit":0}`)}
	w := NewGCTBaselineWaiter(runner)

	if w.Producer() != ProducerGCT {
		t.Fatalf("producer = %q", w.Producer())
	}

	got, err := w.Wait(context.Background(), "agent-manager/am-park-resume")
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if got != `{"exit":0}` {
		t.Fatalf("result = %q", got)
	}
	wantArgs := []string{"baseline", "diff", "--scenario", "agent-manager", "--name", "am-park-resume", "--wait", "--json"}
	if runner.name != "git-control-tower" || strings.Join(runner.args, " ") != strings.Join(wantArgs, " ") {
		t.Fatalf("command = %s %v", runner.name, runner.args)
	}
}

// exitCodeErr is a stub error exposing ExitCode() so the verdict-exit-code path
// can be exercised without spawning a real process (which is what produces an
// *exec.ExitError in production).
type exitCodeErr struct{ code int }

func (e exitCodeErr) Error() string { return fmt.Sprintf("exit status %d", e.code) }
func (e exitCodeErr) ExitCode() int { return e.code }

func TestLifecycleWaiter_ShellsBlockingScenarioWait(t *testing.T) {
	runner := &fakeCommandRunner{out: []byte(`{"verdict":"healthy"}`)}
	w := NewLifecycleWaiter(runner)

	if w.Producer() != ProducerLifecycle {
		t.Fatalf("producer = %q", w.Producer())
	}

	got, err := w.Wait(context.Background(), "web-console/live")
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if got != `{"verdict":"healthy"}` {
		t.Fatalf("result = %q", got)
	}
	wantArgs := []string{"scenario", "wait", "web-console", "--json"}
	if runner.name != "vrooli" || strings.Join(runner.args, " ") != strings.Join(wantArgs, " ") {
		t.Fatalf("command = %s %v", runner.name, runner.args)
	}

	// Non-live variants are addressed with --instance.
	if _, err := w.Wait(context.Background(), "web-console/shadow"); err != nil {
		t.Fatalf("Wait(shadow): %v", err)
	}
	wantArgs = []string{"scenario", "wait", "web-console", "--json", "--instance", "shadow"}
	if strings.Join(runner.args, " ") != strings.Join(wantArgs, " ") {
		t.Fatalf("shadow command args = %v", runner.args)
	}
}

func TestLifecycleWaiter_VerdictExitsAreResolved(t *testing.T) {
	for _, code := range []int{1, 2, 124} {
		runner := &fakeCommandRunner{out: []byte(`{"verdict":"x"}`), err: exitCodeErr{code: code}}
		w := NewLifecycleWaiter(runner)
		got, err := w.Wait(context.Background(), "web-console/live")
		if err != nil {
			t.Fatalf("verdict exit %d must resolve, got error %v", code, err)
		}
		if got != `{"verdict":"x"}` {
			t.Fatalf("result = %q", got)
		}
	}
	// A non-verdict failure is a wait failure.
	runner := &fakeCommandRunner{out: []byte("boom"), err: errors.New("exec: not found")}
	if _, err := NewLifecycleWaiter(runner).Wait(context.Background(), "web-console/live"); err == nil {
		t.Fatal("expected wait failure for non-verdict error")
	}
}

func TestWaiter_VerdictExitCodeIsResolvedNotError(t *testing.T) {
	// A failing test-genie suite exits 1 but the snapshot is the result.
	tg := NewTestGenieWaiter(&fakeCommandRunner{out: []byte(`{"status":"failed"}`), err: exitCodeErr{code: 1}})
	got, err := tg.Wait(context.Background(), "scn/run")
	if err != nil {
		t.Fatalf("test-genie verdict exit 1 should resolve, got err %v", err)
	}
	if got != `{"status":"failed"}` {
		t.Fatalf("result = %q", got)
	}

	// gct regression (exit 1) and not-comparable (exit 2) are verdicts, not errors.
	for _, code := range []int{1, 2} {
		gct := NewGCTBaselineWaiter(&fakeCommandRunner{out: []byte(`{"verdict":"x"}`), err: exitCodeErr{code: code}})
		if _, err := gct.Wait(context.Background(), "scn/name"); err != nil {
			t.Fatalf("gct verdict exit %d should resolve, got err %v", code, err)
		}
	}

	// A non-verdict exit code (e.g. 3 = not-ready) is still a genuine wait error.
	gct := NewGCTBaselineWaiter(&fakeCommandRunner{out: []byte("not ready"), err: exitCodeErr{code: 3}})
	if _, err := gct.Wait(context.Background(), "scn/name"); err == nil {
		t.Fatal("gct exit 3 (not-ready) should be a wait error")
	}
}

func TestWaiter_PropagatesCommandError(t *testing.T) {
	runner := &fakeCommandRunner{out: []byte("boom"), err: errors.New("exit 1")}
	w := NewTestGenieWaiter(runner)

	out, err := w.Wait(context.Background(), "scn/run")
	if err == nil {
		t.Fatal("expected error")
	}
	// Combined output is surfaced for diagnostics, and the wrapped error names
	// the work.
	if !strings.Contains(out, "boom") {
		t.Fatalf("output should carry command output, got %q", out)
	}
	if !strings.Contains(err.Error(), "scn/run") {
		t.Fatalf("error should name the work: %v", err)
	}
}

func TestSplitProducerKey(t *testing.T) {
	scenario, id, err := splitProducerKey("  image-tools/run-7  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scenario != "image-tools" || id != "run-7" {
		t.Fatalf("parsed (%q,%q)", scenario, id)
	}

	for _, bad := range []string{"", "no-slash", "/onlyid", "scn/"} {
		if _, _, err := splitProducerKey(bad); err == nil {
			t.Fatalf("expected error for key %q", bad)
		}
	}
}

// The production constructors must default a nil runner to the os/exec runner.
func TestWaiter_NilRunnerDefaults(t *testing.T) {
	if NewTestGenieWaiter(nil) == nil {
		t.Fatal("nil waiter")
	}
	if NewGCTBaselineWaiter(nil) == nil {
		t.Fatal("nil waiter")
	}
}
