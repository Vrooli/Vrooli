//go:build windows

package process

import "io/fs"

// allocatedBytes falls back to the apparent size: Windows writers do not
// produce sparse log files through truncation.
func allocatedBytes(info fs.FileInfo) int64 {
	return info.Size()
}
