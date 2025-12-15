package main

import "testing"

func TestManifestValidationErrorsAreActionable(t *testing.T) {
	// [REQ:STC-P0-009] preflight errors are actionable
	manifest := CloudManifest{
		Version: "1.0.0",
		Target: ManifestTarget{
			Type: "vps",
			VPS:  &ManifestVPS{},
		},
		Scenario: ManifestScenario{ID: "landing-page-business-suite"},
		Bundle:   ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:    ManifestPorts{UI: 3000, API: 3001, WS: 3002},
		Edge:     ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}

	_, issues := ValidateAndNormalizeManifest(manifest)
	if len(issues) == 0 {
		t.Fatalf("expected issues")
	}

	for _, issue := range issues {
		if issue.Severity == SeverityError && issue.Hint == "" {
			t.Fatalf("expected error issues to include a hint, got: %+v", issue)
		}
	}
}
