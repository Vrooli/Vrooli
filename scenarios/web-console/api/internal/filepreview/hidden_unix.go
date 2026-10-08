//go:build !windows

package filepreview

import (
	"io/fs"
	"strings"
)

// entryHidden reports whether a directory entry is hidden.
//
// On Unix-like systems "hidden" is purely a naming convention: a leading dot,
// exactly what ls conceals without -a. No syscall is involved, so applying the
// filter during the directory scan costs nothing.
func entryHidden(de fs.DirEntry) bool {
	return strings.HasPrefix(de.Name(), ".")
}
