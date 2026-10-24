package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	driverexec "workspace-sandbox/internal/driver/exec"
	"workspace-sandbox/internal/types"
)

func TestAddWritableMounts_UsesRegisteredRoots(t *testing.T) {
	root := t.TempDir()
	codexHome := filepath.Join(root, "runs", "a2a10634-8729-442e-9a66-be86bc2cab1b", "codex")
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := driverexec.DefaultBwrapConfig()
	sb := &types.Sandbox{ProjectRoot: t.TempDir(), AuxiliaryRoots: []string{root}}
	if err := addWritableMounts(&cfg, sb, []WritableMount{{Path: codexHome, Purpose: "codec-state"}}); err != nil {
		t.Fatalf("addWritableMounts() error = %v", err)
	}
	if got := cfg.ReadWriteBinds[codexHome]; got != codexHome {
		t.Fatalf("bind = %q, want %q", got, codexHome)
	}

	outside := t.TempDir()
	if err := addWritableMounts(&cfg, sb, []WritableMount{{Path: outside, Purpose: "codec-state"}}); err == nil {
		t.Fatal("expected mount outside registered roots to be rejected")
	}
}

func TestAddPolicyFilesRequiresRegisteredSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "policy")
	if err := os.WriteFile(source, []byte("policy"), 0600); err != nil {
		t.Fatal(err)
	}
	sb := &types.Sandbox{ProjectRoot: t.TempDir(), AuxiliaryRoots: []string{root}}
	cfg := driverexec.DefaultBwrapConfig()
	files := []types.PolicyFile{{Source: source, Target: "/etc/fixture/requirements.toml", SHA256: "checked by backend"}}
	if err := addPolicyFiles(&cfg, sb, files); err != nil {
		t.Fatal(err)
	}
	if len(cfg.PolicyFiles) != 1 || cfg.PolicyFiles[0].Source != source {
		t.Fatal("policy file was dropped")
	}
	sb.AuxiliaryRoots = nil
	if err := addPolicyFiles(&cfg, sb, files); err == nil {
		t.Fatal("unregistered source was admitted")
	}
}

func TestProcessHandlersDoNotDerivePeerScenarioPaths(t *testing.T) {
	for _, name := range []string{"process.go", "process_start.go"} {
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		peerName := "agent" + "-manager"
		if strings.Contains(string(content), peerName) {
			t.Fatalf("%s must not derive a peer scenario path", name)
		}
	}
}

func TestAddWritableMounts_RejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	cfg := driverexec.DefaultBwrapConfig()
	sb := &types.Sandbox{ProjectRoot: t.TempDir(), AuxiliaryRoots: []string{root}}
	err := addWritableMounts(&cfg, sb, []WritableMount{{Path: link, Purpose: "codec-state"}})
	if err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}
