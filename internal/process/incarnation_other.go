//go:build !linux && !darwin && !windows

package process

import "fmt"

func processIncarnation(pid int) (string, error) {
	return "", fmt.Errorf("stable process incarnation unavailable on this platform (pid %d)", pid)
}

func incarnationFormatSample() string { return "unavailable" }
