package orchestration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"
	"github.com/google/uuid"
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

func TestApplyModelExclusionsRehomesPrimaryToAllowedFallback(t *testing.T) {
	snapshot := &domain.ExecutionPolicySnapshot{Candidates: []domain.ExecutionCandidate{{
		RunnerType: domain.RunnerTypeCodex, SelectionType: domain.ModelSelectionTypeModel,
		Model: "blocked", CanonicalModel: "vendor/blocked", Fallbacks: []string{"allowed", "blocked-two"},
		ExcludedModels: []string{"vendor/blocked", "blocked-two"}, Available: true,
	}}}
	applyModelExclusions(snapshot)
	candidate := snapshot.Candidates[0]
	if candidate.Model != "allowed" || len(candidate.Fallbacks) != 0 || !candidate.Available {
		t.Fatalf("candidate after exclusion = %+v", candidate)
	}
}

func TestValidateExecutionModelRejectsRetainedExcludedModel(t *testing.T) {
	cfg := &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, Model: "blocked", PolicySnapshot: &domain.ExecutionPolicySnapshot{
		SelectedIndex:     0,
		Candidates:        []domain.ExecutionCandidate{{RunnerType: domain.RunnerTypeCodex, SelectionType: domain.ModelSelectionTypeModel, Model: "blocked", ExcludedModels: []string{"blocked"}}},
		SelectedCandidate: domain.ExecutionCandidate{RunnerType: domain.RunnerTypeCodex, SelectionType: domain.ModelSelectionTypeModel, Model: "blocked", ExcludedModels: []string{"blocked"}},
	}}
	if err := validateExecutionModel(cfg); err == nil || !strings.Contains(err.Error(), "excluded") {
		t.Fatalf("retained excluded model error = %v", err)
	}
}

func TestModelExclusionBlocksProviderPrefixedAlias(t *testing.T) {
	if !domain.IsModelExcluded("openrouter/openai/gpt-6-astra", "", []string{"gpt-6-astra"}) {
		t.Fatal("provider-prefixed prohibited model bypassed bare configured exclusion")
	}
	if domain.IsModelExcluded("openrouter/openai/gpt-5.6", "", []string{"gpt-5.6-sol"}) {
		t.Fatal("similar model suffix was incorrectly excluded")
	}
}

type currentDenyResolver struct{}

func (currentDenyResolver) Resolve(_ context.Context, runnerType domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	return rolepolicy.ResolvedRole{Runner: runnerType, Role: role, Model: "allowed-model", ExcludedModels: []string{"blocked-model"}}, nil
}

func TestCurrentPolicyRejectsHistoricalModelWithoutSnapshotExclusion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "role-policy.json")
	const catalog = `{"schemaVersion":1,"metadata":{"catalogId":"current-deny-test","updatedAt":"2026-07-13"},"defaultRole":"code.default","roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"codex","resourceRole":"code.default"}]}}}`
	if err := os.WriteFile(path, []byte(catalog), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	o := &Orchestrator{rolePolicy: state, roleResolver: currentDenyResolver{}}
	cfg := &domain.RunConfig{
		RunnerType: domain.RunnerTypeCodex, RoleRef: "code.default", Model: "blocked-model",
	}
	if err := o.validateCurrentExecutionModel(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "current resource policy") {
		t.Fatalf("historical model escaped current deny policy: %v", err)
	}
}

func TestCurrentPolicyRejectsUnknownNativeDefaultWhenDenyListIsPresent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "role-policy.json")
	const catalog = `{"schemaVersion":1,"metadata":{"catalogId":"current-default-deny-test","updatedAt":"2026-07-13"},"defaultRole":"code.default","roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"codex","resourceRole":"code.default"}]}}}`
	if err := os.WriteFile(path, []byte(catalog), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	o := &Orchestrator{rolePolicy: state, roleResolver: currentDenyResolver{}}
	cfg := &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, RoleRef: "code.default"}
	if err := o.validateCurrentExecutionModel(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "unknown native model") {
		t.Fatalf("unknown native default escaped current deny policy: %v", err)
	}
}

type recordingCurrentPolicyResolver struct {
	roles map[domain.RunnerType]rolepolicy.ResolvedRole
	calls map[domain.RunnerType]int
}

func (r *recordingCurrentPolicyResolver) Resolve(_ context.Context, runnerType domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	r.calls[runnerType]++
	resolved := r.roles[runnerType]
	resolved.Runner = runnerType
	resolved.Role = role
	return resolved, nil
}

func TestCurrentPolicyReadsEveryRetainedFallbackRunner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "role-policy.json")
	const catalog = `{"schemaVersion":1,"metadata":{"catalogId":"current-fallback-deny-test","updatedAt":"2026-07-13"},"defaultRole":"code.default","roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"codex","resourceRole":"code.default"},{"runner":"opencode","resourceRole":"code.default"}]}}}`
	if err := os.WriteFile(path, []byte(catalog), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &recordingCurrentPolicyResolver{
		roles: map[domain.RunnerType]rolepolicy.ResolvedRole{
			domain.RunnerTypeCodex:    {ExcludedModels: []string{}},
			domain.RunnerTypeOpenCode: {ExcludedModels: []string{"forbidden-fallback"}},
		},
		calls: make(map[domain.RunnerType]int),
	}
	cfg := &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, RoleRef: "code.default", Model: "allowed-model", PolicySnapshot: &domain.ExecutionPolicySnapshot{Candidates: []domain.ExecutionCandidate{{RunnerType: domain.RunnerTypeGrok, ResourceRole: "code.default"}}}}
	exclusions, err := currentModelExclusions(context.Background(), cfg, state, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if resolver.calls[domain.RunnerTypeCodex] != 1 || resolver.calls[domain.RunnerTypeOpenCode] != 1 || resolver.calls[domain.RunnerTypeGrok] != 1 {
		t.Fatalf("current policy calls = %v, want one read per retained runner", resolver.calls)
	}
	if len(exclusions[domain.RunnerTypeOpenCode]) != 1 || exclusions[domain.RunnerTypeOpenCode][0] != "forbidden-fallback" || exclusions[domain.RunnerTypeGrok] == nil {
		t.Fatalf("fallback exclusion overlay = %v", exclusions)
	}
}

func TestCurrentPolicyMergesOmittedRunnerOverlayWithHistoricalFallbacks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "role-policy.json")
	const catalog = `{"schemaVersion":1,"metadata":{"catalogId":"current-merge-deny-test","updatedAt":"2026-07-13"},"defaultRole":"code.default","roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"codex","resourceRole":"code.default"}]}}}`
	if err := os.WriteFile(path, []byte(catalog), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &recordingCurrentPolicyResolver{
		roles: map[domain.RunnerType]rolepolicy.ResolvedRole{
			domain.RunnerTypeCodex:       {ExcludedModels: []string{"codex-deny"}},
			domain.RunnerTypeGrok:        {ExcludedModels: []string{"grok-deny"}},
			domain.RunnerTypeAntigravity: {ExcludedModels: []string{"antigravity-deny"}},
		},
		calls: make(map[domain.RunnerType]int),
	}
	cfg := &domain.RunConfig{
		RunnerType: domain.RunnerTypeAntigravity, RoleRef: "code.default", Model: "allowed-model",
		PolicySnapshot: &domain.ExecutionPolicySnapshot{Candidates: []domain.ExecutionCandidate{{RunnerType: domain.RunnerTypeGrok, ResourceRole: "legacy.role"}}},
	}
	exclusions, err := currentModelExclusions(context.Background(), cfg, state, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if len(exclusions[domain.RunnerTypeGrok]) != 1 || exclusions[domain.RunnerTypeGrok][0] != "grok-deny" {
		t.Fatalf("historical fallback exclusion was lost: %v", exclusions)
	}
	if len(exclusions[domain.RunnerTypeAntigravity]) != 1 || exclusions[domain.RunnerTypeAntigravity][0] != "antigravity-deny" {
		t.Fatalf("omitted runner exclusion was not merged: %v", exclusions)
	}
}

func TestApplyModelOverrideRejectsExcludedModelBeforeRunnerProbe(t *testing.T) {
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeCodex)
	probed := false
	mock.ProbeFunc = func(context.Context, string) error { probed = true; return nil }
	if err := registry.Register(mock); err != nil {
		t.Fatal(err)
	}
	o := &Orchestrator{runners: registry}
	cfg := configWithSelectedCandidate(domain.RunnerTypeCodex, "allowed", 0)
	cfg.PolicySnapshot.Candidates[0].ExcludedModels = []string{"blocked"}
	cfg.PolicySnapshot.SelectedCandidate.ExcludedModels = []string{"blocked"}
	requested := "blocked"
	if err := o.applyModelOverride(context.Background(), cfg, &requested); err == nil || !strings.Contains(err.Error(), "excluded") {
		t.Fatalf("excluded override error = %v", err)
	}
	if probed {
		t.Fatal("excluded override reached runner model probe")
	}
}

func TestSelectInitialCandidateRejectsExcludedModelBeforeRunnerProbe(t *testing.T) {
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeCodex)
	probed := false
	mock.ProbeFunc = func(context.Context, string) error {
		probed = true
		return nil
	}
	if err := registry.Register(mock); err != nil {
		t.Fatal(err)
	}
	o := &Orchestrator{runners: registry}
	candidate := domain.ExecutionCandidate{
		RunnerType:     domain.RunnerTypeCodex,
		SelectionType:  domain.ModelSelectionTypeModel,
		Model:          "blocked-model",
		ExcludedModels: []string{"blocked-model"},
		Available:      true,
	}
	snapshot := &domain.ExecutionPolicySnapshot{Candidates: []domain.ExecutionCandidate{candidate}}
	applyModelExclusions(snapshot)
	if _, _, err := o.selectInitialCandidate(context.Background(), snapshot.Candidates); err == nil || !strings.Contains(err.Error(), "excluded") {
		t.Fatalf("excluded candidate error = %v", err)
	}
	if probed {
		t.Fatal("excluded candidate reached runner model probe")
	}
}

func TestResumeConversationRejectsRetainedExcludedModelBeforeRunnerContinuation(t *testing.T) {
	registry := runner.NewRegistry()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	continued := false
	mock.ContinueFunc = func(context.Context, runner.ContinueRequest) (*runner.ExecuteResult, error) {
		continued = true
		return &runner.ExecuteResult{Success: true}, nil
	}
	if err := registry.Register(mock); err != nil {
		t.Fatal(err)
	}
	o := &Orchestrator{runners: registry}
	run := &domain.Run{
		ID:        uuid.New(),
		Status:    domain.RunStatusComplete,
		SessionID: "retained-session",
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
			Model:      "blocked-model",
			PolicySnapshot: &domain.ExecutionPolicySnapshot{SelectedCandidate: domain.ExecutionCandidate{
				RunnerType:     domain.RunnerTypeClaudeCode,
				SelectionType:  domain.ModelSelectionTypeModel,
				Model:          "blocked-model",
				ExcludedModels: []string{"blocked-model"},
			}},
		},
	}
	if _, err := o.resumeConversation(context.Background(), run, "continue", nil, "test", continuationOverrides{}); err == nil || !domain.IsPreEffectRefusal(err) || !strings.Contains(err.Error(), "excluded") {
		t.Fatalf("retained excluded continuation error = %v", err)
	}
	if continued {
		t.Fatal("excluded continuation reached runner")
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
