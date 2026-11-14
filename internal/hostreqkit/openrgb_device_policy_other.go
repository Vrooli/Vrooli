//go:build !linux

package hostreqkit

import (
	"context"
	"fmt"

	platform "github.com/vrooli/platform-go"
)

func ObserveLinuxOpenRGBDevicePolicy(context.Context, OpenRGBLinuxPolicy, OpenRGBLinuxCheck, func(int) (string, error)) (OpenRGBLinuxEvidence, error) {
	return OpenRGBLinuxEvidence{}, fmt.Errorf("Linux OpenRGB device policy: %w", platform.ErrUnsupported)
}

func DiscoverLinuxOpenRGBDevices(context.Context) ([]OpenRGBLinuxNode, error) {
	return nil, fmt.Errorf("Linux OpenRGB discovery: %w", platform.ErrUnsupported)
}

func ResolveLinuxOpenRGBDevices(context.Context, []OpenRGBLinuxDevice) ([]OpenRGBLinuxNode, error) {
	return nil, fmt.Errorf("Linux OpenRGB discovery: %w", platform.ErrUnsupported)
}
