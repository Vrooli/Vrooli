//go:build linux

package executor

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func childLeakDetectionAvailable() bool {
	info, err := os.Stat("/proc")
	return err == nil && info.IsDir()
}

func processGroupHasMembers(groupID int) bool {
	if groupID <= 0 {
		return false
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue
		}
		closeParen := strings.LastIndexByte(string(raw), ')')
		if closeParen < 0 || closeParen+2 >= len(raw) {
			continue
		}
		fields := strings.Fields(string(raw)[closeParen+2:])
		// After comm, fields are state, ppid, pgrp, session, ... .
		if len(fields) >= 3 && fields[2] == strconv.Itoa(groupID) {
			return true
		}
	}
	return false
}

func waitForProcessGroupExit(groupID int) bool {
	for attempt := 0; attempt < 20; attempt++ {
		if !processGroupHasMembers(groupID) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return !processGroupHasMembers(groupID)
}
