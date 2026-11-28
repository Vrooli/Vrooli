package claimtypes

import "testing"

func TestGeometryFloorsRemainKnownWithoutClaimingEXMEvaluation(t *testing.T) {
	for _, claimType := range []string{
		"no-document-horizontal-overflow", "viewport-fill", "chrome-pinned",
		"safe-area-tap-targets", "single-line-chrome", "tap-target-size",
	} {
		if !IsKnown(claimType) {
			t.Errorf("geometry claim %q must remain parseable", claimType)
		}
		if IsImplemented(claimType) {
			t.Errorf("geometry claim %q must be owned by ui-health, not EXM", claimType)
		}
	}
	if !IsImplemented("accessible-name") {
		t.Fatal("non-geometry Experience Manager claim ownership must remain intact")
	}
}
