//go:build windows

package hostfs

import "context"

// restoreOwnerWritableDirs is a no-op on Windows. Directory deletion there is
// governed by ACLs rather than the parent directory's write bit, so the unix
// read-only-directory repair has no equivalent to perform.
func restoreOwnerWritableDirs(context.Context, string) (int, error) { return 0, nil }
