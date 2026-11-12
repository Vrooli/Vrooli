package lightingruntime

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/config"
)

func (c *Controller) nativeStatePolicy() (config.PrivatePathPolicy, error) {
	if c == nil || !filepath.IsAbs(c.home) || filepath.Clean(c.home) != c.home {
		return config.PrivatePathPolicy{}, errors.New("native lighting state requires its original logical operator home")
	}
	// Bind authority to the logical operator's actual home owner, even when a
	// machine broker has a different effective account or HOME environment.
	info, err := os.Lstat(c.home)
	if err != nil {
		return config.PrivatePathPolicy{}, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return config.PrivatePathPolicy{}, errors.New("native lighting operator home is an alias or non-directory")
	}
	owner, err := config.PrivatePathOwner(c.home, info)
	return config.PrivatePathPolicy{OwnerPrincipal: owner}, err
}

func (c *Controller) prepareNativeStateDirectory(p config.PrivatePathPolicy) error {
	var missing []string
	for path := filepath.Dir(c.path); ; path = filepath.Dir(path) {
		info, err := os.Lstat(path)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return errors.New("native lighting state has an aliased parent")
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		missing = append(missing, path)
		if filepath.Dir(path) == path {
			return errors.New("native lighting state has no existing owner namespace")
		}
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if err := config.ConfigurePrivateDirectoryAccess(missing[i], p); err != nil {
			return err
		}
	}
	// This exact lighting directory is private management state. No daemon
	// principal receives HOME traversal or access to unrelated operator state.
	return config.ConfigurePrivateDirectoryAccess(filepath.Dir(c.path), p)
}
