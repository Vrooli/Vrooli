//go:build linux

package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// sessionAgentMemory sums the resident and swapped memory of a session's agent
// processes and everything they started (MCP servers, an app-server daemon,
// running commands). Swap counts because an idle Codex is mostly swapped out:
// 0.2 GB resident can sit on 1.6 GB of swap. ok is false when no agent process
// of the session is running.
func sessionAgentMemory(sessionID string) (int64, bool) {
	agents, err := discoverAgentProcesses()
	if err != nil {
		return 0, false
	}
	var queue []int
	for _, p := range agents {
		if p.SessionID == sessionID {
			queue = append(queue, p.PID)
		}
	}
	if len(queue) == 0 {
		return 0, false
	}
	children := procChildren()
	seen := map[int]bool{}
	var total int64
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if seen[pid] {
			continue
		}
		seen[pid] = true
		total += procMemoryBytes(pid)
		queue = append(queue, children[pid]...)
	}
	return total, true
}

// procChildren maps each pid to its children from /proc/<pid>/stat.
func procChildren() map[int][]int {
	out := map[int][]int{}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return out
	}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue
		}
		end := bytes.LastIndexByte(data, ')')
		if end < 0 || end+2 >= len(data) {
			continue
		}
		fields := bytes.Fields(data[end+2:])
		if len(fields) < 2 {
			continue
		}
		if ppid, err := strconv.Atoi(string(fields[1])); err == nil {
			out[ppid] = append(out[ppid], pid)
		}
	}
	return out
}

// procMemoryBytes is VmRSS plus VmSwap of one process.
func procMemoryBytes(pid int) int64 {
	f, err := os.Open(filepath.Join("/proc", strconv.Itoa(pid), "status"))
	if err != nil {
		return 0
	}
	defer f.Close()
	var total int64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "VmRSS:") && !strings.HasPrefix(line, "VmSwap:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				total += kb * 1024
			}
		}
	}
	return total
}
