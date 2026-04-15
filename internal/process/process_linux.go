//go:build linux

package process

import (
	"os"
	"strconv"
	"syscall"
)

func processIsAlive(process *os.Process) bool {
	return process.Signal(syscall.Signal(0)) == nil
}

func readProcessEnvironment(pid int) (map[string]string, error) {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/environ")
	if err != nil {
		return nil, err
	}
	return parseEnvironmentEntries(data), nil
}
