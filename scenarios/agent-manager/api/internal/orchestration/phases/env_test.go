package phases

import (
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestSandboxEnvVars_NilForInPlace(t *testing.T) {
	if got := SandboxEnvVars(SandboxEnvInput{RunMode: domain.RunModeInPlace}); got != nil {
		t.Errorf("expected nil for in-place run, got %v", got)
	}
}

func TestSandboxEnvVars_NilWhenIDOrWorkDirMissing(t *testing.T) {
	id := uuid.New()
	if got := SandboxEnvVars(SandboxEnvInput{
		RunMode:   domain.RunModeSandboxed,
		SandboxID: nil,
		WorkDir:   "/tmp/x",
	}); got != nil {
		t.Errorf("expected nil when sandboxID missing, got %v", got)
	}
	if got := SandboxEnvVars(SandboxEnvInput{
		RunMode:   domain.RunModeSandboxed,
		SandboxID: &id,
		WorkDir:   "",
	}); got != nil {
		t.Errorf("expected nil when workDir empty, got %v", got)
	}
}

func TestSandboxEnvVars_PopulatedForSandboxed(t *testing.T) {
	id := uuid.New()
	got := SandboxEnvVars(SandboxEnvInput{
		RunMode:   domain.RunModeSandboxed,
		SandboxID: &id,
		WorkDir:   "/tmp/sbx/merged",
		RepoRoot:  "/repo/Vrooli",
		ScopePath: "scenarios/foo",
	})
	if got["VROOLI_SANDBOX_ID"] != id.String() {
		t.Errorf("VROOLI_SANDBOX_ID mismatch: %q", got["VROOLI_SANDBOX_ID"])
	}
	if got["VROOLI_SANDBOX_MERGED"] != "/tmp/sbx/merged" {
		t.Errorf("VROOLI_SANDBOX_MERGED mismatch: %q", got["VROOLI_SANDBOX_MERGED"])
	}
	if got["VROOLI_SANDBOX_MERGED_HOST"] != got["VROOLI_SANDBOX_MERGED"] {
		t.Errorf("merged path pair disagrees at source: %v", got)
	}
	if got["VROOLI_SANDBOX_REPO_ROOT"] != "/repo/Vrooli" {
		t.Errorf("VROOLI_SANDBOX_REPO_ROOT mismatch: %q", got["VROOLI_SANDBOX_REPO_ROOT"])
	}
	if got["VROOLI_SANDBOX_SCOPE"] != "scenarios/foo" {
		t.Errorf("VROOLI_SANDBOX_SCOPE mismatch: %q", got["VROOLI_SANDBOX_SCOPE"])
	}
}

func TestSandboxEnvVars_EmitsDotScopeWhenEmpty(t *testing.T) {
	id := uuid.New()
	got := SandboxEnvVars(SandboxEnvInput{
		RunMode:   domain.RunModeSandboxed,
		SandboxID: &id,
		WorkDir:   "/tmp/sbx/merged",
	})
	if got["VROOLI_SANDBOX_SCOPE"] != "." {
		t.Errorf("VROOLI_SANDBOX_SCOPE = %q, want .", got["VROOLI_SANDBOX_SCOPE"])
	}
}

func TestIdentityEnvVars(t *testing.T) {
	if got := IdentityEnvVars(""); got != nil {
		t.Errorf("expected nil for empty token, got %v", got)
	}

	got := IdentityEnvVars("tok-1")
	want := map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": "tok-1"}
	if len(got) != len(want) {
		t.Fatalf("IdentityEnvVars returned %v, want %v", got, want)
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %q, want %q", key, got[key], value)
		}
	}

	for _, key := range []string{
		"VROOLI_AGENT_MANAGER_API_BASE",
		"AGENT_MANAGER_API_BASE",
		"SWARM_MANAGER_API_BASE",
		"WORKSPACE_SANDBOX_API_BASE",
	} {
		if _, ok := got[key]; ok {
			t.Errorf("IdentityEnvVars should not synthesize %s", key)
		}
	}
}

func TestMergeEnvVars_SystemOverridesCustom(t *testing.T) {
	got := MergeEnvVars(MergeEnvInput{
		Custom: map[string]string{
			"VROOLI_SANDBOX_ID":      "evil",
			"SWARM_MANAGER_API_BASE": "http://caller.example",
			"USER_KEY":               "1",
		},
		Sandbox:  map[string]string{"VROOLI_SANDBOX_ID": "real"},
		Identity: IdentityEnvVars("tok"),
	})
	if got["VROOLI_SANDBOX_ID"] != "real" {
		t.Errorf("custom must not shadow sandbox env: %v", got["VROOLI_SANDBOX_ID"])
	}
	if got["USER_KEY"] != "1" {
		t.Errorf("user key dropped: %v", got)
	}
	if got["VROOLI_AGENT_IDENTITY_TOKEN"] != "tok" {
		t.Errorf("identity not merged: %v", got)
	}
	if got["SWARM_MANAGER_API_BASE"] != "http://caller.example" {
		t.Errorf("caller-owned API base should not be shadowed: %v", got["SWARM_MANAGER_API_BASE"])
	}
	if _, ok := got["VROOLI_AGENT_MANAGER_API_BASE"]; ok {
		t.Errorf("identity merge should not add agent-manager API base: %v", got)
	}
}

func TestMergeEnvVars_NilWhenAllEmpty(t *testing.T) {
	if got := MergeEnvVars(MergeEnvInput{}); got != nil {
		t.Errorf("expected nil for empty merge, got %v", got)
	}
}

// TestAssembleRunEnv_AllThreeSources is the Phase 0 contract test: the single
// env assembler used by both Execute and continue/wake must merge custom env,
// sandbox routing, and the identity token, with system vars (sandbox +
// identity) taking precedence over caller-supplied custom env.
func TestAssembleRunEnv_AllThreeSources(t *testing.T) {
	sandboxID := uuid.New()
	got := AssembleRunEnv(AssembleRunEnvInput{
		Custom: map[string]string{
			"VROOLI_CUSTOM_FLAG": "on",
			"VROOLI_SANDBOX_ID":  "evil", // must not shadow the real sandbox id
		},
		RunMode:       domain.RunModeSandboxed,
		SandboxID:     &sandboxID,
		WorkDir:       "/work/merged",
		ScopePath:     "scenarios/foo",
		IdentityToken: "tok-123",
	})

	if got["VROOLI_CUSTOM_FLAG"] != "on" {
		t.Errorf("custom env dropped: %v", got)
	}
	if got["VROOLI_SANDBOX_ID"] != sandboxID.String() {
		t.Errorf("custom must not shadow sandbox id: %v", got["VROOLI_SANDBOX_ID"])
	}
	if got["VROOLI_SANDBOX_MERGED"] != "/work/merged" {
		t.Errorf("sandbox merged path missing: %v", got)
	}
	if got["VROOLI_SANDBOX_SCOPE"] != "scenarios/foo" {
		t.Errorf("sandbox scope missing: %v", got)
	}
	if got["VROOLI_AGENT_IDENTITY_TOKEN"] != "tok-123" {
		t.Errorf("identity token missing: %v", got)
	}
}

// TestAssembleRunEnv_NonSandboxedDropsSandboxVars proves the host-run / no-token
// case carries only custom env (no synthesized VROOLI_SANDBOX_* trio).
func TestAssembleRunEnv_NonSandboxedDropsSandboxVars(t *testing.T) {
	got := AssembleRunEnv(AssembleRunEnvInput{
		Custom:  map[string]string{"VROOLI_CUSTOM_FLAG": "on"},
		RunMode: domain.RunModeInPlace,
	})
	if got["VROOLI_CUSTOM_FLAG"] != "on" {
		t.Errorf("custom env dropped: %v", got)
	}
	for _, k := range []string{"VROOLI_SANDBOX_ID", "VROOLI_SANDBOX_MERGED", "VROOLI_SANDBOX_SCOPE", "VROOLI_AGENT_IDENTITY_TOKEN"} {
		if _, ok := got[k]; ok {
			t.Errorf("non-sandboxed/no-token run should not carry %s: %v", k, got)
		}
	}
}
