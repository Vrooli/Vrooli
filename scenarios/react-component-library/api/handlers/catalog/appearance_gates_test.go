package catalog

import "testing"

func TestAppearanceGatesUseVersionPinnedBrowserStories(t *testing.T) {
	for _, gate := range []string{"unit", "interaction", "accessibility", "responsive"} {
		if !appearanceGate(gate) {
			t.Errorf("%s must run against the version-pinned browser story report", gate)
		}
	}
	for _, gate := range []string{"visual", "composition", "tokens"} {
		if appearanceGate(gate) {
			t.Errorf("%s must keep its owning non-appearance evaluator", gate)
		}
	}
}
