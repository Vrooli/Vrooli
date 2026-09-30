package resourcetemplate

import (
	"strings"
	"testing"

	resourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/resource_template"
)

func TestValidateReportSurfacesPerTemplateStatusAndIssues(t *testing.T) {
	h := &handlers{}
	msg := &resourcev1.ValidateResourceTemplatesResponse{
		Count:       2,
		Status:      "fail",
		IssuesCount: 2,
		Results: []*resourcev1.ResourceTemplateValidationResult{
			{Name: "native-cli", Driver: "native", Status: "pass"},
			{Name: "docker-service", Driver: "docker", Status: "fail", Issues: []string{"generated validation: go mod tidy failed"}},
		},
		Issues: []string{"missing canonical resource template \"manual-resource\""},
	}
	report := h.validateReport(nil, msg)
	if len(report.Summary) != 1 || !strings.Contains(report.Summary[0], "status=fail") || !strings.Contains(report.Summary[0], "issues=2") {
		t.Fatalf("summary = %#v, want status=fail issues=2", report.Summary)
	}
	joined := strings.Join(report.Results, "\n")
	if !strings.Contains(joined, "native-cli [pass]") || !strings.Contains(joined, "docker-service [fail]") {
		t.Fatalf("results = %#v, want per-template status", report.Results)
	}
	if !strings.Contains(joined, "go mod tidy failed") {
		t.Fatalf("results = %#v, want the per-template issue detail", report.Results)
	}
	if !strings.Contains(joined, "fleet: missing canonical") {
		t.Fatalf("results = %#v, want the fleet-level issue", report.Results)
	}
}

func TestValidateOutcomeFailsOnIssues(t *testing.T) {
	h := &handlers{}
	if err := h.validateOutcome(&resourcev1.ValidateResourceTemplatesResponse{IssuesCount: 0}); err != nil {
		t.Fatalf("clean validation should not error: %v", err)
	}
	if err := h.validateOutcome(&resourcev1.ValidateResourceTemplatesResponse{IssuesCount: 1}); err == nil {
		t.Fatalf("validation with issues should error")
	}
}
