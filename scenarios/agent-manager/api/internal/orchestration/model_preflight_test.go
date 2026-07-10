package orchestration_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/modelpolicy"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

// TestOrchestrator_CreateRun_PreflightRejectsBadModel proves the
// resolveRunConfig preflight converts a runner ProbeModel failure into a
// clear validation error at run-creation time, instead of letting the model
// fail opaquely once the runner is launched.
func TestOrchestrator_CreateRun_PreflightRejectsBadModel(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	reg := runner.NewRegistry()
	policyState, err := modelpolicy.NewState(modelpolicy.ResolvePath(), modelpolicy.Requirement{Required: true})
	if err != nil {
		t.Fatalf("load model policy catalog: %v", err)
	}
	mr := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mr.SetAvailable(true, "mock runner available")
	mr.ProbeFunc = func(_ context.Context, modelID string) error {
		if modelID == "claude-sonnet-5" {
			return fmt.Errorf("model %q is not available in catalog", modelID)
		}
		return nil
	}
	mustRegisterRunner(t, reg, mr)

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          30 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: true,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(reg),
		orchestration.WithModelPolicyState(policyState),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
	)
	ctx := context.Background()

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "preflight-bad-model",
		RunnerType: domain.RunnerTypeClaudeCode,
		Model:      "claude-sonnet-5",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		ID:        uuid.New(),
		Title:     "preflight task",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	_, err = svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
	})
	if err == nil {
		t.Fatal("expected preflight to reject a run whose model fails ProbeModel")
	}
	if !strings.Contains(err.Error(), "not available in catalog") {
		t.Fatalf("expected the ProbeModel message to surface in the error, got: %v", err)
	}
}
