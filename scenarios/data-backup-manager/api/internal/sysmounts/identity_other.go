//go:build !linux && !darwin && !windows

package sysmounts

import (
	"context"
	"fmt"
)

func platformVolumeIdentity(context.Context, string) (VolumeIdentity, error) {
	return VolumeIdentity{}, fmt.Errorf("volume identity adapter unavailable on this platform")
}

func platformDeviceByIdentity(context.Context, string, string) (Volume, error) {
	return Volume{}, fmt.Errorf("stable device identity inventory unavailable on this platform")
}
