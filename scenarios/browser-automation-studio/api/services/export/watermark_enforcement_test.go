package export

import (
	"testing"

	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
)

func TestEnforceWatermarkRequirementsUsesGeneratedReplaySpec(t *testing.T) {
	spec := &exportsv1.ReplaySpec{Watermark: &exportsv1.ReplayWatermark{AssetId: "custom-logo"}}
	result := EnforceWatermarkRequirements(spec, true)
	if !result.WasEnforced || result.OriginalAssetID != "custom-logo" || result.OriginalEnabled {
		t.Fatalf("unexpected enforcement result: %+v", result)
	}
	if !spec.GetWatermark().GetEnabled() || spec.GetWatermark().GetAssetId() != VrooliAscensionAssetID {
		t.Fatalf("generated watermark did not enforce built-in asset: %v", spec.GetWatermark())
	}
	if spec.GetWatermark().GetPosition() != "bottom-right" || spec.GetWatermark().GetSize() != 15 || spec.GetWatermark().GetOpacity() != 80 || spec.GetWatermark().GetMargin() != 16 {
		t.Fatalf("generated watermark defaults were not applied: %v", spec.GetWatermark())
	}
}

func TestEnforceWatermarkRequirementsCreatesGeneratedSettings(t *testing.T) {
	spec := &exportsv1.ReplaySpec{}
	result := EnforceWatermarkRequirements(spec, true)
	if !result.WasEnforced || !spec.GetWatermark().GetEnabled() || spec.GetWatermark().GetAssetId() != VrooliAscensionAssetID {
		t.Fatalf("required generated watermark was not created: result=%+v watermark=%v", result, spec.GetWatermark())
	}
}
