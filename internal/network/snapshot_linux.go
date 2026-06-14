//go:build linux

package network

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Overridable in tests so the capture logic can be pinned against fake
// procfs trees.
var (
	procNetTCPv4Path = "/proc/net/tcp"
	procNetTCPv6Path = "/proc/net/tcp6"
)

// captureTCPListenerSnapshot builds the global listening-port set from
// /proc/net/tcp{,6} — kernel-global, permission-free, fork-free. PID
// attribution is optional enrichment via a single `ss -ltnpH` invocation;
// when ss is absent or unprivileged the ports stay Known with empty
// attribution.
//
// Both address families must contribute or the snapshot is not Known: a
// readable tcp6 with an unreadable tcp (or vice versa) would report every
// listener of the missing family as known-absent, and reconcile expires
// claims on known-absent listeners. The only tolerated gap is a missing
// /proc/net/tcp6 (ENOENT — IPv6 disabled), because then no IPv6 listener can
// exist for the snapshot to miss.
func captureTCPListenerSnapshot() TCPListenerSnapshot {
	ports := make(map[int][]SnapshotListener)
	dataV4, err := os.ReadFile(procNetTCPv4Path)
	if err != nil {
		return TCPListenerSnapshot{Reason: fmt.Sprintf("cannot read %s: %v", procNetTCPv4Path, err), Tool: "procfs"}
	}
	for _, port := range parseProcNetTCPListenPorts(dataV4) {
		if _, ok := ports[port]; !ok {
			ports[port] = nil
		}
	}
	dataV6, err := os.ReadFile(procNetTCPv6Path)
	if err != nil {
		if !os.IsNotExist(err) {
			return TCPListenerSnapshot{Reason: fmt.Sprintf("cannot read %s: %v", procNetTCPv6Path, err), Tool: "procfs"}
		}
		// IPv6 disabled on this host: no v6 listeners can exist, so the v4
		// set alone is complete.
	} else {
		for _, port := range parseProcNetTCPListenPorts(dataV6) {
			if _, ok := ports[port]; !ok {
				ports[port] = nil
			}
		}
	}
	tool := "procfs"
	if enrichListenerPIDsWithSS(ports) {
		tool = "procfs+ss"
	}
	return TCPListenerSnapshot{Known: true, Tool: tool, Ports: ports}
}

// enrichListenerPIDsWithSS attributes listening ports to PIDs with one
// `ss -ltnpH` fork. ss only reports process info for sockets the invoking
// user can see, so attribution may be partial; missing attribution leaves a
// port Known and Listening with an empty listener list. Returns whether ss
// contributed anything.
func enrichListenerPIDsWithSS(ports map[int][]SnapshotListener) bool {
	path, err := exec.LookPath("ss")
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), listenerEnrichTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, path, "-ltnpH").Output()
	if err != nil {
		return false
	}
	attributed := parseSSListenerAttribution(output, readCmdlineLabel)
	if len(attributed) == 0 {
		return false
	}
	for port, listeners := range attributed {
		if _, ok := ports[port]; ok {
			ports[port] = listeners
		}
	}
	return true
}
