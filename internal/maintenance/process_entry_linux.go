//go:build linux

package maintenance

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	processEntryLinuxParameterA = 4
)

// readProcessEntry reads a single process's table entry straight from /proc,
// fork-free. /proc/<pid>/stat is world-readable, so this also works for
// processes owned by other users.
func readProcessEntry(pid int) (processTableEntry, bool) {
	if pid <= 0 {
		return processTableEntry{}, false
	}
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return processTableEntry{}, false
	}
	entry, ok := parseProcStatEntry(stat)
	if !ok || entry.PID != pid {
		return processTableEntry{}, false
	}
	entry.Command = readProcCmdline(pid, entry.Command)
	entry.Executable, _ = os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	entry.Cwd, _ = os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	return entry, true
}

// parseProcStatEntry parses /proc/<pid>/stat: `pid (comm) state ppid pgrp
// session ...`. comm may contain spaces and parentheses, so the fields after
// it are located from the LAST closing parenthesis. The returned entry's
// Command holds the comm name as a fallback until cmdline is read.
func parseProcStatEntry(stat []byte) (processTableEntry, bool) {
	text := strings.TrimSpace(string(stat))
	open := strings.Index(text, "(")
	closing := strings.LastIndex(text, ")")
	if open < 1 || closing < open || closing+2 >= len(text) {
		return processTableEntry{}, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(text[:open-1]))
	if err != nil {
		return processTableEntry{}, false
	}
	comm := text[open+1 : closing]
	fields := strings.Fields(text[closing+1:])
	// fields: [0]=state [1]=ppid [2]=pgrp [3]=session ...
	if len(fields) < processEntryLinuxParameterA {
		return processTableEntry{}, false
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return processTableEntry{}, false
	}
	pgid, err := strconv.Atoi(fields[2])
	if err != nil {
		return processTableEntry{}, false
	}
	sid, err := strconv.Atoi(fields[3])
	if err != nil {
		return processTableEntry{}, false
	}
	return processTableEntry{
		PID:     pid,
		PPID:    ppid,
		PGID:    pgid,
		SID:     sid,
		State:   fields[0],
		Command: comm,
	}, true
}

// readProcCmdline returns the process's full argv joined with spaces, falling
// back to the supplied comm name when cmdline is empty (zombies, kernel
// threads).
func readProcCmdline(pid int, fallback string) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil || len(bytes.TrimRight(data, "\x00")) == 0 {
		return fallback
	}
	return string(bytes.ReplaceAll(bytes.TrimRight(data, "\x00"), []byte{0}, []byte{' '}))
}
