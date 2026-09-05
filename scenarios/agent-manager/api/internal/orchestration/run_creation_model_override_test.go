package orchestration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

func TestApplyModelOverridePreservesPolicySnapshotAuthority(t *testing.T) {
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeCodex)
	var probed string
	mock.ProbeFunc = func(_ context.Context, model string) error {
		probed = model
		return nil
	}
	if err := registry.Register(mock); err != nil {
		t.Fatal(err)
	}
	o := &Orchestrator{runners: registry}
	requested := "  local-model  "
	cfg := configWithSelectedCandidate(domain.RunnerTypeCodex, "old-model", 0)

	if err := o.applyModelOverride(context.Background(), cfg, &requested); err != nil {
		t.Fatalf("applyModelOverride: %v", err)
	}
	if probed != "local-model" {
		t.Fatalf("ProbeModel received %q, want trimmed override", probed)
	}
	if cfg.Model != "local-model" {
		t.Fatalf("resolved model = %q", cfg.Model)
	}
	selected := cfg.PolicySnapshot.SelectedCandidate
	if selected.Model != "local-model" || selected.SelectionType != domain.ModelSelectionTypeModel {
		t.Fatalf("selected candidate = %+v", selected)
	}
	listed := cfg.PolicySnapshot.Candidates[0]
	if listed.Model != "local-model" || listed.SelectionType != domain.ModelSelectionTypeModel {
		t.Fatalf("candidate list was not updated: %+v", cfg.PolicySnapshot.Candidates)
	}
}

func TestApplyModelOverrideRejectsInvalidOrUnavailableOverrides(t *testing.T) {
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeCodex)
	mock.ProbeFunc = func(context.Context, string) error { return errors.New("not installed") }
	if err := registry.Register(mock); err != nil {
		t.Fatal(err)
	}
	o := &Orchestrator{runners: registry}

	tests := []struct {
		name  string
		cfg   *domain.RunConfig
		value string
		want  string
		bare  bool
	}{
		{"empty", configWithSelectedCandidate(domain.RunnerTypeCodex, "old", 0), " ", "must not be empty", false},
		{"missing policy", &domain.RunConfig{RunnerType: domain.RunnerTypeCodex}, "model", "without a resolved execution policy", false},
		{"invalid selection", configWithSelectedCandidate(domain.RunnerTypeCodex, "old", 2), "model", "invalid selected candidate", true},
		{"probe failure", configWithSelectedCandidate(domain.RunnerTypeCodex, "old", 0), "model", "model is not available", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			underTest := o
			if tt.bare {
				underTest = &Orchestrator{}
			}
			err := underTest.applyModelOverride(context.Background(), tt.cfg, &tt.value)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}

	if err := o.applyModelOverride(context.Background(), configWithSelectedCandidate(domain.RunnerTypeCodex, "old", 0), nil); err != nil {
		t.Fatalf("nil override: %v", err)
	}
}

func configWithSelectedCandidate(runnerType domain.RunnerType, model string, selectedIndex int) *domain.RunConfig {
	candidate := domain.ExecutionCandidate{RunnerType: runnerType, SelectionType: domain.ModelSelectionTypeModel, Model: model}
	return &domain.RunConfig{
		RunnerType: runnerType,
		Model:      model,
		PolicySnapshot: &domain.ExecutionPolicySnapshot{
			Candidates:        []domain.ExecutionCandidate{candidate},
			SelectedIndex:     selectedIndex,
			SelectedCandidate: candidate,
		},
	}
}
