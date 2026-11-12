package resources

import (
	"errors"
	"fmt"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// OpenRGBNativeResourceRoot names the maintained machine-level native resource
// tree. This read-only accessor creates nothing. Explicit setup provisions the
// operator-private admissions. Immutable artifact publication has its own owner.
// Fixture overrides stay inside their leased resource root.
func OpenRGBNativeResourceRoot() (string, error) {
	if override := os.Getenv("VROOLI_RESOURCE_STORAGE_ROOT"); override != "" {
		if !filepath.IsAbs(override) || filepath.Clean(override) != override || strings.TrimSpace(override) != override || strings.ContainsAny(override, "\x00\r\n") {
			return "", errors.New("native OpenRGB resource storage override must be canonical and absolute")
		}
		return filepath.Join(override, "native", "openrgb"), nil
	}
	return nativeOpenRGBMachineRoot()
}

func nativeOpenRGBStoragePaths() (runtimestorage.Paths, bool, error) {
	return nativeOpenRGBStoragePathsFor(runtime.GOOS)
}
func nativeOpenRGBStoragePathsFor(goos string) (runtimestorage.Paths, bool, error) {
	if goos != "darwin" && goos != "windows" {
		return runtimestorage.Paths{}, false, nil
	}
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return runtimestorage.Paths{}, true, err
	}
	// Native mutable storage is one enrolled namespace. Per-class variables may
	// only repeat its exact directories; arbitrary layouts require the coherent
	// global root override instead. Linux and other resources keep their resolver.
	for _, item := range []struct{ key, class string }{{"VROOLI_RESOURCE_CONFIG_ROOT", "config"}, {"VROOLI_RESOURCE_DATA_ROOT", "data"}, {"VROOLI_RESOURCE_CACHE_ROOT", "cache"}, {"VROOLI_RESOURCE_LOGS_ROOT", "logs"}, {"VROOLI_RESOURCE_STATE_ROOT", "state"}} {
		if value := strings.TrimSpace(os.Getenv(item.key)); value != "" {
			expected := filepath.Join(root, item.class)
			same := value == expected
			if goos == "windows" {
				same = strings.EqualFold(value, expected)
			}
			if !filepath.IsAbs(value) || filepath.Clean(value) != value || !same {
				return runtimestorage.Paths{}, true, fmt.Errorf("native OpenRGB %s must equal its canonical directory %s; use VROOLI_RESOURCE_STORAGE_ROOT to move the whole native resource", item.key, expected)
			}
		}
	}
	return runtimestorage.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), CacheDir: filepath.Join(root, "cache"), LogsDir: filepath.Join(root, "logs"), StateDir: filepath.Join(root, "state")}, true, nil
}

// Setup validates endpoint reachability before creating its private namespace.
// Read/status resolution stays separate so an invalid new layout never causes
// effects or prevents inspecting retained state for exact cleanup.
func validateOpenRGBNativeSetupStorage() error {
	_, handled, err := nativeOpenRGBStoragePaths()
	if err != nil || !handled {
		return err
	}
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return err
	}
	return validateOpenRGBNativeEndpointRoot(root)
}
func validateOpenRGBNativeEndpointRoot(root string) error {
	// Every claim endpoint has this fixed hash-name width, irrespective of ID.
	if _, err := openRGBNativeCapsuleEndpointAt(root, "storage-validation"); err != nil {
		return fmt.Errorf("native OpenRGB IPC root exceeds its supported byte bound; use a shorter VROOLI_RESOURCE_STORAGE_ROOT: %w", err)
	}
	return nil
}
