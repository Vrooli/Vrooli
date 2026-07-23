package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/testutil"
	"agent-manager/internal/testutil/fixtures"

	"github.com/google/uuid"
)

func TestPurgeDataDryRunThenDeletesMatchingDurableRecords(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(eventStore))

	removeProfile := fixtures.NewAgentProfile(t, fixtures.WithAgentProfileName("remove-profile"))
	keepProfile := fixtures.NewAgentProfile(t, fixtures.WithAgentProfileName("keep-profile"))
	if err := repos.Profiles.Create(ctx, removeProfile); err != nil {
		t.Fatalf("create removable profile: %v", err)
	}
	if err := repos.Profiles.Create(ctx, keepProfile); err != nil {
		t.Fatalf("create retained profile: %v", err)
	}
	removeTask := fixtures.NewTask(t, fixtures.WithTaskTitle("remove task"))
	keepTask := fixtures.NewTask(t, fixtures.WithTaskTitle("keep task"))
	if err := repos.Tasks.Create(ctx, removeTask); err != nil {
		t.Fatalf("create removable task: %v", err)
	}
	if err := repos.Tasks.Create(ctx, keepTask); err != nil {
		t.Fatalf("create retained task: %v", err)
	}
	removeRun := fixtures.NewRun(t, removeTask.ID, removeProfile.ID)
	removeRun.Tag = "remove-run"
	keepRun := fixtures.NewRun(t, keepTask.ID, keepProfile.ID)
	keepRun.Tag = "keep-run"
	if err := repos.Runs.Create(ctx, removeRun); err != nil {
		t.Fatalf("create removable run: %v", err)
	}
	if err := repos.Runs.Create(ctx, keepRun); err != nil {
		t.Fatalf("create retained run: %v", err)
	}
	if err := eventStore.Append(ctx, removeRun.ID, fixtures.NewRunEvent(t, removeRun.ID)); err != nil {
		t.Fatalf("append removable event: %v", err)
	}

	req := PurgeRequest{Pattern: "^remove", Targets: []PurgeTarget{PurgeTargetProfiles, PurgeTargetTasks, PurgeTargetRuns}, DryRun: true}
	dryRun, err := o.PurgeData(ctx, req)
	if err != nil {
		t.Fatalf("dry-run purge: %v", err)
	}
	if dryRun.Matched != (PurgeCounts{Profiles: 1, Tasks: 1, Runs: 1}) || dryRun.Deleted != (PurgeCounts{}) {
		t.Fatalf("dry-run result = %+v, want one match per target and no deletion", dryRun)
	}
	if got, err := repos.Runs.Get(ctx, removeRun.ID); err != nil || got == nil {
		t.Fatalf("dry run removed run: got=%+v err=%v", got, err)
	}

	req.DryRun = false
	deleted, err := o.PurgeData(ctx, req)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if deleted.Deleted != (PurgeCounts{Profiles: 1, Tasks: 1, Runs: 1}) {
		t.Fatalf("deleted counts = %+v, want one deletion per target", deleted.Deleted)
	}
	if got, err := repos.Runs.Get(ctx, removeRun.ID); err != nil || got != nil {
		t.Fatalf("removed run = %+v err=%v, want nil", got, err)
	}
	if got, err := repos.Tasks.Get(ctx, removeTask.ID); err != nil || got != nil {
		t.Fatalf("removed task = %+v err=%v, want nil", got, err)
	}
	if got, err := repos.Profiles.Get(ctx, removeProfile.ID); err != nil || got != nil {
		t.Fatalf("removed profile = %+v err=%v, want nil", got, err)
	}
	if events, err := eventStore.Get(ctx, removeRun.ID, event.GetOptions{}); err != nil || len(events) != 0 {
		t.Fatalf("removed run events = %+v err=%v, want none", events, err)
	}
	if got, err := repos.Runs.Get(ctx, keepRun.ID); err != nil || got == nil || got.Status != domain.RunStatusPending {
		t.Fatalf("retained run = %+v err=%v", got, err)
	}
}

func TestPurgeDataRejectsIncompleteRequests(t *testing.T) {
	o := newDeclarationOrchestrator(t)
	ctx := context.Background()
	for _, req := range []PurgeRequest{{}, {Pattern: "[", Targets: []PurgeTarget{PurgeTargetRuns}}, {Pattern: "run"}} {
		if _, err := o.PurgeData(ctx, req); err == nil {
			t.Fatalf("PurgeData(%+v) succeeded, want validation error", req)
		}
	}
}

func TestOptionalOperationalSurfacesReturnSafeDefaultsWhenUnconfigured(t *testing.T) {
	o := New(nil, nil, nil, WithConfig(OrchestratorConfig{DefaultProjectRoot: "/project"}))
	ctx := context.Background()

	snapshot, err := o.GetModelHealthSnapshot(ctx)
	if err != nil || len(snapshot.Models) != 0 || len(snapshot.Runners) != 0 {
		t.Fatalf("empty health snapshot=%+v err=%v", snapshot, err)
	}
	if got := o.GetDefaultProjectRoot(); got != "/project" {
		t.Fatalf("default project root=%q", got)
	}
	if _, err := o.ValidatePath(ctx, "src", "/project"); err == nil {
		t.Fatal("ValidatePath succeeded without sandbox provider")
	}
	probe, err := o.ProbeRunner(ctx, domain.RunnerTypeCodex)
	if err != nil || probe.Success || probe.Message != "no runner registry configured" {
		t.Fatalf("probe=%+v err=%v", probe, err)
	}
	if _, err := o.StreamRunEvents(ctx, uuid.New(), event.StreamOptions{}); err == nil {
		t.Fatal("StreamRunEvents succeeded without event store")
	}

	settings, err := o.GetInvestigationSettings(ctx)
	if err != nil || settings == nil {
		t.Fatalf("default investigation settings=%+v err=%v", settings, err)
	}
	if err := o.UpdateInvestigationSettings(ctx, settings); err == nil {
		t.Fatal("UpdateInvestigationSettings succeeded without repository")
	}
	if err := o.ResetInvestigationSettings(ctx); err == nil {
		t.Fatal("ResetInvestigationSettings succeeded without repository")
	}

	orchSettings, err := o.GetOrchestrationSettings(ctx)
	if err != nil || orchSettings == nil {
		t.Fatalf("default orchestration settings=%+v err=%v", orchSettings, err)
	}
	if err := o.UpdateOrchestrationSettings(ctx, orchSettings); err == nil {
		t.Fatal("UpdateOrchestrationSettings succeeded without store")
	}
	if err := o.ResetOrchestrationSettings(ctx); err == nil {
		t.Fatal("ResetOrchestrationSettings succeeded without store")
	}
	defaults := agentconfig.DefaultOrchestrationSettings()
	if defaults.RunExecution.MaxConcurrentRuns == 0 {
		t.Fatal("orchestration defaults unexpectedly empty")
	}
}

func TestGetHealthReportsMissingCoreDependenciesAsDegraded(t *testing.T) {
	o := New(nil, nil, nil)
	status, err := o.GetHealth(context.Background())
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	if status.Status != "degraded" || status.Readiness {
		t.Fatalf("health status=%+v, want degraded and unready", status)
	}
	if status.Dependencies.Database == nil || status.Dependencies.Database.Connected {
		t.Fatalf("database dependency=%+v, want disconnected", status.Dependencies.Database)
	}
	if status.Dependencies.WorkflowRuntime == nil || status.Dependencies.WorkflowRuntime.Connected {
		t.Fatalf("workflow dependency=%+v, want disconnected", status.Dependencies.WorkflowRuntime)
	}
	if status.Dependencies.Sandbox == nil || status.Dependencies.Sandbox.Connected {
		t.Fatalf("sandbox dependency=%+v, want disconnected", status.Dependencies.Sandbox)
	}
	if len(status.Dependencies.Runners) != 0 {
		t.Fatalf("runners=%+v, want empty without registry", status.Dependencies.Runners)
	}
	runners, err := o.GetRunnerStatus(context.Background())
	if err != nil || runners != nil {
		t.Fatalf("runner statuses=%+v err=%v, want nil", runners, err)
	}
}

func TestProbeRunnerUsesSupportedCLIContractsWithoutLiveAgents(t *testing.T) {
	binDir := t.TempDir()
	writeProbeCommand := func(name, script string) {
		t.Helper()
		path := filepath.Join(binDir, name)
		if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
			t.Fatalf("write %s stub: %v", name, err)
		}
	}
	writeProbeCommand("claude", "#!/bin/sh\nprintf 'PROBE_OK\\n'\n")
	writeProbeCommand("opencode", "#!/bin/sh\nprintf 'PROBE_OK\\n'\n")
	writeProbeCommand("grok", "#!/bin/sh\nprintf 'PROBE_OK\\n'\n")
	writeProbeCommand("codex", "#!/bin/sh\nprevious=''\nfor arg in \"$@\"; do\n  if [ \"$previous\" = '-o' ]; then\n    printf 'PROBE_OK\\n' > \"$arg\"\n    exit 0\n  fi\n  previous=\"$arg\"\ndone\nprintf 'missing -o output path' >&2\nexit 1\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	registry := runner.NewRegistry()
	for _, runnerType := range []domain.RunnerType{domain.RunnerTypeClaudeCode, domain.RunnerTypeCodex, domain.RunnerTypeOpenCode, domain.RunnerTypeGrok} {
		mock := runner.NewMockRunner(runnerType)
		mock.SetAvailable(true, "available")
		if err := registry.Register(mock); err != nil {
			t.Fatalf("register %s: %v", runnerType, err)
		}
	}
	o := New(nil, nil, nil, WithRunners(registry))
	for _, runnerType := range []domain.RunnerType{domain.RunnerTypeClaudeCode, domain.RunnerTypeCodex, domain.RunnerTypeOpenCode, domain.RunnerTypeGrok} {
		result, err := o.ProbeRunner(context.Background(), runnerType)
		if err != nil || !result.Success || result.Response != "PROBE_OK" {
			t.Fatalf("probe %s result=%+v err=%v", runnerType, result, err)
		}
	}
	unknown, err := o.ProbeRunner(context.Background(), domain.RunnerType("missing"))
	if err != nil || unknown.Success {
		t.Fatalf("unknown probe=%+v err=%v", unknown, err)
	}
}

func TestProbeRunnerReportsUnavailableRegisteredRunner(t *testing.T) {
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeCodex)
	mock.SetAvailable(false, "not installed")
	if err := registry.Register(mock); err != nil {
		t.Fatalf("register runner: %v", err)
	}
	result, err := New(nil, nil, nil, WithRunners(registry)).ProbeRunner(context.Background(), domain.RunnerTypeCodex)
	if err != nil || result.Success || result.Message != "not installed" {
		t.Fatalf("unavailable probe=%+v err=%v", result, err)
	}
}

func TestExplainRunPolicyReturnsOnlyPersistedSnapshot(t *testing.T) {
	ctx := context.Background()
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	o := New(repos.Profiles, repos.Tasks, repos.Runs)
	task := fixtures.NewTask(t)
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	snapshot := &domain.ExecutionPolicySnapshot{CatalogDigest: "sha256:catalog", RoleRef: "code.review", Explanation: domain.PolicyResolutionExplanation{Source: "persisted", Summary: "chosen before run creation"}}
	withSnapshot := fixtures.NewRun(t, task.ID, uuid.Nil)
	withSnapshot.ResolvedConfig = &domain.RunConfig{PolicySnapshot: snapshot}
	withoutSnapshot := fixtures.NewRun(t, task.ID, uuid.Nil)
	if err := repos.Runs.Create(ctx, withSnapshot); err != nil {
		t.Fatalf("create run with snapshot: %v", err)
	}
	if err := repos.Runs.Create(ctx, withoutSnapshot); err != nil {
		t.Fatalf("create run without snapshot: %v", err)
	}

	got, err := o.ExplainRunPolicy(ctx, withSnapshot.ID)
	if err != nil || got == nil || got.CatalogDigest != snapshot.CatalogDigest || got.Explanation.Source != "persisted" {
		t.Fatalf("persisted policy=%+v err=%v", got, err)
	}
	missing, err := o.ExplainRunPolicy(ctx, withoutSnapshot.ID)
	if err != nil || missing != nil {
		t.Fatalf("missing policy=%+v err=%v, want nil", missing, err)
	}
}
