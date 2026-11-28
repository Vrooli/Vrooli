package ai

import (
	"testing"

	"image-tools/internal/backends"
)

// The submit-time tier is the plan; the job's meta says what actually ran, so a
// job that admission moved to the CPU is never reported as a GPU run.
func TestExecutionDeviceMetaNamesTheTierThatRan(t *testing.T) {
	for _, tc := range []struct {
		tier         backends.Tier
		tierName     string
		deviceName   string
		wantRecorded bool
	}{
		{backends.TierLocalGPU, "local-gpu", "gpu", true},
		{backends.TierLocalCPU, "local-cpu", "cpu", true},
		{backends.TierBYOK, "", "", false},
		{backends.TierGatewayLocal, "", "", false},
	} {
		meta := withExecutionDevice(map[string]string{"backend": "onnxruntime"}, tc.tier)
		if meta["backend"] != "onnxruntime" {
			t.Errorf("%v: existing meta was lost: %v", tc.tier, meta)
		}
		if got := meta[MetaExecutionTier]; got != tc.tierName {
			t.Errorf("%v: %s = %q, want %q", tc.tier, MetaExecutionTier, got, tc.tierName)
		}
		if got := meta[MetaExecutionDevice]; got != tc.deviceName {
			t.Errorf("%v: %s = %q, want %q", tc.tier, MetaExecutionDevice, got, tc.deviceName)
		}
	}
	if meta := withExecutionDevice(nil, backends.TierLocalCPU); meta[MetaExecutionDevice] != "cpu" {
		t.Errorf("a provider that returned no meta still gets the device recorded: %v", meta)
	}
	if meta := withExecutionDevice(nil, backends.TierBYOK); meta != nil {
		t.Errorf("a hosted route adds nothing: %v", meta)
	}
}
