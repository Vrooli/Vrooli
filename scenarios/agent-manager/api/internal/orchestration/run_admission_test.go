package orchestration

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

// launchInfoStub satisfies runner.AgentLaunchInfo and
// runner.RuntimeVersionReporter on top of MockRunner so admission capture can
// be exercised without a real codec.
type launchInfoStub struct {
	*runner.MockRunner
	args            []string
	err             error
	runtimeVersion  string
	runtimeVersionE error
}

func (s *launchInfoStub) TagEnvKey() string { return "STUB_TAG" }

func (s *launchInfoStub) BinaryPath() string { return "/usr/bin/stub" }

func (s *launchInfoStub) ControlArgs(*domain.RunConfig) ([]string, error) {
	return s.args, s.err
}

func (s *launchInfoStub) RuntimeVersion(context.Context) (string, error) {
	return s.runtimeVersion, s.runtimeVersionE
}

func TestBuildRunAdmissionCapturesRequestedAndEffective(t *testing.T) {
	effort := domain.EffortHigh
	model := "gpt-5.6-luna"
	roleRef := "code.default"
	timeout := 45 * time.Minute
	maxTurns := 12
	req := CreateRunRequest{
		PreferredRunner: "codex",
		Model:           &model,
		RoleRef:         &roleRef,
		Effort:          &effort,
		Timeout:         &timeout,
		MaxTurns:        &maxTurns,
		Until:           "all requested outcomes have evidence",
	}
	cfg := &domain.RunConfig{
		RunnerType: domain.RunnerTypeCodex,
		Model:      "gpt-5.6-luna",
		Effort:     domain.EffortMedium,
		Timeout:    time.Hour,
		MaxTurns:   20,
		Until:      "all requested outcomes have evidence",
		PolicySnapshot: &domain.ExecutionPolicySnapshot{
			CatalogDigest: "catalog-abc",
			SelectedCandidate: domain.ExecutionCandidate{
				PolicyDigest: "policy-def",
				PolicyPath:   "resources/codex/model-policy.json",
			},
			SelectionReason: "role_policy",
		},
	}

	got := buildRunAdmission(req, cfg)
	if got == nil {
		t.Fatal("admission is nil")
	}
	if got.RequestedRunner != "codex" || got.RequestedModel != "gpt-5.6-luna" ||
		got.RequestedRoleRef != "code.default" || got.RequestedEffort != "high" ||
		got.RequestedTimeout != timeout || got.RequestedMaxTurns != maxTurns ||
		got.RequestedGoalMode != "until" {
		t.Fatalf("requested side mismatch: %#v", got)
	}
	if got.EffectiveRunner != string(domain.RunnerTypeCodex) || got.EffectiveModel != "gpt-5.6-luna" ||
		got.EffectiveEffort != string(domain.EffortMedium) || got.EffectiveTimeout != time.Hour ||
		got.EffectiveMaxTurns != 20 || got.EffectiveUntil != "all requested outcomes have evidence" {
		t.Fatalf("effective side mismatch: %#v", got)
	}
	if got.CatalogDigest != "catalog-abc" || got.PolicyDigest != "policy-def" ||
		got.PolicyPath != "resources/codex/model-policy.json" || got.SelectionReason != "role_policy" {
		t.Fatalf("policy provenance mismatch: %#v", got)
	}
}

func TestBuildRunAdmissionLeavesUnrequestedFieldsEmpty(t *testing.T) {
	cfg := &domain.RunConfig{
		RunnerType: domain.RunnerTypeCodex,
		Model:      "gpt-5.6-luna",
		Effort:     domain.EffortMedium,
		Timeout:    time.Hour,
		MaxTurns:   20,
		PolicySnapshot: &domain.ExecutionPolicySnapshot{
			SelectedCandidate: domain.ExecutionCandidate{PolicyDigest: "policy-def"},
		},
	}

	got := buildRunAdmission(CreateRunRequest{}, cfg)
	if got == nil {
		t.Fatal("admission is nil")
	}
	if got.RequestedRunner != "" || got.RequestedModel != "" || got.RequestedRoleRef != "" ||
		got.RequestedEffort != "" || got.RequestedTimeout != 0 || got.RequestedMaxTurns != 0 ||
		got.RequestedGoalMode != "" {
		t.Fatalf("unrequested side should be empty: %#v", got)
	}
	if got.EffectiveRunner != string(domain.RunnerTypeCodex) || got.EffectiveTimeout != time.Hour {
		t.Fatalf("effective side mismatch: %#v", got)
	}
}

func TestRecordPassedInvocationCapturesControlArgs(t *testing.T) {
	registry := runner.NewRegistry()
	want := []string{"-m", "gpt-5.6-luna", "-c", "model_reasoning_effort=medium"}
	if err := registry.Register(&launchInfoStub{
		MockRunner:     runner.NewMockRunner(domain.RunnerTypeCodex),
		args:           want,
		runtimeVersion: "codex-cli 0.153.4",
	}); err != nil {
		t.Fatalf("register stub: %v", err)
	}
	o := &Orchestrator{runners: registry}
	admission := &domain.RunAdmission{}

	o.recordPassedInvocation(context.Background(), admission, &domain.RunConfig{RunnerType: domain.RunnerTypeCodex})

	if !reflect.DeepEqual(admission.PassedControlArgs, want) {
		t.Fatalf("passed control args = %#v, want %#v", admission.PassedControlArgs, want)
	}
	if admission.RuntimeVersion != "codex-cli 0.153.4" {
		t.Fatalf("runtime version = %q, want codex-cli 0.153.4", admission.RuntimeVersion)
	}
	if len(admission.TranslationDiagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", admission.TranslationDiagnostics)
	}
}

func TestRecordPassedInvocationRecordsRuntimeVersionDiagnostic(t *testing.T) {
	registry := runner.NewRegistry()
	if err := registry.Register(&launchInfoStub{
		MockRunner:      runner.NewMockRunner(domain.RunnerTypeCodex),
		runtimeVersionE: errors.New("binary missing"),
	}); err != nil {
		t.Fatalf("register stub: %v", err)
	}
	o := &Orchestrator{runners: registry}
	admission := &domain.RunAdmission{}

	o.recordPassedInvocation(context.Background(), admission, &domain.RunConfig{RunnerType: domain.RunnerTypeCodex})

	if admission.RuntimeVersion != "" {
		t.Fatalf("unobserved runtime version must stay empty, got %q", admission.RuntimeVersion)
	}
	found := false
	for _, d := range admission.TranslationDiagnostics {
		if strings.Contains(d, "runtime version not observed") {
			found = true
		}
	}
	if !found {
		t.Fatalf("want a runtime-version diagnostic, got %#v", admission.TranslationDiagnostics)
	}
}

func TestRecordPassedInvocationRecordsRefusal(t *testing.T) {
	registry := runner.NewRegistry()
	if err := registry.Register(&launchInfoStub{
		MockRunner: runner.NewMockRunner(domain.RunnerTypeCodex),
		err:        errors.New("codex has no native reasoning effort for \"max\""),
	}); err != nil {
		t.Fatalf("register stub: %v", err)
	}
	o := &Orchestrator{runners: registry}
	admission := &domain.RunAdmission{}

	o.recordPassedInvocation(context.Background(), admission, &domain.RunConfig{RunnerType: domain.RunnerTypeCodex})

	if len(admission.PassedControlArgs) != 0 {
		t.Fatalf("refused translation must not record passed args: %#v", admission.PassedControlArgs)
	}
	if len(admission.TranslationDiagnostics) != 1 {
		t.Fatalf("want one refusal diagnostic, got %#v", admission.TranslationDiagnostics)
	}
}

func TestRecordPassedInvocationWithoutRegistryRecordsDiagnostic(t *testing.T) {
	admission := &domain.RunAdmission{}
	(&Orchestrator{}).recordPassedInvocation(context.Background(), admission, &domain.RunConfig{RunnerType: domain.RunnerTypeCodex})
	if len(admission.TranslationDiagnostics) != 1 {
		t.Fatalf("want one diagnostic when the registry is unavailable, got %#v", admission.TranslationDiagnostics)
	}
}
