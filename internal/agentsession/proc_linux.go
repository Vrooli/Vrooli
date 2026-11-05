//go:build linux

package agentsession

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// procRoot is the proc mount; a variable so fixtures can stand in for it.
var procRoot = "/proc"

// clockTicksPerSecond is USER_HZ, which Linux fixes at 100 for every
// userspace-visible tick count in /proc.
const clockTicksPerSecond = 100

// statFields reads /proc/<pid>/stat and returns the fields after the comm,
// which may itself contain spaces and parentheses.
func statFields(pid int) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH) {
			return nil, ErrProcessGone
		}
		return nil, err
	}
	text := string(data)
	end := strings.LastIndex(text, ")")
	if end < 0 {
		return nil, fmt.Errorf("agentsession: malformed stat for pid %d", pid)
	}
	fields := strings.Fields(text[end+1:])
	// fields[0]=state [1]=ppid ... [19]=starttime
	if len(fields) < 20 {
		return nil, fmt.Errorf("agentsession: short stat for pid %d", pid)
	}
	return fields, nil
}

func processFact(pid int) (ProcessFact, error) {
	fields, err := statFields(pid)
	if err != nil {
		return ProcessFact{}, err
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return ProcessFact{}, fmt.Errorf("agentsession: ppid of %d: %w", pid, err)
	}
	return ProcessFact{PID: pid, PPID: ppid, StartTime: fields[19], TTY: controllingTTY(pid)}, nil
}

func processStartTime(pid int) (string, error) {
	fields, err := statFields(pid)
	if err != nil {
		return "", err
	}
	return fields[19], nil
}

// controllingTTY reads which terminal device the process's standard streams
// are attached to; "" when none is a terminal or the fds are unreadable.
func controllingTTY(pid int) string {
	for _, fd := range []string{"0", "1", "2"} {
		target, err := os.Readlink(filepath.Join(procRoot, strconv.Itoa(pid), "fd", fd))
		if err != nil {
			continue
		}
		if strings.HasPrefix(target, "/dev/pts/") || strings.HasPrefix(target, "/dev/tty") {
			return target
		}
	}
	return ""
}

func bootTime() (time.Time, error) {
	data, err := os.ReadFile(filepath.Join(procRoot, "stat"))
	if err != nil {
		return time.Time{}, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if value, ok := strings.CutPrefix(line, "btime "); ok {
			seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			return time.Unix(seconds, 0).UTC(), nil
		}
	}
	return time.Time{}, errors.New("agentsession: no btime in /proc/stat")
}

func processStartedAt(pid int) (time.Time, error) {
	start, err := processStartTime(pid)
	if err != nil {
		return time.Time{}, err
	}
	ticks, err := strconv.ParseInt(start, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	boot, err := bootTime()
	if err != nil {
		return time.Time{}, err
	}
	return boot.Add(time.Duration(ticks) * time.Second / clockTicksPerSecond), nil
}

// ttyActivity is the newest of the device's access and modification times:
// the kernel stamps them on terminal input and output, which is how `w`
// computes idle time.
func ttyActivity(tty string) (time.Time, error) {
	info, err := os.Stat(tty)
	if err != nil {
		return time.Time{}, err
	}
	at := info.ModTime()
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		if accessed := time.Unix(stat.Atim.Sec, stat.Atim.Nsec); accessed.After(at) {
			at = accessed
		}
	}
	return at.UTC(), nil
}

// desktopAppProcesses lists live processes, outside the agent slice, whose
// environment carries dataDir as VROOLI_DATA or APP_DATA_DIR. Processes of
// other users are unreadable and skipped: a desktop app runs as the operator.
func desktopAppProcesses(dataDir string) ([]int, error) {
	dataDir = cleanPath(dataDir)
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, err
	}
	self := os.Getpid()
	var out []int
	for _, entry := range entries {
		pid, convErr := strconv.Atoi(entry.Name())
		if convErr != nil || pid == self {
			continue
		}
		environ, readErr := os.ReadFile(filepath.Join(procRoot, entry.Name(), "environ"))
		if readErr != nil || len(environ) == 0 {
			continue
		}
		if !environHolds(environ, dataDir) {
			continue
		}
		cgroup, cgErr := os.ReadFile(filepath.Join(procRoot, entry.Name(), "cgroup"))
		if cgErr != nil || bytes.Contains(cgroup, []byte("/vrooli-agents.slice/")) {
			continue
		}
		out = append(out, pid)
	}
	return out, nil
}

func environHolds(environ []byte, dataDir string) bool {
	for _, entry := range bytes.Split(environ, []byte{0}) {
		key, value, ok := strings.Cut(string(entry), "=")
		if !ok || (key != envDesktopData && key != envAppDataDir) {
			continue
		}
		if cleanPath(value) == dataDir {
			return true
		}
	}
	return false
}

// processMemory reads VmRSS and VmSwap from /proc/<pid>/status. A process
// with no such lines (a zombie, a kernel thread) holds none; a status that
// cannot be read is an error, never a zero.
func processMemory(pid int) (int64, int64, error) {
	data, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "status"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH) {
			return 0, 0, ErrProcessGone
		}
		return 0, 0, err
	}
	var rss, swap int64
	for _, line := range strings.Split(string(data), "\n") {
		key, rest, ok := strings.Cut(line, ":")
		if !ok || (key != "VmRSS" && key != "VmSwap") {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) < 1 {
			continue
		}
		kib, convErr := strconv.ParseInt(fields[0], 10, 64)
		if convErr != nil {
			return 0, 0, fmt.Errorf("agentsession: %s of pid %d: %w", key, pid, convErr)
		}
		if key == "VmRSS" {
			rss = kib * 1024
		} else {
			swap = kib * 1024
		}
	}
	return rss, swap, nil
}
