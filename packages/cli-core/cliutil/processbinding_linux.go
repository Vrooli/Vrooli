//go:build linux

package cliutil

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ProcessBinding is a read-only kernel observation, never human identity.
// Start time and boot ID prevent PID reuse; cgroup pins the session boundary.
type ProcessBinding struct {
	PID    int    `json:"pid"`
	UID    int    `json:"uid"`
	Start  string `json:"start"`
	Boot   string `json:"boot"`
	Cgroup string `json:"cgroup"`
}

func ObserveProcessBinding(pid int) (ProcessBinding, error) {
	if pid <= 1 {
		return ProcessBinding{}, fmt.Errorf("invalid process ID")
	}
	base := "/proc/" + strconv.Itoa(pid)
	stat, err := os.ReadFile(base + "/stat")
	if err != nil {
		return ProcessBinding{}, err
	}
	end := strings.LastIndex(string(stat), ") ")
	if end < 0 {
		return ProcessBinding{}, fmt.Errorf("invalid process stat")
	}
	fields := strings.Fields(string(stat)[end+2:])
	if len(fields) < 20 || fields[0] == "Z" || fields[0] == "X" {
		return ProcessBinding{}, fmt.Errorf("invalid process stat")
	}
	status, err := os.ReadFile(base + "/status")
	if err != nil {
		return ProcessBinding{}, err
	}
	uid := -1
	for _, line := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(line, "Uid:") {
			values := strings.Fields(line)
			if len(values) >= 2 {
				uid, err = strconv.Atoi(values[1])
				if err != nil {
					return ProcessBinding{}, err
				}
			}
		}
	}
	if uid < 0 {
		return ProcessBinding{}, fmt.Errorf("process UID unavailable")
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return ProcessBinding{}, err
	}
	cgroup, err := os.ReadFile(base + "/cgroup")
	if err != nil {
		return ProcessBinding{}, err
	}
	return ProcessBinding{pid, uid, fields[19], strings.TrimSpace(string(boot)), strings.TrimSpace(string(cgroup))}, nil
}

// ProcessDescendsFrom walks kernel parentage and rechecks the root incarnation.
// A matching UID or a copied environment marker alone cannot satisfy it.
func ProcessDescendsFrom(pid int, root ProcessBinding) bool {
	current, err := ObserveProcessBinding(root.PID)
	if err != nil || current != root {
		return false
	}
	for hops := 0; pid > 1 && hops < 256; hops++ {
		peer, err := ObserveProcessBinding(pid)
		if err != nil || peer.UID != root.UID || peer.Boot != root.Boot || peer.Cgroup != root.Cgroup {
			return false
		}
		if pid == root.PID {
			return true
		}
		stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
		if err != nil {
			return false
		}
		end := strings.LastIndex(string(stat), ") ")
		if end < 0 {
			return false
		}
		fields := strings.Fields(string(stat)[end+2:])
		if len(fields) < 2 {
			return false
		}
		parent, err := strconv.Atoi(fields[1])
		if err != nil || parent == pid {
			return false
		}
		pid = parent
	}
	return false
}

func ProcessParent(pid int) int {
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0
	}
	end := strings.LastIndex(string(stat), ") ")
	if end < 0 {
		return 0
	}
	fields := strings.Fields(string(stat)[end+2:])
	if len(fields) < 2 {
		return 0
	}
	parent, err := strconv.Atoi(fields[1])
	if err != nil || parent == pid {
		return 0
	}
	return parent
}
