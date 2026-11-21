package resources

import (
	"errors"
	"fmt"
	platform "github.com/vrooli/platform-go"
	"os"
	"runtime"
)

// Runtime authority is observed before native launch. Setup publication is a
// separate explicit operation; unknown security evidence cannot grant runtime.
func validateOpenRGBNativeRuntimeAuthority(goos, arch string, observe func() (platform.ProcessSecurityContext, error)) error {
	if !((goos == openRGBPlatformDarwin && (arch == "amd64" || arch == "arm64")) || (goos == openRGBPlatformWindows && arch == "amd64")) {
		return fmt.Errorf("OpenRGB has no qualified native release profile for %s/%s", goos, arch)
	}
	return validateOpenRGBNativeOperatorAuthority(goos, observe)
}

// Ordinary native lifecycle never executes mutable scenario code with setup
// authority. Disable still works on unsupported artifact architectures.
func requireOpenRGBNativeOperatorAuthority() error {
	if runtime.GOOS != openRGBPlatformDarwin && runtime.GOOS != openRGBPlatformWindows {
		return nil
	}
	return validateOpenRGBNativeOperatorAuthority(runtime.GOOS, func() (platform.ProcessSecurityContext, error) { return platform.ProcessSecurity(os.Getpid()) })
}
func validateOpenRGBNativeOperatorAuthority(goos string, observe func() (platform.ProcessSecurityContext, error)) error {
	security, err := observe()
	if err != nil {
		return errors.Join(errors.New("native OpenRGB runtime security is unobserved"), err)
	}
	if security.Principal == "" || security.Elevated || (goos == openRGBPlatformDarwin && security.EffectiveUID <= 0) || (goos == openRGBPlatformWindows && security.Principal == "S-1-5-18") {
		return errors.New("headless OpenRGB requires the actual unprivileged operator")
	}
	return nil
}
