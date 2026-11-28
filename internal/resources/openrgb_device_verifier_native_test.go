package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestOpenRGBNativeVerifierSeparatesMetadataAdmissionAndConsumedAccess(t *testing.T) {
	p, _, _ := nativeResourcePolicyFixture(t)
	id, err := p.ID()
	if err != nil {
		t.Fatal(err)
	}
	check := OpenRGBDevicePolicyCheck{Account: p.Account, PolicyID: id, Artifact: ManagedServiceArtifact{Resource: openRGBResourceName, SHA256: p.ArtifactSHA256}, DetectorSHA256: p.DetectorSHA256}
	metadataCalls, liveCalls := 0, 0
	generation := updateTestDigest("current native HID birth")
	metadata := func(_ context.Context, policy hostreqkit.OpenRGBNativePolicy) (hostreqkit.OpenRGBNativeEvidence, error) {
		metadataCalls++
		if policy.Account != p.Account {
			t.Fatal("metadata account substituted")
		}
		return hostreqkit.OpenRGBNativeEvidence{PolicyID: id, Account: p.Account, DeviceGenerationSHA256: generation, SelectedPaths: []string{"native HID current path"}}, nil
	}
	live := func(_ context.Context, request OpenRGBDevicePolicyCheck) error {
		liveCalls++
		if request.DeviceGenerationSHA256 != generation || request.Account != p.Account {
			t.Fatal("live verification lost refreshed native generation/account")
		}
		return nil
	}
	if err = verifyOpenRGBNativeDevicePolicyWith(t.Context(), p, check, metadata, live); err != nil || metadataCalls != 1 || liveCalls != 0 {
		t.Fatal("prospective metadata admission replayed runtime HID access", err)
	}
	check.PID = 123
	check.Incarnation = "native-birth"
	if err = verifyOpenRGBNativeDevicePolicyWith(t.Context(), p, check, metadata, nil); !errors.Is(err, hostreqkit.ErrOpenRGBLiveDeviceAccessUnverified) {
		t.Fatal("live process admitted without its consumed access witness", err)
	}
	if err = verifyOpenRGBNativeDevicePolicyWith(t.Context(), p, check, metadata, live); err != nil || liveCalls != 1 {
		t.Fatal("live native witness refused", err)
	}
	before := metadataCalls
	check.Account = "uid:999"
	if err = verifyOpenRGBNativeDevicePolicyWith(t.Context(), p, check, metadata, live); err == nil || metadataCalls != before {
		t.Fatal("different-account request reached metadata observation")
	}
}
