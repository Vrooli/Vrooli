//go:build !linux

package codex

// Hosts without /proc report no daemon processes, so stopping a session's
// daemon is a no-op there. Codex's own daemon management is unchanged.
func scanProcessesUnder([]string) ([]DaemonProcess, error) { return nil, nil }

func processRunning(DaemonProcess) bool { return false }

func terminateProcess(DaemonProcess, bool) error { return nil }
