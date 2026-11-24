package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	applydomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/apply"
)

func TestApplyRunnerRequestParsesBothFlagForms(t *testing.T) {
	for _, testCase := range []struct {
		name string
		args []string
		want string
		mode bool
	}{
		{name: "server start", args: []string{}, want: "", mode: false},
		{name: "server start with other flags", args: []string{"--verbose"}, want: "", mode: false},
		{name: "separate value", args: []string{"--apply-run", "apply-1"}, want: "apply-1", mode: true},
		{name: "equals value", args: []string{"--apply-run=apply-2"}, want: "apply-2", mode: true},
		{name: "flag with no value", args: []string{"--apply-run"}, want: "", mode: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			id, ok := applyRunnerRequest(testCase.args)
			if ok != testCase.mode {
				t.Fatalf("runner mode = %v, want %v", ok, testCase.mode)
			}
			if id != testCase.want {
				t.Fatalf("id = %q, want %q", id, testCase.want)
			}
		})
	}
}

func TestPrepareApplyRunnerExecutableCreatesStateOwnedCopy(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	previous := applyRunnerExecutablePath
	t.Cleanup(func() { applyRunnerExecutablePath = previous })

	if err := prepareApplyRunnerExecutable(); err != nil {
		t.Fatalf("prepareApplyRunnerExecutable: %v", err)
	}
	if applyRunnerExecutablePath == "" {
		t.Fatal("apply runner executable path is empty")
	}
	info, err := os.Stat(applyRunnerExecutablePath)
	if err != nil {
		t.Fatalf("stat stable runner: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("stable runner mode = %o, want executable", info.Mode().Perm())
	}
}

// TestApplyRunnerModeExecutesAPersistedRun covers the handoff itself: a run
// written by one process is picked up and executed by another, with nothing
// passed between them but the run id.
func TestApplyRunnerModeExecutesAPersistedRun(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	stubExternalReadinessProbes(t)

	fake := &recordingApplyExecutor{}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })

	run := applyRun{
		ID:        "apply-handoff-fixture",
		Status:    "pending",
		StartedAt: operatorStateNow().UTC().Format(time.RFC3339),
		Items: []applyItemResult{
			{applyItem: applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"}, Outcome: "pending"},
		},
	}
	if err := persistApplyRun(run); err != nil {
		t.Fatalf("persist run: %v", err)
	}

	if err := runApplyRunner(context.Background(), run.ID); err != nil {
		t.Fatalf("runApplyRunner: %v", err)
	}

	executed, ok := applyRunSnapshot(run.ID)
	if !ok {
		t.Fatal("the run disappeared")
	}
	if !isTerminalApplyStatus(executed.Status) {
		t.Fatalf("status = %q, want terminal", executed.Status)
	}
	if calls := fake.snapshotCalls(); len(calls) != 1 || calls[0] != "resource:postgres" {
		t.Fatalf("calls = %#v, want the persisted item to have been executed", calls)
	}
	if executed.RunnerPID != os.Getpid() {
		t.Fatalf("runner pid = %d, want the executing process %d recorded", executed.RunnerPID, os.Getpid())
	}
}

// TestApplyRunnerRefusesARunItDoesNotOwn keeps the handoff single-shot. Two
// runners on one run would repeat host mutations that already happened.
func TestApplyRunnerRefusesARunItDoesNotOwn(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)

	fake := &recordingApplyExecutor{}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })

	run := applyRun{ID: "apply-already-done", Status: "applied"}
	if err := persistApplyRun(run); err != nil {
		t.Fatalf("persist run: %v", err)
	}
	err := runApplyRunner(context.Background(), run.ID)
	if err == nil {
		t.Fatal("a runner claimed a run that was already terminal")
	}
	if !strings.Contains(err.Error(), "not pending") {
		t.Fatalf("error = %v, want it to name the reason", err)
	}
	if calls := fake.snapshotCalls(); len(calls) != 0 {
		t.Fatalf("a refused runner mutated the host: %#v", calls)
	}
}

func TestConcurrentApplyRunnerHandoffsExecuteRunOnce(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)

	fake := &blockingApplyExecutor{started: make(chan struct{}), release: make(chan struct{})}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })

	run := applyRun{
		ID:        "apply-concurrent-handoff",
		Status:    applyRunPending,
		StartedAt: operatorStateNow().UTC().Format(time.RFC3339),
		Items:     []applyItemResult{{applyItem: applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"}, Outcome: applyItemPending}},
	}
	if err := persistApplyRun(run); err != nil {
		t.Fatalf("persist run: %v", err)
	}

	first := make(chan error, 1)
	go func() { first <- runApplyRunner(context.Background(), run.ID) }()
	select {
	case <-fake.started:
	case <-time.After(time.Second):
		t.Fatal("first runner did not reach the host effect")
	}

	second := make(chan error, 1)
	go func() { second <- runApplyRunner(context.Background(), run.ID) }()
	// The duplicate handoff should leave immediately while the first runner
	// keeps owning the run. Without the run lock, the target lock serializes both
	// processes but the second still executes from its stale pending snapshot.
	select {
	case err := <-second:
		if err != nil {
			t.Fatalf("duplicate runner handoff: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("duplicate runner waited instead of exiting while the run was owned")
	}
	if got := fake.calls.Load(); got != 1 {
		t.Fatalf("host calls while the first runner owns the run = %d, want 1", got)
	}
	close(fake.release)
	if err := <-first; err != nil {
		t.Fatalf("first runner: %v", err)
	}
	if got := fake.calls.Load(); got != 1 {
		t.Fatalf("host effects = %d, want exactly 1", got)
	}
}

func TestInterruptedBetweenItemsCanRetryRemainingState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)
	previousAlive := processAliveObserved
	processAliveObserved = func(int) (bool, error) { return false, nil }
	t.Cleanup(func() { processAliveObserved = previousAlive })

	stoppedRun := observedApplyRun(applyRun{
		ID: "interrupted-between-items", Target: "local", Status: applyRunApplying,
		RunnerPID: 4242, Heartbeat: operatorStateNow().UTC().Add(-2 * staleApplyHeartbeat).Format(time.RFC3339),
		Items: []applyItemResult{
			{applyItem: applyItem{ID: "tool:git", Kind: "tool", Name: "git"}, Outcome: applyItemApplied},
			{applyItem: applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"}, Outcome: applyItemPending},
		},
	})
	if stoppedRun.Status != applyRunInterrupted || stoppedRun.Items[0].Outcome != applyItemApplied || stoppedRun.Items[1].Outcome != applyItemPending {
		t.Fatalf("interrupted-between-items projection = %+v", stoppedRun)
	}

	fake := &recordingApplyExecutor{failOn: "resource:postgres"}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })

	run, err := (&Server{}).buildApplyRun(context.Background(), applydomain.Plan{
		Target: "local", Digest: "resume-plan",
		Items: []applydomain.Item{
			{ID: "tool:git", Kind: "tool", Name: "git", ObservedState: applyStateSatisfied},
			{ID: "resource:postgres", Kind: "resource", Name: "postgres", ObservedState: applyStateUnknown},
		},
	})
	if err != nil {
		t.Fatalf("build apply run: %v", err)
	}
	if run.Items[0].Outcome != applyItemAlreadySatisfied || run.Items[1].Outcome != applyItemPending {
		t.Fatalf("run outcomes = %#v, want observed item satisfied and unknown item pending", run.Items)
	}
	if err := executeApplyRun(context.Background(), run); err != nil {
		t.Fatalf("execute apply run: %v", err)
	}
	if calls := fake.snapshotCalls(); len(calls) != 1 || calls[0] != "resource:postgres" {
		t.Fatalf("control-plane effects = %#v, want only the unknown resource action", calls)
	}
	finished, ok := applyRunSnapshot(run.ID)
	if !ok || finished.Status != applyRunPartiallyApplied || finished.Items[0].Outcome != applyItemAlreadySatisfied {
		t.Fatalf("persisted recovery run = %+v, want satisfied item retained and pending work reported", finished)
	}
}

// TestInterruptedRecoveryRechecksDurableEffectsBeforeExecutingRemainingSetup
// joins the recovery owners: a durable interrupted journal is projected to
// the user-facing uncertainty state, fresh typed readback controls replay
// admission, and the approved retry executes only newly pending setup work.
// The executor is fake; this proves the source contract, not a host effect.
func TestInterruptedRecoveryRechecksDurableEffectsBeforeExecutingRemainingSetup(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)

	previousAlive := processAliveObserved
	processAliveObserved = func(int) (bool, error) { return false, nil }
	t.Cleanup(func() { processAliveObserved = previousAlive })

	interrupted := applyRun{
		ID: "recovered-setup-run", Target: "local", Status: applyRunApplying,
		RunnerPID: 4242, Heartbeat: operatorStateNow().UTC().Add(-2 * staleApplyHeartbeat).Format(time.RFC3339),
		Items: []applyItemResult{
			{applyItem: applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"}, Outcome: applyItemApplying},
			{applyItem: applyItem{ID: "scenario:writer", Kind: "scenario", Name: "writer"}, Outcome: applyItemApplying},
		},
	}
	if err := persistApplyRun(interrupted); err != nil {
		t.Fatalf("persist interrupted setup: %v", err)
	}
	projected, err := (&Server{}).getApplyRun(context.Background(), interrupted.ID)
	if err != nil || projected.Status != string(applyRunInterrupted) {
		t.Fatalf("recovery projection = %+v, %v; want interrupted run visible to caller", projected, err)
	}

	closure := closureResult{
		Resources: []closureMember{{Name: "postgres"}},
		Scenarios: []closureMember{{Name: "writer"}},
	}
	uncertain := map[string]struct{}{"resource:postgres": {}, "scenario:writer": {}}
	reader := fakeApplyPlanStatusReader{
		resources: map[string]*cliv1.ResourceStatusResponse{
			"postgres": {Success: true, Name: "postgres", Resource: &cliv1.ResourceStatus{Resource: &cliv1.Resource{Name: "postgres", Enabled: true}}},
		},
		scenarios: &cliv1.ScenarioStatusListResponse{Success: true, Scenarios: []*cliv1.ScenarioStatusItem{{Name: "writer", Status: "stopped"}}},
	}
	observePlan := func() []applydomain.Item {
		observed := observeApplyStates(context.Background(), "local", t.TempDir(), hostRequirementsResponse{}, closure, uncertain, reader)
		items := buildApplyPlan(applyPlanInput{
			Closure:      closure,
			Requirements: hostRequirementsResponse{Tools: []hostItem{{hostRequirement: hostRequirement{Name: "git"}, Status: "required"}}},
			Observed:     map[string]string{"tool:git": applyStatePending, "resource:postgres": observedState(observed, "resource:postgres"), "scenario:writer": observedState(observed, "scenario:writer")},
		})
		planItems := make([]applydomain.Item, 0, len(items))
		for _, item := range items {
			planItems = append(planItems, toApplyDomainItem(item))
		}
		return planItems
	}

	partialPlan := observePlan()
	if err := checkApplyReplayEligibility("local", partialPlan); err == nil || !strings.Contains(err.Error(), "scenario:writer") {
		t.Fatalf("partial target recheck admitted retry: %v; want unresolved scenario to remain blocked", err)
	}
	reader.scenarios = &cliv1.ScenarioStatusListResponse{Success: true, Scenarios: []*cliv1.ScenarioStatusItem{{Name: "writer", Status: "running"}}}
	refreshedPlan := observePlan()
	if err := checkApplyReplayEligibility("local", refreshedPlan); err != nil {
		t.Fatalf("positive target recheck did not admit retry: %v", err)
	}

	fake := &recordingApplyExecutor{}
	previousExecutor := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previousExecutor })
	run, err := (&Server{}).buildApplyRun(context.Background(), applydomain.Plan{Target: "local", Digest: "refreshed-recovery-plan", Items: refreshedPlan})
	if err != nil {
		t.Fatalf("build retry from refreshed setup plan: %v", err)
	}
	if err := executeApplyRun(context.Background(), run); err != nil {
		t.Fatalf("execute fake setup retry: %v", err)
	}
	if calls := fake.snapshotCalls(); len(calls) != 1 || calls[0] != "tool:git" {
		t.Fatalf("retry effects = %#v; want only newly pending setup work", calls)
	}

	durable, err := loadPersistedApplyRun(interrupted.ID)
	if err != nil || durable.Items[0].Outcome != applyItemApplying || durable.Items[1].Outcome != applyItemApplying {
		t.Fatalf("recheck rewrote the uncertain journal: %+v, %v", durable.Items, err)
	}
}

type blockingApplyExecutor struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (e *blockingApplyExecutor) InstallTool(context.Context, string) error    { return nil }
func (e *blockingApplyExecutor) ApplySafeguard(context.Context, string) error { return nil }
func (e *blockingApplyExecutor) EnableResource(context.Context, string) error {
	e.calls.Add(1)
	close(e.started)
	<-e.release
	return nil
}
func (e *blockingApplyExecutor) StartScenario(context.Context, string) error { return nil }

// TestObservedApplyRunReportsAnAbandonedRun is the other half of surviving a
// restart. A run whose executor was killed leaves "applying" behind, and a
// client polling that status waits forever for a process that no longer exists.
func TestObservedApplyRunReportsAnAbandonedRun(t *testing.T) {
	fresh := operatorStateNow().UTC().Format(time.RFC3339)
	stale := operatorStateNow().UTC().Add(-2 * staleApplyHeartbeat).Format(time.RFC3339)

	previousAlive := processAliveObserved
	t.Cleanup(func() { processAliveObserved = previousAlive })

	t.Run("a working runner is left alone", func(t *testing.T) {
		processAliveObserved = func(int) (bool, error) { return true, nil }
		got := observedApplyRun(applyRun{Status: "applying", Heartbeat: fresh, RunnerPID: 4242})
		if got.Status != "applying" {
			t.Fatalf("status = %q, want applying", got.Status)
		}
	})

	t.Run("a slow runner that is still alive is left alone", func(t *testing.T) {
		// Both signals are required. A heartbeat can lag on a slow filesystem,
		// and burying a live run would lose work that is still in progress.
		processAliveObserved = func(int) (bool, error) { return true, nil }
		got := observedApplyRun(applyRun{Status: "applying", Heartbeat: stale, RunnerPID: 4242})
		if got.Status != "applying" {
			t.Fatalf("status = %q, want applying for a stale heartbeat with a live process", got.Status)
		}
	})

	t.Run("a run accepted moments ago is not yet judged", func(t *testing.T) {
		processAliveObserved = func(int) (bool, error) { return false, nil }
		got := observedApplyRun(applyRun{Status: "pending"})
		if got.Status != "pending" {
			t.Fatalf("status = %q, want pending; the runner is still starting", got.Status)
		}
	})

	t.Run("a dead runner is reported as interrupted", func(t *testing.T) {
		processAliveObserved = func(int) (bool, error) { return false, nil }
		got := observedApplyRun(applyRun{Status: "applying", Heartbeat: stale, RunnerPID: 4242, Items: []applyItemResult{
			{applyItem: applyItem{ID: "tool:git", Kind: "tool", Name: "git"}, Outcome: applyItemApplied},
			{applyItem: applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"}, Outcome: applyItemApplying},
			{applyItem: applyItem{ID: "scenario:alpha", Kind: "scenario", Name: "alpha"}, Outcome: applyItemPending},
		}})
		if got.Status != "interrupted" {
			t.Fatalf("status = %q, want interrupted", got.Status)
		}
		if got.Error == "" {
			t.Fatal("an interrupted run must say what happened")
		}
		if len(got.Blockers) != 2 || !strings.Contains(got.Blockers[len(got.Blockers)-1].Remediation, "apply the retained selection again") {
			t.Fatalf("blockers = %#v, want a recovery instruction", got.Blockers)
		}
		if len(got.Items) != 3 || got.Items[0].Outcome != applyItemApplied || got.Items[1].Outcome != applyItemIndeterminate || got.Items[2].Outcome != applyItemPending {
			t.Fatalf("per-item interruption projection = %#v, want applied / indeterminate / pending", got.Items)
		}
		if got.Items[1].ErrorCode != "effect_outcome_unknown" || !strings.Contains(got.Items[1].Error, "cannot confirm whether its effect completed") {
			t.Fatalf("uncertain item = %+v, want explicit unknown-effect reason", got.Items[1])
		}
		if got.Items[1].CompletedAt != "" {
			t.Fatalf("uncertain item completed_at = %q, must remain unset", got.Items[1].CompletedAt)
		}
	})

	t.Run("an unavailable liveness observation cannot prove interruption", func(t *testing.T) {
		processAliveObserved = func(int) (bool, error) { return false, os.ErrPermission }
		got := observedApplyRun(applyRun{Status: "applying", Heartbeat: stale, RunnerPID: 4242})
		if got.Status != "applying" {
			t.Fatalf("status = %q, want applying while process absence is unverified", got.Status)
		}
	})

	t.Run("a terminal run is never second-guessed", func(t *testing.T) {
		processAliveObserved = func(int) (bool, error) { return false, nil }
		got := observedApplyRun(applyRun{Status: "applied", Heartbeat: stale, RunnerPID: 4242})
		if got.Status != "applied" {
			t.Fatalf("status = %q, want applied", got.Status)
		}
	})
}
