package dependencygovernance

import (
	"os"
	"path/filepath"
	"testing"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

func TestScanGoModMarksFilesystemReplacementsAsLocal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go.mod")
	content := `module demo

go 1.25.0

require (
	github.com/vrooli/local-module v0.0.0
	github.com/third-party/module v1.2.3
)

replace (
	github.com/vrooli/local-module => ../local-module
	github.com/third-party/module v1.2.3 => v1.2.4
)
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	deps, err := scanGoMod(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 2 {
		t.Fatalf("dependencies = %d, want 2", len(deps))
	}
	for _, dep := range deps {
		switch dep.GetPackageName() {
		case "github.com/vrooli/local-module":
			if got := dep.GetSignalCategory(); got != "local_replace" {
				t.Fatalf("local signal category = %q, want local_replace", got)
			}
		case "github.com/third-party/module":
			if got := dep.GetSignalCategory(); got != "" {
				t.Fatalf("version replacement signal category = %q, want empty", got)
			}
		default:
			t.Fatalf("unexpected dependency %q", dep.GetPackageName())
		}
	}

	var localDependency *governancev1.ObservedDependency
	for _, dep := range deps {
		if dep.GetPackageName() == "github.com/vrooli/local-module" {
			localDependency = dep
		}
	}
	if localDependency == nil {
		t.Fatal("local replacement was not observed")
	}

	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{"records":[]}`)
	response, err := NewRegistry(repoRoot).ValidateObserved("demo", []*governancev1.ObservedDependency{localDependency, {
		Ecosystem: "npm", PackageName: "third-party", Version: "1.0.0", FilePath: "package.json",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.GetFindings()) != 1 || response.GetFindings()[0].GetPackageName() != "third-party" {
		t.Fatalf("governance findings = %#v, want only the unrecorded external package", response.GetFindings())
	}
}
