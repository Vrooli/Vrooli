package findingledger

import (
	"testing"
	"time"
)

func TestMergeDeduplicatesByStableValidationTupleAndOrdersByPriority(t *testing.T) {
	now := time.Date(2026, 8, 17, 23, 0, 0, 0, time.UTC)
	base := Finding{Asset: "overlays.dialog", Version: "1.1.0", Check: "content-not-clipped", Viewport: "ultrawide", Theme: "dark", Kit: "vrooli-default", Severity: "error", Adoptions: 15, TargetRung: 3, FirstSeen: now.Add(-time.Hour), LastSeen: now}
	merged := Merge(nil, []Finding{base, base})
	if len(merged) != 1 || merged[0].Identity != Identity(base.Asset, base.Version, base.Check, base.Viewport, base.Theme, base.Kit) {
		t.Fatalf("merged = %+v, want one stable identity", merged)
	}
	other := Finding{Asset: "primitives.surface", Version: "1.0.0", Check: "content-not-clipped", Viewport: "ultrawide", Theme: "dark", Kit: "vrooli-default", Severity: "error", Adoptions: 0, TargetRung: 4}
	ordered := Merge(merged, []Finding{other})
	if ordered[0].Asset != "overlays.dialog" || ordered[0].RankReason == "" {
		t.Fatalf("ordered = %+v, want high-adoption finding first with explanation", ordered)
	}
}
