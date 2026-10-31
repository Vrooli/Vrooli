//go:build !unix

package evidenceretention

import "io/fs"

type fileKey struct{}

// keyOf has no inode identity off unix; every file is counted.
func keyOf(fs.FileInfo) (fileKey, bool) { return fileKey{}, false }
