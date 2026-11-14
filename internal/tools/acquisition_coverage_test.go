package tools_test

import (
	"testing"

	"github.com/vrooli/vrooli/internal/deployability"
	"github.com/vrooli/vrooli/internal/tools"
)

// TestToolManifestsDeclareAcquisitionRouteForEveryPlatform holds every
// embedded tool manifest to the one acquisition-route rule owned by
// internal/deployability: each claimed OS has a route or an explicit
// unsupported reason, and each OS with arch-selected targets accounts for
// linux/arm64 as well as amd64. Scenario Dependency Analyzer applies the same
// function at runtime, so there is no second copy of the rule to drift.
func TestToolManifestsDeclareAcquisitionRouteForEveryPlatform(t *testing.T) {
	if err := deployability.ValidateToolManifestAcquisitionCoverage(tools.Manifests); err != nil {
		t.Fatalf("tool acquisition coverage:\n%v", err)
	}
}
