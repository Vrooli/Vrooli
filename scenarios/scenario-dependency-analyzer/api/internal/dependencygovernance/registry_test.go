package dependencygovernance

import (
	"context"
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
	if got := resp.GetObservedDependencies()[0].GetSignalCategory(); got != "indirect" {
		t.Fatalf("signal category = %q, want indirect", got)
	}
}

func TestUsageGroupsExposeNormalizedSignalCategories(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"records": [
			{"ecosystem":"npm","package_name":"react","version_range":"*","state":"approved","rationale":"Approved React."}
		]
	}`)
	writeScenarioPackage(t, repoRoot, "demo", `{"dependencies":{"react":"^19.0.0"},"devDependencies":{"react":"^19.0.0"}}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateFleet()
	if err != nil {
		t.Fatal(err)
	}
	var react *governancev1.DependencyUsageGroup
	for _, group := range resp.GetUsageGroups() {
		if group.GetEcosystem() == "npm" && group.GetPackageName() == "react" {
			react = group
			break
		}
	}
	if react == nil {
		t.Fatalf("react usage group not found: %#v", resp.GetUsageGroups())
	}
	if got := strings.Join(react.GetSignalCategories(), ","); got != "direct_dev,direct_runtime" {
		t.Fatalf("signal categories = %q, want direct_dev,direct_runtime", got)
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

func TestValidateObservedTreatsApprovalsAsGlobalAcrossSurfacesAndGroups(t *testing.T) {
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
		t.Fatalf("global approval should pass advisory mode")
	}
	if got := len(resp.GetFindings()); got != 0 {
		t.Fatalf("findings = %d, want 0 because surface/group fields are ignored", got)
	}
	if resp.GetSummary().GetOutOfScope() != 0 || resp.GetSummary().GetOutOfRange() != 0 {
		t.Fatalf("summary = %#v, want no out-of-scope or out-of-range findings", resp.GetSummary())
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

func TestValidateObservedHonorsMajorLineRangePolicy(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "react",
				"version_range": "18.2.0",
				"range_policy": "major_line",
				"state": "approved",
				"rationale": "React 18 line is approved."
			}
		]
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "react", Version: "^18.3.1", FilePath: "scenarios/demo/ui/package.json", DependencyGroup: "dependencies"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetFindings()); got != 0 {
		t.Fatalf("findings = %d, want 0 for same major line", got)
	}

	resp, err = registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "react", Version: "^19.0.0", FilePath: "scenarios/demo/ui/package.json", DependencyGroup: "dependencies"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetFindings()); got != 1 || resp.GetFindings()[0].GetFindingClass() != "VERSION_OUT_OF_RANGE" {
		t.Fatalf("findings = %#v, want one out-of-range finding for next major", resp.GetFindings())
	}
}

func TestValidateObservedHonorsMinimumRangePolicyWithExplicitUpperBound(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "react",
				"version_range": ">=18.0.0 <20.0.0",
				"range_policy": "minimum",
				"state": "approved",
				"rationale": "React 18 and 19 are approved."
			}
		]
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "react", Version: "^19.1.0", FilePath: "scenarios/demo/ui/package.json", DependencyGroup: "dependencies"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetFindings()); got != 0 {
		t.Fatalf("findings = %d, want 0 within explicit minimum range", got)
	}
}

func TestValidateObservedWarnsForUnparseableVersionPolicy(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "vite",
				"version_range": "workspace:*",
				"range_policy": "minimum",
				"state": "approved",
				"rationale": "Local workspace declaration needs normalization."
			}
		]
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "vite", Version: "workspace:*", FilePath: "scenarios/demo/ui/package.json", DependencyGroup: "dependencies"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetFindings()); got != 0 {
		t.Fatalf("matching literal workspace range should pass, got %d findings", got)
	}

	resp, err = registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "vite", Version: "catalog:", FilePath: "scenarios/demo/ui/package.json", DependencyGroup: "dependencies"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetFindings()); got != 1 || resp.GetFindings()[0].GetFindingClass() != "VERSION_RANGE_UNPARSEABLE" {
		t.Fatalf("findings = %#v, want one unparseable range finding", resp.GetFindings())
	}
}

func TestValidateObservedHonorsSecurityDeniedRangePolicy(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "vite",
				"version_range": ">=5.0.0 <6.4.2",
				"range_policy": "security_denied",
				"state": "denied",
				"rationale": "Vulnerable Vite range.",
				"replacement": "Update to >=6.4.2."
			}
		]
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "vite", Version: "^6.4.3", FilePath: "scenarios/demo/ui/package.json", DependencyGroup: "dependencies"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetFindings()); got != 0 {
		t.Fatalf("findings = %d, want 0 outside security-denied range", got)
	}

	resp, err = registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "vite", Version: "6.4.1", FilePath: "scenarios/demo/ui/package.json", DependencyGroup: "dependencies"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetFindings()); got != 1 || resp.GetFindings()[0].GetFindingClass() != "DENIED_IN_USE" {
		t.Fatalf("findings = %#v, want one denied finding inside vulnerable range", resp.GetFindings())
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

func TestListFindingsFiltersFleetGovernanceFindings(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"records": []
	}`)
	writeScenarioPackage(t, repoRoot, "alpha", `{"dependencies":{"react":"^19.0.0"}}`)
	writeScenarioPackage(t, repoRoot, "beta", `{"dependencies":{"left-pad":"^1.3.0"}}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ListFindings(&governancev1.ListApprovedDependencyFindingsRequest{
		Scenario:    "alpha",
		Ecosystem:   "npm",
		Severity:    "WARNING",
		PackageName: "react",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetFindings()); got != 1 {
		t.Fatalf("findings = %d, want 1", got)
	}
	if resp.GetFindings()[0].GetScenario() != "alpha" || resp.GetFindings()[0].GetPackageName() != "react" {
		t.Fatalf("finding = %#v, want alpha react", resp.GetFindings()[0])
	}
	if resp.GetSummary().GetScenarioCount() != 1 || resp.GetSummary().GetDependencyCount() != 1 || resp.GetSummary().GetFindingCount() != 1 {
		t.Fatalf("summary = %#v, want one scenario/dependency/finding", resp.GetSummary())
	}
}

func TestGetUsageReturnsOneDependencyUsageAndFindings(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"records": []
	}`)
	writeScenarioPackage(t, repoRoot, "alpha", `{"dependencies":{"react":"^19.0.0"}}`)
	writeScenarioPackage(t, repoRoot, "beta", `{"devDependencies":{"react":"^19.0.0"}}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.GetUsage(&governancev1.GetApprovedDependencyUsageRequest{
		Ecosystem:   "npm",
		PackageName: "react",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetFound() {
		t.Fatalf("usage should be found")
	}
	if resp.GetUsageGroup().GetUsageCount() != 2 || resp.GetUsageGroup().GetScenarioCount() != 2 {
		t.Fatalf("usage group = %#v, want two usages across two scenarios", resp.GetUsageGroup())
	}
	if got := len(resp.GetFindings()); got != 2 {
		t.Fatalf("findings = %d, want 2 unrecorded findings for react", got)
	}
	if resp.GetSummary().GetObserved() != 2 || resp.GetSummary().GetUnrecorded() != 2 {
		t.Fatalf("summary = %#v, want observed/unrecorded counts for selected dependency", resp.GetSummary())
	}
}

func TestGetTriageGroupsFleetFindingsByActionSection(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "react",
				"version_range": "^18.0.0",
				"state": "approved",
				"rationale": "Approved React 18 runtime."
			},
			{
				"ecosystem": "npm",
				"package_name": "left-pad",
				"version_range": "*",
				"state": "denied",
				"rationale": "Use native padding.",
				"replacement": "String.prototype.padStart/padEnd"
			}
		]
	}`)
	writeScenarioPackage(t, repoRoot, "alpha", `{"dependencies":{"vite":"^5.0.0","react":"^19.0.0"}}`)
	writeScenarioPackage(t, repoRoot, "beta", `{"dependencies":{"vite":"^5.0.0","left-pad":"^1.3.0"}}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.GetTriage(&governancev1.GetApprovedDependencyTriageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetSummary().GetFindingCount() != 4 {
		t.Fatalf("finding count = %d, want 4", resp.GetSummary().GetFindingCount())
	}
	if got := len(resp.GetRegistrySeeding()); got != 1 {
		t.Fatalf("registry seeding groups = %d, want 1", got)
	}
	seeding := resp.GetRegistrySeeding()[0]
	if seeding.GetPackageName() != "vite" || seeding.GetFindingCount() != 2 || seeding.GetScenarioCount() != 2 {
		t.Fatalf("seeding group = %#v, want vite across two scenarios", seeding)
	}
	if seeding.GetActionType() != "approve_or_review" || !strings.Contains(seeding.GetRecommendedCommand(), "deps approved approve-observed npm/vite --from-findings") {
		t.Fatalf("seeding action/command = %q / %q", seeding.GetActionType(), seeding.GetRecommendedCommand())
	}
	if got := len(resp.GetRangePolicy()); got != 1 {
		t.Fatalf("range policy groups = %d, want 1", got)
	}
	if resp.GetRangePolicy()[0].GetPackageName() != "react" || resp.GetRangePolicy()[0].GetActionType() != "widen_range" || !strings.Contains(resp.GetRangePolicy()[0].GetRecommendedCommand(), "deps approved widen-range npm/react --to-major-line") {
		t.Fatalf("range group = %#v, want react widen_range", resp.GetRangePolicy()[0])
	}
	if got := len(resp.GetSecurityActions()); got != 1 {
		t.Fatalf("security action groups = %d, want 1", got)
	}
	if resp.GetSecurityActions()[0].GetPackageName() != "left-pad" || resp.GetSecurityActions()[0].GetHighestSeverity() != "ERROR" {
		t.Fatalf("security group = %#v, want left-pad error", resp.GetSecurityActions()[0])
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
		AllowedSurfaces:         []string{"ui"},
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
	if len(record.GetAllowedSurfaces()) != 0 || len(record.GetAllowedDependencyGroups()) != 0 {
		t.Fatalf("ignored scope fields should not be persisted: %#v", record)
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

func TestPreviewVulnerabilityRemediationBuildsSecurityDerivedDeniedRecord(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{"schema_version":"1","policy":{"mode":"advisory"},"records":[]}`)
	installFakeSecurityHealth(t, `{
		"found": true,
		"vulnerability": {
			"vulnerability_id": "GHSA-demo",
			"aliases": ["CVE-2026-0001"],
			"ecosystem": "ECOSYSTEM_NPM",
			"name": "vite",
			"version": "5.0.0",
			"affected_ranges": [{"range": "<5.1.0", "fixed": "5.1.0"}],
			"fixed_ranges": [{"range": ">=5.1.0", "version": "5.1.0"}],
			"normalized_severity": "high",
			"advisory_url": "https://osv.dev/vulnerability/GHSA-demo",
			"summary": "Demo vulnerability",
			"source": "VULNERABILITY_SOURCE_OSV",
			"reachability": "REACHABILITY_LOCKFILE_AFFECTED",
			"confidence": "EVIDENCE_CONFIDENCE_ADVISORY",
			"scenarios": ["demo"],
			"source_files": ["ui/pnpm-lock.yaml"],
			"remediation": "Upgrade vite to >=5.1.0."
		}
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.PreviewVulnerabilityRemediation(context.Background(), &governancev1.PreviewVulnerabilityRemediationRequest{
		Ecosystem:       "npm",
		PackageName:     "vite",
		VulnerabilityId: "GHSA-demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetFound() {
		t.Fatalf("found = false, want true")
	}
	record := resp.GetSuggestedRecord()
	if record.GetState() != "denied" || record.GetVersionRange() != "<5.1.0" || record.GetRangePolicy() != "security_denied" {
		t.Fatalf("suggested record = %#v, want denied <5.1.0 with security_denied policy", record)
	}
	if !strings.Contains(record.GetSecurityNotes(), "vulnerability=GHSA-demo") || !strings.Contains(record.GetSecurityNotes(), "confidence=advisory") {
		t.Fatalf("security notes missing evidence: %q", record.GetSecurityNotes())
	}
	if got := strings.Join(resp.GetAffectedScenarios(), ","); got != "demo" {
		t.Fatalf("affected scenarios = %q, want demo", got)
	}
	_, found, err := registry.Explain("npm", "vite")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatalf("preview should not write registry")
	}
}

func TestPreviewVulnerabilityRemediationUsesMatchingAffectedInterval(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{"schema_version":"1","policy":{"mode":"advisory"},"records":[]}`)
	installFakeSecurityHealth(t, `{
		"found": true,
		"vulnerability": {
			"vulnerability_id": "GHSA-demo",
			"ecosystem": "ECOSYSTEM_NPM",
			"name": "vite",
			"version": "4.5.14",
			"affected_ranges": [
				{"introduced": "8.0.0"},
				{"fixed": "8.0.5"},
				{"introduced": "7.0.0"},
				{"fixed": "7.3.2"},
				{"introduced": "4.0.0"},
				{"fixed": "4.5.15"}
			],
			"fixed_ranges": [{"range": ">=4.5.15", "version": "4.5.15"}],
			"confidence": "EVIDENCE_CONFIDENCE_ADVISORY"
		}
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.PreviewVulnerabilityRemediation(context.Background(), &governancev1.PreviewVulnerabilityRemediationRequest{
		Ecosystem:       "npm",
		PackageName:     "vite",
		VulnerabilityId: "GHSA-demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resp.GetSuggestedRecord().GetVersionRange(), ">=4.0.0 <4.5.15"; got != want {
		t.Fatalf("suggested range = %q, want %q", got, want)
	}
	if !strings.Contains(resp.GetSuggestedRecord().GetReplacement(), ">= 4.5.15") || !strings.Contains(resp.GetRemediation(), ">= 4.5.15") {
		t.Fatalf("remediation should recommend fixed version for selected interval, replacement=%q remediation=%q", resp.GetSuggestedRecord().GetReplacement(), resp.GetRemediation())
	}
}

func TestPreviewVulnerabilityRemediationFallsBackToObservedVersionForAmbiguousRanges(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{"schema_version":"1","policy":{"mode":"advisory"},"records":[]}`)
	installFakeSecurityHealth(t, `{
		"found": true,
		"vulnerability": {
			"vulnerability_id": "GHSA-demo",
			"ecosystem": "ECOSYSTEM_NPM",
			"name": "vite",
			"version": "4.5.14",
			"affected_ranges": [
				{"range": ">=8.0.0"},
				{"range": "<8.0.5"},
				{"range": ">=7.0.0"},
				{"range": "<7.3.2"},
				{"range": "<4.5.15"}
			],
			"fixed_ranges": [{"range": ">=4.5.15", "version": "4.5.15"}],
			"confidence": "EVIDENCE_CONFIDENCE_ADVISORY"
		}
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.PreviewVulnerabilityRemediation(context.Background(), &governancev1.PreviewVulnerabilityRemediationRequest{
		Ecosystem:       "npm",
		PackageName:     "vite",
		VulnerabilityId: "GHSA-demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resp.GetSuggestedRecord().GetVersionRange(), "4.5.14"; got != want {
		t.Fatalf("suggested range = %q, want exact observed version %q", got, want)
	}
}

func TestDenyVulnerableDependencyDryRunAndApply(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"policy": {"mode": "advisory"},
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "vite",
				"version_range": ">=5.0.0 <6.0.0",
				"state": "approved",
				"rationale": "Previously approved Vite range."
			}
		]
	}`)
	installFakeSecurityHealth(t, `{
		"found": true,
		"vulnerability": {
			"vulnerabilityId": "GHSA-demo",
			"ecosystem": "ECOSYSTEM_NPM",
			"name": "vite",
			"version": "5.0.0",
			"affectedRanges": [{"range": "<5.1.0"}],
			"fixedRanges": [{"range": ">=5.1.0"}],
			"source": "VULNERABILITY_SOURCE_PNPM_AUDIT",
			"reachability": "REACHABILITY_UNKNOWN",
			"confidence": "EVIDENCE_CONFIDENCE_GATING",
			"scenarios": ["demo"]
		}
	}`)
	registry := NewRegistry(repoRoot)

	dryRun, err := registry.DenyVulnerableDependency(context.Background(), &governancev1.DenyVulnerableDependencyRequest{
		Ecosystem:       "npm",
		PackageName:     "vite",
		VulnerabilityId: "GHSA-demo",
		DryRun:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.GetMutation() == nil || !dryRun.GetMutation().GetDryRun() || dryRun.GetMutation().GetPreviousRecord() == nil {
		t.Fatalf("dry-run mutation = %#v, want dry-run with previous record", dryRun.GetMutation())
	}
	record, found, err := registry.Explain("npm", "vite")
	if err != nil {
		t.Fatal(err)
	}
	if !found || record.GetState() != "approved" {
		t.Fatalf("dry-run changed registry record: %#v found=%t", record, found)
	}

	applied, err := registry.DenyVulnerableDependency(context.Background(), &governancev1.DenyVulnerableDependencyRequest{
		Ecosystem:       "npm",
		PackageName:     "vite",
		VulnerabilityId: "GHSA-demo",
		DryRun:          false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied.GetMutation() == nil || applied.GetMutation().GetDryRun() || !applied.GetMutation().GetChanged() {
		t.Fatalf("apply mutation = %#v, want applied changed mutation", applied.GetMutation())
	}
	validation, err := registry.ValidateObserved("demo", []*governancev1.ObservedDependency{
		{Ecosystem: "npm", PackageName: "vite", Version: "5.0.0", FilePath: "scenarios/demo/ui/package.json", DependencyGroup: "dependencies"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if validation.GetPassed() || validation.GetFindings()[0].GetFindingClass() != "DENIED_IN_USE" {
		t.Fatalf("validation = passed:%t findings:%#v, want denied failure", validation.GetPassed(), validation.GetFindings())
	}
}

func TestListSecurityGovernanceGapsReportsCoverageAndApprovedOverlap(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"policy": {"mode": "advisory"},
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "vite",
				"version_range": ">=5.0.0 <5.1.0",
				"range_policy": "security_denied",
				"state": "denied",
				"rationale": "Vite affected range is denied.",
				"replacement": "Upgrade Vite."
			},
			{
				"ecosystem": "npm",
				"package_name": "react",
				"version_range": "^18.0.0",
				"range_policy": "major_line",
				"state": "approved",
				"rationale": "React 18 line approved."
			}
		]
	}`)
	installFakeSecurityHealth(t, `{
		"total": 2,
		"vulnerabilities": [
			{
				"vulnerability_id": "GHSA-vite",
				"ecosystem": "ECOSYSTEM_NPM",
				"name": "vite",
				"version": "5.0.0",
				"affected_ranges": [{"range": ">=5.0.0 <5.1.0"}],
				"fixed_ranges": [{"range": ">=5.1.0"}],
				"normalized_severity": "high",
				"confidence": "EVIDENCE_CONFIDENCE_ADVISORY",
				"reachability": "REACHABILITY_LOCKFILE_AFFECTED",
				"scenarios": ["demo"],
				"source_files": ["ui/pnpm-lock.yaml"]
			},
			{
				"vulnerability_id": "GHSA-react",
				"ecosystem": "ECOSYSTEM_NPM",
				"name": "react",
				"version": "18.2.0",
				"affected_ranges": [{"range": "<18.3.0"}],
				"fixed_ranges": [{"range": ">=18.3.0"}],
				"normalized_severity": "moderate",
				"confidence": "EVIDENCE_CONFIDENCE_ADVISORY",
				"production": true,
				"scenarios": ["demo"],
				"source_files": ["ui/package.json"]
			}
		]
	}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ListSecurityGovernanceGaps(context.Background(), &governancev1.ListSecurityGovernanceGapsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetTotal() != 2 || resp.GetDeniedCoveredCount() != 1 || resp.GetUncoveredCount() != 1 || resp.GetApprovedOverlapCount() != 1 {
		t.Fatalf("summary total=%d denied=%d uncovered=%d overlap=%d, want 2/1/1/1", resp.GetTotal(), resp.GetDeniedCoveredCount(), resp.GetUncoveredCount(), resp.GetApprovedOverlapCount())
	}
	var react, vite *governancev1.SecurityGovernanceGap
	for _, gap := range resp.GetGaps() {
		switch gap.GetPackageName() {
		case "react":
			react = gap
		case "vite":
			vite = gap
		}
	}
	if vite == nil || !vite.GetDeniedRecordCovers() || vite.GetApprovedRecordOverlaps() {
		t.Fatalf("vite gap = %#v, want denied-covered without approved overlap", vite)
	}
	if react == nil || react.GetDeniedRecordCovers() || !react.GetApprovedRecordOverlaps() || react.GetSignalCategory() != "security_vulnerable" {
		t.Fatalf("react gap = %#v, want uncovered approved-overlap security_vulnerable", react)
	}
	if !strings.Contains(react.GetSuggestedCommand(), "deny-vulnerable npm/react --vulnerability GHSA-react") {
		t.Fatalf("suggested command = %q", react.GetSuggestedCommand())
	}
}

func TestProposeRecordsBuildsDraftsFromTopUnrecordedDirectDependencies(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{"schema_version":"1","policy":{"mode":"advisory"},"records":[]}`)
	writeScenarioPackage(t, repoRoot, "alpha", `{"dependencies":{"react":"^19.0.0","vite":"^6.0.0"},"devDependencies":{"vitest":"^3.0.0"}}`)
	writeScenarioPackage(t, repoRoot, "beta", `{"dependencies":{"react":"^19.0.0"}}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ProposeRecords(&governancev1.ProposeApprovedDependencyRecordsRequest{
		TopUnrecorded:        2,
		MinimumScenarioCount: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(resp.GetRecords()); got != 1 {
		t.Fatalf("records = %d, want only dependency used by two scenarios", got)
	}
	record := resp.GetRecords()[0]
	if record.GetEcosystem() != "npm" || record.GetPackageName() != "react" || record.GetVersionRange() != "^19.0.0" {
		t.Fatalf("record = %#v, want npm/react ^19.0.0", record)
	}
	if record.GetState() != "needs_review" || !strings.Contains(record.GetRationale(), "Reviewer must confirm") {
		t.Fatalf("record review metadata = %#v", record)
	}
	if got := strings.Join(record.GetExampleScenarios(), ","); got != "alpha,beta" {
		t.Fatalf("example scenarios = %q, want alpha,beta", got)
	}
}

func TestApproveObservedBuildsApprovedRecordFromFleetUsage(t *testing.T) {
	repoRoot := t.TempDir()
	original := `{"schema_version":"1","policy":{"mode":"advisory"},"records":[]}`
	writeRegistry(t, repoRoot, original)
	writeScenarioPackage(t, repoRoot, "alpha", `{"dependencies":{"react":"^19.0.0"}}`)
	writeScenarioPackage(t, repoRoot, "beta", `{"dependencies":{"react":"^19.0.0"}}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.ApproveObserved(&governancev1.ApproveObservedDependencyRequest{
		Ecosystem:    "npm",
		PackageName:  "react",
		RangePolicy:  "major_line",
		ApprovedBy:   "dependency-review",
		DryRun:       true,
		FromFindings: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	record := resp.GetRecord()
	if record.GetState() != "approved" || record.GetVersionRange() != "^19.0.0" || record.GetRangePolicy() != "major_line" {
		t.Fatalf("record = %#v, want approved ^19.0.0 major_line", record)
	}
	if resp.GetMutation() == nil || !resp.GetMutation().GetDryRun() || !resp.GetMutation().GetChanged() {
		t.Fatalf("mutation = %#v, want changed dry-run", resp.GetMutation())
	}
	if resp.GetEvidenceGroup().GetScenarioCount() != 2 || resp.GetEvidenceGroup().GetUsageCount() != 2 {
		t.Fatalf("evidence = %#v, want two scenarios/usages", resp.GetEvidenceGroup())
	}
	if got := readRegistry(t, repoRoot); got != original {
		t.Fatalf("dry-run wrote registry:\n%s", got)
	}
}

func TestWidenRangeUsesObservedSingleMajorLine(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"policy": {"mode": "advisory"},
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "react",
				"version_range": "18.2.0",
				"range_policy": "exact",
				"state": "approved",
				"rationale": "Approved React runtime."
			}
		]
	}`)
	writeScenarioPackage(t, repoRoot, "alpha", `{"dependencies":{"react":"^18.3.1"}}`)
	writeScenarioPackage(t, repoRoot, "beta", `{"dependencies":{"react":"^18.2.0"}}`)
	registry := NewRegistry(repoRoot)

	resp, err := registry.WidenRange(&governancev1.WidenApprovedDependencyRangeRequest{
		Ecosystem:    "npm",
		PackageName:  "react",
		TargetPolicy: "major_line",
		DryRun:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	record := resp.GetRecord()
	if record.GetVersionRange() != "^18.2.0" || record.GetRangePolicy() != "major_line" || record.GetState() != "approved" {
		t.Fatalf("record = %#v, want lowest observed major-line approval", record)
	}
	if resp.GetMutation() == nil || !resp.GetMutation().GetDryRun() || !resp.GetMutation().GetChanged() {
		t.Fatalf("mutation = %#v, want changed dry-run", resp.GetMutation())
	}
}

func TestWidenRangeRefusesMultipleObservedMajors(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"schema_version": "1",
		"policy": {"mode": "advisory"},
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "react",
				"version_range": "18.2.0",
				"state": "approved",
				"rationale": "Approved React runtime."
			}
		]
	}`)
	writeScenarioPackage(t, repoRoot, "alpha", `{"dependencies":{"react":"^18.3.1"}}`)
	writeScenarioPackage(t, repoRoot, "beta", `{"dependencies":{"react":"^19.0.0"}}`)
	registry := NewRegistry(repoRoot)

	_, err := registry.WidenRange(&governancev1.WidenApprovedDependencyRangeRequest{
		Ecosystem:    "npm",
		PackageName:  "react",
		TargetPolicy: "major_line",
		DryRun:       true,
	})
	if err == nil || !strings.Contains(err.Error(), "multiple majors") {
		t.Fatalf("err = %v, want multiple-majors refusal", err)
	}
}

func TestBatchUpsertValidatesAtomicallyBeforeWriting(t *testing.T) {
	repoRoot := t.TempDir()
	original := `{
		"schema_version": "1",
		"policy": {"mode": "advisory"},
		"records": [
			{
				"ecosystem": "npm",
				"package_name": "react",
				"version_range": "^18.0.0",
				"state": "approved",
				"rationale": "Existing approval."
			}
		]
	}`
	writeRegistry(t, repoRoot, original)
	registry := NewRegistry(repoRoot)

	_, err := registry.BatchUpsert([]*governancev1.ApprovedDependencyRecord{
		{
			Ecosystem:    "npm",
			PackageName:  "vite",
			VersionRange: "^6.0.0",
			State:        "needs_review",
			Rationale:    "Draft Vite review.",
		},
		{
			Ecosystem:    "npm",
			PackageName:  "broken",
			VersionRange: "*",
			State:        "approved",
		},
	}, false)
	if err == nil {
		t.Fatalf("BatchUpsert succeeded, want invalid record error")
	}
	data, readErr := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "dependencies", "approved-dependencies.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != original {
		t.Fatalf("invalid batch partially changed registry:\n%s", string(data))
	}
}

func TestBatchUpsertDryRunAndApply(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{"schema_version":"1","policy":{"mode":"advisory"},"records":[]}`)
	registry := NewRegistry(repoRoot)
	records := []*governancev1.ApprovedDependencyRecord{
		{
			Ecosystem:    "npm",
			PackageName:  "vite",
			VersionRange: "^6.0.0",
			State:        "needs_review",
			Rationale:    "Draft Vite review.",
		},
		{
			Ecosystem:    "go",
			PackageName:  "connectrpc.com/connect",
			VersionRange: "v1.19.2",
			State:        "approved",
			Rationale:    "Standard Connect runtime.",
		},
	}

	dryRun, err := registry.BatchUpsert(records, true)
	if err != nil {
		t.Fatal(err)
	}
	if !dryRun.GetDryRun() || !dryRun.GetChanged() || len(dryRun.GetMutations()) != 2 {
		t.Fatalf("dry-run response = %#v, want dry-run changed with two mutations", dryRun)
	}
	if _, found, err := registry.Explain("npm", "vite"); err != nil || found {
		t.Fatalf("dry-run explain found=%t err=%v, want no write", found, err)
	}

	applied, err := registry.BatchUpsert(records, false)
	if err != nil {
		t.Fatal(err)
	}
	if applied.GetDryRun() || !applied.GetChanged() || applied.GetSummary().GetNeedsReview() != 1 || applied.GetSummary().GetApproved() != 1 {
		t.Fatalf("apply response = %#v, want applied summary", applied)
	}
	if _, found, err := registry.Explain("npm", "vite"); err != nil || !found {
		t.Fatalf("apply explain found=%t err=%v, want written record", found, err)
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

func readRegistry(t *testing.T, repoRoot string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "dependencies", "approved-dependencies.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func installFakeSecurityHealth(t *testing.T, response string) {
	t.Helper()
	binDir := t.TempDir()
	path := filepath.Join(binDir, "security-health")
	script := "#!/usr/bin/env sh\ncat <<'JSON'\n" + response + "\nJSON\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
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
