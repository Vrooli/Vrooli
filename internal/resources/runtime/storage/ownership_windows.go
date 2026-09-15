//go:build windows

package storage

import "io/fs"

// Windows ownership is SID-based and requires a different API surface. The
// native Windows storage roots are per-profile, so this guard is intentionally
// a no-op until the Windows ACL implementation is introduced.
func requireFileInfoOwnedByCurrentUser(string, fs.FileInfo) error { return nil }

func restoreCreatedStorageOwnership([]string) error { return nil }
