//go:build !linux

package hostwireless

import (
	"context"
	"errors"
)

// InspectFirmwarePolicy describes the Linux firmware loader and boot archive;
// other operating systems require their own native owner and evidence.
func InspectFirmwarePolicy(ctx context.Context, req FirmwarePolicyRequest) (FirmwarePolicyEvidence, error) {
	e := FirmwarePolicyEvidence{KernelRelease: req.KernelRelease, Reasons: []string{"Linux firmware lookup/initramfs policy is unsupported on this platform"}}
	if err := ctx.Err(); err != nil {
		return e, err
	}
	return e, errors.New(e.Reasons[0])
}
