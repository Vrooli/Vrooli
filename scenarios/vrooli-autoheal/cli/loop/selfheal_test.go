package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	langrecover "vrooli-autoheal-langrecover"
)

// The exact text the 2026-09-01 outage produced. The recovery floor exists to
// recognise this string from inside the loop, because the API that used to own
// the detector could not build while it was true.
const outageFailureOutput = `running setup phase for vrooli-autoheal...
../../../packages/api-core/nodereach/client.go:19:2: missing go.sum entry for module providing package github.com/gorilla/websocket (imported by github.com/vrooli/api-core/nodereach); to add:
	go get github.com/vrooli/api-core/nodereach@v0.0.0
error: build component api: exit status 1`

func TestSelfHealAllowedFreshState(t *testing.T) {
	allowed, reason := selfHealAllowed(selfHealState{}, time.Now())
	if !allowed {
		t.Fatalf("fresh state must permit an attempt, got %q", reason)
	}
}

func TestSelfHealBudgetExhaustsWithinWindow(t *testing.T) {
	now := time.Now()
	state := selfHealState{WindowStartedAt: now, Attempts: selfHealMaxAttempts}

	allowed, reason := selfHealAllowed(state, now.Add(time.Minute))
	if allowed {
		t.Fatal("budget must be exhausted after max attempts inside the window")
	}
	if !strings.Contains(reason, "budget exhausted") {
		t.Errorf("reason should explain the budget, got %q", reason)
	}
}

func TestSelfHealBudgetResetsAfterWindow(t *testing.T) {
	now := time.Now()
	state := selfHealState{WindowStartedAt: now, Attempts: selfHealMaxAttempts}

	allowed, _ := selfHealAllowed(state, now.Add(selfHealWindow+time.Minute))
	if !allowed {
		t.Fatal("a new window must restore the attempt budget")
	}
}

// The breaker must survive process restart. Without persistence a crash-loop
// would grant a fresh budget every cycle, which defeats the whole point.
func TestSelfHealSuspensionBlocksEvenInNewWindow(t *testing.T) {
	now := time.Now()
	state := selfHealState{
		WindowStartedAt: now.Add(-2 * selfHealWindow),
		Attempts:        selfHealMaxAttempts,
		SuspendedUntil:  now.Add(selfHealSuspension),
	}

	allowed, reason := selfHealAllowed(state, now)
	if allowed {
		t.Fatal("suspension must outrank window expiry")
	}
	if !strings.Contains(reason, "suspended") {
		t.Errorf("reason should mention suspension, got %q", reason)
	}
}

func TestRecordSelfHealAttemptTripsBreaker(t *testing.T) {
	now := time.Now()
	state := selfHealState{}
	for i := 0; i < selfHealMaxAttempts; i++ {
		state = recordSelfHealAttempt(state, now, "go", "healed")
	}
	if state.Attempts != selfHealMaxAttempts {
		t.Errorf("want %d attempts, got %d", selfHealMaxAttempts, state.Attempts)
	}
	if state.SuspendedUntil.IsZero() {
		t.Fatal("reaching the attempt cap must set a suspension")
	}
	if allowed, _ := selfHealAllowed(state, now); allowed {
		t.Error("breaker must be closed immediately after tripping")
	}
}

func TestSelfHealStateRoundTrips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "recovery-floor.json")

	now := time.Now().Truncate(time.Second)
	want := recordSelfHealAttempt(selfHealState{}, now, "go", "healed")
	saveSelfHealState(path, want)

	got := loadSelfHealState(path)
	if got.Attempts != want.Attempts || got.LastStrategy != "go" || got.LastOutcome != "healed" {
		t.Fatalf("state did not round-trip: %+v vs %+v", got, want)
	}
}

func TestLoadSelfHealStateToleratesCorruption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "recovery-floor.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Must fail open to an empty budget rather than wedging recovery forever.
	if state := loadSelfHealState(path); state.Attempts != 0 {
		t.Errorf("corrupt state must read as empty, got %+v", state)
	}
	if allowed, _ := selfHealAllowed(loadSelfHealState(path), time.Now()); !allowed {
		t.Error("corrupt state must not block recovery")
	}
}

func TestAttemptSelfHealIgnoresUnrelatedFailure(t *testing.T) {
	config := &Config{VrooliRoot: t.TempDir(), ScenarioName: "vrooli-autoheal"}
	outcome := attemptSelfHeal(config, "connection refused talking to postgres")
	if outcome.Attempted {
		t.Fatal("a non-drift failure must not trigger recovery")
	}
	if !strings.Contains(outcome.Detail, "no healable") {
		t.Errorf("unexpected detail: %q", outcome.Detail)
	}
}

func TestAttemptSelfHealRequiresOutput(t *testing.T) {
	config := &Config{VrooliRoot: t.TempDir(), ScenarioName: "vrooli-autoheal"}
	if outcome := attemptSelfHeal(config, "   "); outcome.Attempted {
		t.Fatal("empty output must not trigger recovery")
	}
}

// End-to-end through the floor: the outage signature is detected, the Go
// strategy runs against autoheal's own module, and the modified go.sum makes
// the outcome Healed so the caller retries the start.
func TestAttemptSelfHealRecoversOutageSignature(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "scenarios", "vrooli-autoheal", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goMod := filepath.Join(apiDir, "go.mod")
	goSum := filepath.Join(apiDir, "go.sum")
	if err := os.WriteFile(goMod, []byte("module m\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goSum, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Fake runner stands in for `go mod download`, mutating go.sum the way a
	// real recovery would.
	restore := selfHealRunner
	selfHealRunner = func(_ context.Context, dir, name string, args ...string) ([]byte, error) {
		if name != "go" {
			t.Errorf("unexpected command %q", name)
		}
		_ = os.WriteFile(filepath.Join(dir, "go.sum"), []byte("old\nrepaired\n"), 0o644)
		return []byte("ok"), nil
	}
	t.Cleanup(func() { selfHealRunner = restore })

	// Isolate breaker state so the test never touches the real home directory.
	t.Setenv("HOME", filepath.Join(root, "home"))

	config := &Config{VrooliRoot: root, ScenarioName: "vrooli-autoheal"}
	outcome := attemptSelfHeal(config, outageFailureOutput)

	if !outcome.Attempted {
		t.Fatalf("outage signature must engage the floor: %s", outcome.Detail)
	}
	if !outcome.Healed {
		t.Fatalf("modified go.sum must count as healed: %s", outcome.Detail)
	}
	if !strings.Contains(outcome.Detail, "go.sum") {
		t.Errorf("detail should name the modified file, got %q", outcome.Detail)
	}
}

// A recovery command that succeeds but changes nothing must NOT report healed:
// retrying the start would fail identically and burn the attempt budget.
func TestAttemptSelfHealNoOpIsNotHealed(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "scenarios", "vrooli-autoheal", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module m\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.sum"), []byte("unchanged\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	restore := selfHealRunner
	selfHealRunner = func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
		return []byte("nothing to do"), nil
	}
	t.Cleanup(func() { selfHealRunner = restore })
	t.Setenv("HOME", filepath.Join(root, "home"))

	config := &Config{VrooliRoot: root, ScenarioName: "vrooli-autoheal"}
	outcome := attemptSelfHeal(config, outageFailureOutput)

	if !outcome.Attempted {
		t.Fatal("floor should have engaged")
	}
	if outcome.Healed {
		t.Fatal("a no-op recovery must not be reported as healed")
	}
	if !strings.Contains(outcome.Detail, "no-op") {
		t.Errorf("detail should say it was a no-op, got %q", outcome.Detail)
	}
}

// Signature detection is the load-bearing part; assert it directly against the
// real outage text so a refactor of langrecover cannot silently stop matching.
func TestOutageOutputMatchesMissingSumSignature(t *testing.T) {
	if got := langrecover.DetectGoSignature(outageFailureOutput); got != langrecover.GoSignatureMissingSum {
		t.Fatalf("outage output must classify as MissingSum, got %v", got)
	}
}

// The gap the live test exposed on 2026-09-01.
//
// `vrooli scenario start` prints only a summary to stdout ("build component
// api: exit status 1"); the compiler error naming the missing go.sum entry
// goes to ~/.vrooli/logs/<scenario>.log. The first version of the floor read
// only stdout and declined to act, while its unit tests passed because they
// were handed the raw compiler text directly.
const summaryOnlyOutput = `starting vrooli-autoheal...
✗ Failed to start 'vrooli-autoheal'
  Error: build component api: exit status 1
  Full log: /home/matthalloran8/.vrooli/logs/vrooli-autoheal.log
Runtime error: build component api: exit status 1`

func TestSummaryOnlyOutputCarriesNoSignature(t *testing.T) {
	// Guards the premise: if this ever starts matching, the log fallback below
	// is no longer what makes the floor work and the test is misleading.
	if langrecover.DetectGoSignature(summaryOnlyOutput) != langrecover.GoSignatureNone {
		t.Fatal("summary output unexpectedly carries a drift signature")
	}
}

func TestDecideFromSourcesFallsBackToLifecycleLog(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	logDir := filepath.Join(home, ".vrooli", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "vrooli-autoheal.log"), []byte(outageFailureOutput), 0o644); err != nil {
		t.Fatal(err)
	}
	apiDir := filepath.Join(root, "scenarios", "vrooli-autoheal", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	decision, source := decideFromSources(summaryOnlyOutput,
		filepath.Join(root, "scenarios", "vrooli-autoheal"), root, "vrooli-autoheal")

	if !decision.Has() {
		t.Fatal("signature in the lifecycle log must be found when stdout has none")
	}
	if source != "lifecycle-log" {
		t.Errorf("source = %q, want lifecycle-log", source)
	}
}

// Command output wins when both carry a signature: it is always current, while
// the log tail can hold an older failure.
func TestDecideFromSourcesPrefersCommandOutput(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "scenarios", "vrooli-autoheal", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(root, "home"))

	_, source := decideFromSources(outageFailureOutput,
		filepath.Join(root, "scenarios", "vrooli-autoheal"), root, "vrooli-autoheal")
	if source != "command-output" {
		t.Errorf("source = %q, want command-output", source)
	}
}

func TestReadLifecycleLogTailMissingFileIsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if got := readLifecycleLogTail("nope", 1024); got != "" {
		t.Errorf("missing log must read empty, got %q", got)
	}
}
