// Phase G of agent-sandbox-completion: profile/request AllowedPaths and
// DeniedPaths must surface as enforced SandboxConfig.Acceptance globs at
// resolve time. The runner-side advisory env vars are kept but the
// load-bearing enforcement is now at the workspace-sandbox apply-at-run-end
// boundary.

package orchestration

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestResolveSandboxConfig_PromotesProfilePathsToAcceptance(t *testing.T) {
	o := &Orchestrator{}
	profile := &domain.AgentProfile{
		AllowedPaths:  []string{"src/**", "docs/**"},
		DeniedPaths:   []string{"vendor/**"},
		SandboxConfig: domain.DefaultSandboxConfig(), RoleRef: "code.default",
	}
	cfg, err := o.resolveSandboxConfig(CreateRunRequest{}, profile)
	if err != nil {
		t.Fatalf("resolveSandboxConfig: %v", err)
	}
	if got := cfg.Acceptance.Allow.PathGlobs; !samePathGlobs(got, []string{"src/**", "docs/**"}) {
		t.Errorf("Acceptance.Allow.PathGlobs = %v, want [src/**, docs/**]", got)
	}
	if got := cfg.Acceptance.Deny.PathGlobs; !samePathGlobs(got, []string{"vendor/**"}) {
		t.Errorf("Acceptance.Deny.PathGlobs = %v, want [vendor/**]", got)
	}
}

func TestResolveSandboxConfig_RequestPathsOverrideProfile(t *testing.T) {
	o := &Orchestrator{}
	profile := &domain.AgentProfile{
		AllowedPaths:  []string{"src/**"},
		SandboxConfig: domain.DefaultSandboxConfig(), RoleRef: "code.default",
	}
	req := CreateRunRequest{AllowedPaths: []string{"tests/**"}}
	cfg, err := o.resolveSandboxConfig(req, profile)
	if err != nil {
		t.Fatalf("resolveSandboxConfig: %v", err)
	}
	if got := cfg.Acceptance.Allow.PathGlobs; !samePathGlobs(got, []string{"tests/**"}) {
		t.Errorf("Acceptance.Allow.PathGlobs = %v, want [tests/**] (req should override profile)", got)
	}
}

func TestResolveSandboxConfig_MergesWithExistingAcceptance(t *testing.T) {
	o := &Orchestrator{}
	cfg := domain.DefaultSandboxConfig()
	cfg.Acceptance.Allow.PathGlobs = []string{"existing/**"}
	profile := &domain.AgentProfile{
		AllowedPaths:  []string{"src/**"},
		SandboxConfig: cfg, RoleRef: "code.default",
	}
	resolved, err := o.resolveSandboxConfig(CreateRunRequest{}, profile)
	if err != nil {
		t.Fatalf("resolveSandboxConfig: %v", err)
	}
	if got := resolved.Acceptance.Allow.PathGlobs; !samePathGlobs(got, []string{"existing/**", "src/**"}) {
		t.Errorf("Acceptance.Allow.PathGlobs = %v, want merged [existing/**, src/**]", got)
	}
}

func samePathGlobs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
