package hygiene

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPathAuthorityProviderRejectsHomeRuntimeJoin(t *testing.T) {
	root := t.TempDir()
	for _, scenario := range []string{"vrooli-autoheal", "agent-manager", "prompt-manager"} {
		if err := os.MkdirAll(filepath.Join(root, "scenarios", scenario), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	bad := filepath.Join(root, "scenarios", "agent-manager", "bad.go")
	if err := os.WriteFile(bad, []byte("package bad\nvar _ = filepath.Join(home, \".vrooli\")\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var report Report
	if err := (pathAuthorityProvider{root: root}).Run(context.Background(), Request{}, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 || report.Findings[0].Code != "runtime_path_authority_bypass" {
		t.Fatalf("provider findings = %+v", report.Findings)
	}
}

func TestPathAuthorityProviderPassesMigratedTree(t *testing.T) {
	root := t.TempDir()
	for _, scenario := range []string{"vrooli-autoheal", "agent-manager", "prompt-manager"} {
		if err := os.MkdirAll(filepath.Join(root, "scenarios", scenario), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var report Report
	if err := (pathAuthorityProvider{root: root}).Run(context.Background(), Request{}, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("migrated tree findings = %+v", report.Findings)
	}
}

func TestPathAuthorityProviderCurrentRepositoryIsClean(t *testing.T) {
	var report Report
	if err := (pathAuthorityProvider{root: filepath.Clean(filepath.Join("..", "..", ".."))}).Run(context.Background(), Request{}, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("repository findings = %+v", report.Findings)
	}
}
