//go:build windows

package hostfs

// pidAlive is unavailable on Windows, where there is no cheap signal-0 probe.
// Reporting every recorded pid as alive keeps callers fail-closed: storage is
// never reclaimed on the strength of a check this platform cannot make.
func pidAlive(int) bool { return true }
