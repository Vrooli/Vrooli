package resources

import (
	"context"
	"errors"
	"fmt"
	"runtime"
)

// ValidateOpenRGBNativeHostProfile is a read-only pre-effect setup gate. It
// checks the maintained release matrix and native process/host architecture;
// it does not inspect devices, execute a candidate or provision an account.
func ValidateOpenRGBNativeHostProfile(ctx context.Context) error {
	if ctx == nil {
		return errors.New("OpenRGB host profile validation requires a context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	p, err := openRGBReleaseProfileForCurrentHost()
	if err != nil {
		return err
	}
	return p.validate()
}

// Pure cross-target profiles remain available for offline measurement. Host
// admission additionally proves that its process is native, so an emulated
// amd64 CLI cannot turn an unsupported ARM host into an x64 qualification.
func openRGBReleaseProfileForCurrentHost() (openRGBReleaseProfile, error) {
	p, err := openRGBReleaseProfileFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return p, fmt.Errorf("OpenRGB1.1 has no native %s/%s release profile: %w", runtime.GOOS, runtime.GOARCH, err)
	}
	if err := verifyOpenRGBNativeHostArchitecture(); err != nil {
		return openRGBReleaseProfile{}, err
	}
	return p, nil
}

func openRGBWindowsVersionAtLeast(major, minor, build, minimum uint32) bool {
	return minimum != 0 && (major > 10 || major == 10 && (minor > 0 || build >= minimum))
}
