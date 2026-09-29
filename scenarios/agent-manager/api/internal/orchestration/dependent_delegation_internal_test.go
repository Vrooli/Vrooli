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

func TestDependentDelegationAdmissionError_PreservesExecutionRestrictions(t *testing.T) {
	boundedConfig := func() *domain.RunConfig {
		cfg := matchingChildConfig()
		cfg.NetworkAccess = domain.NetworkAccessNone
		cfg.AllowedTools = []string{"read", "edit", "shell"}
		cfg.DeniedTools = []string{"web_fetch"}
		cfg.AllowedPaths = []string{"src/**"}
		cfg.DeniedPaths = []string{"docs/**", ".git/**"}
		cfg.AllowedEffects = []string{"filesystem.write[paths=src/**]"}
		cfg.RequireEffectContainment = true
		cfg.SandboxConfig = domain.DefaultSandboxConfig()
		cfg.SandboxConfig.ManualReview = true
		cfg.SandboxConfig.AutoApply = new(bool)
		cfg.SandboxConfig.ApplyOnFailure = new(bool)
		cfg.SandboxConfig.NetworkMode = domain.NetworkAccessNone
		cfg.SandboxConfig.Acceptance.Deny.PathGlobs = []string{"docs/**"}
		cfg.SandboxConfig.WritePolicy = &domain.WorkspaceWritePolicy{Paths: []string{"src"}}
		return cfg
	}
	for _, tc := range []struct {
		name   string
		change func(*domain.RunConfig)
	}{
		{"network", func(c *domain.RunConfig) { c.NetworkAccess = domain.NetworkAccessFull }},
		{"sandbox-network", func(c *domain.RunConfig) { c.SandboxConfig.NetworkMode = domain.NetworkAccessFull }},
		{"sandbox-off", func(c *domain.RunConfig) { c.SandboxConfig.Mode = domain.SandboxModeOff }},
		{"sandbox-tracking", func(c *domain.RunConfig) { c.SandboxConfig.Mode = domain.SandboxModeTracking }},
		{"sandbox-omitted", func(c *domain.RunConfig) { c.SandboxConfig = nil }},
		{"manual-review", func(c *domain.RunConfig) { c.SandboxConfig.ManualReview = false }},
		{"automatic-apply", func(c *domain.RunConfig) { c.SandboxConfig.AutoApply = nil }},
		{"apply-on-failure", func(c *domain.RunConfig) { c.SandboxConfig.ApplyOnFailure = nil }},
		{"acceptance", func(c *domain.RunConfig) { c.SandboxConfig.Acceptance.Deny.PathGlobs = nil }},
		{"write-policy-omitted", func(c *domain.RunConfig) { c.SandboxConfig.WritePolicy = nil }},
		{"write-policy-widened", func(c *domain.RunConfig) { c.SandboxConfig.WritePolicy.Paths = []string{"docs"} }},
		{"allowed-tools", func(c *domain.RunConfig) { c.AllowedTools = nil }},
		{"denied-tools", func(c *domain.RunConfig) { c.DeniedTools = nil }},
		{"tool-enforcement", func(c *domain.RunConfig) { c.ToolRestrictionPolicy = domain.ToolRestrictionPolicyAdvisory }},
		{"allowed-paths", func(c *domain.RunConfig) { c.AllowedPaths = nil }},
		{"denied-paths", func(c *domain.RunConfig) { c.DeniedPaths = nil }},
		{"approval-bypass", func(c *domain.RunConfig) { c.SkipPermissionPrompt = true }},
		{"browser", func(c *domain.RunConfig) { c.Features.EnableBrowser = true }},
		{"raw-flags", func(c *domain.RunConfig) {
			c.ExtraFlags = domain.RunnerExtraFlags{domain.RunnerTypeOpenCode: {"--dangerous"}}
		}},
		{"effect-grant", func(c *domain.RunConfig) {
			c.AllowedEffects = append(c.AllowedEffects, "publication.write[target=release]")
		}},
		{"effect-containment", func(c *domain.RunConfig) { c.RequireEffectContainment = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := qualifiedParentRun()
			admission := parent.ResolvedConfig.Admission
			parent.ResolvedConfig = boundedConfig()
			parent.ResolvedConfig.Admission = admission
			child := boundedConfig()
			if err := dependentDelegationAdmissionError(parent, child); err != nil {
				t.Fatalf("unchanged restrictions refused: %v", err)
			}
			tc.change(child)
			if err := dependentDelegationAdmissionError(parent, child); err == nil {
				t.Fatal("matching model admitted a child with changed execution authority")
			}
		})
	}
	t.Run("narrowing", func(t *testing.T) {
		parent := qualifiedParentRun()
		admission := parent.ResolvedConfig.Admission
		parent.ResolvedConfig = boundedConfig()
		parent.ResolvedConfig.Admission = admission
		parent.ResolvedConfig.NetworkAccess = domain.NetworkAccessFull
		parent.ResolvedConfig.SkipPermissionPrompt = true
		parent.ResolvedConfig.Features.EnableBrowser = true
		parent.ResolvedConfig.ToolRestrictionPolicy = domain.ToolRestrictionPolicyAdvisory
		parent.ResolvedConfig.AllowedTools = append(parent.ResolvedConfig.AllowedTools, "web_search")
		parent.ResolvedConfig.DeniedTools = nil
		parent.ResolvedConfig.AllowedPaths = append(parent.ResolvedConfig.AllowedPaths, "tests/**")
		parent.ResolvedConfig.DeniedPaths = nil
		parent.ResolvedConfig.AllowedEffects = append(parent.ResolvedConfig.AllowedEffects, "process.test[scope=example]")
		parent.ResolvedConfig.RequireEffectContainment = false
		parent.ResolvedConfig.SandboxConfig.Mode = domain.SandboxModeTracking
		parent.ResolvedConfig.SandboxConfig.ManualReview = false
		parent.ResolvedConfig.SandboxConfig.AutoApply = nil
		parent.ResolvedConfig.SandboxConfig.ApplyOnFailure = nil
		parent.ResolvedConfig.SandboxConfig.NetworkMode = domain.NetworkAccessFull
		if err := dependentDelegationAdmissionError(parent, boundedConfig()); err != nil {
			t.Fatalf("a child may reduce its execution authority: %v", err)
		}
	})
}

func TestAdmitDependentDelegation_NonChildRunIsUnaffected(t *testing.T) {
	// A run without a parent is not dependent delegation; the gate must not
	// require a receipt or a repository for it.
	svc := &Orchestrator{}
	if err := svc.admitDependentDelegation(context.Background(), CreateRunRequest{}, matchingChildConfig()); err != nil {
		t.Fatalf("expected non-child run to be admitted unchanged: %v", err)
	}
}
