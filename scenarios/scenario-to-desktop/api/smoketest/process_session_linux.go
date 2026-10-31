//go:build linux

package smoketest

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// cleanupProcessSession includes children that move groups or sanitize their
// environment. The optional root is allocated by the smoke owner before launch.
func cleanupProcessSession(sessionID, scopeRoot string, grace time.Duration) ([]int, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("empty process session marker")
	}
	find := func() ([]int, error) {
		entries, err := os.ReadDir("/proc")
		if err != nil {
			return nil, err
		}
		var pids []int
		for _, entry := range entries {
			pid, err := strconv.Atoi(entry.Name())
			if err != nil || pid == os.Getpid() {
				continue
			}
			data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "environ"))
			owned := err == nil && hasSessionMarker(data, sessionID)
			if !owned && scopeRoot != "" {
				for _, link := range []string{"exe", "cwd"} {
					target, readErr := os.Readlink(filepath.Join("/proc", entry.Name(), link))
					target = strings.TrimSuffix(target, " (deleted)")
					if readErr == nil && (target == scopeRoot || isUnderPath(target, scopeRoot)) {
						owned = true
						break
					}
				}
			}
			if !owned {
				continue
			}
			if zombie, _ := processIsZombie(entry.Name()); !zombie {
				pids = append(pids, pid)
			}
		}
		return pids, nil
	}
	pids, err := find()
	if err != nil {
		return nil, fmt.Errorf("scan /proc for launch session: %w", err)
	}
	for _, pid := range pids {
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	if len(pids) > 0 && grace > 0 {
		deadline := time.Now().Add(grace)
		for time.Now().Before(deadline) {
			remaining, scanErr := find()
			if scanErr != nil {
				return nil, fmt.Errorf("rescan /proc for launch session: %w", scanErr)
			}
			if len(remaining) == 0 {
				return nil, nil
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	remaining, err := find()
	if err != nil {
		return nil, fmt.Errorf("verify /proc launch session cleanup: %w", err)
	}
	for _, pid := range remaining {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	if len(remaining) > 0 {
		time.Sleep(25 * time.Millisecond)
	}
	remaining, err = find()
	if err != nil {
		return nil, fmt.Errorf("final /proc launch session assertion: %w", err)
	}
	return remaining, nil
}

func hasSessionMarker(environ []byte, sessionID string) bool {
	return strings.Contains("\x00"+string(environ)+"\x00", "\x00S2D_PROCESS_SESSION="+sessionID+"\x00")
}

func processIsZombie(pid string) (bool, error) {
	data, err := os.ReadFile(filepath.Join("/proc", pid, "stat"))
	if err != nil {
		return false, err
	}
	end := strings.LastIndex(string(data), ")")
	if end < 0 || end+2 >= len(data) {
		return false, fmt.Errorf("malformed /proc/%s/stat", pid)
	}
	return string(data[end+2]) == "Z", nil
}
