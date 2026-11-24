//go:build !linux

package cliutil

import "io/fs"

// statChangeToken is unavailable here; the watch then relies on size and
// mtime, as the manifest stat-cache does.
func statChangeToken(fs.FileInfo) (int64, uint64) { return 0, 0 }

// runningExecutableIs cannot be established portably here, so the runtime
// never rewrites the installed manifest; the installer's stamp stands.
func runningExecutableIs(string) bool { return false }
