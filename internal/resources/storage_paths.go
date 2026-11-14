package resources

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

// resourceStorageResolver resolves resource state for the operator who invoked
// Vrooli. Under `sudo vrooli setup`, that is the SUDO_USER rather than root.
// This keeps the XDG paths used by resource lifecycle and the child process's
// HOME aligned with the same user.
func resourceStorageResolver() (*runtimestorage.Resolver, error) {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return nil, err
	}
	_, _, sudoed := hostreqkit.InvokingUserIDs()
	return resourceStorageResolverForInvoker(home, sudoed, runtimestorage.ResolverConfig{})
}

// The invoking identity is explicit so sudo defaults can be verified without
// changing process identity or touching either user's filesystem.
func resourceStorageResolverForInvoker(home string, sudoed bool, cfg runtimestorage.ResolverConfig) (*runtimestorage.Resolver, error) {
	goos := cfg.RuntimeOS
	if goos == "" {
		goos = runtime.GOOS
	}
	getenv := cfg.EnvGet
	if getenv == nil {
		getenv = os.Getenv
	}
	cfg.AppID = "vrooli"
	cfg.UserHomeDir = func() (string, error) { return home, nil }
	if sudoed && goos == string(hostreqspec.PlatformLinux) {
		// os.UserConfigDir/os.UserCacheDir otherwise resolve sudo's HOME
		// (usually /root), even though the other roots use the invoker.
		cfg.UserConfigDir = func() (string, error) { return filepath.Join(home, ".config"), nil }
		cfg.UserCacheDir = func() (string, error) { return filepath.Join(home, ".cache"), nil }
	}
	cfg.EnvGet = func(key string) string {
		if sudoed && goos == string(hostreqspec.PlatformLinux) {
			switch key {
			case "XDG_DATA_HOME":
				return filepath.Join(home, ".local", "share")
			case "XDG_STATE_HOME":
				return filepath.Join(home, ".local", "state")
			}
		}
		return getenv(key)
	}
	return runtimestorage.NewResolver(cfg)
}

func resourceStoragePaths(resource string) (runtimestorage.Paths, error) {
	if resource == openRGBResourceName {
		if paths, handled, err := nativeOpenRGBStoragePaths(); handled {
			return paths, err
		}
	}
	resolver, err := resourceStorageResolver()
	if err != nil {
		return runtimestorage.Paths{}, err
	}
	return resolver.Resolve(runtimestorage.Options{ResourceID: resource})
}
