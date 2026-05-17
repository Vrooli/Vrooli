//go:build windows

package hwprobe

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// readWindowsRAM uses wmic. PowerShell's Get-CimInstance would be more
// modern but adds startup cost; wmic is universally present on supported
// Windows SKUs.
func readSystemRAM() (uint64, error) {
	out, err := exec.Command("wmic", "ComputerSystem", "get", "TotalPhysicalMemory", "/Value").Output()
	if err != nil {
		return 0, fmt.Errorf("hwprobe: wmic: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "TotalPhysicalMemory=") {
			continue
		}
		v, err := strconv.ParseUint(strings.TrimPrefix(line, "TotalPhysicalMemory="), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("hwprobe: parse wmic output: %w", err)
		}
		return v, nil
	}
	return 0, fmt.Errorf("hwprobe: TotalPhysicalMemory not in wmic output")
}
