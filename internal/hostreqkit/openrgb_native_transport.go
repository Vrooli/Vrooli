package hostreqkit

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	platform "github.com/vrooli/platform-go"
	"os"
	"path/filepath"
)

const openRGBNativeDarwinTransportProfile = `(version 1)
(allow default)
(deny process-fork)
(deny iokit-open
 (iokit-user-client-class "macUSPCIOUserClient")
 (iokit-connection "macUSPCIO"))`

var openRGBNativeBlockedWindowsModules = []string{"LedsValve.bin", "LpcIO.bin", "SmbusI801.bin", "SmbusPIIX4.bin", "SmbusNCT6793.bin", "SmbusIntelSkylakeIMC.bin"}

func NativeOpenRGBTransportPolicySHA256(goos string) (string, error) {
	switch goos {
	case "darwin":
		return fmt.Sprintf("%x", sha256.Sum256([]byte(openRGBNativeDarwinTransportProfile))), nil
	case "windows":
		return fmt.Sprintf("%x", sha256.Sum256([]byte("OpenRGB1.1WindowsHID-modules-omitted:"+fmt.Sprint(openRGBNativeBlockedWindowsModules)))), nil
	default:
		return "", fmt.Errorf("native OpenRGB transport policy for %s: %w", goos, platform.ErrUnsupported)
	}
}

// ConfineNativeOpenRGBBackendProcess runs only in the fresh owned backend
// wrapper, before its device witness and exec. Darwin confinement is permanent
// for that process and inherited by exec. Windows verifies that the six fixed
// privileged bus modules are absent from the separately verified immutable HID
// derivation. It does not rely on transient driver ACLs or same-user file denial.
// Neither path loads a driver, sends a report, or performs bus transactions.
func ConfineNativeOpenRGBBackendProcess(ctx context.Context, payloadDirectory string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if _, _, err := nativeOpenRGBIdentity(); err != nil {
		return "", err
	}
	return confineNativeOpenRGBTransport(ctx, payloadDirectory)
}

func verifyNativeOpenRGBBlockedModules(ctx context.Context, root string, stat func(string) (os.FileInfo, error)) (string, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return "", errors.New("native OpenRGB transport proof requires exact payload directory")
	}
	for _, name := range openRGBNativeBlockedWindowsModules {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		_, err := stat(filepath.Join(root, name))
		if !errors.Is(err, os.ErrNotExist) {
			return "", errors.Join(fmt.Errorf("unsupported PawnIO module must be omitted from the immutable HID artifact: %s", name), err)
		}
	}
	return NativeOpenRGBTransportPolicySHA256("windows")
}
