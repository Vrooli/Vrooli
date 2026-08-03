//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package orchestration

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

func gracefulTerminateProcess(process *os.Process) bool {
	return process.Signal(syscall.SIGTERM) == nil
}

func processGroupID(pid int) int {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 5 {
		return 0
	}
	pgid, err := strconv.Atoi(fields[4])
	if err != nil {
		return 0
	}
	return pgid
}

func killProcessGroupID(pgid int) bool {
	return syscall.Kill(-pgid, syscall.SIGKILL) == nil
}
