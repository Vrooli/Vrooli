//go:build linux

package securestore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/shell"
)

const (
	keyringdaemonLinuxParameterA = 19
)

var (
	keyringDaemonStartTime = readKeyringDaemonStartTime
	procClockTicks         = readProcClockTicks
)

func readKeyringDaemonStartTime() (time.Time, bool) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return time.Time{}, false
	}
	boot, err := procBootTime()
	if err != nil {
		return time.Time{}, false
	}
	clockTicks, ok := procClockTicks()
	if !ok {
		return time.Time{}, false
	}
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		comm, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "comm"))
		if err != nil || !isKeyringDaemonComm(strings.TrimSpace(string(comm))) {
			continue
		}
		stat, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue
		}
		closeParen := strings.LastIndex(string(stat), ")")
		if closeParen < 0 {
			continue
		}
		fields := strings.Fields(string(stat)[closeParen+1:])
		if len(fields) <= keyringdaemonLinuxParameterA {
			continue
		}
		ticks, err := strconv.ParseInt(fields[19], 10, 64)
		if err != nil {
			continue
		}
		return boot.Add(time.Duration(float64(ticks) * float64(time.Second) / clockTicks)), true
	}
	return time.Time{}, false
}

// keyringDaemonComm is the process name to match. procCommLimit is the kernel's
// TASK_COMM_LEN minus its NUL terminator: /proc/<pid>/comm is truncated to that
// many bytes, so "gnome-keyring-daemon" appears there as "gnome-keyring-d".
//
// Comparing the full name against that truncated value never matched, which
// silently disabled the stale-daemon check on every Linux host — it reported
// "not-run" rather than a wrong answer, so nothing ever failed loudly.
const (
	keyringDaemonComm = "gnome-keyring-daemon"
	procCommLimit     = 15
)

func isKeyringDaemonComm(comm string) bool {
	if comm == keyringDaemonComm {
		return true
	}
	// Only accept the truncated form at exactly the kernel's limit. A shorter
	// prefix would match an unrelated "gnome-keyring-d" helper.
	return len(comm) == procCommLimit && strings.HasPrefix(keyringDaemonComm, comm)
}

func readProcClockTicks() (float64, bool) {
	output, err := shell.NewCommand("getconf", "CLK_TCK").Output()
	if err != nil {
		return 0, false
	}
	ticks, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	return ticks, err == nil && ticks > 0
}

func procBootTime() (time.Time, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return time.Time{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[0] == "btime" {
			seconds, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			return time.Unix(seconds, 0), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return time.Time{}, err
	}
	return time.Time{}, fmt.Errorf("/proc/stat has no btime")
}

func addStaleDaemonReport(report *KeyringReport, info os.FileInfo) {
	start, ok := keyringDaemonStartTime()
	if !ok {
		report.StaleDaemonCheck = "not-run"
		return
	}
	report.StaleDaemonCheck = "checked"
	if !info.ModTime().After(start) {
		return
	}
	report.StaleDaemon = true
	report.StaleDaemonDetail = fmt.Sprintf("keyring file is newer than the running gnome-keyring-daemon (file %s; daemon %s); log out and back in so the daemon reloads the repaired file", info.ModTime().Format(time.RFC3339), start.Format(time.RFC3339))
}
