package orchestration_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
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
		orchestration.WithRunStateRoot(t.TempDir()),
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

func TestContinueRun_FailurePreservesPreviousStructuredResult(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	mockRunner, _, _, done := newContinuationRunner(t)
	mockRunner.ContinueFunc = func(_ context.Context, _ runner.ContinueRequest) (*runner.ExecuteResult, error) {
		close(done)
		return nil, errors.New("no rollout found for thread id test-thread")
	}
	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatal(err)
	}
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: time.Minute, MaxConcurrentRuns: 1, RequireSandboxByDefault: false}),
		orchestration.WithEvents(eventStore), orchestration.WithRunners(registry), orchestration.WithRunStateRoot(t.TempDir()))
	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{Name: "preserve-result", ProfileKey: "preserve-result-" + uuid.NewString()[:8], RoleRef: "code.default", SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}})
	task := mustCreateTask(t, svc, ctx, &domain.Task{Title: "preserve-result", ScopePath: "src/"})
	now := time.Now()
	raw := []byte(`{"outcome":"complete"}`)
	run := &domain.Run{
		ID: uuid.New(), TaskID: task.ID, AgentProfileID: &profile.ID, Tag: "preserve-result", RunMode: domain.RunModeInPlace,
		Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, SessionID: "test-thread", StartedAt: &now, EndedAt: &now,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode}, ApprovalState: domain.ApprovalStateNone,
		Result:  &domain.RunResult{Success: true, Structured: &domain.StructuredResult{Status: domain.StructuredResultSuccess, Value: raw}},
		Summary: &domain.RunSummary{TurnsUsed: 1}, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ContinueRun(ctx, orchestration.ContinueRunRequest{RunID: run.ID, Message: "continue"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("continuation did not run")
	}
	var got *domain.Run
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var err error
		got, err = repos.Runs.Get(ctx, run.ID)
		if err == nil && got.Status == domain.RunStatusFailed {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got == nil || got.Status != domain.RunStatusFailed {
		t.Fatalf("run status = %#v, want failed", got)
	}
	if got.Result == nil || got.Result.Structured == nil || string(got.Result.Structured.Value) != string(raw) {
		t.Fatalf("previous structured result was lost: %#v", got.Result)
	}
	events, err := eventStore.Get(ctx, run.ID, event.GetOptions{})
	if err != nil {
		t.Fatalf("get events: %v", err)
	}
	found := false
	for _, evt := range events {
		data, ok := evt.Data.(*domain.LogEventData)
		if ok && strings.Contains(data.Message, "failed on turn 2; preserved structured result from successful turn 1") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing numbered partial-result preservation event: %#v", events)
	}
}
