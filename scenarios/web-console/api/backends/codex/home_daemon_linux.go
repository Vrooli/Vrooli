//go:build linux

package codex

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// scanProcessesUnder lists processes whose executable path (argv[0]) starts
// with one of the prefixes. It reads only /proc/<pid>/cmdline, stat and
// status; a process that vanishes mid-scan is skipped.
func scanProcessesUnder(prefixes []string) ([]DaemonProcess, error) {
	if len(prefixes) == 0 {
		return nil, nil
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var out []DaemonProcess
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil || len(cmdline) == 0 {
			continue
		}
		args := strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
		if !hasAnyPrefix(args[0], prefixes) {
			continue
		}
		ticks, state, ok := procStat(pid)
		if !ok || state == 'Z' {
			continue
		}
		out = append(out, DaemonProcess{PID: pid, StartTicks: ticks, Cmdline: args, MemoryBytes: procMemory(pid)})
	}
	return out, nil
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

// procStat returns the start ticks (field 22) and state (field 3). The process
// name in field 2 can contain spaces and parentheses, so parsing starts after
// the final ')'.
func procStat(pid int) (ticks uint64, state byte, ok bool) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, 0, false
	}
	end := bytes.LastIndexByte(data, ')')
	if end < 0 || end+2 >= len(data) {
		return 0, 0, false
	}
	fields := bytes.Fields(data[end+2:])
	const startTicksIndex = 19 // field 22 overall; state is index 0 here
	if len(fields) <= startTicksIndex || len(fields[0]) == 0 {
		return 0, 0, false
	}
	ticks, err = strconv.ParseUint(string(fields[startTicksIndex]), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return ticks, fields[0][0], true
}

// procMemory is VmRSS plus VmSwap: an idle Codex is mostly swapped out.
func procMemory(pid int) int64 {
	f, err := os.Open(filepath.Join("/proc", strconv.Itoa(pid), "status"))
	if err != nil {
		return 0
	}
	defer f.Close()
	var total int64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "VmRSS:") && !strings.HasPrefix(line, "VmSwap:") {
			continue
		}
		if fields := strings.Fields(line); len(fields) >= 2 {
			if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				total += kb * 1024
			}
		}
	}
	return total
}

// processRunning is true while the exact process (pid and start time) is alive
// and not a zombie.
func processRunning(proc DaemonProcess) bool {
	ticks, state, ok := procStat(proc.PID)
	return ok && ticks == proc.StartTicks && state != 'Z'
}

// terminateProcess signals the process only while it is still the one that was
// listed, so a reused pid is never hit.
func terminateProcess(proc DaemonProcess, force bool) error {
	if !processRunning(proc) {
		return nil
	}
	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}
	err := syscall.Kill(proc.PID, sig)
	if err == syscall.ESRCH {
		return nil
	}
	return err
}
