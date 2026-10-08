package trials

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// fakeCmd scripts agent-manager's CLI: it records every invocation and returns
// canned snake_case JSON per subcommand, so the Runner is exercised end-to-end
// without a live model. runGets is a queue (the last entry repeats) so a
// non-terminal→terminal poll sequence can be modelled.
type fakeCmd struct {
	calls        [][]string
	profileOut   string
	taskOut      string
	runCreateOut string
	runGets      []string
	diffOut      string
	errOn        string // "profile reconcile-scenario" | "task create" | "run create" | "run get" | "run diff"
	getIdx       int
}

func (f *fakeCmd) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	key := strings.Join(args[:min(2, len(args))], " ")
	if f.errOn == key {
		return nil, fmt.Errorf("boom: %s", key)
	}
	switch key {
	case "profile reconcile-scenario":
		return []byte(f.profileOut), nil
	case "task create":
		return []byte(f.taskOut), nil
	case "run create":
		return []byte(f.runCreateOut), nil
	case "run get":
		out := f.runGets[f.getIdx]
		if f.getIdx < len(f.runGets)-1 {
			f.getIdx++
		}
		return []byte(out), nil
	case "run diff":
		return []byte(f.diffOut), nil
	}
	return nil, fmt.Errorf("unexpected command: %v", args)
}

func completeRunJSON() string {
	return `{"run":{"id":"run-1","status":"RUN_STATUS_COMPLETE","sandbox_id":"sbx-9",` +
		`"changed_files":2,"started_at":"2026-06-24T12:00:00Z","ended_at":"2026-06-24T12:00:05Z",` +
		`"summary":{"tokens_used":1234,"cost_estimate":0.01},"extra_unknown_field":"ignored"}}`
}

func happyFake() *fakeCmd {
	return &fakeCmd{
		profileOut:   `{"scenario":"meta-optimization-manager","results":[{"profile_key":"meta-optimization-manager/trials","profile_id":"prof-1"}]}`,
		taskOut:      `{"task":{"id":"task-1"}}`,
		runCreateOut: `{"run":{"id":"run-1","status":"RUN_STATUS_RUNNING"},"queue_depth":0}`,
		runGets:      []string{completeRunJSON()},
		diffOut:      "diff --git a/x.txt b/x.txt\n@@ -0,0 +1 @@\n+hello\n",
	}
}

func fixtureForTest() Fixture {
	return Fixture{Family: SuiteAddFeature, TargetDir: "/tmp/fx/target", Rev: "rev1", Prompt: "do the thing", Oracle: []string{"bash", "check.sh"}}
}

func TestRunnerHappyPathCollectsEvidence(t *testing.T) {
	f := happyFake()
	r := NewRunnerWithCommand(f.run)
	res := r.RunTask(context.Background(), TrialTask{ID: "trial/g1", Suite: SuiteAddFeature}, fixtureForTest())

	if res.Verdict != VerdictUnspecified {
		t.Fatalf("runner must NOT decide a verdict on success, got %v (detail=%q)", res.Verdict, res.Detail)
	}
	if res.Tokens != 1234 || res.DurationMs != 5000 || res.ChangedFiles != 2 {
		t.Fatalf("metrics parsed wrong: %+v", res)
	}
	if res.SandboxDiffRef != "sbx-9" || res.RunID != "run-1" {
		t.Fatalf("evidence pointers wrong: %+v", res)
	}
	if !strings.Contains(res.Diff, "+hello") {
		t.Fatalf("diff not captured: %q", res.Diff)
	}

	// Command sequencing: reconcile → task create → run create → run get → run diff.
	wantSeq := []string{"profile reconcile-scenario", "task create", "run create", "run get", "run diff"}
	if len(f.calls) != len(wantSeq) {
		t.Fatalf("expected %d calls, got %d: %v", len(wantSeq), len(f.calls), f.calls)
	}
	for i, want := range wantSeq {
		got := strings.Join(f.calls[i][1:3], " ")
		if got != want {
			t.Fatalf("call %d = %q want %q", i, got, want)
		}
	}
	// Scope path + sandboxed mode threaded through.
	if !argsContain(f.calls, "--scope-path", "/tmp/fx/target") {
		t.Fatalf("task create missing fixture scope path: %v", f.calls)
	}
	if !argsContain(f.calls, "--run-mode", "sandboxed") {
		t.Fatalf("run create missing sandboxed mode: %v", f.calls)
	}
}

func TestRunnerPollsUntilTerminal(t *testing.T) {
	f := happyFake()
	f.runGets = []string{
		`{"run":{"id":"run-1","status":"RUN_STATUS_RUNNING"}}`,
		`{"run":{"id":"run-1","status":"RUN_STATUS_SANDBOX_CREATING"}}`,
		completeRunJSON(),
	}
	r := newRunnerForTest(f.run, time.Millisecond)
	res := r.RunTask(context.Background(), TrialTask{ID: "g", Suite: SuiteAddFeature}, fixtureForTest())
	if res.Verdict != VerdictUnspecified || res.Tokens != 1234 {
		t.Fatalf("poll-to-terminal failed: %+v", res)
	}
	getCalls := 0
	for _, c := range f.calls {
		if len(c) >= 3 && c[1] == "run" && c[2] == "get" {
			getCalls++
		}
	}
	if getCalls != 3 {
		t.Fatalf("expected 3 run get polls, got %d", getCalls)
	}
}

func TestRunnerStepErrorsDegradeToVerdictError(t *testing.T) {
	steps := []string{"profile reconcile-scenario", "task create", "run create", "run get", "run diff"}
	for _, step := range steps {
		f := happyFake()
		f.errOn = step
		r := NewRunnerWithCommand(f.run)
		res := r.RunTask(context.Background(), TrialTask{ID: "g", Suite: SuiteAddFeature}, fixtureForTest())
		if res.Verdict != VerdictError {
			t.Fatalf("step %q error must yield VerdictError, got %v", step, res.Verdict)
		}
		if res.Detail == "" {
			t.Fatalf("step %q error must carry a detail", step)
		}
	}
}

func TestRunnerFailedStatusIsErrorAndSkipsDiff(t *testing.T) {
	f := happyFake()
	f.runGets = []string{`{"run":{"id":"run-1","status":"RUN_STATUS_FAILED","error_msg":"agent crashed","changed_files":3}}`}
	r := NewRunnerWithCommand(f.run)
	res := r.RunTask(context.Background(), TrialTask{ID: "g", Suite: SuiteAddFeature}, fixtureForTest())
	if res.Verdict != VerdictError {
		t.Fatalf("failed run must be VerdictError, got %v", res.Verdict)
	}
	if !strings.Contains(res.Detail, "agent crashed") {
		t.Fatalf("detail should carry the error_msg: %q", res.Detail)
	}
	for _, c := range f.calls {
		if len(c) >= 3 && c[1] == "run" && c[2] == "diff" {
			t.Fatalf("run diff must not be called for a failed run")
		}
	}
}

func TestRunnerAbstentionCollectsNoDiff(t *testing.T) {
	f := happyFake()
	f.runGets = []string{`{"run":{"id":"run-1","status":"RUN_STATUS_COMPLETE","changed_files":0,"summary":{"tokens_used":50}}}`}
	r := NewRunnerWithCommand(f.run)
	res := r.RunTask(context.Background(), TrialTask{ID: "neg", Suite: SuiteNegative, Negative: true}, fixtureForTest())
	if res.Verdict != VerdictUnspecified {
		t.Fatalf("evidence run should leave verdict to the evaluator, got %v", res.Verdict)
	}
	if res.ChangedFiles != 0 || res.Diff != "" {
		t.Fatalf("abstention should produce no diff: %+v", res)
	}
	for _, c := range f.calls {
		if len(c) >= 3 && c[1] == "run" && c[2] == "diff" {
			t.Fatalf("run diff must be skipped when changed_files==0")
		}
	}
}

func TestRunnerNonTerminalTimesOut(t *testing.T) {
	f := happyFake()
	f.runGets = []string{`{"run":{"id":"run-1","status":"RUN_STATUS_RUNNING"}}`}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so the first poll select hits ctx.Done deterministically
	r := newRunnerForTest(f.run, time.Hour)
	res := r.RunTask(ctx, TrialTask{ID: "g", Suite: SuiteAddFeature}, fixtureForTest())
	if res.Verdict != VerdictError {
		t.Fatalf("a run that never reaches terminal must be VerdictError, got %v", res.Verdict)
	}
}

func TestRunnerReconcilesDeclaredProfile(t *testing.T) {
	f := happyFake()
	r := NewRunnerWithCommand(f.run)
	_ = r.RunTask(context.Background(), TrialTask{ID: "g", Suite: SuiteAddFeature}, fixtureForTest())
	if !argsContain(f.calls, "--scenario", trialScenario) {
		t.Fatalf("profile reconciliation missing scenario: %v", f.calls)
	}
}

func argsContain(calls [][]string, flag, value string) bool {
	for _, c := range calls {
		for i := 0; i+1 < len(c); i++ {
			if c[i] == flag && c[i+1] == value {
				return true
			}
		}
	}
	return false
}
