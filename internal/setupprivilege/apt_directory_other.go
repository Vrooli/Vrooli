//go:build !linux

package setupprivilege

import "os"

func isRootOwnedDirectory(os.FileInfo) bool { return false }
