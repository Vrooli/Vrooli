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
	if resp.GetSummary().GetStatus() != "warn" {
		t.Fatalf("status = %q, want warn", resp.GetSummary().GetStatus())
	}
	if len(resp.GetFindings()) != 1 || resp.GetFindings()[0].GetSeverity() != "WARNING" {
		t.Fatalf("findings = %#v, want one warning", resp.GetFindings())
	}
	if !strings.Contains(resp.GetGuidance(), "not an exhaustive allowlist") {
		t.Fatalf("response guidance missing non-allowlist wording: %s", resp.GetGuidance())
	}
}

func TestValidateObservedDoesNotWarnForUnrecordedIndirectGoDependency(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{"records":[]}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{
			Ecosystem:       "go",
			PackageName:     "golang.org/x/sys",
			Version:         "v0.29.0",
			FilePath:        "scenarios/demo/api/go.mod",
			DependencyGroup: "require_indirect",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetPassed() {
		t.Fatalf("unrecorded indirect Go dependency should not fail")
	}
	if got := len(resp.GetFindings()); got != 0 {
		t.Fatalf("findings = %d, want 0 for unrecorded indirect Go dependency", got)
	}
	if got := resp.GetSummary().GetObserved(); got != 1 {
		t.Fatalf("observed = %d, want indirect dependency retained in observed output", got)
	}
	if got := resp.GetSummary().GetUnrecorded(); got != 0 {
		t.Fatalf("unrecorded = %d, want 0 for suppressed indirect dependency", got)
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
				"rationale": "Unmaintained package with safer native replacement.",
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

func TestValidateObservedFailsForUnrecordedDependencyInStrictMode(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"policy": {"mode": "strict"},
		"records": []
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "react", Version: "^19.0.0", FilePath: "scenarios/demo/ui/package.json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetPassed() {
		t.Fatalf("unrecorded direct dependency should fail in strict mode")
	}
	if resp.GetSummary().GetPolicyMode() != "strict" {
		t.Fatalf("policy mode = %q, want strict", resp.GetSummary().GetPolicyMode())
	}
	if len(resp.GetFindings()) != 1 || resp.GetFindings()[0].GetFindingClass() != "UNRECORDED_DIRECT" || resp.GetFindings()[0].GetSeverity() != "ERROR" {
		t.Fatalf("findings = %#v, want strict unrecorded error", resp.GetFindings())
	}
}

func TestValidateObservedEnforcesVersionRangeAndScope(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"policy": {"mode": "advisory"},
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "react",
				"version_range": ">=18.0.0 <20.0.0",
				"state": "approved_with_constraints",
				"allowed_surfaces": ["ui"],
				"allowed_dependency_groups": ["dependencies"],
				"rationale": "Approved React range for UI runtime dependencies."
			}
		]
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{
			Ecosystem:       "npm",
			PackageName:     "react",
			Version:         "^19.0.0",
			SurfaceId:       "api",
			FilePath:        "scenarios/demo/api/package.json",
			DependencyGroup: "devDependencies",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetPassed() {
		t.Fatalf("scope warning should not fail advisory mode")
	}
	if got := len(resp.GetFindings()); got != 1 {
		t.Fatalf("findings = %d, want 1 scope finding because version range allows ^19.0.0", got)
	}
	if resp.GetFindings()[0].GetFindingClass() != "SCOPE_VIOLATION" {
		t.Fatalf("finding class = %q, want SCOPE_VIOLATION", resp.GetFindings()[0].GetFindingClass())
	}
	if resp.GetSummary().GetOutOfScope() != 1 || resp.GetSummary().GetOutOfRange() != 0 {
		t.Fatalf("summary = %#v, want one out-of-scope and no out-of-range", resp.GetSummary())
	}
}

func TestValidateObservedWarnsForOutOfRangeVersion(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"records": [
			{
				"ecosystem": "go",
				"package_name": "example.com/demo",
				"version_range": ">=v1.2.0 <v2.0.0",
				"state": "approved",
				"rationale": "Approved stable v1 range."
			}
		]
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "go", PackageName: "example.com/demo", Version: "v2.1.0", FilePath: "scenarios/demo/api/go.mod", DependencyGroup: "require"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetFindings()); got != 1 {
		t.Fatalf("findings = %d, want one out-of-range finding", got)
	}
	if resp.GetFindings()[0].GetFindingClass() != "VERSION_OUT_OF_RANGE" {
		t.Fatalf("finding class = %q, want VERSION_OUT_OF_RANGE", resp.GetFindings()[0].GetFindingClass())
	}
}

func TestValidateObservedFailsForBlockedIndirectGoDependency(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"records": [
			{
				"ecosystem": "go",
				"package_name": "example.com/blocked",
				"version_range": "*",
				"state": "blocked",
				"rationale": "Blocked transitive module must not appear in the graph.",
				"replacement": "Use a maintained module."
			}
		]
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{
			Ecosystem:       "go",
			PackageName:     "example.com/blocked",
			Version:         "v0.1.0",
			FilePath:        "scenarios/demo/api/go.mod",
			DependencyGroup: "require_indirect",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetPassed() {
		t.Fatalf("blocked indirect Go dependency should fail")
	}
	if len(resp.GetFindings()) != 1 || resp.GetFindings()[0].GetSeverity() != "ERROR" {
		t.Fatalf("findings = %#v, want one error", resp.GetFindings())
	}
}

func TestValidateFleetAggregatesScenariosAndUsage(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "react",
				"version_range": "^19.0.0",
				"state": "approved",
				"rationale": "Approved UI runtime."
			}
		]
	}`)
	writeScenarioPackage(t, repoRoot, "alpha", `{"dependencies":{"react":"^19.0.0"}}`)
	writeScenarioPackage(t, repoRoot, "beta", `{"dependencies":{"left-pad":"^1.3.0"}}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateFleet()
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetPassed() {
		t.Fatalf("fleet should warn but pass in advisory mode")
	}
	if resp.GetSummary().GetScenarioCount() != 2 {
		t.Fatalf("scenario count = %d, want 2", resp.GetSummary().GetScenarioCount())
	}
	if resp.GetSummary().GetDependencyCount() != 2 {
		t.Fatalf("dependency count = %d, want 2", resp.GetSummary().GetDependencyCount())
	}
	if resp.GetSummary().GetUnrecorded() != 1 {
		t.Fatalf("unrecorded = %d, want 1", resp.GetSummary().GetUnrecorded())
	}
	if len(resp.GetUsageGroups()) != 2 {
		t.Fatalf("usage groups = %d, want 2", len(resp.GetUsageGroups()))
	}
}

func TestScanGoModMarksIndirectRequirements(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte(`module example.com/demo

go 1.24

require (
	github.com/direct/module v1.2.3
	golang.org/x/sys v0.29.0 // indirect
)
`), 0o644); err != nil {
		t.Fatal(err)
	}

	deps, err := scanGoMod(path)
	if err != nil {
		t.Fatal(err)
	}
	groups := map[string]string{}
	for _, dep := range deps {
		groups[dep.GetPackageName()] = dep.GetDependencyGroup()
	}
	if groups["github.com/direct/module"] != "require" {
		t.Fatalf("direct group = %q, want require", groups["github.com/direct/module"])
	}
	if groups["golang.org/x/sys"] != "require_indirect" {
		t.Fatalf("indirect group = %q, want require_indirect", groups["golang.org/x/sys"])
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

func TestUpsertDryRunDoesNotWriteRegistry(t *testing.T) {
	repoRoot := t.TempDir()
	original := `{
		"schema_version": "1",
		"policy": {"mode": "advisory"},
		"records": []
	}`
	writeRegistry(t, repoRoot, original)
	registry := NewRegistry(repoRoot)

	resp, err := registry.Upsert(&governancev1.ApprovedDependencyRecord{
		Ecosystem:    "npm",
		PackageName:  "left-pad",
		VersionRange: "^1.3.0",
		State:        "denied",
		Rationale:    "Use native string padding instead.",
		Replacement:  "Native String.prototype.padStart/padEnd",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetDryRun() || !resp.GetChanged() {
		t.Fatalf("dry-run response = dry_run:%t changed:%t, want true/true", resp.GetDryRun(), resp.GetChanged())
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "dependencies", "approved-dependencies.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("dry run changed registry:\n%s", string(data))
	}
}

func TestUpsertWritesNormalizedRecordAndValidationUsesIt(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{"schema_version":"1","policy":{"mode":"advisory"},"records":[]}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.Upsert(&governancev1.ApprovedDependencyRecord{
		Ecosystem:               "NPM",
		PackageName:             "React",
		VersionRange:            ">=18.0.0 <20.0.0",
		State:                   "approved_with_constraints",
		AllowedDependencyGroups: []string{"dependencies"},
		Rationale:               "Approved UI runtime framework range.",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetDryRun() || !resp.GetChanged() {
		t.Fatalf("apply response = dry_run:%t changed:%t, want false/true", resp.GetDryRun(), resp.GetChanged())
	}
	record, found, err := registry.Explain("npm", "react")
	if err != nil {
		t.Fatal(err)
	}
	if !found || record.GetEcosystem() != "npm" || record.GetPackageName() != "React" {
		t.Fatalf("record = %#v found=%t, want normalized npm/React", record, found)
	}
	validation, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{
			Ecosystem:       "npm",
			PackageName:     "react",
			Version:         "^19.0.0",
			FilePath:        "scenarios/demo/ui/package.json",
			DependencyGroup: "dependencies",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(validation.GetFindings()); got != 0 {
		t.Fatalf("findings = %d, want 0 after approved upsert", got)
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

func writeScenarioPackage(t *testing.T, repoRoot, scenario, packageJSON string) {
	t.Helper()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", scenario)
	if err := os.MkdirAll(filepath.Join(scenarioRoot, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioRoot, ".vrooli", "service.json"), []byte(`{"name":"`+scenario+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	uiRoot := filepath.Join(scenarioRoot, "ui")
	if err := os.MkdirAll(uiRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiRoot, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}
}
