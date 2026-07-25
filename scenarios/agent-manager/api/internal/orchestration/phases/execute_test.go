// Tests for ExecuteWithModelFallback — immutable persisted-candidate execution
// and advancement on model-unavailable errors.

package phases

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
)

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

func TestToolRestrictionCandidateReasonRejectsUnsupportedEnforcedFallback(t *testing.T) {
	cfg := &domain.RunConfig{
		AllowedTools:          []string{"read"},
		ToolRestrictionPolicy: domain.ToolRestrictionPolicyEnforced,
	}
	unsupported := runner.NewMockRunner(domain.RunnerTypeCodex)
	if got := toolRestrictionCandidateReason(cfg, unsupported); !strings.Contains(got, "cannot enforce allowedTools") {
		t.Fatalf("unsupported enforced fallback reason = %q", got)
	}
	cfg.ToolRestrictionPolicy = domain.ToolRestrictionPolicyAdvisory
	if got := toolRestrictionCandidateReason(cfg, unsupported); got != "" {
		t.Fatalf("advisory fallback must proceed, got %q", got)
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
				RoleRef:       "code.smart",
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
