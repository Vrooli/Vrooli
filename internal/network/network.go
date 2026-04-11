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

func LockPath(home string, port int) string {
	return filepath.Join(process.ScenarioStateDir(home), fmt.Sprintf(".port_%d.lock", port))
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
