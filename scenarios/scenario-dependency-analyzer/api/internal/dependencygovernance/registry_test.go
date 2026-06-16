package dependencygovernance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

func TestGuidanceStatesApprovedDependenciesAreNotAllowlist(t *testing.T) {
	for _, phrase := range []string{
		"not an exhaustive allowlist",
		"suggest it with purpose",
		"security/license notes",
	} {
		if !strings.Contains(Guidance, phrase) {
			t.Fatalf("guidance missing %q: %s", phrase, Guidance)
		}
	}
}

func TestValidateObservedWarnsForUnrecordedDependencyWithoutFailing(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{"records":[]}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "react", Version: "^19.0.0", FilePath: "scenarios/demo/ui/package.json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetPassed() {
		t.Fatalf("unrecorded dependency should warn, not fail")
	}
	if resp.GetSummary().GetStatus() != "not_configured" {
		t.Fatalf("status = %q, want not_configured", resp.GetSummary().GetStatus())
	}
	if len(resp.GetFindings()) != 1 || resp.GetFindings()[0].GetSeverity() != "WARNING" {
		t.Fatalf("findings = %#v, want one warning", resp.GetFindings())
	}
	if !strings.Contains(resp.GetGuidance(), "not an exhaustive allowlist") {
		t.Fatalf("response guidance missing non-allowlist wording: %s", resp.GetGuidance())
	}
}

func TestValidateObservedFailsForBlockedDependency(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "left-pad",
				"version_range": "*",
				"state": "blocked",
				"replacement": "Use native string padding."
			}
		]
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "left-pad", Version: "^1.3.0", FilePath: "scenarios/demo/ui/package.json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetPassed() {
		t.Fatalf("blocked dependency should fail")
	}
	if resp.GetSummary().GetStatus() != "fail" {
		t.Fatalf("status = %q, want fail", resp.GetSummary().GetStatus())
	}
	if len(resp.GetFindings()) != 1 || resp.GetFindings()[0].GetSeverity() != "ERROR" {
		t.Fatalf("findings = %#v, want one error", resp.GetFindings())
	}
}

func TestSearchFindsRecordsByUseCaseAndPackage(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "reactflow",
				"version_range": "^11.0.0",
				"state": "approved",
				"use_cases": ["React graph library"],
				"rationale": "Maintained graph UI package."
			}
		]
	}`)
	registry := NewRegistry(repoRoot)

	records, _, err := registry.Search(&governancev1.SearchApprovedDependenciesRequest{Query: "React graph library"})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].GetPackageName() != "reactflow" {
		t.Fatalf("records = %#v, want reactflow", records)
	}
}

func writeRegistry(t *testing.T, repoRoot, content string) {
	t.Helper()
	path := filepath.Join(repoRoot, ".vrooli", "dependencies", "approved-dependencies.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
