package eligibility

import "testing"

func TestPathDecision_IsRoutedAndIsRefused(t *testing.T) {
	routed := PathDecision{Path: PathRouted}
	if !routed.IsRouted() || routed.IsRefused() {
		t.Fatalf("PathRouted should be routed and not refused; got %+v", routed)
	}

	for _, p := range []Path{
		PathRefusedIsolation,
		PathRefusedProviderUnreachable,
		PathRefusedPreflight,
		PathRefusedProductionMode,
	} {
		d := PathDecision{Path: p}
		if d.IsRouted() {
			t.Errorf("%s should not be routed", p)
		}
		if !d.IsRefused() {
			t.Errorf("%s should be refused", p)
		}
	}
}
