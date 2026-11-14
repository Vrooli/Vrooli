//go:build linux

package config

import "path/filepath"

func platformStoragePaths(home, temp string) StoragePaths {
	transient := filepath.Join(temp, "workspace-sandbox", StorageNamespace(home))
	return StoragePaths{
		PersistentData: filepath.Join(home, ".local", "share", "workspace-sandbox"),
		Transient:      filepath.Join(transient, "overlays"),
		Runtime:        filepath.Join(transient, "runtime"),
	}
}
