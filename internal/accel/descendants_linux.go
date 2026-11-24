//go:build linux

package accel

import (
	"os"
	"strconv"
	"strings"
)

// processDescendantPIDs returns the live descendants of root by reading the
// process table under /proc. The pid that holds a device is frequently a child
// of the pid the control plane supervises (Ollaya's CUDA runner, Ollama's
// llama-server), and nvidia-smi reports the child's basename rather than a path,
// so ancestry is the one reliable join key.
func processDescendantPIDs(root int) map[int]struct{} {
	if root <= 1 {
		return nil
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	children := make(map[int][]int, len(entries))
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		ppid, ok := readParentPID(pid)
		if !ok {
			continue
		}
		children[ppid] = append(children[ppid], pid)
	}
	out := map[int]struct{}{}
	queue := append([]int(nil), children[root]...)
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if _, seen := out[pid]; seen {
			continue
		}
		out[pid] = struct{}{}
		queue = append(queue, children[pid]...)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// readParentPID parses field 4 of /proc/<pid>/stat. The comm field can contain
// spaces and parentheses, so split after its final ')'.
func readParentPID(pid int) (int, bool) {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, false
	}
	text := string(raw)
	closeIdx := strings.LastIndexByte(text, ')')
	if closeIdx < 0 || closeIdx+2 >= len(text) {
		return 0, false
	}
	fields := strings.Fields(text[closeIdx+2:])
	if len(fields) < 2 {
		return 0, false
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil || ppid < 0 {
		return 0, false
	}
	return ppid, true
}
