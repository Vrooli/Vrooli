//go:build windows

package sysmounts

import (
	"context"
	"fmt"
)

// Windows identity probing is kept behind this seam until a native volume
// adapter is available. Returning uncertainty is safer than treating a reused
// drive letter as the same device.
func platformVolumeIdentity(context.Context, string) (VolumeIdentity, error) {
	return VolumeIdentity{}, fmt.Errorf("native Windows volume identity adapter unavailable")
}
