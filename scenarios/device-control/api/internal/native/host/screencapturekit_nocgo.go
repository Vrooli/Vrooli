//go:build (darwin && !cgo) || !darwin

package host

import (
	"context"
	"errors"
)

// screenCaptureKitCapture is unavailable when the helper is built without the
// Darwin Objective-C toolchain. The caller may use its compatibility command
// seam, which still classifies an empty or failed capture as permission data.
func screenCaptureKitCapture(context.Context) ([]byte, error) {
	return nil, errors.New("ScreenCaptureKit requires a Darwin cgo build")
}
