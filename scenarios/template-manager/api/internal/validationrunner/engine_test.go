package validationrunner

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func TestFindingFromValidationIssueDerivesStableKeyFromFailureClass(t *testing.T) {
	issue := templatecontracts.TemplateValidationIssue{
		Template: "react-vite",
		Path:     "test-genie/deep-validation/phase-results",
		Message:  "test-genie deep validation failed: 3 passed, 1 failed, 4 total; failed phases: build /tmp/vrooli-template-deep-9182",
	}

	finding := findingFromValidationIssue("react-vite", issue)

	if finding.Key != "react-vite.test-genie.deep-validation.phase-results" {
		t.Fatalf("Key = %q, want the failure-class key, not a slugified message", finding.Key)
	}
	if finding.Severity != "error" {
		t.Fatalf("Severity = %q, want error for a failed phase", finding.Severity)
	}
	if finding.Summary != issue.Message {
		t.Fatalf("Summary = %q, want the full prose preserved", finding.Summary)
	}
}

func TestFindingFromValidationIssueKeyIsStableAcrossVolatileMessages(t *testing.T) {
	first := findingFromValidationIssue("react-vite", templatecontracts.TemplateValidationIssue{
		Path:    "test-genie/deep-validation/protocol",
		Message: "no JSON output at /tmp/vrooli-template-deep-111",
	})
	second := findingFromValidationIssue("react-vite", templatecontracts.TemplateValidationIssue{
		Path:    "test-genie/deep-validation/protocol",
		Message: "no JSON output at /tmp/vrooli-template-deep-222",
	})
	if first.Key != second.Key {
		t.Fatalf("keys diverged across runs: %q vs %q", first.Key, second.Key)
	}
}

func TestFindingFromValidationIssueFallsBackToStableKeyWithoutFailureClass(t *testing.T) {
	// A path-less issue (e.g. a workspace-prep failure) must NOT slugify its
	// message into the key, or every run mints a new transient debt entry.
	finding := findingFromValidationIssue("react-vite", templatecontracts.TemplateValidationIssue{
		Message: "prepare deep validation workspace: symlink /tmp/vrooli-template-deep-333/packages: file exists",
	})
	if finding.Key != "react-vite.template-validate.issue" {
		t.Fatalf("Key = %q, want the stable fallback key", finding.Key)
	}
	if finding.Severity != "error" {
		t.Fatalf("Severity = %q, want error", finding.Severity)
	}
}
