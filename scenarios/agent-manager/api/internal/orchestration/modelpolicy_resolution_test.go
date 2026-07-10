package orchestration

import (
	"context"
	"errors"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/modelpolicy"
)

func TestResolveRunConfigPersistsNamedPolicySnapshot(t *testing.T) {
	state, err := modelpolicy.NewState(modelpolicy.ResolvePath(), modelpolicy.Requirement{Required: true})
	if err != nil {
		t.Fatalf("new model policy state: %v", err)
	}
	orchestrator := &Orchestrator{modelPolicy: state}
	runnerType := domain.RunnerTypeCodex
	policyRef := "codex.smart"

	cfg, profile, err := orchestrator.resolveRunConfig(context.Background(), CreateRunRequest{
		RunnerType: &runnerType,
		PolicyRef:  &policyRef,
	})
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if profile != nil {
		t.Fatalf("profile = %+v, want nil for inline config", profile)
	}
	if cfg.PolicySnapshot == nil {
		t.Fatal("resolved config has no policy snapshot")
	}
	if cfg.PolicySnapshot.PolicyRef != "codex.smart" {
		t.Fatalf("policy ref = %q", cfg.PolicySnapshot.PolicyRef)
	}
	if cfg.PolicySnapshot.CatalogDigest == "" || len(cfg.PolicySnapshot.Candidates) < 2 {
		t.Fatalf("snapshot = %+v", cfg.PolicySnapshot)
	}
	if cfg.RunnerType != cfg.PolicySnapshot.SelectedCandidate.RunnerType ||
		cfg.Model != cfg.PolicySnapshot.SelectedCandidate.Model {
		t.Fatalf("config selection runner/model = %s/%q, snapshot = %+v", cfg.RunnerType, cfg.Model, cfg.PolicySnapshot.SelectedCandidate)
	}
}

func TestResolveRunConfigPersistsExplicitRunnerDefaultSnapshot(t *testing.T) {
	state, err := modelpolicy.NewState(modelpolicy.ResolvePath(), modelpolicy.Requirement{Required: true})
	if err != nil {
		t.Fatalf("new model policy state: %v", err)
	}
	orchestrator := &Orchestrator{modelPolicy: state}
	runnerType := domain.RunnerTypeClaudeCode

	cfg, _, err := orchestrator.resolveRunConfig(context.Background(), CreateRunRequest{RunnerType: &runnerType})
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if cfg.PolicySnapshot == nil ||
		cfg.PolicySnapshot.SelectedCandidate.SelectionType != domain.ModelSelectionTypeRunnerDefault {
		t.Fatalf("snapshot = %+v, want explicit runner_default", cfg.PolicySnapshot)
	}
	if cfg.Model != "" {
		t.Fatalf("runner default config model = %q, want omitted", cfg.Model)
	}
}

func TestResolveRunConfigRejectsUndeclaredDirectModel(t *testing.T) {
	state, err := modelpolicy.NewState(modelpolicy.ResolvePath(), modelpolicy.Requirement{Required: true})
	if err != nil {
		t.Fatalf("new model policy state: %v", err)
	}
	orchestrator := &Orchestrator{modelPolicy: state}
	runnerType := domain.RunnerTypeCodex
	model := "retired-model"

	if _, _, err := orchestrator.resolveRunConfig(context.Background(), CreateRunRequest{
		RunnerType: &runnerType,
		Model:      &model,
	}); err == nil {
		t.Fatal("expected undeclared direct model rejection")
	}
}

// [REQ:REQ-P1-004] Stale declared models fall through to an explicit runner default.
func TestResolveRunConfigFallsBackToExplicitRunnerDefaultWhenCatalogModelsAreStale(t *testing.T) {
	state, err := modelpolicy.NewState(modelpolicy.ResolvePath(), modelpolicy.Requirement{Required: true})
	if err != nil {
		t.Fatalf("new model policy state: %v", err)
	}
	registry := runner.NewRegistry()
	codex := runner.NewMockRunner(domain.RunnerTypeCodex)
	codex.ProbeFunc = func(context.Context, string) error {
		return errors.New("model is unavailable in installed runner")
	}
	if err := registry.Register(codex); err != nil {
		t.Fatalf("register codex: %v", err)
	}
	orchestrator := &Orchestrator{modelPolicy: state, runners: registry}
	runnerType := domain.RunnerTypeCodex
	policyRef := "codex.fast"

	cfg, _, err := orchestrator.resolveRunConfig(context.Background(), CreateRunRequest{
		RunnerType: &runnerType,
		PolicyRef:  &policyRef,
	})
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if cfg.PolicySnapshot == nil {
		t.Fatal("resolved config has no policy snapshot")
	}
	selected := cfg.PolicySnapshot.SelectedCandidate
	if selected.RunnerType != domain.RunnerTypeCodex ||
		selected.SelectionType != domain.ModelSelectionTypeRunnerDefault {
		t.Fatalf("selected = %+v, want codex runner_default", selected)
	}
	if cfg.Model != "" {
		t.Fatalf("runner-default model = %q, want omitted", cfg.Model)
	}
	if cfg.PolicySnapshot.SelectedIndex != 3 {
		t.Fatalf("selected index = %d, want 3 after three stale catalog models", cfg.PolicySnapshot.SelectedIndex)
	}
	if len(cfg.PolicySnapshot.Explanation.Preflight) != 4 {
		t.Fatalf("preflight = %+v, want three failures and selected runner default", cfg.PolicySnapshot.Explanation.Preflight)
	}
	for index, check := range cfg.PolicySnapshot.Explanation.Preflight {
		if index < 3 && check.Available {
			t.Fatalf("preflight[%d] unexpectedly available: %+v", index, check)
		}
	}
}
