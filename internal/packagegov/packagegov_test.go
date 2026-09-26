package packagegov

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackageKindsAndRefreshVocabulary(t *testing.T) {
	if KindGoRuntime == "" || KindJSRuntime == "" {
		t.Fatal("package kind vocabulary must remain available")
	}
	if RefreshScenarioSetup == "" || RefreshNone == "" {
		t.Fatal("refresh lifecycle vocabulary must remain available")
	}
}

func TestValidationIssueRemainsRegistryDiagnosticShape(t *testing.T) {
	issue := ValidationIssue{Severity: "warning", Code: "registry-warning", Message: "example", Path: "packages/example"}
	if issue.Code == "" || issue.Message == "" {
		t.Fatal("registry diagnostics must retain code and message")
	}
}

func TestValidateConsumerClassBoundaryRejectsUndeclaredClass(t *testing.T) {
	pkg := Package{
		Name: "agentharness",
		Manifest: Manifest{Package: ManifestEntry{Adoption: AdoptionPolicy{
			AllowedConsumers: []ConsumerClass{ConsumerResourceRuntime, ConsumerInternalPlatform},
		}}},
	}
	dependents := []Dependent{{
		PackageName:    pkg.Name,
		ConsumerName:   "example-scenario",
		ConsumerClass:  ConsumerScenarioAPI,
		DependencyFile: "scenarios/example/api/go.mod",
	}}

	violations := ValidateConsumerClassBoundary(pkg, dependents)
	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1", len(violations))
	}
	issue := violations[0].ValidationIssue()
	if issue.Code != "PACKAGE_CONSUMER_CLASS_VIOLATION" || issue.Severity != "error" {
		t.Fatalf("unexpected issue: %#v", issue)
	}
}

func TestValidateConsumerClassBoundaryAllowsDeclaredClass(t *testing.T) {
	pkg := Package{
		Name: "agentharness",
		Manifest: Manifest{Package: ManifestEntry{Adoption: AdoptionPolicy{
			AllowedConsumers: []ConsumerClass{ConsumerResourceRuntime, ConsumerInternalPlatform},
		}}},
	}
	dependents := []Dependent{{
		PackageName:    pkg.Name,
		ConsumerName:   "resource-runtime",
		ConsumerClass:  ConsumerResourceRuntime,
		DependencyFile: "resources/example/runtime/go.mod",
	}}

	if violations := ValidateConsumerClassBoundary(pkg, dependents); len(violations) != 0 {
		t.Fatalf("got unexpected violations: %#v", violations)
	}
}

func TestPlanRefreshForArtifactNarrowsExactExportImpact(t *testing.T) {
	root := t.TempDir()
	demoPath := filepath.Join(root, "scenarios", "demo", "ui")
	otherPath := filepath.Join(root, "scenarios", "other", "ui")
	for _, dir := range []string{filepath.Join(demoPath, "src"), filepath.Join(otherPath, "src")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(demoPath, "src", "App.tsx"), []byte(`import { Button } from "@vrooli/react-component-library/Button/1.0.0"; export const App = Button;`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherPath, "src", "App.tsx"), []byte(`import { Card } from "@vrooli/react-component-library/Card/1.0.0"; export const App = Card;`), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := Package{Name: "react-component-library", Manifest: Manifest{Package: ManifestEntry{Refresh: RefreshPolicy{Strategy: RefreshScenarioSetup}}}}
	dependents := []Dependent{
		{PackageName: pkg.Name, ConsumerName: "demo", ConsumerPath: demoPath, ConsumerClass: ConsumerScenarioUI},
		{PackageName: pkg.Name, ConsumerName: "other", ConsumerPath: otherPath, ConsumerClass: ConsumerScenarioUI},
	}

	plan, err := PlanRefreshForArtifact(root, pkg, dependents, "all", []string{"./Button/1.0.0", "./foundation/1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].ConsumerName != "demo" || plan.Actions[0].Impact != ImpactAffected {
		t.Fatalf("exact changed-export plan = %#v", plan)
	}
	if len(plan.Impacts) != 2 || plan.Impacts[0].Status != ImpactAffected || plan.Impacts[1].Status != ImpactUnaffected {
		t.Fatalf("impact report = %#v", plan.Impacts)
	}
}

func TestPlanRefreshForArtifactTreatsUnknownImpactConservatively(t *testing.T) {
	root := t.TempDir()
	consumer := filepath.Join(root, "scenarios", "demo", "ui")
	if err := os.MkdirAll(consumer, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := Package{Name: "react-component-library", Manifest: Manifest{Package: ManifestEntry{Refresh: RefreshPolicy{Strategy: RefreshScenarioSetup}}}}
	plan, err := PlanRefreshForArtifact(root, pkg, []Dependent{{PackageName: pkg.Name, ConsumerName: "demo", ConsumerPath: consumer, ConsumerClass: ConsumerScenarioUI}}, "all", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Impact != ImpactUnknown || plan.Actions[0].Reason == "" {
		t.Fatalf("unknown impact plan = %#v", plan)
	}
}
