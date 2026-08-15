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
