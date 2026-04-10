//go:build !linux

package process

import "os"

func processIsAlive(process *os.Process) bool {
	return false
}

func readProcessEnvironment(pid int) (map[string]string, error) {
	return map[string]string{}, nil
}
