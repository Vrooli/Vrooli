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
		ScopePath: "scenarios/foo",
	})
	if got["VROOLI_SANDBOX_ID"] != id.String() {
		t.Errorf("VROOLI_SANDBOX_ID mismatch: %q", got["VROOLI_SANDBOX_ID"])
	}
	if got["VROOLI_SANDBOX_MERGED"] != "/tmp/sbx/merged" {
		t.Errorf("VROOLI_SANDBOX_MERGED mismatch: %q", got["VROOLI_SANDBOX_MERGED"])
	}
	if got["VROOLI_SANDBOX_SCOPE"] != "scenarios/foo" {
		t.Errorf("VROOLI_SANDBOX_SCOPE mismatch: %q", got["VROOLI_SANDBOX_SCOPE"])
	}
}

func TestSandboxEnvVars_OmitsScopeWhenEmpty(t *testing.T) {
	id := uuid.New()
	got := SandboxEnvVars(SandboxEnvInput{
		RunMode:   domain.RunModeSandboxed,
		SandboxID: &id,
		WorkDir:   "/tmp/sbx/merged",
	})
	if _, ok := got["VROOLI_SANDBOX_SCOPE"]; ok {
		t.Errorf("expected VROOLI_SANDBOX_SCOPE omitted when empty, got %q", got["VROOLI_SANDBOX_SCOPE"])
	}
}

func TestIdentityEnvVars(t *testing.T) {
	t.Setenv("VROOLI_AGENT_MANAGER_API_BASE", "")
	t.Setenv("API_BASE_URL", "")
	t.Setenv("API_PORT", "")

	if got := IdentityEnvVars(""); got != nil {
		t.Errorf("expected nil for empty token, got %v", got)
	}
	got := IdentityEnvVars("tok-1")
	if got["VROOLI_AGENT_IDENTITY_TOKEN"] != "tok-1" {
		t.Errorf("token not forwarded: %v", got)
	}
	if got["VROOLI_AGENT_MANAGER_API_BASE"] != "http://localhost:18800" {
		t.Errorf("agent-manager API base = %q, want default base", got["VROOLI_AGENT_MANAGER_API_BASE"])
	}
}

func TestIdentityEnvVars_UsesConfiguredAgentManagerBase(t *testing.T) {
	t.Setenv("VROOLI_AGENT_MANAGER_API_BASE", "http://agent-manager.internal:18800")
	t.Setenv("API_BASE_URL", "http://wrong.example")
	t.Setenv("API_PORT", "19999")

	got := IdentityEnvVars("tok-1")
	if got["VROOLI_AGENT_MANAGER_API_BASE"] != "http://agent-manager.internal:18800" {
		t.Errorf("agent-manager API base = %q, want configured VROOLI_AGENT_MANAGER_API_BASE", got["VROOLI_AGENT_MANAGER_API_BASE"])
	}
}

func TestIdentityEnvVars_FallsBackToScenarioAPIBase(t *testing.T) {
	t.Setenv("VROOLI_AGENT_MANAGER_API_BASE", "")
	t.Setenv("API_BASE_URL", "http://localhost:19901")
	t.Setenv("API_PORT", "19999")

	got := IdentityEnvVars("tok-1")
	if got["VROOLI_AGENT_MANAGER_API_BASE"] != "http://localhost:19901" {
		t.Errorf("agent-manager API base = %q, want API_BASE_URL fallback", got["VROOLI_AGENT_MANAGER_API_BASE"])
	}
}

func TestMergeEnvVars_SystemOverridesCustom(t *testing.T) {
	t.Setenv("VROOLI_AGENT_MANAGER_API_BASE", "http://agent-manager.internal:18800")
	got := MergeEnvVars(MergeEnvInput{
		Custom:   map[string]string{"VROOLI_SANDBOX_ID": "evil", "VROOLI_AGENT_MANAGER_API_BASE": "evil", "USER_KEY": "1"},
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
	if got["VROOLI_AGENT_MANAGER_API_BASE"] != "http://agent-manager.internal:18800" {
		t.Errorf("custom must not shadow identity API base: %v", got["VROOLI_AGENT_MANAGER_API_BASE"])
	}
}

func TestMergeEnvVars_NilWhenAllEmpty(t *testing.T) {
	if got := MergeEnvVars(MergeEnvInput{}); got != nil {
		t.Errorf("expected nil for empty merge, got %v", got)
	}
}
