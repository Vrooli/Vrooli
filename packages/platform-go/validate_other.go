//go:build !unix

package platform

// ownRuntimeDir has no meaning off Unix: systemd validation runs only there.
func ownRuntimeDir(string) bool { return true }
