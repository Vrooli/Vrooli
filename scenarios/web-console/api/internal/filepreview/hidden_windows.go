//go:build windows

package filepreview

import (
	"io/fs"
	"strings"
	"syscall"
)

// entryHidden reports whether a directory entry is hidden.
//
// Windows has a real hidden attribute rather than a naming convention, so this
// consults the file attributes as well as the leading dot that cross-platform
// tooling still uses (.git, .env). The attribute lookup is free here: the
// directory scan that produced this entry already carried its metadata, so
// Info() does not cost an extra syscall on this platform.
//
// SYSTEM is treated as hidden alongside HIDDEN, matching what `dir` conceals
// by default.
func entryHidden(de fs.DirEntry) bool {
	if strings.HasPrefix(de.Name(), ".") {
		return true
	}
	info, err := de.Info()
	if err != nil {
		return false
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return false
	}
	const hiddenMask = syscall.FILE_ATTRIBUTE_HIDDEN | syscall.FILE_ATTRIBUTE_SYSTEM
	return data.FileAttributes&hiddenMask != 0
}
