// Tests for ExecuteWithModelFallback — preset-chain walk on
// model-unavailable errors.

package phases

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/modelregistry"
	"agent-manager/internal/testutil/mocks/modelchain"

	"github.com/google/uuid"
)

// fallbackHarness wires up a per-test fixture for ExecuteWithModelFallback.
type fallbackHarness struct {
	t        *testing.T
	mock     *runner.MockRunner
	run      *domain.Run
	deps     Deps
	resolver ModelChainResolver
	attempts []string
}

func newFallbackHarness(t *testing.T, runnerType domain.RunnerType, chain modelregistry.PresetChain, preset domain.ModelPreset) *fallbackHarness {
	t.Helper()
	store := &filterableEventStore{}
	run := &domain.Run{
		ID:      uuid.New(),
		Status:  domain.RunStatusPending,
		Phase:   domain.RunPhaseQueued,
		RunMode: domain.RunModeInPlace,
		ResolvedConfig: &domain.RunConfig{
			RunnerType:  runnerType,
			Model:       chain.Primary(),
			ModelPreset: preset,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mock := runner.NewMockRunner(runnerType)
	deps := Deps{Events: store, Levers: config.DefaultLevers()}
	return &fallbackHarness{
		t:        t,
		mock:     mock,
		run:      run,
		deps:     deps,
		resolver: modelchain.NewFakeResolver(chain),
	}
}

func (h *fallbackHarness) programResponses(results []*runner.ExecuteResult) {
	index := 0
	h.mock.ExecuteFunc = func(_ context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		h.attempts = append(h.attempts, req.ResolvedConfig.Model)
		if index >= len(results) {
			h.t.Fatalf("mock runner invoked %d times, only %d responses programmed", index+1, len(results))
		}
		r := results[index]
		index++
		return r, nil
	}
}

func (h *fallbackHarness) run_(t *testing.T) ExecuteAgentOutput {
	t.Helper()
	return ExecuteWithModelFallback(context.Background(), ExecuteWithModelFallbackInput{
		ExecuteAgentInput: ExecuteAgentInput{
			Deps:        h.deps,
			Run:         h.run,
			Runner:      h.mock,
			ModelChains: h.resolver,
		},
	})
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

func TestModelFallback_SuccessOnPrimary(t *testing.T) {
	chain := modelregistry.PresetChain{"primary", "secondary", ""}
	h := newFallbackHarness(t, domain.RunnerTypeCodex, chain, domain.ModelPresetSmart)
	h.programResponses([]*runner.ExecuteResult{successResult()})

	h.run_(t)

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

	h.run_(t)

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

	h.run_(t)

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

// TestModelFallback_UseCliDefaultModelLeverSkipsChain asserts the
// runner-CLI-default escape hatch: when Levers.Runners.UseCliDefaultModel
// is on, ExecuteWithModelFallback bypasses the preset chain entirely,
// clears cfg.Model so the codec emits no -m/--model flag, and records
// ActualModel="" so observability reflects "runner default."
func TestModelFallback_UseCliDefaultModelLeverSkipsChain(t *testing.T) {
	chain := modelregistry.PresetChain{"primary", "secondary", ""}
	h := newFallbackHarness(t, domain.RunnerTypeCodex, chain, domain.ModelPresetSmart)
	h.deps.Levers.Runners.UseCliDefaultModel = true
	h.programResponses([]*runner.ExecuteResult{successResult()})

	h.run_(t)

	if len(h.attempts) != 1 {
		t.Fatalf("expected exactly 1 attempt (no chain walk), got %d: %v", len(h.attempts), h.attempts)
	}
	if h.attempts[0] != "" {
		t.Fatalf("expected cfg.Model cleared on the single attempt, got %q", h.attempts[0])
	}
	if h.run.ResolvedConfig.Model != "" {
		t.Fatalf("expected run.ResolvedConfig.Model cleared, got %q", h.run.ResolvedConfig.Model)
	}
	if h.run.ActualModel != "" {
		t.Fatalf("expected ActualModel=\"\" (runner default), got %q", h.run.ActualModel)
	}
}

func TestModelFallback_NonModelErrorSkipsRetry(t *testing.T) {
	chain := modelregistry.PresetChain{"primary", "secondary", ""}
	h := newFallbackHarness(t, domain.RunnerTypeCodex, chain, domain.ModelPresetSmart)
	h.programResponses([]*runner.ExecuteResult{genericFailureResult()})

	h.run_(t)

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

	out := h.run_(t)

	if len(h.attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d: %v", len(h.attempts), h.attempts)
	}
	if h.run.ActualModel != "secondary" {
		t.Fatalf("expected ActualModel=secondary (last attempted), got %q", h.run.ActualModel)
	}
	if out.Result == nil || out.Result.Success {
		t.Fatal("expected result to reflect the terminal failure after chain exhaustion")
	}
}

func TestModelFallback_NoResolverFallsBackToSingleShot(t *testing.T) {
	h := newFallbackHarness(t, domain.RunnerTypeCodex, modelregistry.PresetChain{}, domain.ModelPresetUnspecified)
	h.run.ResolvedConfig.Model = "explicit-choice"
	h.run.ResolvedConfig.ModelPreset = domain.ModelPresetUnspecified
	h.resolver = nil
	h.programResponses([]*runner.ExecuteResult{successResult()})

	h.run_(t)

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

	h.run_(t)

	if len(h.attempts) != 1 {
		t.Fatalf("transient failure must not trigger chain walk; got %d attempts", len(h.attempts))
	}
	if h.run.ActualModel != "primary" {
		t.Fatalf("ActualModel should record the model that returned the transient error; got %q", h.run.ActualModel)
	}
}

// Sanity: the classifier is the source of truth for model errors.
func TestModelFallback_UsesClassifier(t *testing.T) {
	mr := runner.NewMockRunner(domain.RunnerTypeCodex)
	got := mr.Classify("unknown model", 1)
	if got == nil || !got.IsModelUnavailable() {
		t.Fatalf("classifier drift: expected model-unavailable classification, got %+v", got)
	}
	var _ error = errors.New("classifier test placeholder")
}

// groundWorkingDir — strengthens cwd path grounding in the system prompt.

func TestGroundWorkingDir(t *testing.T) {
	const dir = "/tmp/work-abc"

	t.Run("empty dir leaves prompt untouched", func(t *testing.T) {
		if got := groundWorkingDir("sys", ""); got != "sys" {
			t.Fatalf("got %q, want unchanged %q", got, "sys")
		}
		if got := groundWorkingDir("sys", "   "); got != "sys" {
			t.Fatalf("whitespace dir should be treated as empty, got %q", got)
		}
	})

	t.Run("appends directive carrying the absolute dir", func(t *testing.T) {
		got := groundWorkingDir("existing system prompt", dir)
		if !strings.HasPrefix(got, "existing system prompt") {
			t.Fatalf("original system prompt must be preserved at the front: %q", got)
		}
		if !strings.Contains(got, dir) {
			t.Fatalf("directive must contain the absolute dir %q: %q", dir, got)
		}
		if !strings.Contains(got, "<working-directory>") || !strings.Contains(got, "</working-directory>") {
			t.Fatalf("directive must be wrapped in a working-directory block: %q", got)
		}
		if !strings.Contains(got, "Do not invent or target any other directory") {
			t.Fatalf("directive must instruct against other directories: %q", got)
		}
	})

	t.Run("empty system prompt yields just the directive", func(t *testing.T) {
		got := groundWorkingDir("", dir)
		if strings.HasPrefix(got, "\n") {
			t.Fatalf("must not lead with blank separator when system prompt empty: %q", got)
		}
		if !strings.Contains(got, dir) {
			t.Fatalf("directive must contain the dir: %q", got)
		}
	})
}
