//go:build !windows

package agentharness

import "os"

// POSIX security is represented by the mode carried in the snapshot.
func permissionFileSecurity(path string) (string, error) { return "", nil }

// SecurePermissionFile supplements POSIX mode handling with native ACL handling
// on Windows. security is empty for a private new file, or the snapshot ACL whose
// access policy must be preserved. Call before writing any configuration bytes.
func SecurePermissionFile(path, security string) error { return nil }

// CreatePermissionTemp creates a unique file with private access at creation.
func CreatePermissionTemp(directory, prefix string) (*os.File, error) {
	return os.CreateTemp(directory, prefix)
}
