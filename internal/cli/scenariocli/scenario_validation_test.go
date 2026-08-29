package scenariocli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/cliout"
)

func TestValidateScenarioManifestsResolvesEveryEdge(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json")
	testkitgo.WriteFile(t, path, `{
  "version": "1.0.0",
  "service": {"name": "alpha"},
  "dependencies": {
    "resources": {"redis": {"enabled": true, "required": false, "startup_policy": "try_start"}},
    "scenarios": {"beta": {"enabled": true, "required": true, "startup_policy": "try_start", "supervision_precedence": "required"}}
  }
}`)

	report := ValidateScenarioManifests(root)
	if !report.Passed || report.ManifestCount != 1 || report.DependencyEdgeCount != 2 {
		t.Fatalf("report = %#v, want one passing manifest with two edges", report)
	}
	if report.IntentCounts.MustStart != 1 || report.IntentCounts.TryStart != 1 || report.IntentCounts.Ignore != 0 {
		t.Fatalf("intent counts = %#v", report.IntentCounts)
	}
}

func TestValidateScenarioManifestsRejectsUnmarkedDisagreement(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json")
	testkitgo.WriteFile(t, path, `{
  "version": "1.0.0",
  "service": {"name": "alpha"},
  "dependencies": {"scenarios": {"beta": {"enabled": true, "required": true, "startup_policy": "try_start"}}}
}`)

	report := ValidateScenarioManifests(root)
	if report.Passed || len(report.Issues) != 1 {
		t.Fatalf("report = %#v, want one validation issue", report)
	}
}

func TestRenderValidateResponseJSONContract(t *testing.T) {
	var output bytes.Buffer
	err := RenderValidateResponse(&output, cliout.FormatJSON, ValidateResponse{Report: ManifestValidationReport{
		Passed: true, ManifestCount: 120, DependencyEdgeCount: 385,
		IntentCounts: SupervisionIntentCounts{MustStart: 136, TryStart: 175, Ignore: 74},
		Issues:       []ManifestValidationIssue{},
	}})
	if err != nil {
		t.Fatalf("RenderValidateResponse: %v", err)
	}
	var response ValidateResponse
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !response.Success || response.Report.DependencyEdgeCount != 385 {
		t.Fatalf("response = %#v", response)
	}
}
