//go:build darwin

package network

import (
	"context"
	"os/exec"
)

// captureTCPListenerSnapshot builds the global listening-port set from
// `netstat -an -p tcp` (one fork, unprivileged, sees ALL users' sockets).
// `lsof` is used ONLY for PID attribution on top of that set: unprivileged
// lsof cannot see other users' processes, so an lsof-only port set would
// misreport another user's bound port as not-listening and downstream
// cleanup would expire a live claim.
func captureTCPListenerSnapshot() TCPListenerSnapshot {
	netstatPath, err := exec.LookPath("netstat")
	if err != nil {
		return TCPListenerSnapshot{Reason: "netstat is not installed", Tool: "netstat"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), listenerEnrichTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, netstatPath, "-an", "-p", "tcp").Output()
	if err != nil {
		return TCPListenerSnapshot{Reason: "netstat -an -p tcp failed: " + err.Error(), Tool: "netstat"}
	}
	ports := make(map[int][]SnapshotListener)
	for _, port := range parseNetstatListenPorts(output) {
		if _, ok := ports[port]; !ok {
			ports[port] = nil
		}
	}
	tool := "netstat"
	if enrichListenerPIDsWithLsof(ports) {
		tool = "netstat+lsof"
	}
	return TCPListenerSnapshot{Known: true, Tool: tool, Ports: ports}
}

// enrichListenerPIDsWithLsof attributes ports to PIDs with one
// `lsof -nP -iTCP -sTCP:LISTEN -Fpcn` fork. Attribution only — the port set
// itself always comes from netstat. Returns whether lsof contributed.
func enrichListenerPIDsWithLsof(ports map[int][]SnapshotListener) bool {
	path, err := exec.LookPath("lsof")
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), listenerEnrichTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, path, "-nP", "-iTCP", "-sTCP:LISTEN", "-Fpcn").Output()
	if err != nil {
		return false
	}
	attributed := parseLsofFieldAttribution(output)
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
