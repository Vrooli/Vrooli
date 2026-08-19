package packagegov

import (
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
