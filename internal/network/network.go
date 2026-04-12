package network

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

const (
	mutationLockTimeout     = 2 * time.Second
	mutationLockRetry       = 10 * time.Millisecond
	mutationLockStaleWindow = 30 * time.Second
)

type LockInfo struct {
	Port         int       `json:"port"`
	Scenario     string    `json:"scenario,omitempty"`
	PID          int       `json:"pid,omitempty"`
	Timestamp    time.Time `json:"timestamp,omitempty"`
	Path         string    `json:"path"`
	OwnerRunning bool      `json:"owner_running"`
	Stale        bool      `json:"stale"`
}

type PortListener struct {
	PID     int    `json:"pid"`
	Command string `json:"command,omitempty"`
	Zombie  bool   `json:"zombie"`
}

type ListenerInspection struct {
	Available bool   `json:"available"`
	Tool      string `json:"tool,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type PortInspection struct {
	Listeners  []PortListener     `json:"listeners,omitempty"`
	Inspection ListenerInspection `json:"inspection"`
}

func LockPath(home string, port int) string {
	return filepath.Join(process.ScenarioStateDir(home), fmt.Sprintf(".port_%d.lock", port))
}

func mutationLockPath(home string, port int) string {
	return filepath.Join(process.ScenarioStateDir(home), fmt.Sprintf(".port_%d.guard", port))
}

func ListLocks(home string) ([]LockInfo, error) {
	pattern := filepath.Join(process.ScenarioStateDir(home), ".port_*.lock")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	locks := make([]LockInfo, 0, len(files))
	for _, path := range files {
		lock, err := ReadLockFile(path)
		if err != nil {
			return nil, err
		}
		locks = append(locks, lock)
	}
	return locks, nil
}

func ReadLockFile(path string) (LockInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LockInfo{}, err
	}
	base := filepath.Base(path)
	portText := strings.TrimSuffix(strings.TrimPrefix(base, ".port_"), ".lock")
	port, err := strconv.Atoi(portText)
	if err != nil {
		return LockInfo{}, fmt.Errorf("parse lock port from %s: %w", path, err)
	}

	lock := LockInfo{
		Port: port,
		Path: path,
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		lock.Stale = true
		return lock, nil
	}

	parts := strings.Split(raw, ":")
	if len(parts) > 0 {
		lock.Scenario = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		lock.PID, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
	}
	if len(parts) > 2 {
		if seconds, parseErr := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64); parseErr == nil {
			lock.Timestamp = time.Unix(seconds, 0).UTC()
		}
	}
	lock.OwnerRunning = lock.PID > 0 && process.IsPIDRunning(lock.PID)
	lock.Stale = !lock.OwnerRunning
	return lock, nil
}

func ListenerInspectionStatus() ListenerInspection {
	return listenerInspectionStatus()
}

func InspectPortListeners(port int) (PortInspection, error) {
	return inspectPortListeners(port)
}

// PruneStaleLocks removes stale port locks while holding a per-port mutation
// guard, so maintenance cleanup and runtime lock writers share the same safety
// contract for lock-file mutation.
func PruneStaleLocks(home string) ([]LockInfo, error) {
	stateDir := process.ScenarioStateDir(home)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, err
	}

	locks, err := ListLocks(home)
	if err != nil {
		return nil, err
	}

	cleaned := make([]LockInfo, 0)
	for _, lock := range locks {
		if !lock.Stale {
			continue
		}
		removed, err := removeStaleLock(home, lock.Port)
		if err != nil {
			return nil, err
		}
		if removed {
			cleaned = append(cleaned, lock)
		}
	}
	return cleaned, nil
}

type mutationGuard struct {
	PID       int
	Timestamp time.Time
}

func removeStaleLock(home string, port int) (bool, error) {
	removed := false
	err := withMutationLock(home, port, func() error {
		lockPath := LockPath(home, port)
		lock, err := ReadLockFile(lockPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if lock.PID > 0 && process.IsPIDRunning(lock.PID) {
			return nil
		}
		if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		removed = true
		return nil
	})
	return removed, err
}

func withMutationLock(home string, port int, fn func() error) error {
	release, err := acquireMutationLock(home, port)
	if err != nil {
		return err
	}
	defer release()
	return fn()
}

func acquireMutationLock(home string, port int) (func(), error) {
	path := mutationLockPath(home, port)
	deadline := time.Now().Add(mutationLockTimeout)
	payload := []byte(fmt.Sprintf("%d:%d\n", os.Getpid(), time.Now().UTC().Unix()))

	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			if _, writeErr := file.Write(payload); writeErr != nil {
				_ = file.Close()
				_ = os.Remove(path)
				return nil, writeErr
			}
			if closeErr := file.Close(); closeErr != nil {
				_ = os.Remove(path)
				return nil, closeErr
			}
			return func() {
				_ = os.Remove(path)
			}, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}

		guard, exists, readErr := readMutationGuard(path)
		if readErr == nil && exists {
			age := time.Since(guard.Timestamp)
			if (guard.PID > 0 && !process.IsPIDRunning(guard.PID)) || age > mutationLockStaleWindow {
				_ = os.Remove(path)
				continue
			}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for port %d mutation lock", port)
		}
		time.Sleep(mutationLockRetry)
	}
}

func readMutationGuard(path string) (mutationGuard, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return mutationGuard{}, false, nil
		}
		return mutationGuard{}, false, err
	}
	parts := strings.Split(strings.TrimSpace(string(data)), ":")
	guard := mutationGuard{}
	if len(parts) > 0 {
		guard.PID, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
	}
	if len(parts) > 1 {
		if seconds, parseErr := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); parseErr == nil {
			guard.Timestamp = time.Unix(seconds, 0).UTC()
		}
	}
	return guard, true, nil
}
