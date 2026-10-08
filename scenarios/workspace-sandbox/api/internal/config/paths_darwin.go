//go:build darwin

package config

import "path/filepath"

func platformStoragePaths(home, temp string) StoragePaths {
	transient := filepath.Join(temp, "workspace-sandbox", StorageNamespace(home))
	return StoragePaths{
		PersistentData: filepath.Join(home, "Library", "Application Support", "workspace-sandbox"),
		Transient:      filepath.Join(transient, "overlays"),
		Runtime:        filepath.Join(transient, "runtime"),
	}
}
