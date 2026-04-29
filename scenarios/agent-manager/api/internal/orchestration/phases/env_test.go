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
	if got := IdentityEnvVars(""); got != nil {
		t.Errorf("expected nil for empty token, got %v", got)
	}
	got := IdentityEnvVars("tok-1")
	if got["VROOLI_AGENT_IDENTITY_TOKEN"] != "tok-1" {
		t.Errorf("token not forwarded: %v", got)
	}
}

func TestMergeEnvVars_SystemOverridesCustom(t *testing.T) {
	got := MergeEnvVars(MergeEnvInput{
		Custom:   map[string]string{"VROOLI_SANDBOX_ID": "evil", "USER_KEY": "1"},
		Sandbox:  map[string]string{"VROOLI_SANDBOX_ID": "real"},
		Identity: map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": "tok"},
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
}

func TestMergeEnvVars_NilWhenAllEmpty(t *testing.T) {
	if got := MergeEnvVars(MergeEnvInput{}); got != nil {
		t.Errorf("expected nil for empty merge, got %v", got)
	}
}
