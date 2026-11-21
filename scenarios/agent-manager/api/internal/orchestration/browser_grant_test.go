package orchestration

import (
	"context"
	"strings"
	"testing"

	"agent-manager/internal/browsercapability"
)

func TestResolveBrowserSessionProfileModePersistsCanonicalMode(t *testing.T) {
	for _, tc := range []struct {
		mode browsercapability.Mode
		want browsercapability.SessionProfileMode
	}{
		{browsercapability.ModeManaged, browsercapability.SessionProfileFresh},
		{browsercapability.ModeInstalled, browsercapability.SessionProfileIsolated},
		{browsercapability.ModeAttached, browsercapability.SessionProfileUserAttached},
		{browsercapability.ModeRemote, browsercapability.SessionProfileRemote},
	} {
		capability := browsercapability.Capability{Mode: tc.mode}
		if err := resolveBrowserSessionProfileMode(&capability); err != nil {
			t.Errorf("mode %q: %v", tc.mode, err)
		} else if capability.Permissions.ProfileMode != tc.want {
			t.Errorf("mode %q persisted profile mode %q, want %q", tc.mode, capability.Permissions.ProfileMode, tc.want)
		}
	}
	conflict := browsercapability.Capability{Mode: browsercapability.ModeInstalled,
		Permissions: browsercapability.PermissionProfileSpec{ProfileMode: browsercapability.SessionProfileFresh}}
	if err := resolveBrowserSessionProfileMode(&conflict); err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("caller-selected profile mode conflicting with backend accepted: %v", err)
	}
	unknown := browsercapability.Capability{Mode: "unknown"}
	if err := resolveBrowserSessionProfileMode(&unknown); err == nil {
		t.Fatal("unknown browser backend produced a session profile mode")
	}
}

func TestResolveRunConfigRejectsBrowserCapabilityWithoutPolicyResolvedBrowserRole(t *testing.T) {
	f := newModeFixture(t, nil)
	role := "code.default"
	requested := browsercapability.Capability{
		Enabled:      true,
		GrantReceipt: "caller-chosen-value-must-not-survive",
		Candidates:   []browsercapability.Candidate{{Harness: "codex", ResourceRole: "browser.session"}},
		Mode:         browsercapability.ModeManaged,
		Permissions:  browsercapability.PermissionProfileSpec{ProfileID: "profile-browser-observe", Level: browsercapability.ProfileObserve, AllowedOrigins: []string{"https://example.test"}},
		Budget:       browsercapability.Budget{SessionMinutes: 5, Actions: 20, Bytes: 100_000},
	}
	if _, _, err := f.o.resolveRunConfig(context.Background(), CreateRunRequest{RoleRef: &role, BrowserCapability: &requested}); err == nil || !strings.Contains(err.Error(), "no eligible browser capability candidate") {
		t.Fatalf("role with only code.default must not confer browser authority: %v", err)
	}
	requested.Candidates = nil
	if _, _, err := f.o.resolveRunConfig(context.Background(), CreateRunRequest{RoleRef: &role, BrowserCapability: &requested}); err == nil {
		t.Fatal("omitted candidates must not infer browser access from an ordinary execution role")
	}
}

func TestResolveRunConfigBindsBrowserGrantToExplicitRoleProfileBudgetAndModelCandidate(t *testing.T) {
	f := newModeFixture(t, nil)
	role := "browser.agent"
	requested := browsercapability.Capability{
		Enabled: true, GrantReceipt: "caller-chosen-value-must-not-survive",
		Candidates:  []browsercapability.Candidate{{Harness: "codex", ResourceRole: browsercapability.ResourceRoleSession}},
		Mode:        browsercapability.ModeManaged,
		Permissions: browsercapability.PermissionProfileSpec{ProfileID: "profile-browser-act", Level: browsercapability.ProfileAct, AllowedOrigins: []string{"https://example.test"}},
		Budget:      browsercapability.Budget{SessionMinutes: 7, Actions: 80, Bytes: 200_000, InferenceTokens: 4096},
	}
	cfg, _, err := f.o.resolveRunConfig(context.Background(), CreateRunRequest{RoleRef: &role, BrowserCapability: &requested})
	if err != nil {
		t.Fatalf("explicit browser role failed to resolve: %v", err)
	}
	capability := cfg.Features.BrowserCapability
	if capability == nil || !capability.Enabled || capability.GrantReceipt == "" || capability.GrantReceipt == requested.GrantReceipt {
		t.Fatalf("run-scoped grant was not minted from resolved capability: %+v", capability)
	}
	if capability.Permissions.ProfileID != "profile-browser-act" || capability.Permissions.ProfileMode != browsercapability.SessionProfileFresh || capability.Budget.SessionMinutes != 7 || capability.Budget.Actions != 80 || capability.Budget.Bytes != 200_000 || capability.Budget.InferenceTokens != 4096 {
		t.Fatalf("saved browser permission profile or budget changed: %+v", capability)
	}
	if cfg.PolicySnapshot == nil || cfg.PolicySnapshot.RoleRef != "browser.agent" || len(cfg.PolicySnapshot.Candidates) != 1 || cfg.PolicySnapshot.Candidates[0].ResourceRole != "code.default" {
		t.Fatalf("browser capability must not become the selected model candidate: %+v", cfg.PolicySnapshot)
	}
	if len(cfg.PolicySnapshot.BrowserCandidates) != 1 || !cfg.PolicySnapshot.BrowserCandidates[0].Available || cfg.PolicySnapshot.BrowserCandidates[0].ResourceRole != browsercapability.ResourceRoleSession {
		t.Fatalf("browser candidate was not separately persisted: %+v", cfg.PolicySnapshot.BrowserCandidates)
	}
}
