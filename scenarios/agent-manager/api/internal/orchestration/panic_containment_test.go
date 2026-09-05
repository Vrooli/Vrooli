package orchestration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

// A panic inside the continuation runner turn must be contained: the run
// reaches the normal failed terminal state (with the panic recorded) instead
// of the goroutine killing the whole API.
func TestContinuation_RunnerPanicFailsRunWithoutCrash(t *testing.T) {
	ctx := context.Background()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "available")
	mockRunner.SetCapabilities(runner.Capabilities{
		SupportsMessages:     true,
		SupportsStreaming:    true,
		SupportsCancellation: true,
		SupportsContinuation: true,
		MaxTurns:             100,
		SupportedModels:      []string{"mock-model"},
	})
	mockRunner.ContinueFunc = func(ctx context.Context, req runner.ContinueRequest) (*runner.ExecuteResult, error) {
		panic("simulated runner defect during continuation")
	}

	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: false,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
	)

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:       "panic-containment-profile",
		ProfileKey: "panic-containment-" + uuid.New().String()[:8],

		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, RoleRef: "code.default",
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "panic-containment-task",
		Description: "task for continuation panic containment test",
		ScopePath:   "src/",
	})

	now := time.Now()
	runID := uuid.New()
	directRun := &domain.Run{
		ID:             runID,
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Tag:            runID.String(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusComplete,
		Phase:          domain.RunPhaseCompleted,
		SessionID:      "sess-original",
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
		},
		ApprovalState: domain.ApprovalStateNone,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repos.Runs.Create(ctx, directRun); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if _, err := svc.ContinueRun(ctx, orchestration.ContinueRunRequest{
		RunID:   runID,
		Message: "please continue",
	}); err != nil {
		t.Fatalf("ContinueRun: %v", err)
	}

	// The contained panic must drive the run to the failed terminal state.
	deadline := time.Now().Add(10 * time.Second)
	var got *domain.Run
	for time.Now().Before(deadline) {
		var err error
		got, err = repos.Runs.Get(ctx, runID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}
		if got.Status == domain.RunStatusFailed {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got == nil || got.Status != domain.RunStatusFailed {
		status := domain.RunStatus("<missing>")
		if got != nil {
			status = got.Status
		}
		t.Fatalf("run status = %q, want %q after contained continuation panic", status, domain.RunStatusFailed)
	}
	if !strings.Contains(got.ErrorMsg, "panic") {
		t.Errorf("run.ErrorMsg = %q, want it to record the recovered panic", got.ErrorMsg)
	}
}
