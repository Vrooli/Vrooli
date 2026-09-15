package orchestration

import (
	"context"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func qualifiedParentRun() *domain.Run {
	return &domain.Run{
		ID: uuid.New(),
		ResolvedConfig: &domain.RunConfig{
			Admission: &domain.RunAdmission{
				RequestedRoleRef: "code.flatrate",
				EffectiveRunner:  "opencode",
				EffectiveModel:   "opencode-go/deepseek-v4.1-flash",
				EffectiveEffort:  "medium",
				Receipt: &domain.QualificationReceipt{
					Route:           "code.flatrate",
					EffectiveRunner: "opencode",
					EffectiveModel:  "opencode-go/deepseek-v4.1-flash",
					EffectiveEffort: "medium",
					RuntimeVersion:  "opencode-go/1.2.3",
					AcceptedOutput:  true,
				},
			},
		},
	}
}

func matchingChildConfig() *domain.RunConfig {
	return &domain.RunConfig{
		RunnerType: domain.RunnerTypeOpenCode,
		Model:      "opencode-go/deepseek-v4.1-flash",
		Effort:     domain.EffortMedium,
	}
}

func TestDependentDelegationAdmissionError_AdmitsOnlyMatchingQualifiedParent(t *testing.T) {
	if err := dependentDelegationAdmissionError(qualifiedParentRun(), matchingChildConfig()); err != nil {
		t.Fatalf("expected a matching qualified parent to admit the child: %v", err)
	}
}

func TestDependentDelegationAdmissionError_ClosesOnMissingEvidence(t *testing.T) {
	cases := []struct {
		name   string
		parent *domain.Run
		cfg    *domain.RunConfig
	}{
		{"nil parent", nil, matchingChildConfig()},
		{"nil resolved config", &domain.Run{ID: uuid.New()}, matchingChildConfig()},
		{"nil admission", &domain.Run{ID: uuid.New(), ResolvedConfig: &domain.RunConfig{}}, matchingChildConfig()},
		{"nil receipt and no live identity", func() *domain.Run {
			r := qualifiedParentRun()
			r.ResolvedConfig.Admission.Receipt = nil
			r.ResolvedConfig.Admission.RuntimeVersion = ""
			r.ResolvedConfig.Admission.PassedControlArgs = nil
			return r
		}(), matchingChildConfig()},
		{"nil child config", qualifiedParentRun(), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := dependentDelegationAdmissionError(tc.parent, tc.cfg); err == nil {
				t.Fatal("expected dependent delegation to stay closed")
			}
		})
	}
}

func TestDependentDelegationAdmissionError_AdmitsLiveCoordinatorBeforeTerminalReceipt(t *testing.T) {
	parent := qualifiedParentRun()
	parent.Status = domain.RunStatusRunning
	parent.ResolvedConfig.Admission.Receipt = nil
	parent.ResolvedConfig.Admission.RuntimeVersion = "codex-cli 0.153.4"
	parent.ResolvedConfig.Admission.PassedControlArgs = []string{"-m", "opencode-go/deepseek-v4.1-flash"}
	if err := dependentDelegationAdmissionError(parent, matchingChildConfig()); err != nil {
		t.Fatalf("expected live coordinator identity to admit child before terminal receipt: %v", err)
	}
}

func TestDependentDelegationAdmissionError_ClosesOnIdentityMismatch(t *testing.T) {
	mismatched := matchingChildConfig()
	mismatched.Model = "openrouter/deepseek/deepseek-v4.1-flash"
	if err := dependentDelegationAdmissionError(qualifiedParentRun(), mismatched); err == nil {
		t.Fatal("expected a mismatched child model to stay closed")
	}
}

func TestAdmitDependentDelegation_NonChildRunIsUnaffected(t *testing.T) {
	// A run without a parent is not dependent delegation; the gate must not
	// require a receipt or a repository for it.
	svc := &Orchestrator{}
	if err := svc.admitDependentDelegation(context.Background(), CreateRunRequest{}, matchingChildConfig()); err != nil {
		t.Fatalf("expected non-child run to be admitted unchanged: %v", err)
	}
}
