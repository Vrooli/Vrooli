//go:build !darwin && !windows

package hostreqkit

import (
	"context"
	"fmt"
	platform "github.com/vrooli/platform-go"
)

func nativeOpenRGBPlatform() string { return "unsupported" }
func confineNativeOpenRGBTransport(context.Context, string) (string, error) {
	return "", fmt.Errorf("native OpenRGB transport isolation: %w", platform.ErrUnsupported)
}
func nativeOpenRGBMachineID() (string, error) {
	return "", fmt.Errorf("native OpenRGB machine identity: %w", platform.ErrUnsupported)
}
func nativeOpenRGBIdentity() (string, string, error) {
	return "", "", fmt.Errorf("native OpenRGB HID identity: %w", platform.ErrUnsupported)
}
func nativeOpenRGBDiscover(context.Context) ([]OpenRGBNativeNode, error) {
	return nil, fmt.Errorf("native OpenRGB HID discovery: %w", platform.ErrUnsupported)
}
func nativeOpenRGBAccess(context.Context, OpenRGBNativeNode) error {
	return fmt.Errorf("native OpenRGB HID access: %w", platform.ErrUnsupported)
}
