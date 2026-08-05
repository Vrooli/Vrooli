//go:build !linux && !darwin && !windows

package sysmounts

import (
	"context"
	"fmt"
)

func platformVolumeIdentity(context.Context, string) (VolumeIdentity, error) {
	return VolumeIdentity{}, fmt.Errorf("volume identity adapter unavailable on this platform")
}
