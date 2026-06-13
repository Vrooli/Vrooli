package network

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// The parsers below are platform-neutral pure functions over captured tool
// output so they stay unit-testable on every GOOS; the per-platform capture
// entry points live in snapshot_{linux,darwin,windows,other}.go. Anything
// host-dependent (label enrichment from /proc) is injected by the caller so
// fixture PIDs in tests never collide with real processes on a busy host.

// parseProcNetTCPListenPorts extracts local ports of sockets in LISTEN state
// (st == 0A) from /proc/net/tcp{,6} content. Malformed lines are skipped.
func parseProcNetTCPListenPorts(data []byte) []int {
	out := make([]int, 0, 32)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		// fields: [0]=sl [1]=local_address(hexaddr:hexport) [2]=rem_address [3]=st ...
		if len(fields) < 4 || !strings.HasSuffix(fields[0], ":") {
			continue
		}
		if !strings.EqualFold(fields[3], "0A") {
			continue
		}
		sep := strings.LastIndex(fields[1], ":")
		if sep < 0 {
			continue
		}
		port, err := strconv.ParseInt(fields[1][sep+1:], 16, 32)
		if err != nil || port <= 0 || port > 65535 {
			continue
		}
		out = append(out, int(port))
	}
	return out
}

// parseSSListenerAttribution parses `ss -ltnpH` lines of the shape:
//
//	LISTEN 0 4096 127.0.0.1:5432 0.0.0.0:* users:(("postgres",pid=1234,fd=5))
//
// labelFor resolves a PID to a richer label (the linux capture passes
// readCmdlineLabel for parity with the process table command column); the ss
// comm name is the fallback when it returns "".
func parseSSListenerAttribution(output []byte, labelFor func(pid int) string) map[int][]SnapshotListener {
	out := make(map[int][]SnapshotListener)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		sep := strings.LastIndex(fields[3], ":")
		if sep < 0 {
			continue
		}
		port, err := strconv.Atoi(fields[3][sep+1:])
		if err != nil || port <= 0 {
			continue
		}
		for _, listener := range parseSSUsersToken(line, labelFor) {
			if !containsListenerPID(out[port], listener.PID) {
				out[port] = append(out[port], listener)
			}
		}
	}
	return out
}

func parseSSUsersToken(line string, labelFor func(pid int) string) []SnapshotListener {
	start := strings.Index(line, "users:((")
	if start < 0 {
		return nil
	}
	segment := line[start+len("users:(("):]
	if end := strings.LastIndex(segment, "))"); end >= 0 {
		segment = segment[:end]
	}
	listeners := make([]SnapshotListener, 0, 2)
	for _, entry := range strings.Split(segment, "),(") {
		name := ""
		pid := 0
		for _, part := range strings.Split(entry, ",") {
			part = strings.TrimSpace(part)
			switch {
			case strings.HasPrefix(part, "\""):
				name = strings.Trim(part, "\"")
			case strings.HasPrefix(part, "pid="):
				pid, _ = strconv.Atoi(strings.TrimPrefix(part, "pid="))
			}
		}
		if pid <= 0 {
			continue
		}
		label := ""
		if labelFor != nil {
			label = labelFor(pid)
		}
		if label == "" {
			label = name
		}
		listeners = append(listeners, SnapshotListener{PID: pid, Label: label})
	}
	return listeners
}

// readCmdlineLabel returns the full argv of a process from /proc, or "" when
// unreadable (non-Linux, vanished process, or zombie with an empty cmdline).
func readCmdlineLabel(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	data = bytes.TrimRight(data, "\x00")
	if len(data) == 0 {
		return ""
	}
	return string(bytes.ReplaceAll(data, []byte{0}, []byte{' '}))
}

// pidIsZombie reports whether /proc/<pid>/stat shows state Z. Best-effort:
// false when /proc is unavailable (non-Linux) or the process vanished.
func pidIsZombie(pid int) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return false
	}
	text := strings.TrimSpace(string(data))
	closing := strings.LastIndex(text, ")")
	if closing < 0 || closing+2 >= len(text) || text[closing+1] != ' ' {
		return false
	}
	return text[closing+2] == 'Z'
}

// parseNetstatListenPorts extracts local ports from darwin netstat lines:
//
//	tcp4  0  0  127.0.0.1.5432  *.*  LISTEN
//
// The local-address port is the segment after the LAST dot (darwin uses dots
// as address separators).
func parseNetstatListenPorts(output []byte) []int {
	out := make([]int, 0, 32)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 || fields[len(fields)-1] != "LISTEN" {
			continue
		}
		if !strings.HasPrefix(fields[0], "tcp") {
			continue
		}
		local := fields[3]
		sep := strings.LastIndex(local, ".")
		if sep < 0 {
			continue
		}
		port, err := strconv.Atoi(local[sep+1:])
		if err != nil || port <= 0 || port > 65535 {
			continue
		}
		out = append(out, port)
	}
	return out
}

// parseLsofFieldAttribution parses `lsof -F pcn` records: `p<pid>` starts a
// process record, `c<command>` names it, each `n<addr>:<port>` line is a
// listening socket of the current process.
func parseLsofFieldAttribution(output []byte) map[int][]SnapshotListener {
	out := make(map[int][]SnapshotListener)
	pid := 0
	label := ""
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'p':
			pid, _ = strconv.Atoi(line[1:])
			label = ""
		case 'c':
			label = line[1:]
		case 'n':
			if pid <= 0 {
				continue
			}
			sep := strings.LastIndex(line, ":")
			if sep < 0 {
				continue
			}
			port, err := strconv.Atoi(line[sep+1:])
			if err != nil || port <= 0 {
				continue
			}
			if !containsListenerPID(out[port], pid) {
				out[port] = append(out[port], SnapshotListener{PID: pid, Label: label})
			}
		}
	}
	return out
}

func containsListenerPID(listeners []SnapshotListener, pid int) bool {
	for _, listener := range listeners {
		if listener.PID == pid {
			return true
		}
	}
	return false
}
