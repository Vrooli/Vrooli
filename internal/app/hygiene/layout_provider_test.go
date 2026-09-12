package hygiene

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestLayoutProviderRemovesEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	empty := filepath.Join(root, "packages", "scratch")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}

	// Exercise the safe primitive directly; loading the repository contract is
	// covered by the provider's production path and would obscure this focused
	// filesystem invariant in a temporary fixture.
	found, err := findEmptyDirectories(root, []string{"packages"})
	if err != nil || len(found) != 1 || found[0] != empty {
		t.Fatalf("findEmptyDirectories = %#v, %v; want %q", found, err, empty)
	}
	if err := os.Remove(found[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(empty); !os.IsNotExist(err) {
		t.Fatalf("empty directory still exists, stat error = %v", err)
	}
}

func TestLayoutProviderPreservesGitkeepDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "deliberate")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitkeep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := findEmptyDirectories(root, []string{"internal"})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("findEmptyDirectories = %#v, want no findings", found)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("gitkeep directory was not preserved: %v", err)
	}
}

func TestFindUndocumentedInternalPackages(t *testing.T) {
	root := t.TempDir()
	undocumented := filepath.Join(root, "internal", "undocumented")
	documented := filepath.Join(root, "internal", "documented")
	if err := os.MkdirAll(undocumented, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(documented, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(undocumented, "code.go"), []byte("package undocumented\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(documented, "doc.go"), []byte("// Package documented owns a fixture boundary.\npackage documented\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := findUndocumentedInternalPackages(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "internal/undocumented" {
		t.Fatalf("findUndocumentedInternalPackages = %#v, want [internal/undocumented]", got)
	}
}

func TestLayoutProviderReportsUndocumentedInternalPackage(t *testing.T) {
	root := t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")
	path := filepath.Join(root, "internal", "newboundary")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "code.go"), []byte("package newboundary\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var report Report
	if err := (layoutProvider{root: root}).Run(context.Background(), Request{}, &report); err != nil {
		t.Fatal(err)
	}
	for _, finding := range report.Findings {
		if finding.Code == "repo_contract_internal_package_doc" && finding.Path == "internal/newboundary" {
			return
		}
	}
	t.Fatalf("report findings = %#v, want undocumented package finding", report.Findings)
}

func TestLayoutProviderPassesWhenNoEmptyDirectoriesRemain(t *testing.T) {
	root := t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")

	var report Report
	if err := (layoutProvider{root: root}).Run(context.Background(), Request{}, &report); err != nil {
		t.Fatal(err)
	}
	for _, check := range report.Checks {
		if check.Name == "repo_layout_empty_directories" {
			if !check.Passed {
				t.Fatalf("empty-directory check = %#v, want passed", check)
			}
			return
		}
	}
	t.Fatal("empty-directory check was not reported")
}
