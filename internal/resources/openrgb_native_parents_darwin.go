//go:build darwin

package resources

import (
	"github.com/vrooli/vrooli/internal/config"
	"path/filepath"
)

func prepareOpenRGBNativeRootParents(root string) error {
	parent := filepath.Dir(root)
	// Common machine namespaces are traversable, never writable by the daemon
	// or by arbitrary local users. Private access begins at exact resource root.
	for _, path := range []string{filepath.Dir(parent), parent} {
		if err := config.EnsureTraversableServiceNamespace(path); err != nil {
			return err
		}
	}
	return nil
}
