package handlers

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestProcessHandlersDoNotDerivePeerScenarioPaths(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	for _, name := range []string{"process.go", "process_start.go"} {
		content, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), name))
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
