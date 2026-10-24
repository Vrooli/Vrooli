package orchestration_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"github.com/google/uuid"
)

func TestCreateRunRejectsIncompatibleExplicitSpawnBeforeAdmission(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.SetAvailable(true, "fixture runner")
	mock.SetCapabilities(runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{
		ExecutionMode: "interactive",
		SandboxModes:  []string{"off"},
	}}})
	if err := registry.Register(mock); err != nil {
		t.Fatalf("register mock runner: %v", err)
	}

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithEvents(events),
		orchestration.WithRunners(registry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		fixtureOwnerIdentityOption(),
		newTestRolePolicyOption(t),
	)
	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		ID:      uuid.New(),
		Name:    "explicit-spawn-policy",
		RoleRef: "code.default",
		SpawnPolicy: &domain.SpawnPolicy{
			AxisOrder:     []string{"executionMode", "sandboxMode"},
			ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive", "codec_pipe"}},
			SandboxMode:   domain.PreferenceAxis{Prefer: []string{"off", "protected"}},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		ID:        uuid.New(),
		Title:     "Explicit protected runner requirement",
		ScopePath: ".",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	runMode := domain.RunModeSandboxed
	_, err := svc.CreateRun(ctx, authenticatedCreateRunFixture(orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "must refuse before run admission",
		RunMode:        &runMode,
		ExecutionMode:  domain.ExecutionMode("codec_pipe"),
		SandboxConfig:  &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}))
	var validationErr *domain.ValidationError
	if !errors.As(err, &validationErr) || validationErr.Field != "spawnPolicy" {
		t.Fatalf("expected spawnPolicy validation refusal, got %T: %v", err, err)
	}
	if !strings.Contains(validationErr.Message, "explicit caller constraints") {
		t.Fatalf("expected explicit-capability mismatch cause, got %q", validationErr.Message)
	}

	runs, listErr := svc.ListRuns(ctx, orchestration.RunListOptions{})
	if listErr != nil {
		t.Fatalf("list runs after refused admission: %v", listErr)
	}
	if len(runs) != 0 {
		t.Fatalf("incompatible explicit spawn was refused only after creating runs: got %d", len(runs))
	}
}
