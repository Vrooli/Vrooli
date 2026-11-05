//go:build linux

package collectors

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// attributeSocketOwners maps TCP socket inodes to the processes holding them.
// It reads no subprocess: /proc/net/tcp supplies the inode set and /proc/<pid>/fd
// supplies the ownership, so this stays honest about fork rate even while
// diagnosing a fork storm.
//
// Cost is proportional to open file descriptors host-wide, which is exactly why
// callers gate it behind shouldAttributeSockets rather than running it every
// cycle.
func attributeSocketOwners(ctx context.Context, established int, limit int) SocketAttribution {
	result := SocketAttribution{Supported: true, Total: established}

	inodes, err := establishedSocketInodes(ctx)
	if err != nil {
		return SocketAttribution{Supported: false, Reason: "read socket inodes: " + err.Error(), Total: established}
	}
	if len(inodes) == 0 {
		return result
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return SocketAttribution{Supported: false, Reason: "read /proc: " + err.Error(), Total: established}
	}

	counts := map[int]int{}
	names := map[int]string{}
	for _, entry := range entries {
		if ctx.Err() != nil {
			break
		}
		if !entry.IsDir() {
			continue
		}
		pid, convErr := strconv.Atoi(entry.Name())
		if convErr != nil {
			continue
		}
		// Permission failures are expected for other users' processes and are
		// intentionally skipped rather than failing the whole reading; the
		// Attributed/Total pair reports the resulting coverage.
		fds, readErr := os.ReadDir(filepath.Join("/proc", entry.Name(), "fd"))
		if readErr != nil {
			continue
		}
		for _, fd := range fds {
			link, linkErr := os.Readlink(filepath.Join("/proc", entry.Name(), "fd", fd.Name()))
			if linkErr != nil {
				continue
			}
			inode, ok := socketInodeFromLink(link)
			if !ok || !inodes[inode] {
				continue
			}
			counts[pid]++
			result.Attributed++
		}
		if counts[pid] > 0 {
			names[pid] = processComm(entry.Name())
		}
	}

	result.Owners = topSocketOwners(counts, names, limit)
	result.Endpoints = topSocketEndpoints(ctx, limit)
	return result
}

func topSocketEndpoints(ctx context.Context, limit int) []SocketEndpoint {
	counts := map[int]int{}
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n")[1:] {
			if ctx.Err() != nil {
				return nil
			}
			fields := strings.Fields(line)
			if len(fields) < 4 || strings.ToUpper(fields[3]) != "01" {
				continue
			}
			parts := strings.Split(fields[2], ":")
			if len(parts) != 2 {
				continue
			}
			port, err := strconv.ParseInt(parts[1], 16, 32)
			if err != nil || port <= 0 {
				continue
			}
			counts[int(port)]++
		}
	}
	endpoints := make([]SocketEndpoint, 0, len(counts))
	for port, count := range counts {
		endpoints = append(endpoints, SocketEndpoint{Scope: "external", Direction: "outbound", Port: port, Connections: count})
	}
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Connections != endpoints[j].Connections {
			return endpoints[i].Connections > endpoints[j].Connections
		}
		return endpoints[i].Port < endpoints[j].Port
	})
	if limit > 0 && len(endpoints) > limit {
		endpoints = endpoints[:limit]
	}
	return endpoints
}

// establishedSocketInodes collects the inode of every established TCP socket.
func establishedSocketInodes(ctx context.Context) (map[uint64]bool, error) {
	inodes := map[uint64]bool{}
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		if ctx.Err() != nil {
			return inodes, ctx.Err()
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n")[1:] {
			fields := strings.Fields(line)
			// inode is field 9; established is state code 01.
			if len(fields) < 10 || strings.ToUpper(fields[3]) != "01" {
				continue
			}
			inode, convErr := strconv.ParseUint(fields[9], 10, 64)
			if convErr != nil {
				continue
			}
			inodes[inode] = true
		}
	}
	return inodes, nil
}

// socketInodeFromLink parses the inode out of a "socket:[12345]" fd link.
func socketInodeFromLink(link string) (uint64, bool) {
	const prefix = "socket:["
	if !strings.HasPrefix(link, prefix) || !strings.HasSuffix(link, "]") {
		return 0, false
	}
	inode, err := strconv.ParseUint(link[len(prefix):len(link)-1], 10, 64)
	if err != nil {
		return 0, false
	}
	return inode, true
}

func processComm(pid string) string {
	raw, err := os.ReadFile(filepath.Join("/proc", pid, "comm"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
