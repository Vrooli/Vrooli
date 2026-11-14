package resources

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestOpenRGBDeviceVerifierRequiresPersistentGrantAndNativeLiveWitness(t *testing.T) {
	ctx := context.Background()
	policy := hostreqkit.OpenRGBLinuxPolicy{MachineID: strings.Repeat("a", 32), ArtifactSHA256: updateTestDigest("archive"), DetectorSHA256: updateTestDigest("catalog"), RuleSHA256: updateTestDigest("rule"), GroupID: 1001, Devices: []hostreqkit.OpenRGBLinuxDevice{{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/usb1/1-7/1-7:1.0", HardwareID: "0b05:19af:00"}}}
	id, err := policy.ID()
	if err != nil {
		t.Fatal(err)
	}
	check := OpenRGBDevicePolicyCheck{PolicyID: id, Artifact: ManagedServiceArtifact{Resource: openRGBResourceName, SHA256: policy.ArtifactSHA256}, DetectorSHA256: policy.DetectorSHA256, EffectiveUID: 1000}
	observed, liveCalls := 0, 0
	observe := func(_ context.Context, p hostreqkit.OpenRGBLinuxPolicy, request hostreqkit.OpenRGBLinuxCheck, _ func(int) (string, error)) (hostreqkit.OpenRGBLinuxEvidence, error) {
		observed++
		if p.ArtifactSHA256 != policy.ArtifactSHA256 || request.PID != 0 || request.Incarnation != "" || request.PolicyID != id {
			t.Fatal("device verifier changed the selected policy or transferred a caller access claim")
		}
		return hostreqkit.OpenRGBLinuxEvidence{DeviceGenerationSHA256: updateTestDigest("selected kernel device generation"), StaticGrantConfigured: true, CurrentScopeVerified: true, SelectedNodes: []string{"/dev/hidraw0"}}, nil
	}
	live := func(_ context.Context, request OpenRGBDevicePolicyCheck) error {
		liveCalls++
		if request.PID != check.PID || request.Incarnation != check.Incarnation {
			t.Fatal("native witness target differs from the requested live identity")
		}
		return nil
	}
	if err := verifyOpenRGBDevicePolicyRequest(ctx, policy, check, observe, live); err != nil || liveCalls != 0 {
		t.Fatalf("prospective qualification failed or observed a live process: %v", err)
	}
	check.PID, check.Incarnation = 77, "boot:123"
	if err := verifyOpenRGBDevicePolicyRequest(ctx, policy, check, observe, nil); !errors.Is(err, hostreqkit.ErrOpenRGBLiveDeviceAccessUnverified) {
		t.Fatalf("caller observation was accepted as child evidence: %v", err)
	}
	if err := verifyOpenRGBDevicePolicyRequest(ctx, policy, check, observe, live); err != nil || liveCalls != 1 {
		t.Fatalf("exact native witness was not required: %v", err)
	}
	check.DetectorSHA256 = updateTestDigest("different catalog")
	before := observed
	if err := verifyOpenRGBDevicePolicyRequest(ctx, policy, check, observe, live); err == nil || observed != before || liveCalls != 1 {
		t.Fatal("mismatched detector policy reached access or native observation")
	}
}
