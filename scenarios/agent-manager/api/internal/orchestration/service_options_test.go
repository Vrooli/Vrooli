package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

func TestOrchestratorOptionsApplyAndPreserveConstructorInvariants(t *testing.T) {
	clock := func() time.Time { return time.Unix(123, 0) }
	dispatcher := spawn.New(spawn.Config{MaxStartingConcurrency: 1, QueueCapacity: 3})
	config := OrchestratorConfig{DefaultTimeout: 2 * time.Minute, MaxConcurrentRuns: 3, RequireSandboxByDefault: false}
	o := New(nil, nil, nil,
		WithConfig(config),
		WithClock(clock),
		WithRunners(nil),
		WithSandbox(nil),
		WithWorkspaceSandboxEnsurer(nil),
		WithPolicy(nil),
		WithEvents(nil),
		WithArtifacts(nil),
		WithLocks(nil),
		WithCheckpoints(nil),
		WithIdempotency(nil),
		WithWorkflowRepository(nil),
		WithWorkflowExecutionRepository(nil),
		WithBroadcaster(nil),
		WithTerminator(nil),
		WithStorageLabel("  sqlite  "),
		WithRolePolicyState(nil, nil),
		WithStructuredExtractor(nil),
		WithHealthStore(nil),
		WithInvestigationSettings(nil),
		WithFlagValidator(nil),
		WithPromptClient(nil),
		WithAttachmentStorage(nil),
		WithOrchestrationSettings(nil),
		WithIdentitySecret([]byte("test-secret")),
		WithSpawnDispatcher(dispatcher),
		WithInteractiveSessions(nil),
		WithWebConsoleUIBase("https://console.example"),
	)

	if o.config != config || o.clock != nil && !o.now().Equal(clock()) {
		t.Fatalf("config or clock option was not retained: %#v", o.config)
	}
	if o.storageLabel != "sqlite" || o.webConsoleUIBase != "https://console.example" {
		t.Fatalf("string options were not normalized/preserved: label=%q ui=%q", o.storageLabel, o.webConsoleUIBase)
	}
	if string(o.identitySecret) != "test-secret" || o.dispatcher != dispatcher {
		t.Fatal("identity or dispatcher option was not retained")
	}
	if o.workflowWaiters == nil || o.interactiveDrivers == nil || o.structuredResults == nil {
		t.Fatal("constructor invariants were not initialized")
	}
	if got := o.SpawnStats(); got.QueueDepth != 0 || got.ActiveCount != 0 || got.StartingCount != 0 {
		t.Fatalf("SpawnStats = %#v, want an idle configured dispatcher", got)
	}

	var nilClock func() time.Time
	WithClock(nilClock)(o)
	if !o.now().Equal(clock()) {
		t.Fatal("nil clock must not replace the configured clock")
	}
}

func TestResumeRunExecutesPendingInPlaceRunThroughDispatcher(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	now := time.Now()
	task := &domain.Task{ID: uuid.New(), Title: "resume", ScopePath: "src", ProjectRoot: t.TempDir(), Status: domain.TaskStatusQueued, CreatedAt: now, UpdatedAt: now}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, Tag: "resume-run", RunMode: domain.RunModeInPlace,
		Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued, ApprovalState: domain.ApprovalStateNone,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode}, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	registry := runner.NewRegistry()
	if err := registry.Register(mock); err != nil {
		t.Fatal(err)
	}
	svc := New(repos.Profiles, repos.Tasks, repos.Runs,
		WithEvents(events), WithRunners(registry),
		WithConfig(OrchestratorConfig{DefaultTimeout: time.Minute, MaxConcurrentRuns: 1}),
	)
	accepted, err := svc.ResumeRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != domain.RunStatusRunning {
		t.Fatalf("accepted status = %s, want running", accepted.Status)
	}
	deadline := time.Now().Add(5 * time.Second)
	var last *domain.Run
	for time.Now().Before(deadline) {
		stored, err := repos.Runs.Get(ctx, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if stored.Status == domain.RunStatusComplete {
			return
		}
		last = stored
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("resumed run did not reach complete: %#v", last)
}

func TestOrchestratorReadOnlyRunOperationsCoverConfiguredAndMissingSeams(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	now := time.Now()
	heartbeat := now.Add(-2 * time.Hour)
	task := &domain.Task{ID: uuid.New(), Title: "read operations", ScopePath: "src", Status: domain.TaskStatusQueued, CreatedAt: now, UpdatedAt: now}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, Tag: "read-operations", Status: domain.RunStatusRunning,
		Phase: domain.RunPhaseExecuting, ProgressPercent: 55, StartedAt: &heartbeat, LastHeartbeat: &heartbeat,
		CreatedAt: now, UpdatedAt: now, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	svc := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events), WithConfig(OrchestratorConfig{DefaultProjectRoot: "/project"}))

	progress, err := svc.GetRunProgress(ctx, run.ID)
	if err != nil || progress.CurrentAction != "Agent is working on the task" || progress.PercentComplete != 55 || progress.ElapsedTime <= 0 {
		t.Fatalf("progress=%#v err=%v", progress, err)
	}
	stale, err := svc.ListStaleRuns(ctx, time.Hour)
	if err != nil || len(stale) != 1 || stale[0].ID != run.ID {
		t.Fatalf("stale=%#v err=%v", stale, err)
	}
	if err := events.Append(ctx, run.ID, domain.NewLogEvent(run.ID, "info", "event")); err != nil {
		t.Fatal(err)
	}
	gotEvents, err := svc.GetRunEvents(ctx, run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(gotEvents) != 1 {
		t.Fatalf("events=%#v err=%v", gotEvents, err)
	}
	if _, err := svc.GetRunDiff(ctx, run.ID); err == nil {
		t.Fatal("GetRunDiff without sandbox unexpectedly succeeded")
	}
	if _, err := svc.ValidatePath(ctx, "src", "/project"); err == nil {
		t.Fatal("ValidatePath without provider unexpectedly succeeded")
	}
	health, err := svc.GetModelHealthSnapshot(ctx)
	if err != nil || len(health.Models) != 0 || len(health.Runners) != 0 {
		t.Fatalf("health=%#v err=%v", health, err)
	}
	if svc.GetDefaultProjectRoot() != "/project" {
		t.Fatal("default project root was not preserved")
	}
	policy, err := svc.ExplainRunPolicy(ctx, run.ID)
	if err != nil || policy != nil {
		t.Fatalf("policy=%#v err=%v", policy, err)
	}
	withoutEvents := New(repos.Profiles, repos.Tasks, repos.Runs)
	if _, err := withoutEvents.GetRunEvents(ctx, run.ID, event.GetOptions{}); err == nil {
		t.Fatal("GetRunEvents without event store unexpectedly succeeded")
	}
}

func TestDefaultConfigIsSafeForUnconfiguredOrchestrators(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultTimeout <= 0 || cfg.MaxConcurrentRuns < 1 || !cfg.RequireSandboxByDefault {
		t.Fatalf("DefaultConfig = %#v, want positive timeout/concurrency and sandbox protection", cfg)
	}
	o := New(nil, nil, nil)
	if o.dispatcher == nil {
		t.Fatal("default constructor must install a dispatcher")
	}
}
