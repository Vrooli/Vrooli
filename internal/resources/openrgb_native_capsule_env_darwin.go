//go:build darwin

package resources

import (
	"context"
	platform "github.com/vrooli/platform-go"
	"os"
)

// launchd can add metadata to the original environment. Reexec the already
// verified immutable launcher with its exact persisted environment; no unknown
// manager variable is silently whitelisted or passed to OpenRGB.
func normalizeOpenRGBNativeEnvironment(ctx context.Context, ref platform.ServiceRef) error {
	expected, err := platform.FrozenOwnedServiceEnvironment(ctx, ref)
	if err != nil {
		return err
	}
	want, err := validateOpenRGBNativeCapsuleEnvironment(expected, "darwin")
	if err != nil {
		return err
	}
	actual, err := platform.ReadProcessEnvironment(os.Getpid())
	if err != nil {
		return err
	}
	equal := len(want) == len(actual)
	for key, value := range want {
		if actual[key] != value {
			equal = false
			break
		}
	}
	if equal {
		return nil
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	image, err := platform.OpenOwnedLauncherImage(ref)
	if err != nil {
		return err
	}
	defer image.Close()
	return platform.ReplaceProcessImage(image, append([]string(nil), os.Args...), expected)
}
