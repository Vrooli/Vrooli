//go:build darwin && !cgo

package hostreqkit

import (
	"context"
	"fmt"
	platform "github.com/vrooli/platform-go"
)

func nativeOpenRGBPlatform() string { return "darwin" }
func confineNativeOpenRGBTransport(context.Context, string) (string, error) {
	return "", fmt.Errorf("native OpenRGB transport isolation requires IOKit CGO: %w", platform.ErrUnsupported)
}
func nativeOpenRGBMachineID() (string, error) {
	return "", fmt.Errorf("native OpenRGB machine identity requires IOKit CGO: %w", platform.ErrUnsupported)
}
func nativeOpenRGBIdentity() (string, string, error) {
	return "", "", fmt.Errorf("native OpenRGB HID requires the IOKit CGO build: %w", platform.ErrUnsupported)
}
func nativeOpenRGBDiscover(context.Context) ([]OpenRGBNativeNode, error) {
	return nil, fmt.Errorf("native OpenRGB HID requires the IOKit CGO build: %w", platform.ErrUnsupported)
}
func nativeOpenRGBAccess(context.Context, OpenRGBNativeNode) error {
	return fmt.Errorf("native OpenRGB HID requires the IOKit CGO build: %w", platform.ErrUnsupported)
}
