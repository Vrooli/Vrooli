//go:build !windows

package providers

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// openHandlePaths returns paths held by a live process on Linux. macOS has no
// stable procfs equivalent, so it reports unverifiable there and callers fail
// closed. The scan happens once per provider preview, never once per file.
func openHandlePaths(root string) (map[string]struct{}, bool) {
	if runtime.GOOS != "linux" {
		return nil, false
	}
	processes, err := os.ReadDir("/proc")
	if err != nil {
		return nil, false
	}
	root = filepath.Clean(root)
	handles := make(map[string]struct{})
	for _, process := range processes {
		if !process.IsDir() || process.Name() == "self" || strings.Trim(process.Name(), "0123456789") != "" {
			continue
		}
		fds, readErr := os.ReadDir(filepath.Join("/proc", process.Name(), "fd"))
		if readErr != nil {
			continue
		}
		for _, fd := range fds {
			target, linkErr := os.Readlink(filepath.Join("/proc", process.Name(), "fd", fd.Name()))
			if linkErr != nil {
				continue
			}
			target = strings.TrimSuffix(target, " (deleted)")
			clean := filepath.Clean(target)
			if pathContains(root, clean) {
				handles[clean] = struct{}{}
			}
		}
	}
	return handles, true
}

func hasLeaseFile(path, root string) bool {
	cleanRoot := filepath.Clean(root)
	current := filepath.Clean(path)
	if info, err := os.Stat(current); err == nil && !info.IsDir() {
		current = filepath.Dir(current)
	}
	for {
		for _, marker := range []string{".building", ".lease", ".active", ".running", ".in-progress"} {
			if _, err := os.Stat(filepath.Join(current, marker)); err == nil {
				return true
			}
		}
		if current == cleanRoot || !pathContains(cleanRoot, current) {
			return false
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false
		}
		current = parent
	}
}
