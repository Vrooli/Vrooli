package handlers

import (
	"os"
	"path/filepath"
	"testing"

	driverexec "workspace-sandbox/internal/driver/exec"
)

func TestAddAgentManagerSessionHomeBinds_AllowsOnlyRunScopedCodecHomes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	runID := "a2a10634-8729-442e-9a66-be86bc2cab1b"
	codexHome := filepath.Join(home, ".vrooli", "data", "vrooli", "agent-manager", "runs", runID, "codex")
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := driverexec.DefaultBwrapConfig()
	if err := addAgentManagerSessionHomeBinds(&cfg, map[string]string{"CODEX_HOME": codexHome}); err != nil {
		t.Fatalf("addAgentManagerSessionHomeBinds() error = %v", err)
	}
	if got := cfg.ReadWriteBinds[codexHome]; got != codexHome {
		t.Fatalf("bind = %q, want %q", got, codexHome)
	}

	if err := addAgentManagerSessionHomeBinds(&cfg, map[string]string{"CODEX_HOME": filepath.Join(home, ".codex")}); err == nil {
		t.Fatal("expected arbitrary CODEX_HOME to be rejected")
	}
}
