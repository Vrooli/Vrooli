//go:build linux

package hwprobe

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readLinuxRAM parses MemTotal from /proc/meminfo. The kB-suffix is
// canonical even though kernels report KiB.
func readSystemRAM() (uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, fmt.Errorf("hwprobe: open /proc/meminfo: %w", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("hwprobe: parse MemTotal: %w", err)
		}
		return kb * 1024, nil
	}
	return 0, fmt.Errorf("hwprobe: MemTotal not found in /proc/meminfo")
}
