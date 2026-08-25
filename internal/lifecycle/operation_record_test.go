package lifecycle

import (
	"context"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func latestStartOperation(t *testing.T, home, scenario string) scenarioruntime.StartOperation {
	t.Helper()
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("open runtime registry: %v", err)
	}
	defer store.Close()
	op, err := store.GetLatestStartOperation(context.Background(), scenario, "")
	if err != nil {
		t.Fatalf("GetLatestStartOperation: %v", err)
	}
	return op
}

func stepStatuses(op scenarioruntime.StartOperation) map[string]string {
	out := map[string]string{}
	for _, step := range op.Steps() {
		out[step.Name] = step.Status
	}
	return out
}

// TestStartWritesOperationRecord proves a real successful start leaves a
// succeeded record with done steps, a verdict, and phase-duration history.
func TestStartWritesOperationRecord(t *testing.T) {
	if runtime.GOOS != "linux" {
		testkitgo.SkipPlatform(t, "lifecycle process management currently targets linux")
	}
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")
	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	cleanupRunner(t, runner, "alpha", StopOptions{})

	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	op := latestStartOperation(t, home, "alpha")
	if op.Status != scenarioruntime.StartOperationStatusSucceeded {
		t.Fatalf("status = %q, want succeeded (op=%+v)", op.Status, op)
	}
	if op.Verdict != "healthy" {
		t.Fatalf("verdict = %q, want healthy", op.Verdict)
	}
	if op.Operation != "start" {
		t.Fatalf("operation = %q, want start", op.Operation)
	}
	if op.InitiatorPID == nil {
		t.Fatal("initiator pid must be recorded")
	}
	if op.FinishedAt == nil {
		t.Fatal("finished_at must be set on success")
	}
	statuses := stepStatuses(op)
	for _, step := range []string{startStepSetup, startStepDevelop, startStepHealth} {
		if statuses[step] != scenarioruntime.StartStepDone {
			t.Fatalf("step %s = %q, want done (steps=%v)", step, statuses[step], statuses)
		}
	}

	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("open runtime registry: %v", err)
	}
	defer store.Close()
	estimates, err := store.PhaseDurationEstimates(context.Background(), "alpha", "")
	if err != nil {
		t.Fatalf("PhaseDurationEstimates: %v", err)
	}
	for _, phase := range []string{startStepSetup, startStepDevelop, startStepHealth} {
		if _, ok := estimates[phase]; !ok {
			t.Fatalf("phase %s has no recorded duration (estimates=%v)", phase, estimates)
		}
	}

	// A warm second start (reuse) also records a succeeded operation.
	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start(warm): %v", err)
	}
	warm := latestStartOperation(t, home, "alpha")
	if warm.OperationID == op.OperationID {
		t.Fatal("warm start must create a new operation record")
	}
	if warm.Status != scenarioruntime.StartOperationStatusSucceeded {
		t.Fatalf("warm status = %q, want succeeded", warm.Status)
	}

	// Restart records operation=restart including the stop step.
	if _, err := runner.Restart("alpha", StartOptions{}); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	restart := latestStartOperation(t, home, "alpha")
	if restart.Operation != "restart" {
		t.Fatalf("restart operation = %q, want restart", restart.Operation)
	}
	if got := stepStatuses(restart)[startStepStop]; got != scenarioruntime.StartStepDone {
		t.Fatalf("restart stop step = %q, want done", got)
	}
}

// TestRecorderStopStepMatchesInstanceSlug proves the recorder opens its stop
// step for exactly its own instance: stop events carry the INSTANCE slug
// (scenario@variant for non-live), so a variant's own stop is recorded and a
// dependency's stop never opens the top-level record's stop step.
func TestRecorderStopStepMatchesInstanceSlug(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("open runtime registry: %v", err)
	}
	defer store.Close()

	op, err := store.BeginStartOperation(ctx, scenarioruntime.StartOperation{Scenario: "alpha", Variant: "shadow"})
	if err != nil {
		t.Fatalf("BeginStartOperation: %v", err)
	}
	rec := &startOperationRecorder{
		runner: &Runner{}, store: store, op: op, scenario: "alpha",
		instanceSlug: scenarioruntime.InstanceKey{Scenario: "alpha", Variant: "shadow"}.Slug(),
	}
	at := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	// A dependency's stop must not open this record's stop step.
	rec.Publish(ProgressEvent{Kind: EventStopStarted, Scenario: "some-dependency", At: at})
	if got := stepStatuses(rec.op); got[startStepStop] != "" {
		t.Fatalf("dependency stop opened the stop step: %v", got)
	}
	// The bare scenario slug is not this instance either (live vs shadow).
	rec.Publish(ProgressEvent{Kind: EventStopStarted, Scenario: "alpha", At: at})
	if got := stepStatuses(rec.op); got[startStepStop] != "" {
		t.Fatalf("live-instance stop opened the shadow record's stop step: %v", got)
	}
	// This instance's own stop (restart's leading step / mid-plan stop) is.
	rec.Publish(ProgressEvent{Kind: EventStopStarted, Scenario: "alpha@shadow", At: at})
	if got := stepStatuses(rec.op); got[startStepStop] != scenarioruntime.StartStepRunning {
		t.Fatalf("own stop step = %q, want running (%v)", got[startStepStop], got)
	}

	persisted, err := store.GetLatestStartOperation(ctx, "alpha", "shadow")
	if err != nil {
		t.Fatalf("GetLatestStartOperation: %v", err)
	}
	if got := stepStatuses(persisted)[startStepStop]; got != scenarioruntime.StartStepRunning {
		t.Fatalf("persisted stop step = %q, want running", got)
	}
}

func TestStartOperationContextCancellationMarksAbandoned(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner, err := NewRunner(t.TempDir(), t.TempDir(), io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	rec := runner.beginStartOperationRecord("alpha", StartOptions{Context: ctx})
	if rec == nil {
		t.Fatal("beginStartOperationRecord returned nil")
	}
	cancel()
	select {
	case <-rec.cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("context cancellation did not mark the operation")
	}
	op := latestStartOperation(t, runner.Home, "alpha")
	if op.Status != scenarioruntime.StartOperationStatusAbandoned || op.Error != "start cancelled" {
		t.Fatalf("operation = %+v, want abandoned/start cancelled", op)
	}
	rec.close()
}

// TestFailedStartWritesFailedOperationRecord proves a health-failing start
// leaves a failed record carrying the error.
func TestFailedStartWritesFailedOperationRecord(t *testing.T) {
	if runtime.GOOS != "linux" {
		testkitgo.SkipPlatform(t, "lifecycle process management currently targets linux")
	}
	root := t.TempDir()
	home := t.TempDir()
	manifest := lifecycleFixtureManifest("alpha")
	// Break the declared component command so the start fails fast.
	component := manifest.Components["api"]
	component.Run.Argv = []string{"false"}
	manifest.Components["api"] = component
	writeLifecycleFixtureManifest(t, root, manifest)
	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	cleanupRunner(t, runner, "alpha", StopOptions{})

	if _, err := runner.Start("alpha", StartOptions{}); err == nil {
		t.Fatal("expected start to fail")
	}
	op := latestStartOperation(t, home, "alpha")
	if op.Status != scenarioruntime.StartOperationStatusFailed {
		t.Fatalf("status = %q, want failed", op.Status)
	}
	if op.Error == "" {
		t.Fatal("failed record must carry the error")
	}
	if got := stepStatuses(op)[startStepDevelop]; got != scenarioruntime.StartStepFailed {
		t.Fatalf("develop step = %q, want failed", got)
	}
}

// --- EvaluateStartOperation (reader view) ---

func runningOperationFixture(startedAt time.Time) scenarioruntime.StartOperation {
	pid := 12345
	op := scenarioruntime.StartOperation{
		OperationID:  "startop-1",
		Scenario:     "alpha",
		Variant:      "live",
		Operation:    "start",
		Status:       scenarioruntime.StartOperationStatusRunning,
		InitiatorPID: &pid,
		StartedAt:    startedAt,
	}
	return op
}

func TestEvaluateStartOperationAbandonsDeadInitiator(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 1, 0, 0, time.UTC)
	op := runningOperationFixture(now.Add(-time.Minute))

	view := EvaluateStartOperation(op, func(int) bool { return false }, now, nil)
	if view.Status != scenarioruntime.StartOperationStatusAbandoned {
		t.Fatalf("status = %q, want abandoned for dead initiator", view.Status)
	}
	if !view.Terminal() || view.RecommendedNextCheckSeconds != 0 {
		t.Fatalf("abandoned must be terminal with next-check 0, got %+v", view)
	}
	if view.ElapsedSeconds != 60 {
		t.Fatalf("elapsed = %d, want 60", view.ElapsedSeconds)
	}

	// Unknown initiator PID is also not trusted.
	op.InitiatorPID = nil
	view = EvaluateStartOperation(op, func(int) bool { return true }, now, nil)
	if view.Status != scenarioruntime.StartOperationStatusAbandoned {
		t.Fatalf("status = %q, want abandoned for unknown initiator", view.Status)
	}
}

func TestEvaluateStartOperationSurfacesInitiatorPID(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 1, 0, 0, time.UTC)
	op := runningOperationFixture(now.Add(-time.Minute))

	// The PID is the reader's link between an in-flight operation and the
	// "already running (held by pid N)" lock error, so it must survive
	// evaluation in both directions — live initiator and dead one.
	view := EvaluateStartOperation(op, func(int) bool { return true }, now, nil)
	if view.InitiatorPID != 12345 {
		t.Fatalf("initiator pid = %d, want 12345", view.InitiatorPID)
	}
	view = EvaluateStartOperation(op, func(int) bool { return false }, now, nil)
	if view.InitiatorPID != 12345 {
		t.Fatalf("abandoned view dropped the initiator pid: %d", view.InitiatorPID)
	}

	op.InitiatorPID = nil
	view = EvaluateStartOperation(op, func(int) bool { return true }, now, nil)
	if view.InitiatorPID != 0 {
		t.Fatalf("absent initiator pid = %d, want 0", view.InitiatorPID)
	}
}

func TestInFlightSummaryReportsRunningOperationOnly(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 1, 0, 0, time.UTC)
	alive := func(int) bool { return true }

	op := runningOperationFixture(now.Add(-time.Minute))
	op.CurrentStep = startStepSetup
	view := EvaluateStartOperation(op, alive, now, nil)
	summary := view.InFlightSummary()
	for _, want := range []string{"start in progress", "pid 12345", "setup step", "60s elapsed"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary %q missing %q", summary, want)
		}
	}

	// A dependency-step operation names the dependency instead of the step.
	op.CurrentStep = startStepDependencies
	op.DependencyCurrent = "postgres"
	op.DependencyIndex = 2
	op.DependencyTotal = 3
	view = EvaluateStartOperation(op, alive, now, nil)
	if summary := view.InFlightSummary(); !strings.Contains(summary, "dependency postgres (2/3)") {
		t.Fatalf("summary %q missing dependency detail", summary)
	}

	// Terminal operations are not in flight and must render nothing: a
	// succeeded corpse reported as activity is the same lie in reverse.
	for _, status := range []string{
		scenarioruntime.StartOperationStatusSucceeded,
		scenarioruntime.StartOperationStatusFailed,
		scenarioruntime.StartOperationStatusAbandoned,
	} {
		terminal := runningOperationFixture(now.Add(-time.Minute))
		terminal.Status = status
		if summary := EvaluateStartOperation(terminal, alive, now, nil).InFlightSummary(); summary != "" {
			t.Fatalf("terminal %q rendered in-flight summary %q", status, summary)
		}
	}

	// A running record whose initiator is dead evaluates to abandoned, so it
	// must not be advertised as in flight either.
	dead := runningOperationFixture(now.Add(-time.Minute))
	if summary := EvaluateStartOperation(dead, func(int) bool { return false }, now, nil).InFlightSummary(); summary != "" {
		t.Fatalf("dead-initiator record rendered in-flight summary %q", summary)
	}
}

func TestEvaluateStartOperationRunningETAAndNextCheck(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 30, 0, time.UTC)
	alive := func(int) bool { return true }

	// No history → ETA unknown → next check 30.
	op := runningOperationFixture(now.Add(-30 * time.Second))
	view := EvaluateStartOperation(op, alive, now, nil)
	if view.Status != scenarioruntime.StartOperationStatusRunning {
		t.Fatalf("status = %q, want running", view.Status)
	}
	if view.ETAKnown {
		t.Fatal("ETA must be unknown without history")
	}
	if view.RecommendedNextCheckSeconds != 30 {
		t.Fatalf("next check = %d, want 30 for unknown ETA", view.RecommendedNextCheckSeconds)
	}

	// Full history, develop currently running for 10s of an estimated 20s:
	// remaining = (20-10) + health 15 = 25s.
	developStart := now.Add(-10 * time.Second)
	op.CurrentStep = startStepDevelop
	setupEnd := developStart
	op.WithSteps([]scenarioruntime.StartOperationStep{
		{Name: startStepSetup, Status: scenarioruntime.StartStepDone, StartedAt: now.Add(-30 * time.Second), EndedAt: &setupEnd},
		{Name: startStepDevelop, Status: scenarioruntime.StartStepRunning, StartedAt: developStart},
	})
	estimates := map[string]time.Duration{
		startStepSetup:   40 * time.Second,
		startStepDevelop: 20 * time.Second,
		startStepHealth:  15 * time.Second,
	}
	view = EvaluateStartOperation(op, alive, now, estimates)
	if !view.ETAKnown || view.ETASeconds != 25 {
		t.Fatalf("eta = (%v, %d), want (true, 25)", view.ETAKnown, view.ETASeconds)
	}
	if view.RecommendedNextCheckSeconds != 25 {
		t.Fatalf("next check = %d, want 25 (remaining within clamp)", view.RecommendedNextCheckSeconds)
	}

	// Clamping: tiny remaining clamps to 5, huge clamps to 60.
	estimates[startStepDevelop] = 1 * time.Second
	estimates[startStepHealth] = 1 * time.Second
	view = EvaluateStartOperation(op, alive, now, estimates)
	if view.RecommendedNextCheckSeconds != recommendedNextCheckMin {
		t.Fatalf("next check = %d, want clamp min %d", view.RecommendedNextCheckSeconds, recommendedNextCheckMin)
	}
	estimates[startStepHealth] = 10 * time.Minute
	view = EvaluateStartOperation(op, alive, now, estimates)
	if view.RecommendedNextCheckSeconds != recommendedNextCheckMax {
		t.Fatalf("next check = %d, want clamp max %d", view.RecommendedNextCheckSeconds, recommendedNextCheckMax)
	}

	// A plain start that has not entered setup excludes setup from the ETA
	// (it may be skipped); a restart includes it, and missing develop history
	// makes the whole ETA unknown.
	fresh := runningOperationFixture(now.Add(-time.Second))
	view = EvaluateStartOperation(fresh, alive, now, map[string]time.Duration{
		startStepDevelop: 20 * time.Second,
		startStepHealth:  10 * time.Second,
	})
	if !view.ETAKnown || view.ETASeconds != 30 {
		t.Fatalf("fresh start eta = (%v, %d), want (true, 30) excluding unstarted setup", view.ETAKnown, view.ETASeconds)
	}
	fresh.Operation = "restart"
	view = EvaluateStartOperation(fresh, alive, now, map[string]time.Duration{
		startStepDevelop: 20 * time.Second,
		startStepHealth:  10 * time.Second,
	})
	if view.ETAKnown {
		t.Fatal("restart without setup history must render unknown ETA")
	}
}

func TestEvaluateStartOperationTerminalStates(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 5, 0, 0, time.UTC)
	finished := now.Add(-time.Minute)
	op := runningOperationFixture(now.Add(-2 * time.Minute))
	op.Status = scenarioruntime.StartOperationStatusSucceeded
	op.Verdict = "healthy"
	op.FinishedAt = &finished

	view := EvaluateStartOperation(op, func(int) bool { return false }, now, nil)
	if view.Status != scenarioruntime.StartOperationStatusSucceeded {
		t.Fatalf("status = %q, want succeeded (dead pid must not reclassify terminal records)", view.Status)
	}
	if view.RecommendedNextCheckSeconds != 0 {
		t.Fatalf("next check = %d, want 0 for terminal", view.RecommendedNextCheckSeconds)
	}
	if view.ElapsedSeconds != 60 {
		t.Fatalf("elapsed = %d, want 60 (start → finish, not start → now)", view.ElapsedSeconds)
	}
}
