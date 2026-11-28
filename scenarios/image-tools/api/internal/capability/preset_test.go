package capability

import "testing"

func TestPresetTargetNamesTheRunsOperationModelAndAdapters(t *testing.T) {
	runs, model, adapters, ok := PresetTarget("pose_generate")
	if !ok {
		t.Fatal("pose_generate must be recognized as a preset")
	}
	if runs != "text_to_image" || model != "juggernaut-xl-v9" {
		t.Fatalf("pose_generate target = %q/%q, want text_to_image/juggernaut-xl-v9", runs, model)
	}
	if len(adapters) != 2 {
		t.Fatalf("pose_generate adapters = %d, want 2 (OpenPose + IP-Adapter)", len(adapters))
	}
	if runs, _, _, ok := PresetTarget("text_to_image"); ok || runs != "" {
		t.Fatal("a plain operation must not be reported as a preset")
	}
}
