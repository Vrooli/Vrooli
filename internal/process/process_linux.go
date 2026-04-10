//go:build linux

package process

import (
	"os"
	"strconv"
	"strings"
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

	values := make(map[string]string)
	for _, entry := range strings.Split(string(data), "\x00") {
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		values[parts[0]] = parts[1]
	}
	return values, nil
}
