//go:build darwin

package hwprobe

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// readDarwinRAM invokes `sysctl -n hw.memsize`.
func readSystemRAM() (uint64, error) {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, fmt.Errorf("hwprobe: sysctl hw.memsize: %w", err)
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("hwprobe: parse hw.memsize: %w", err)
	}
	return v, nil
}
