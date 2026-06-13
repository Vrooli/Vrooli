//go:build linux

package network

import (
	"fmt"
	"os"
	"os/exec"
)

// captureTCPListenerSnapshot builds the global listening-port set from
// /proc/net/tcp{,6} — kernel-global, permission-free, fork-free. PID
// attribution is optional enrichment via a single `ss -ltnpH` invocation;
// when ss is absent or unprivileged the ports stay Known with empty
// attribution.
func captureTCPListenerSnapshot() TCPListenerSnapshot {
	ports := make(map[int][]SnapshotListener)
	parsedAny := false
	var firstErr error
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		data, err := os.ReadFile(path)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, port := range parseProcNetTCPListenPorts(data) {
			if _, ok := ports[port]; !ok {
				ports[port] = nil
			}
		}
		parsedAny = true
	}
	if !parsedAny {
		reason := "cannot read /proc/net/tcp"
		if firstErr != nil {
			reason = fmt.Sprintf("cannot read /proc/net/tcp: %v", firstErr)
		}
		return TCPListenerSnapshot{Reason: reason, Tool: "procfs"}
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
	output, err := exec.Command(path, "-ltnpH").Output()
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
