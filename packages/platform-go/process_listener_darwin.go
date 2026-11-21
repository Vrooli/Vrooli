//go:build darwin

package platform

import (
	"fmt"
	"golang.org/x/sys/unix"
)

// pcblist_n is the global read-only TCP PCB snapshot, unlike per-process
// descriptor queries which require privilege for foreign users' processes.
func nativeTCPListeners(port uint16) ([]ProcessTCPListener, error) {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		data, err := unix.SysctlRaw("net.inet.tcp.pcblist_n")
		if err != nil {
			return nil, fmt.Errorf("platform: global TCP PCB snapshot: %w", err)
		}
		if len(data) > 32<<20 {
			return nil, fmt.Errorf("platform: TCP PCB snapshot exceeds bound")
		}
		listeners, err := parseDarwinTCPPCBList(data, port)
		if err == nil {
			return listeners, nil
		}
		last = err
	}
	return nil, last
}
