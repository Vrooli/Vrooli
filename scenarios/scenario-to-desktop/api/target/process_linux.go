//go:build linux

package target

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// killElectronDescendants handles Electron runtimes that intentionally place
// bundled services in their own process groups. The launcher process group is
// not a sufficient ownership boundary for those descendants, but their
// parentage remains rooted at the target until the launcher is killed.
func killElectronDescendants(rootPID int) error {
	parents, err := linuxProcessParents()
	if err != nil {
		return err
	}
	children := descendantsOf(rootPID, parents)
	var errs []error
	for _, pid := range children {
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			errs = append(errs, fmt.Errorf("kill Electron descendant %d: %w", pid, err))
		}
	}
	return errors.Join(errs...)
}

func linuxProcessParents() (map[int]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	parents := make(map[int]int)
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue
		}
		text := string(data)
		closeParen := strings.LastIndexByte(text, ')')
		if closeParen < 0 || closeParen+2 >= len(text) {
			continue
		}
		fields := strings.Fields(text[closeParen+2:])
		if len(fields) < 2 {
			continue
		}
		parentPID, err := strconv.Atoi(fields[1])
		if err != nil || parentPID < 0 {
			continue
		}
		parents[pid] = parentPID
	}
	return parents, nil
}

func descendantsOf(rootPID int, parents map[int]int) []int {
	depth := make(map[int]int)
	for pid := range parents {
		for current, level := pid, 0; current != rootPID && level < len(parents); level++ {
			parent, ok := parents[current]
			if !ok {
				break
			}
			if parent == rootPID {
				depth[pid] = level + 1
				break
			}
			current = parent
		}
	}
	children := make([]int, 0, len(depth))
	for pid := range depth {
		children = append(children, pid)
	}
	sort.Slice(children, func(i, j int) bool { return depth[children[i]] > depth[children[j]] })
	return children
}
