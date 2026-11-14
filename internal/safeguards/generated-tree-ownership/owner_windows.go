//go:build windows

package generatedtreeownership

import "os"

// Windows has no root-owned file case.
func ownedByRoot(os.FileInfo) bool { return false }
