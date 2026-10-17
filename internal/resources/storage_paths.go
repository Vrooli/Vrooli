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
	return runtimestorage.NewResolver(runtimestorage.ResolverConfig{
		AppID: "vrooli",
		UserHomeDir: func() (string, error) {
			return home, nil
		},
		EnvGet: func(key string) string {
			if sudoed && runtime.GOOS == string(hostreqspec.PlatformLinux) {
				switch key {
				case "XDG_DATA_HOME":
					return filepath.Join(home, ".local", "share")
				case "XDG_STATE_HOME":
					return filepath.Join(home, ".local", "state")
				}
			}
			return os.Getenv(key)
		},
	})
}

func resourceStoragePaths(resource string) (runtimestorage.Paths, error) {
	resolver, err := resourceStorageResolver()
	if err != nil {
		return runtimestorage.Paths{}, err
	}
	return resolver.Resolve(runtimestorage.Options{ResourceID: resource})
}
