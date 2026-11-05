//go:build linux

package process

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func processIncarnation(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid process ID %d", pid)
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "", err
	}
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", err
	}
	// comm can itself contain spaces and parentheses. Field 22 is starttime;
	// the suffix begins with field 3 after the final closing parenthesis.
	end := strings.LastIndexByte(string(stat), ')')
	if end < 0 {
		return "", fmt.Errorf("invalid process stat")
	}
	fields := strings.Fields(string(stat)[end+1:])
	if len(fields) <= 19 || strings.TrimSpace(string(boot)) == "" {
		return "", fmt.Errorf("incomplete process identity")
	}
	start, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil || start == 0 {
		return "", fmt.Errorf("invalid process start identity")
	}
	return strings.TrimSpace(string(boot)) + ":" + fields[19], nil
}
