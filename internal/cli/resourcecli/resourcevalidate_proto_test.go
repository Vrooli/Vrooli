package resourcecli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vrooli/vrooli/internal/resources"
)

// TestWriteValidationReportJSONContract pins the `resource validate --json`
// wire shape, including the success-mirrors-passed envelope rule.
func TestWriteValidationReportJSONContract(t *testing.T) {
	report := resources.ResourceValidationReport{
		Count:  2,
		Passed: false,
		Items: []resources.ResourceValidationItem{
			{
				Name:         "postgres",
				ManifestPath: "resources/postgres/manifest.json",
				Driver:       "compose",
				Issues: []resources.ValidationIssue{
					{Severity: "error", Message: "missing port"},
				},
			},
		},
		Issues: []resources.ValidationIssue{
			{Severity: "warning", Message: "catalog stale"},
		},
	}

	var buf bytes.Buffer
	if err := WriteValidationReportJSON(&buf, report); err != nil {
		t.Fatalf("WriteValidationReportJSON: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}

	// success mirrors report.passed.
	if got["success"] != false {
		t.Errorf("success: want false (mirrors passed), got %v", got["success"])
	}

	rep, ok := got["report"].(map[string]any)
	if !ok {
		t.Fatalf("report missing/wrong type: %v", got["report"])
	}
	if rep["count"].(float64) != 2 {
		t.Errorf("report.count: %v", rep["count"])
	}
	if rep["passed"] != false {
		t.Errorf("report.passed: %v", rep["passed"])
	}

	items, ok := rep["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("report.items: want 1, got %v", rep["items"])
	}
	item := items[0].(map[string]any)
	if item["name"] != "postgres" || item["manifest_path"] != "resources/postgres/manifest.json" || item["driver"] != "compose" {
		t.Errorf("item mismatch: %v", item)
	}
	issues := item["issues"].([]any)
	first := issues[0].(map[string]any)
	if first["severity"] != "error" || first["message"] != "missing port" {
		t.Errorf("item issue mismatch: %v", first)
	}

	repIssues := rep["issues"].([]any)
	if repIssues[0].(map[string]any)["severity"] != "warning" {
		t.Errorf("report.issues mismatch: %v", repIssues)
	}
}
