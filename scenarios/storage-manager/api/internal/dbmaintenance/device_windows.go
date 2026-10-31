//go:build windows

package dbmaintenance

// sameFilesystem assumes a shared volume, which demands the larger margin.
func sameFilesystem(string, string) bool { return true }
