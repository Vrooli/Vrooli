package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/modelregistry"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

// stubResolver is a ModelChainResolver backed by a fixed chain, for in-package tests.
type stubResolver struct {
	chain modelregistry.PresetChain
}

func (s stubResolver) ResolvePreset(_ string, _ string) (modelregistry.PresetChain, bool) {
	if len(s.chain) == 0 {
		return nil, false
	}
	return append(modelregistry.PresetChain(nil), s.chain...), true
}

// fallbackHarness assembles a minimal RunExecutor capable of exercising
// executeAgentWithModelFallback without touching sandboxing or the full Execute loop.
type fallbackHarness struct {
	t        *testing.T
	executor *RunExecutor
	runner   *runner.MockRunner
	run      *domain.Run
	attempts []string
}

func newFallbackHarness(t *testing.T, runnerType domain.RunnerType, chain modelregistry.PresetChain, preset domain.ModelPreset) *fallbackHarness {
	t.Helper()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "fallback-test",
		RunnerType: runnerType,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "Fallback Task",
		Description: "Task used for model fallback tests",
		Status:      domain.TaskStatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	ctx := context.Background()
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	run := &domain.Run{
		ID:             uuid.New(),
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Status:         domain.RunStatusPending,
		Phase:          domain.RunPhaseQueued,
		RunMode:        domain.RunModeInPlace,
		ResolvedConfig: &domain.RunConfig{
			RunnerType:  runnerType,
			Model:       chain.Primary(),
			ModelPreset: preset,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("seed run: %v", err)
	}

	mock := runner.NewMockRunner(runnerType)
	executor := NewRunExecutor(
		repos.Runs,
		nil, // runner registry not used; we call executeAgent directly
		nil, // sandbox not used for in-place
		eventStore,
		run,
		task,
		profile,
		"prompt", "systemPrompt",
	).WithModelChainResolver(stubResolver{chain: chain})

	return &fallbackHarness{
		t:        t,
		executor: executor,
		runner:   mock,
		run:      run,
	}
}

// programResponses sets the mock runner's ExecuteFunc to produce the given sequence
// of outcomes, one per attempt, and records the model used each time.
func (h *fallbackHarness) programResponses(results []*runner.ExecuteResult) {
	index := 0
	h.runner.ExecuteFunc = func(_ context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		h.attempts = append(h.attempts, req.ResolvedConfig.Model)
		if index >= len(results) {
			h.t.Fatalf("mock runner invoked %d times, only %d responses programmed", index+1, len(results))
		}
		r := results[index]
		index++
		return r, nil
	}
}

func successResult() *runner.ExecuteResult {
	return &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Summary: &domain.RunSummary{
			Description: "ok",
		},
	}
}

func modelUnavailableResult(runnerType domain.RunnerType) *runner.ExecuteResult {
	msg := "error: unknown model 'does-not-exist'"
	if runnerType == domain.RunnerTypeCodex {
		msg = "error: model 'gpt-X-codex' is deprecated"
	}
	return &runner.ExecuteResult{
		Success:      false,
		ExitCode:     1,
		ErrorMessage: msg,
	}
}

func genericFailureResult() *runner.ExecuteResult {
	return &runner.ExecuteResult{
		Success:      false,
		ExitCode:     2,
		ErrorMessage: "agent crashed unexpectedly",
	}
}

// =============================================================================
// Tests
// =============================================================================

func TestModelFallback_SuccessOnPrimary(t *testing.T) {
	chain := modelregistry.PresetChain{"primary", "secondary", ""}
	h := newFallbackHarness(t, domain.RunnerTypeCodex, chain, domain.ModelPresetSmart)
	h.programResponses([]*runner.ExecuteResult{successResult()})

	h.executor.executeAgentWithModelFallback(context.Background(), h.runner)

	if len(h.attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d: %v", len(h.attempts), h.attempts)
	}
	if h.attempts[0] != "primary" {
		t.Fatalf("expected first attempt to use 'primary', got %q", h.attempts[0])
	}
	if h.run.ActualModel != "primary" {
		t.Fatalf("expected ActualModel=primary, got %q", h.run.ActualModel)
	}
}

func TestModelFallback_WalksChainOnUnavailable(t *testing.T) {
	chain := modelregistry.PresetChain{"primary", "secondary"}
	h := newFallbackHarness(t, domain.RunnerTypeCodex, chain, domain.ModelPresetSmart)
	h.programResponses([]*runner.ExecuteResult{
		modelUnavailableResult(domain.RunnerTypeCodex),
		successResult(),
	})

	h.executor.executeAgentWithModelFallback(context.Background(), h.runner)

	if len(h.attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d: %v", len(h.attempts), h.attempts)
	}
	if h.attempts[0] != "primary" || h.attempts[1] != "secondary" {
		t.Fatalf("attempt order unexpected: %v", h.attempts)
	}
	if h.run.ActualModel != "secondary" {
		t.Fatalf("expected ActualModel=secondary, got %q", h.run.ActualModel)
	}
	if h.run.ResolvedConfig.Model != "secondary" {
		t.Fatalf("expected ResolvedConfig.Model=secondary, got %q", h.run.ResolvedConfig.Model)
	}
}

func TestModelFallback_RunnerDefaultSentinelOmitsFlag(t *testing.T) {
	chain := modelregistry.PresetChain{"primary", ""}
	h := newFallbackHarness(t, domain.RunnerTypeCodex, chain, domain.ModelPresetSmart)
	h.programResponses([]*runner.ExecuteResult{
		modelUnavailableResult(domain.RunnerTypeCodex),
		successResult(),
	})

	h.executor.executeAgentWithModelFallback(context.Background(), h.runner)

	if len(h.attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d: %v", len(h.attempts), h.attempts)
	}
	if h.attempts[1] != "" {
		t.Fatalf("expected second attempt to omit the model flag (empty cfg.Model), got %q", h.attempts[1])
	}
	if h.run.ActualModel != "" {
		t.Fatalf("expected ActualModel to reflect runner-default sentinel (empty), got %q", h.run.ActualModel)
	}
}

func TestModelFallback_NonModelErrorSkipsRetry(t *testing.T) {
	chain := modelregistry.PresetChain{"primary", "secondary", ""}
	h := newFallbackHarness(t, domain.RunnerTypeCodex, chain, domain.ModelPresetSmart)
	h.programResponses([]*runner.ExecuteResult{genericFailureResult()})

	h.executor.executeAgentWithModelFallback(context.Background(), h.runner)

	if len(h.attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d: %v", len(h.attempts), h.attempts)
	}
	if h.run.ActualModel != "primary" {
		t.Fatalf("expected ActualModel=primary (generic failure should still mark executed model), got %q", h.run.ActualModel)
	}
}

func TestModelFallback_ChainExhausted(t *testing.T) {
	chain := modelregistry.PresetChain{"primary", "secondary"}
	h := newFallbackHarness(t, domain.RunnerTypeCodex, chain, domain.ModelPresetSmart)
	h.programResponses([]*runner.ExecuteResult{
		modelUnavailableResult(domain.RunnerTypeCodex),
		modelUnavailableResult(domain.RunnerTypeCodex),
	})

	h.executor.executeAgentWithModelFallback(context.Background(), h.runner)

	if len(h.attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d: %v", len(h.attempts), h.attempts)
	}
	if h.run.ActualModel != "secondary" {
		t.Fatalf("expected ActualModel=secondary (last attempted), got %q", h.run.ActualModel)
	}
	if h.executor.result == nil || h.executor.result.Success {
		t.Fatal("expected executor result to reflect the terminal failure after chain exhaustion")
	}
}

func TestModelFallback_NoResolverFallsBackToSingleShot(t *testing.T) {
	// Harness without a resolver should run exactly once with whatever cfg.Model holds.
	chain := modelregistry.PresetChain{}
	h := newFallbackHarness(t, domain.RunnerTypeCodex, chain, domain.ModelPresetUnspecified)
	h.run.ResolvedConfig.Model = "explicit-choice"
	h.run.ResolvedConfig.ModelPreset = domain.ModelPresetUnspecified
	h.executor.WithModelChainResolver(nil)
	h.programResponses([]*runner.ExecuteResult{successResult()})

	h.executor.executeAgentWithModelFallback(context.Background(), h.runner)

	if len(h.attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(h.attempts))
	}
	if h.attempts[0] != "explicit-choice" {
		t.Fatalf("expected explicit model to be used, got %q", h.attempts[0])
	}
	if h.run.ActualModel != "explicit-choice" {
		t.Fatalf("expected ActualModel=explicit-choice, got %q", h.run.ActualModel)
	}
}

func TestModelFallback_TransientErrorDoesNotAdvanceChain(t *testing.T) {
	chain := modelregistry.PresetChain{"primary", "secondary"}
	h := newFallbackHarness(t, domain.RunnerTypeCodex, chain, domain.ModelPresetSmart)
	h.programResponses([]*runner.ExecuteResult{
		{
			Success:      false,
			ExitCode:     1,
			ErrorMessage: "rate limit exceeded, retry after 30s",
		},
	})

	h.executor.executeAgentWithModelFallback(context.Background(), h.runner)

	if len(h.attempts) != 1 {
		t.Fatalf("transient failure must not trigger chain walk; got %d attempts", len(h.attempts))
	}
	if h.run.ActualModel != "primary" {
		t.Fatalf("ActualModel should record the model that returned the transient error; got %q", h.run.ActualModel)
	}
}

// Sanity: the classifier is the source of truth for model errors. If a future runner
// wraps an unavailable error differently, we want the test to point at the classifier.
func TestModelFallback_UsesClassifier(t *testing.T) {
	if got := runner.ClassifyModelError(domain.RunnerTypeCodex, "unknown model", 1); got != runner.ModelErrorUnavailable {
		t.Fatalf("classifier drift: expected ModelErrorUnavailable, got %v", got)
	}
	var _ error = errors.New("classifier test placeholder") // keep errors import live
}
