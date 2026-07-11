package orchestration_test

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

// TestContinueRun_PreservesCustomEnvAndIdentity is the Phase 0 regression test
// for the latent "continue drops env + identity" bug. Before the fix,
// executeContinuation built a ContinueRequest with Environment=nil, so a
// continued turn lost the caller-supplied custom env AND its identity token
// (VerifyIdentityToken would then fail inside the woken turn). This test pins
// the desired behavior: the env handed to the runner on continue carries the
// persisted custom env and a freshly regenerated, verifiable identity token.
func TestContinueRun_PreservesCustomEnvAndIdentity(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	mockRunner, captured, captureMu, done := newContinuationRunner(t)
	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}

	identitySecret := []byte("phase0-test-identity-secret-0123456789")
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
		orchestration.WithRunners(registry),
		orchestration.WithIdentitySecret(identitySecret),
	)

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:       "continue-env-profile",
		ProfileKey: "continue-env-" + uuid.New().String()[:8],

		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}, RoleRef: "code.default",
	})
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "continue-env-task",
		Description: "task for env/identity continuation regression",
		ScopePath:   "src/",
	})

	// Insert a continuable run with persisted custom env, bypassing CreateRun
	// (which would trigger async execution). This mirrors a run created with
	// CreateRunRequest.Environment set.
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
		SessionID:      "sess-" + uuid.New().String()[:8],
		StartedAt:      &now,
		EndedAt:        &now,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode},
		ApprovalState:  domain.ApprovalStateNone,
		CustomEnv: map[string]string{
			"VROOLI_SHADOW_SCENARIOS":         "agent-manager",
			"VROOLI_SWARM_MANAGER_SESSION_ID": "sess-orig",
		},
		CreatedAt: now,
		UpdatedAt: now,
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

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runner.Continue")
	}

	captureMu.Lock()
	env := captured.Environment
	captureMu.Unlock()

	if env == nil {
		t.Fatal("continue env is nil — the latent bug regressed (env + identity dropped on continue)")
	}

	// Custom env must survive the continuation.
	if env["VROOLI_SHADOW_SCENARIOS"] != "agent-manager" {
		t.Errorf("custom env VROOLI_SHADOW_SCENARIOS dropped: %v", env)
	}
	if env["VROOLI_SWARM_MANAGER_SESSION_ID"] != "sess-orig" {
		t.Errorf("custom env VROOLI_SWARM_MANAGER_SESSION_ID dropped: %v", env)
	}

	// A fresh identity token must be present and verifiable against this run.
	token := env["VROOLI_AGENT_IDENTITY_TOKEN"]
	if token == "" {
		t.Fatal("identity token missing from continued turn env")
	}
	res, err := svc.VerifyIdentityToken(ctx, token)
	if err != nil {
		t.Fatalf("VerifyIdentityToken: %v", err)
	}
	if !res.Valid {
		t.Fatalf("continued-turn identity token is not valid: %q", res.Error)
	}
	if res.Claims == nil || res.Claims.RunID != runID {
		t.Errorf("identity claims do not bind to the run: %+v", res.Claims)
	}
}
