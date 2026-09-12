//go:build windows

package providers

// Windows does not expose a portable standard-library handle enumeration
// equivalent. Spec providers therefore fail closed for open-handle checks.
func openHandlePaths(string) (map[string]struct{}, bool) { return nil, false }

func hasLeaseFile(path, root string) bool { return false }
