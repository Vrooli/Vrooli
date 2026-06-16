package phases

import (
	"sort"
	"testing"
)

// TestComprehensivePresetEqualsCatalog is the anti-drift guard that locks the
// "comprehensive" preset to the full catalog. The forward direction (every
// preset phase exists in the catalog) is covered by TestPresetsResolveAgainstCatalog;
// this is the reverse direction: every catalog phase MUST appear in comprehensive,
// and comprehensive must contain nothing extra. Adding a phase to the catalog
// auto-joins comprehensive (it is computed from ValidPhaseNames), so this test
// fails only if someone reintroduces a hand-maintained comprehensive list that
// drifts from the catalog.
func TestComprehensivePresetEqualsCatalog(t *testing.T) {
	comprehensive, ok := DefaultPresets()[PresetComprehensive.String()]
	if !ok {
		t.Fatalf("DefaultPresets() is missing the %q preset", PresetComprehensive)
	}

	want := append([]string(nil), ValidPhaseNames()...)
	got := append([]string(nil), comprehensive...)
	sort.Strings(want)
	sort.Strings(got)

	if len(got) != len(want) {
		t.Fatalf("comprehensive has %d phases, catalog has %d:\n  comprehensive=%v\n  catalog=%v",
			len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("comprehensive preset drifted from catalog:\n  comprehensive=%v\n  catalog=%v", got, want)
		}
	}
}

// TestComprehensivePresetIncludesPreviouslyDroppedPhases pins the specific
// regression that motivated this guard: ui-health (mandatory) and architecture
// (advisory) were silently omitted from a hand-maintained comprehensive list.
func TestComprehensivePresetIncludesPreviouslyDroppedPhases(t *testing.T) {
	comprehensive := DefaultPresets()[PresetComprehensive.String()]
	have := make(map[string]struct{}, len(comprehensive))
	for _, p := range comprehensive {
		have[p] = struct{}{}
	}
	for _, required := range []Name{UIHealth, Architecture, Quality} {
		if _, ok := have[required.String()]; !ok {
			t.Errorf("comprehensive preset must include %q (regression: it was once silently dropped)", required)
		}
	}
}
