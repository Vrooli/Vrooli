// Tests for ExecuteWithModelFallback — preset-chain walk on
// model-unavailable errors.

package phases

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/modelpolicy"
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

// [REQ:REQ-P1-004] Runtime fallback follows the immutable persisted sequence,
// including explicit runner-default and cross-runner candidates.
func TestPolicySnapshotFallbackUsesPersistedCandidatesAcrossRunners(t *testing.T) {
	run := &domain.Run{
		ID:      uuid.New(),
		Status:  domain.RunStatusPending,
		Phase:   domain.RunPhaseQueued,
		RunMode: domain.RunModeInPlace,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeCodex,
			Model:      "stale-codex-model",
			PolicySnapshot: &domain.ExecutionPolicySnapshot{
				CatalogDigest: "sha256:persisted",
				PolicyRef:     "codex.smart",
				Candidates: []domain.ExecutionCandidate{
					{RunnerType: domain.RunnerTypeCodex, SelectionType: domain.ModelSelectionTypeModel, Model: "stale-codex-model"},
					{RunnerType: domain.RunnerTypeClaudeCode, SelectionType: domain.ModelSelectionTypeRunnerDefault},
				},
				SelectedIndex: 0,
				SelectedCandidate: domain.ExecutionCandidate{
					RunnerType: domain.RunnerTypeCodex, SelectionType: domain.ModelSelectionTypeModel, Model: "stale-codex-model",
				},
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	registry := runner.NewRegistry()
	codex := runner.NewMockRunner(domain.RunnerTypeCodex)
	claude := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	var attempts []string
	codex.ExecuteFunc = func(_ context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		attempts = append(attempts, string(codex.Type())+"/"+req.ResolvedConfig.Model)
		return modelUnavailableResult(domain.RunnerTypeCodex), nil
	}
	claude.ExecuteFunc = func(_ context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		attempts = append(attempts, string(claude.Type())+"/"+req.ResolvedConfig.Model)
		return successResult(), nil
	}
	if err := registry.Register(codex); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(claude); err != nil {
		t.Fatal(err)
	}

	events := &filterableEventStore{}
	out := ExecuteWithModelFallback(context.Background(), ExecuteWithModelFallbackInput{
		ExecuteAgentInput: ExecuteAgentInput{
			Deps:    Deps{Events: events, Levers: config.DefaultLevers()},
			Run:     run,
			Runner:  codex,
			Runners: registry,
		},
	})

	if out.Result == nil || !out.Result.Success {
		t.Fatalf("snapshot fallback result = %+v, err = %v", out.Result, out.ExecErr)
	}
	if got, want := strings.Join(attempts, ","), "codex/stale-codex-model,claude-code/"; got != want {
		t.Fatalf("attempts = %q, want %q", got, want)
	}
	if run.ResolvedConfig.RunnerType != domain.RunnerTypeClaudeCode || run.ResolvedConfig.Model != "" {
		t.Fatalf("resolved candidate = %s/%q", run.ResolvedConfig.RunnerType, run.ResolvedConfig.Model)
	}
	if run.ResolvedConfig.PolicySnapshot.CatalogDigest != "sha256:persisted" ||
		run.ResolvedConfig.PolicySnapshot.SelectedIndex != 0 {
		t.Fatalf("immutable snapshot was mutated: %+v", run.ResolvedConfig.PolicySnapshot)
	}
	policyEvents, err := events.Get(context.Background(), run.ID, event.GetOptions{EventTypes: []domain.RunEventType{domain.EventTypePolicyCandidateAttempt}})
	if err != nil {
		t.Fatal(err)
	}
	if len(policyEvents) != 4 {
		t.Fatalf("policy candidate event count = %d, want 4 (attempt/fail + attempt/select)", len(policyEvents))
	}
}

func TestPolicySnapshotFallbackSkipsUnavailablePersistedCandidate(t *testing.T) {
	run := &domain.Run{
		ID:      uuid.New(),
		Status:  domain.RunStatusPending,
		Phase:   domain.RunPhaseQueued,
		RunMode: domain.RunModeInPlace,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeCodex,
			PolicySnapshot: &domain.ExecutionPolicySnapshot{
				CatalogDigest: "sha256:persisted",
				Candidates: []domain.ExecutionCandidate{
					{RunnerType: domain.RunnerTypeCodex, SelectionType: domain.ModelSelectionTypeRunnerDefault},
					{RunnerType: domain.RunnerTypeClaudeCode, SelectionType: domain.ModelSelectionTypeModel, Model: "claude-current"},
				},
				SelectedIndex: 0,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	registry := runner.NewRegistry()
	codex := runner.NewMockRunner(domain.RunnerTypeCodex)
	codex.SetAvailable(false, "codex binary unavailable")
	claude := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	var attempts int
	claude.ExecuteFunc = func(_ context.Context, _ runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		attempts++
		return successResult(), nil
	}
	if err := registry.Register(codex); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(claude); err != nil {
		t.Fatal(err)
	}

	out := ExecuteWithModelFallback(context.Background(), ExecuteWithModelFallbackInput{
		ExecuteAgentInput: ExecuteAgentInput{
			Deps:    Deps{Events: &filterableEventStore{}, Levers: config.DefaultLevers()},
			Run:     run,
			Runner:  codex,
			Runners: registry,
		},
	})
	if out.Result == nil || !out.Result.Success || attempts != 1 {
		t.Fatalf("result = %+v, err = %v, attempts = %d", out.Result, out.ExecErr, attempts)
	}
	if run.ActualModel != "claude-current" {
		t.Fatalf("actual model = %q", run.ActualModel)
	}
}

// [REQ:REQ-P1-004] Exhausting the immutable sequence is a typed terminal
// failure with an explicit digest/index-qualified exhaustion event.
func TestPolicySnapshotFallbackExhaustionIsTerminalAndObservable(t *testing.T) {
	run := &domain.Run{
		ID:      uuid.New(),
		Status:  domain.RunStatusPending,
		Phase:   domain.RunPhaseQueued,
		RunMode: domain.RunModeInPlace,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeCodex,
			PolicySnapshot: &domain.ExecutionPolicySnapshot{
				CatalogDigest: "sha256:exhaustion",
				Candidates: []domain.ExecutionCandidate{
					{RunnerType: domain.RunnerTypeCodex, SelectionType: domain.ModelSelectionTypeRunnerDefault},
					{RunnerType: domain.RunnerTypeClaudeCode, SelectionType: domain.ModelSelectionTypeModel, Model: "retired-claude-model"},
				},
				SelectedIndex: 0,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	registry := runner.NewRegistry()
	codex := runner.NewMockRunner(domain.RunnerTypeCodex)
	codex.SetAvailable(false, "codex binary unavailable")
	claude := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	claude.ExecuteFunc = func(context.Context, runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		return modelUnavailableResult(domain.RunnerTypeClaudeCode), nil
	}
	if err := registry.Register(codex); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(claude); err != nil {
		t.Fatal(err)
	}

	events := &filterableEventStore{}
	out := ExecuteWithModelFallback(context.Background(), ExecuteWithModelFallbackInput{
		ExecuteAgentInput: ExecuteAgentInput{
			Deps:    Deps{Events: events, Levers: config.DefaultLevers()},
			Run:     run,
			Runner:  codex,
			Runners: registry,
		},
	})

	var runnerErr *domain.RunnerError
	if !errors.As(out.ExecErr, &runnerErr) || runnerErr.Code() != domain.ErrCodeRunnerUnavailable {
		t.Fatalf("terminal error = %T %v, want RUNNER_UNAVAILABLE", out.ExecErr, out.ExecErr)
	}
	policyEvents, err := events.Get(context.Background(), run.ID, event.GetOptions{EventTypes: []domain.RunEventType{domain.EventTypePolicyCandidateAttempt}})
	if err != nil {
		t.Fatal(err)
	}
	if len(policyEvents) != 4 {
		t.Fatalf("policy candidate event count = %d, want skip + attempt/fail + exhaustion", len(policyEvents))
	}
	payload, ok := decodeTypedPayload(t, policyEvents[len(policyEvents)-1]).(*eventlog.PolicyCandidateAttemptPayload)
	if !ok {
		t.Fatalf("terminal payload = %T", decodeTypedPayload(t, policyEvents[len(policyEvents)-1]))
	}
	if payload.Outcome != eventlog.PolicyCandidateOutcomeExhausted ||
		payload.CatalogDigest != "sha256:exhaustion" || payload.SnapshotIndex != 1 ||
		payload.FailureClass != "snapshot_exhausted" {
		t.Fatalf("terminal exhaustion payload = %+v", payload)
	}
}

// [REQ:REQ-P1-004] Restart/resume retries the candidate persisted in the
// resolved config instead of replaying earlier snapshot entries.
func TestPolicySnapshotResumeStartsAtPersistedCandidate(t *testing.T) {
	first := domain.ExecutionCandidate{RunnerType: domain.RunnerTypeCodex, SelectionType: domain.ModelSelectionTypeModel, Model: "first-model"}
	resumed := domain.ExecutionCandidate{RunnerType: domain.RunnerTypeClaudeCode, SelectionType: domain.ModelSelectionTypeRunnerDefault}
	run := &domain.Run{
		ID:      uuid.New(),
		Status:  domain.RunStatusFailed,
		Phase:   domain.RunPhaseExecuting,
		RunMode: domain.RunModeInPlace,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
			Model:      "",
			PolicySnapshot: &domain.ExecutionPolicySnapshot{
				CatalogDigest:     "sha256:before-restart",
				Candidates:        []domain.ExecutionCandidate{first, resumed},
				SelectedIndex:     0,
				SelectedCandidate: first,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	raw, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	var reloaded domain.Run
	if err := json.Unmarshal(raw, &reloaded); err != nil {
		t.Fatal(err)
	}

	registry := runner.NewRegistry()
	codex := runner.NewMockRunner(domain.RunnerTypeCodex)
	codex.ExecuteFunc = func(context.Context, runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		t.Fatal("resume replayed a candidate that was already passed")
		return nil, nil
	}
	claude := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	var attempts int
	claude.ExecuteFunc = func(_ context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		attempts++
		if req.ResolvedConfig.Model != "" {
			t.Fatalf("runner_default resumed with model %q", req.ResolvedConfig.Model)
		}
		return successResult(), nil
	}
	if err := registry.Register(codex); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(claude); err != nil {
		t.Fatal(err)
	}

	out := ExecuteWithModelFallback(context.Background(), ExecuteWithModelFallbackInput{
		ExecuteAgentInput: ExecuteAgentInput{
			Deps:    Deps{Events: &filterableEventStore{}, Levers: config.DefaultLevers()},
			Run:     &reloaded,
			Runner:  codex,
			Runners: registry,
		},
	})
	if out.Result == nil || !out.Result.Success || attempts != 1 {
		t.Fatalf("resume result = %+v, err = %v, attempts = %d", out.Result, out.ExecErr, attempts)
	}
	if reloaded.ResolvedConfig.PolicySnapshot.SelectedIndex != 0 {
		t.Fatalf("resume mutated immutable snapshot: %+v", reloaded.ResolvedConfig.PolicySnapshot)
	}
}

// [REQ:REQ-P1-004] A successful live catalog reload between attempts affects
// only future resolutions; the running execution keeps its old candidates.
func TestPolicySnapshotFallbackIgnoresCatalogReloadDuringExecution(t *testing.T) {
	path, catalog := copyModelPolicyCatalog(t)
	state, err := modelpolicy.NewState(path, modelpolicy.Requirement{Required: true})
	if err != nil {
		t.Fatalf("new policy state: %v", err)
	}
	snapshot, err := state.ResolvePolicy("codex.smart")
	if err != nil {
		t.Fatalf("resolve policy: %v", err)
	}
	originalDigest := snapshot.CatalogDigest
	originalCandidates := append([]domain.ExecutionCandidate(nil), snapshot.Candidates...)
	if len(originalCandidates) < 2 {
		t.Fatalf("test policy has %d candidates, want at least two", len(originalCandidates))
	}
	run := &domain.Run{
		ID:      uuid.New(),
		Status:  domain.RunStatusPending,
		Phase:   domain.RunPhaseQueued,
		RunMode: domain.RunModeInPlace,
		ResolvedConfig: &domain.RunConfig{
			RunnerType:     originalCandidates[0].RunnerType,
			Model:          originalCandidates[0].Model,
			PolicySnapshot: snapshot,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	registry := runner.NewRegistry()
	codex := runner.NewMockRunner(domain.RunnerTypeCodex)
	var attempts []string
	codex.ExecuteFunc = func(_ context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		attempts = append(attempts, req.ResolvedConfig.Model)
		if len(attempts) == 1 {
			catalog.Metadata.CatalogID += "-reloaded"
			policy := catalog.Policies["codex.smart"]
			policy.Candidates = []modelpolicy.Candidate{{
				Runner:    domain.RunnerTypeCodex,
				Selection: modelpolicy.Selection{Type: modelpolicy.SelectionTypeRunnerDefault},
			}}
			catalog.Policies["codex.smart"] = policy
			writeModelPolicyCatalog(t, path, catalog)
			if _, err := state.Reload(); err != nil {
				t.Fatalf("reload catalog during execution: %v", err)
			}
			return modelUnavailableResult(domain.RunnerTypeCodex), nil
		}
		return successResult(), nil
	}
	if err := registry.Register(codex); err != nil {
		t.Fatal(err)
	}

	out := ExecuteWithModelFallback(context.Background(), ExecuteWithModelFallbackInput{
		ExecuteAgentInput: ExecuteAgentInput{
			Deps:    Deps{Events: &filterableEventStore{}, Levers: config.DefaultLevers()},
			Run:     run,
			Runner:  codex,
			Runners: registry,
		},
	})
	if out.Result == nil || !out.Result.Success {
		t.Fatalf("reload execution result = %+v, err = %v", out.Result, out.ExecErr)
	}
	if got, want := strings.Join(attempts, ","), originalCandidates[0].Model+","+originalCandidates[1].Model; got != want {
		t.Fatalf("attempts after reload = %q, want persisted sequence %q", got, want)
	}
	if state.Status().ActiveDigest == originalDigest {
		t.Fatal("test did not activate a new catalog digest")
	}
	if snapshot.CatalogDigest != originalDigest || snapshot.Candidates[1] != originalCandidates[1] {
		t.Fatalf("running snapshot changed across reload: %+v", snapshot)
	}
}

func copyModelPolicyCatalog(t *testing.T) (string, *modelpolicy.Catalog) {
	t.Helper()
	raw, err := os.ReadFile(modelpolicy.ResolvePath())
	if err != nil {
		t.Fatalf("read model policy catalog: %v", err)
	}
	var catalog modelpolicy.Catalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatalf("decode model policy catalog: %v", err)
	}
	path := filepath.Join(t.TempDir(), "model-policy-catalog.json")
	writeModelPolicyCatalog(t, path, &catalog)
	return path, &catalog
}

func writeModelPolicyCatalog(t *testing.T, path string, catalog *modelpolicy.Catalog) {
	t.Helper()
	raw, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("encode model policy catalog: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write model policy catalog: %v", err)
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
