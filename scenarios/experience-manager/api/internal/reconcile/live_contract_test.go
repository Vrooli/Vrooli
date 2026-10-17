package reconcile

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"experience-manager/internal/spec"
)

// TestLiveFleetAffordanceContract keeps the browser capture and the
// experience evaluator joined at the same seam. The fleet page deliberately
// places its search and filter controls beside the table and its sort controls
// inside the table; affordance-present must inspect the complete AX tree.
//
// This is opt-in because it requires the lifecycle-managed BAS and target UI.
func TestLiveFleetAffordanceContract(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("EXPERIENCE_BAS_INTEGRATION_URL"))
	if baseURL == "" {
		t.Skip("set EXPERIENCE_BAS_INTEGRATION_URL to exercise the live fleet capture contract")
	}
	repoRoot := repoRoot(t)
	report, err := spec.ParseScenario(filepath.Join(repoRoot, "scenarios", "experience-manager"))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	page, ok := report.Spec.Pages["fleet"]
	if !ok {
		t.Fatal("experience-manager fleet page is missing")
	}
	profiles, err := CaptureProfilesFromAxes(filepath.Join(repoRoot, "scenarios", "experience-manager", "capabilities", "axes.json"), defaultCaptureBudget)
	if err != nil {
		t.Fatalf("CaptureProfilesFromAxes: %v", err)
	}
	capturer := BASCapturer{Resolve: func(context.Context) (string, error) { return baseURL, nil }}
	for _, profile := range profiles {
		targets := captureTargetsForProfile("experience-manager", page, profile)
		for _, target := range targets {
			snapshot, captureErr := capturer.CaptureAccessibility(context.Background(), target)
			if captureErr != nil {
				t.Logf("profile %q state %q capture unavailable: %v", profile.MatrixID, target.StateID, captureErr)
				continue
			}
			result := reconcileActivePage("experience/pages/fleet.json", pageWithBaselineClaims(page), target, snapshot)
			for _, finding := range result.Findings {
				if (finding.Code == spec.CodeClaimFailed || finding.Code == spec.CodeAffordanceMissing) && strings.Contains(finding.Message, "debt-table-affordances") {
					t.Fatalf("live fleet affordance claim failed for profile %q state %q: %+v", profile.MatrixID, target.StateID, finding)
				}
			}
		}
	}
}
