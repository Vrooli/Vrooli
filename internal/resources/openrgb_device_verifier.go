package resources

import (
	"context"
	"errors"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func verifyOpenRGBDevicePolicyRequest(ctx context.Context, policy hostreqkit.OpenRGBLinuxPolicy, check OpenRGBDevicePolicyCheck, observe func(context.Context, hostreqkit.OpenRGBLinuxPolicy, hostreqkit.OpenRGBLinuxCheck, func(int) (string, error)) (hostreqkit.OpenRGBLinuxEvidence, error), live OpenRGBDevicePolicyVerifier) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	id, err := policy.ID()
	if err != nil {
		return err
	}
	if check.PolicyID != id || check.Artifact.Resource != openRGBResourceName || check.Artifact.SHA256 != policy.ArtifactSHA256 || check.DetectorSHA256 != policy.DetectorSHA256 || check.EffectiveUID <= 0 || check.PID < 0 || (check.PID == 0) != (check.Incarnation == "") {
		return errors.New("OpenRGB launch request differs from the exact machine-local device policy")
	}
	evidence, err := observe(ctx, policy, hostreqkit.OpenRGBLinuxCheck{PolicyID: id, ArtifactSHA256: policy.ArtifactSHA256, DetectorSHA256: policy.DetectorSHA256, EffectiveUID: check.EffectiveUID}, nil)
	if err != nil {
		return err
	}
	if !evidence.StaticGrantConfigured || !evidence.CurrentScopeVerified || len(evidence.SelectedNodes) != len(policy.Devices) || !updateDigest(evidence.DeviceGenerationSHA256) {
		return errors.New("OpenRGB selected persistent device grant is incomplete")
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
