//go:build linux

package namespace

import "syscall"

// getKernelVersion returns the running kernel release string via
// syscall.Uname. Only Linux exposes unprivileged overlayfs, so this is the
// only platform where a real kernel version drives the version gate.
func getKernelVersion() string {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		return "unknown"
	}

	// Convert [65]int8 to string
	var buf []byte
	for _, c := range uname.Release {
		if c == 0 {
			break
		}
		buf = append(buf, byte(c))
	}
	return string(buf)
}
