//go:build windows

package protogen

// handBackOwnership is a no-op: Windows file ownership is inherited from the
// destination directory's ACL.
func handBackOwnership(string, string) error { return nil }
