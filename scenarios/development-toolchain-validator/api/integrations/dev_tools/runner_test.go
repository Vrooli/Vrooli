package dev_tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"development-toolchain-validator/internal/clock"
)

// fakeCommandRunner returns queued CommandResults in order and records
// the argv of each call so tests can assert correct targeting.
type fakeCommandRunner struct {
	results []CommandResult
	errs    []error
	calls   [][]string
	idx     int
}

func (f *fakeCommandRunner) run(_ context.Context, _ string, args ...string) (CommandResult, error) {
	f.calls = append(f.calls, args)
	i := f.idx
	f.idx++
	var res CommandResult
	if i < len(f.results) {
		res = f.results[i]
	}
	var err error
	if i < len(f.errs) {
		err = f.errs[i]
	}
	return res, err
}

func newRunner(t *testing.T, fake *fakeCommandRunner, expectationsDir string) *Runner {
	t.Helper()
	return New(Options{
		Clock:           clock.System{},
		CommandRunner:   fake.run,
		ExpectationsDir: expectationsDir,
	})
}

const absGolden = "/abs/scenarios/reference-react-vite"

func TestInvoke_TestGenie_AllPhasesPass(t *testing.T) {
	fake := &fakeCommandRunner{results: []CommandResult{{
		Launched: true, ExitCode: 0,
		Stdout: []byte(`{"success":true,"phaseSummary":{"total":14,"passed":14,"failed":0},"phases":[{"name":"structure","status":"passed"},{"name":"smoke","status":"passed"}]}`),
	}}}
	r := newRunner(t, fake, t.TempDir())

	res, err := r.Invoke(context.Background(), "test-genie", "reference-react-vite", absGolden)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Ran || !res.ExpectationMet {
		t.Fatalf("want Ran && ExpectationMet, got Ran=%v met=%v detail=%q", res.Ran, res.ExpectationMet, res.Detail)
	}
	// Argv: execute <slug> --scenario-path <abs> --preset comprehensive --json
	want := []string{"execute", "reference-react-vite", "--scenario-path", absGolden, "--preset", "comprehensive", "--json"}
	if len(fake.calls) != 1 || !equal(fake.calls[0], want) {
		t.Fatalf("argv = %v, want %v", fake.calls, want)
	}
	if len(res.RawOutput) == 0 {
		t.Error("expected raw output captured")
	}
}

func TestInvoke_TestGenie_PhaseFailsIsExpectationMiss(t *testing.T) {
	fake := &fakeCommandRunner{results: []CommandResult{{
		// Suite failed (non-zero exit is allowed for the final command —
		// it is the expectation signal, not a run failure).
		Launched: true, ExitCode: 1,
		Stdout: []byte(`{"success":false,"phaseSummary":{"total":14,"passed":13,"failed":1},"phases":[{"name":"smoke","status":"failed"}],"error":"smoke failed"}`),
	}}}
	r := newRunner(t, fake, t.TempDir())

	res, err := r.Invoke(context.Background(), "test-genie", "reference-react-vite", absGolden)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Ran {
		t.Fatalf("tool ran (exited 1) — Ran must be true, got false")
	}
	if res.ExpectationMet {
		t.Fatalf("expectation must be missed when a phase fails")
	}
	if res.Detail == "" || res.ErrorReason == "" {
		t.Errorf("expected detail/error describing the failed phase, got detail=%q reason=%q", res.Detail, res.ErrorReason)
	}
}

func TestInvoke_Completeness_AboveFloorPasses(t *testing.T) {
	fake := &fakeCommandRunner{results: []CommandResult{
		{Launched: true, ExitCode: 0, Stdout: []byte("Score recalculated")},                               // calculate
		{Launched: true, ExitCode: 0, Stdout: []byte(`{"scenario":"reference-react-vite","score":98.5}`)}, // get
	}}
	r := newRunner(t, fake, t.TempDir())

	res, err := r.Invoke(context.Background(), "scenario-completeness-scoring", "reference-react-vite", absGolden)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Ran || !res.ExpectationMet {
		t.Fatalf("want Ran && met, got Ran=%v met=%v detail=%q", res.Ran, res.ExpectationMet, res.Detail)
	}
	// Two commands, both --auto-start; last one is score get --json.
	if len(fake.calls) != 2 {
		t.Fatalf("want 2 commands, got %d: %v", len(fake.calls), fake.calls)
	}
	wantGet := []string{"--auto-start", "score", "get", "reference-react-vite", "--json"}
	if !equal(fake.calls[1], wantGet) {
		t.Fatalf("final argv = %v, want %v", fake.calls[1], wantGet)
	}
}

func TestInvoke_Completeness_BelowFloorIsExpectationMiss(t *testing.T) {
	fake := &fakeCommandRunner{results: []CommandResult{
		{Launched: true, ExitCode: 0, Stdout: []byte("ok")},
		{Launched: true, ExitCode: 0, Stdout: []byte(`{"scenario":"reference-react-vite","score":80}`)},
	}}
	r := newRunner(t, fake, t.TempDir())

	res, err := r.Invoke(context.Background(), "scenario-completeness-scoring", "reference-react-vite", absGolden)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Ran || res.ExpectationMet {
		t.Fatalf("want Ran && !met, got Ran=%v met=%v", res.Ran, res.ExpectationMet)
	}
}

func TestInvoke_Completeness_PrepStepFailureIsRunFailure(t *testing.T) {
	// The calculate step (non-final) exits non-zero → cannot measure → the
	// tool could not run (Ran=false), which the evaluator maps to a run
	// failure rather than a tool/template regression.
	fake := &fakeCommandRunner{results: []CommandResult{
		{Launched: true, ExitCode: 3, Stdout: []byte("service unavailable")},
		{Launched: true, ExitCode: 0, Stdout: []byte(`{"score":99}`)},
	}}
	r := newRunner(t, fake, t.TempDir())

	res, err := r.Invoke(context.Background(), "scenario-completeness-scoring", "reference-react-vite", absGolden)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Ran {
		t.Fatalf("prep-step failure must yield Ran=false, got true")
	}
	// The second (get) command must never run.
	if len(fake.calls) != 1 {
		t.Fatalf("expected to stop after the failed prep step, got calls=%v", fake.calls)
	}
}

func TestInvoke_LaunchFailureIsRunFailure(t *testing.T) {
	fake := &fakeCommandRunner{
		results: []CommandResult{{Launched: false}},
		errs:    []error{os.ErrNotExist},
	}
	r := newRunner(t, fake, t.TempDir())

	res, err := r.Invoke(context.Background(), "test-genie", "reference-react-vite", absGolden)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Ran {
		t.Fatalf("launch failure must yield Ran=false")
	}
	if res.ErrorReason == "" {
		t.Error("expected an error reason on launch failure")
	}
}

func TestInvoke_UnknownToolErrors(t *testing.T) {
	fake := &fakeCommandRunner{}
	r := newRunner(t, fake, t.TempDir())

	res, err := r.Invoke(context.Background(), "scenario-auditor", "reference-react-vite", absGolden)
	if err == nil {
		t.Fatalf("unknown tool must return an error")
	}
	if res.Ran {
		t.Fatalf("unknown tool must yield Ran=false")
	}
	if len(fake.calls) != 0 {
		t.Fatalf("unknown tool must not exec anything, got %v", fake.calls)
	}
}

func TestResolveExpectation_DefaultsWhenMissing(t *testing.T) {
	exp, err := resolveExpectation(t.TempDir(), "test-genie")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.Preset != "comprehensive" {
		t.Errorf("default preset = %q, want comprehensive", exp.Preset)
	}
	if exp.ScoreFloor != defaultScoreFloor {
		t.Errorf("default score floor = %v, want %v", exp.ScoreFloor, defaultScoreFloor)
	}
}

func TestResolveExpectation_FileOverrides(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "scenario-completeness-scoring.json"),
		`{"tool":"scenario-completeness-scoring","score_floor":90}`)
	exp, err := resolveExpectation(dir, "scenario-completeness-scoring")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.ScoreFloor != 90 {
		t.Errorf("score floor = %v, want 90", exp.ScoreFloor)
	}
}

func TestResolveExpectation_MalformedErrors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "test-genie.json"), `{not valid json`)
	if _, err := resolveExpectation(dir, "test-genie"); err == nil {
		t.Fatal("malformed expectation file must error")
	}
}

func TestExtractJSONObject(t *testing.T) {
	got := extractJSONObject([]byte("noise before {\"a\":1} noise after"))
	if string(got) != `{"a":1}` {
		t.Errorf("got %q", string(got))
	}
	if extractJSONObject([]byte("no json here")) != nil {
		t.Error("expected nil when no object present")
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
