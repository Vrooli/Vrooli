//go:build !windows

package providers

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
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
		procRoot := filepath.Join("/proc", process.Name())
		owner, statErr := os.Stat(procRoot)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return nil, false
		}
		stat, ok := owner.Sys().(*syscall.Stat_t)
		if !ok {
			return nil, false
		}
		// Cleanup is restricted to this user's trees. Do not pretend to inspect
		// foreign private processes; unreadable processes of this user block cleanup.
		if stat.Uid != uint32(os.Geteuid()) {
			continue
		}
		fds, readErr := os.ReadDir(filepath.Join(procRoot, "fd"))
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return nil, false
		}
		for _, fd := range fds {
			fdPath := filepath.Join(procRoot, "fd", fd.Name())
			target, linkErr := os.Readlink(fdPath)
			if linkErr != nil {
				if os.IsNotExist(linkErr) {
					continue
				}
				return nil, false
			}
			// The resumable observer itself holds directory streams. Those metadata
			// handles do not represent a writer; self-owned regular handles still do.
			if process.Name() == strconv.Itoa(os.Getpid()) {
				if info, err := os.Stat(fdPath); err == nil && info.IsDir() {
					continue
				}
			}
			target = strings.TrimSuffix(target, " (deleted)")
			clean := filepath.Clean(target)
			if pathContains(root, clean) {
				handles[clean] = struct{}{}
			}
		}
		for _, reference := range []string{"cwd", "exe"} {
			target, err := os.Readlink(filepath.Join(procRoot, reference))
			if os.IsNotExist(err) {
				continue
			}
			environment, envErr := os.ReadFile(filepath.Join(procRoot, "environ"))
			if envErr != nil && !os.IsNotExist(envErr) {
				return nil, false
			}
			if len(environment) > 1<<20 {
				return nil, false
			}
			home, _ := os.UserHomeDir()
			for _, variable := range strings.Split(string(environment), "\x00") {
				key, value, ok := strings.Cut(variable, "=")
				if !ok {
					continue
				}
				switch key {
				case "HOME", "VROOLI_HOME", "GOCACHE", "GOMODCACHE", "GOTMPDIR", "TMPDIR", "TMP", "TEMP", "SCENARIO_DATA_DIR", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CONFIG_HOME":
					clean := filepath.Clean(value)
					if clean == "/tmp" || clean == filepath.Join(home, ".vrooli", "tmp", "go-work") {
						continue
					}
					if filepath.IsAbs(clean) && pathContains(root, clean) {
						handles[clean] = struct{}{}
					}
				}
			}
			if err != nil {
				return nil, false
			}
			clean := filepath.Clean(strings.TrimSuffix(target, " (deleted)"))
			if pathContains(root, clean) {
				handles[clean] = struct{}{}
			}
		}
		maps, err := os.ReadFile(filepath.Join(procRoot, "maps"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, false
		}
		for _, line := range strings.Split(string(maps), "\n") {
			path := line
			for i := 0; i < 5; i++ {
				path = strings.TrimLeft(path, " \t")
				end := strings.IndexAny(path, " \t")
				if end < 0 {
					path = ""
					break
				}
				path = path[end:]
			}
			clean := filepath.Clean(strings.TrimSuffix(strings.TrimLeft(path, " \t"), " (deleted)"))
			if filepath.IsAbs(clean) && pathContains(root, clean) {
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
