//go:build windows

package pstoreobservability

import "os"

// Windows does not expose Unix group ownership through FileInfo. The safeguard
// is unsupported there, so the inspection must safely report non-compliance.
func fileInfoGroupMatches(os.FileInfo, uint32) bool { return false }
