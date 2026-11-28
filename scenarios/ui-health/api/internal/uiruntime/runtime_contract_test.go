package uiruntime

import (
	"strings"
	"testing"

	"ui-health/internal/evidence"
	"ui-health/internal/services/manifestvalidation"
)

func TestFindingsFromEvidenceReportsRCLUsageContractWarnings(t *testing.T) {
	findings := findingsFromEvidence(evidence.Evidence{
		URL:       "http://127.0.0.1:3000/archive",
		Loaded:    true,
		Handshake: evidence.Handshake{Signaled: true},
		Console: []evidence.ConsoleEntry{{
			Level:   "warn",
			Message: "[RCL usage contract] react-component-library:OverlayActionBar@1.0.1: received 5 actions; maximum is 4; call site: at DrawerPage (http://127.0.0.1:3000/src/pages/DrawerPage.tsx:42:9)",
		}},
	}, "mobile")

	for _, finding := range findings {
		if finding.Code == "runtime_rcl_usage_contract" {
			if finding.Severity != manifestvalidation.SeverityError || finding.Location != "http://127.0.0.1:3000/src/pages/DrawerPage.tsx:42:9" {
				t.Fatalf("usage-contract finding should fail at the call-site location, got %+v", finding)
			}
			if !strings.Contains(finding.Message, "react-component-library:OverlayActionBar@1.0.1") || !strings.Contains(finding.Message, "served page: http://127.0.0.1:3000/archive") {
				t.Fatalf("usage-contract finding should identify the asset version and served page, got %+v", finding)
			}
			return
		}
	}
	t.Fatalf("expected UI Health finding for the RCL console warning; got %+v", findings)
}

func TestFindingsFromEvidenceDoesNotReportCleanRCLUsage(t *testing.T) {
	findings := findingsFromEvidence(evidence.Evidence{
		URL:       "http://127.0.0.1:3000/archive",
		Loaded:    true,
		Handshake: evidence.Handshake{Signaled: true},
	}, "mobile")
	for _, finding := range findings {
		if finding.Code == "runtime_rcl_usage_contract" {
			t.Fatalf("clean served evidence must not report an RCL usage violation: %+v", finding)
		}
	}
}

func TestRCLUsageContractCallSiteFallsBackToPageWhenStackIsMissing(t *testing.T) {
	page := "http://127.0.0.1:3000/archive"
	if got := rclUsageContractCallSite(page, "[RCL usage contract] BottomNav: too many items"); got != page {
		t.Fatalf("missing call-site stack should fall back to served page, got %q", got)
	}
	if got := rclUsageContractCallSite(page, "[RCL usage contract] BottomNav: too many items; call site: at BottomNav (file:///tmp/BottomNav.tsx:1:1)"); got != page {
		t.Fatalf("non-HTTP call-site stack should fall back to served page, got %q", got)
	}
}
