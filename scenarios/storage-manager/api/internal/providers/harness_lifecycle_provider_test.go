package providers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	harnesslifecycle "github.com/vrooli/vrooli/packages/harness-lifecycle"
	"storage-manager/hostfs"
	"storage-manager/internal/cleanup"
)

func TestHarnessLifecycleProviderPreservesFoundCopies(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, "tools")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	found := filepath.Join(bin, "codex")
	if err := os.WriteFile(found, []byte("#!/bin/sh\necho 'codex 1.0.0'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+"/usr/bin:/bin")
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_ROOT", root)
	p := NewHarnessLifecycleProvider(hostfs.New(hostfs.Options{}), fixedCleanupClock{time.Now()}, root)
	preview, err := p.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOwner}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 0 {
		t.Fatalf("found copies became cleanup items: %+v", preview.Items)
	}
	foundWarning := false
	for _, warning := range preview.Warnings {
		if strings.Contains(warning, found) {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Fatalf("preview did not state that the found copy is preserved: %+v", preview.Warnings)
	}
	if data, err := os.ReadFile(found); err != nil || string(data) != "#!/bin/sh\necho 'codex 1.0.0'\n" {
		t.Fatalf("found copy changed: %v", err)
	}
}

func TestHarnessLifecycleProviderAppliesExactPreviewAndVerifiesRemovedPaths(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_ROOT", root)
	h, err := harnesslifecycle.LoadHarness(root, "codex")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"1.0.0", "2.0.0", "3.0.0"} {
		prefix, err := harnesslifecycle.NPMInstallPrefix(home, "codex", version)
		if err != nil {
			t.Fatal(err)
		}
		executable := filepath.Join(prefix, filepath.FromSlash(h.Executable))
		if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(executable, []byte("#!/bin/sh\necho 'codex "+version+"'\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		digest, err := binaryfetch.TreeDigest(filepath.Join(prefix, "node_modules"))
		if err != nil {
			t.Fatal(err)
		}
		receipt, _ := json.Marshal(struct{ Resource, Package, Version, TreeSHA256 string }{h.Resource, h.Package, version, digest})
		if err := os.WriteFile(filepath.Join(prefix, "installation.json"), receipt, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	active, err := harnesslifecycle.NPMInstallPrefix(home, "codex", "3.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := harnesslifecycle.LinkNPMBinary(active, "codex", bin); err != nil {
		t.Fatal(err)
	}
	p := NewHarnessLifecycleProvider(hostfs.New(hostfs.Options{}), fixedCleanupClock{time.Now()}, root)
	preview, err := p.Preview(context.Background(), cleanup.PreviewRequest{Policy: cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOwner}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("expected exact obsolete path preview, got %+v", preview.Items)
	}
	approvedPath := preview.Items[0].Path
	result, err := p.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: p.meta.Version, ApprovalMode: cleanup.ApprovalModeOwner, Preview: preview})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied || len(result.AppliedItems) != 1 || result.AppliedItems[0] != approvedPath {
		t.Fatalf("applied result=%+v preview=%s", result, approvedPath)
	}
	verified, err := p.Verify(context.Background(), cleanup.VerifyRequest{ApplyResult: result})
	if err != nil || !verified.Verified {
		t.Fatalf("verify=%+v err=%v", verified, err)
	}
	if _, err := os.Stat(approvedPath); !os.IsNotExist(err) {
		t.Fatalf("previewed obsolete path remains: %v", err)
	}
	for _, version := range []string{"2.0.0", "3.0.0"} {
		prefix, _ := harnesslifecycle.NPMInstallPrefix(home, "codex", version)
		if _, err := os.Stat(prefix); err != nil {
			t.Fatalf("preserved %s: %v", version, err)
		}
	}
}

type fixedCleanupClock struct{ at time.Time }

func (c fixedCleanupClock) Now() time.Time { return c.at }
