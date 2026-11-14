package resources

import (
	"context"
	"errors"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// The caller observes present metadata, never impersonates the runtime account.
// Actual HID access belongs to the consumed pre-exec witness of that account.
func verifyOpenRGBNativeDevicePolicyRequest(ctx context.Context, p hostreqkit.OpenRGBNativePolicy, check OpenRGBDevicePolicyCheck, live OpenRGBDevicePolicyVerifier) error {
	return verifyOpenRGBNativeDevicePolicyWith(ctx, p, check, hostreqkit.ValidateNativeOpenRGBDeviceSelection, live)
}
func verifyOpenRGBNativeDevicePolicyWith(ctx context.Context, p hostreqkit.OpenRGBNativePolicy, check OpenRGBDevicePolicyCheck, metadata func(context.Context, hostreqkit.OpenRGBNativePolicy) (hostreqkit.OpenRGBNativeEvidence, error), live OpenRGBDevicePolicyVerifier) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := hostreqkit.ValidateOpenRGBNativeDetectorScope(p); err != nil {
		return err
	}
	id, err := p.ID()
	if err != nil {
		return err
	}
	// EffectiveUID belongs to the Linux variant. Zero is the unused native
	// sentinel; Account and the observed OS token establish native authority.
	if check.PolicyID != id || check.Artifact.Resource != openRGBResourceName || check.Artifact.SHA256 != p.ArtifactSHA256 || check.DetectorSHA256 != p.DetectorSHA256 || check.Account != p.Account || check.EffectiveUID != 0 || check.PID < 0 || (check.PID == 0) != (check.Incarnation == "") {
		return errors.New("native OpenRGB request differs from exact machine/account/device enrollment")
	}
	if metadata == nil {
		return errors.New("native OpenRGB metadata observer is required")
	}
	evidence, err := metadata(ctx, p)
	if err != nil {
		return err
	}
	if evidence.PolicyID != id || evidence.Account != p.Account || len(evidence.SelectedPaths) != len(p.Devices) || !updateDigest(evidence.DeviceGenerationSHA256) {
		return errors.New("native OpenRGB selected attachment generation is incomplete")
	}
	if check.PID == 0 {
		return nil
	}
	if live == nil {
		return hostreqkit.ErrOpenRGBLiveDeviceAccessUnverified
	}
	check.DeviceGenerationSHA256 = evidence.DeviceGenerationSHA256
	return live(ctx, check)
}
