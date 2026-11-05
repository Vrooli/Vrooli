//go:build !unix

package source

import "os"

// The shared wrapper rejects special files and validates the opened identity.
// Unix additionally protects its named-pipe replacement race with O_NONBLOCK.
func openRecordingFile(path string) (*os.File, error) { return os.Open(path) }
