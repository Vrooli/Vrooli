package ai_test

import (
	"testing"

	"image-tools/internal/ai"
	"image-tools/internal/capability"
	"image-tools/internal/safety"
)

// TestConsentWeightsCoverEveryAIOp is the completeness guard for the consent
// table. Every op the AI engine can run must have an EXPLICIT weight, including
// an explicit WeightNone. Without this, a newly added identity-altering op would
// silently inherit WeightNone and bypass the consent control on a stricter
// deployment.
//
// The table itself now lives in the capability manifest set
// (internal/capability); this test proves the AI engine's catalog is a subset of
// it. The manifest coverage test proves the reverse direction across both
// operation registries, so stale rows are caught there.
func TestConsentWeightsCoverEveryAIOp(t *testing.T) {
	for _, op := range ai.Names() {
		if _, ok := capability.Get(op); !ok {
			t.Errorf("AI operation %q has no capability manifest/consent weight; "+
				"add it to internal/capability/manifests.seed.json", op)
		}
		if safety.OpWeight(op) != safety.Weight(capability.ConsentWeightFor(op)) {
			t.Errorf("AI operation %q: safety weight disagrees with the manifest", op)
		}
	}
}
