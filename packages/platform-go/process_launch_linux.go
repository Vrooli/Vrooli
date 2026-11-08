//go:build linux

package platform

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

func processArguments(pid int) ([]string, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("platform: invalid pid %d", pid)
	}
	file, err := os.Open("/proc/" + strconv.Itoa(pid) + "/cmdline")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxProcessArgumentBytes+1))
	if err != nil {
		return nil, err
	}
	return parseNULProcessArguments(data)
}

func openProcessExecutable(pid int) (*os.File, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("platform: invalid pid %d", pid)
	}
	return os.Open("/proc/" + strconv.Itoa(pid) + "/exe")
}
