//go:build linux

package health

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// readHostMetrics reads memory and swap from /proc/meminfo, and memory stall
// time from /proc/pressure/memory when the kernel exposes PSI.
func readHostMetrics() hostMetrics {
	m := hostMetrics{PSISomeAvg60: -1}
	values, err := readMeminfo("/proc/meminfo")
	if err != nil {
		m.ReadError = "could not read /proc/meminfo: " + err.Error()
		m.SwapReadError = m.ReadError
		return m
	}
	total, hasTotal := values["MemTotal"]
	available, hasAvailable := values["MemAvailable"]
	if hasTotal && hasAvailable && total > 0 {
		m.MemoryKnown = true
		m.MemTotal = total
		m.MemAvailable = available
	} else {
		m.ReadError = "/proc/meminfo has no MemTotal/MemAvailable"
	}
	swapTotal, hasSwapTotal := values["SwapTotal"]
	swapFree, hasSwapFree := values["SwapFree"]
	if hasSwapTotal && hasSwapFree {
		m.SwapKnown = true
		m.SwapTotal = swapTotal
		if swapTotal > swapFree {
			m.SwapUsed = swapTotal - swapFree
		}
	} else {
		m.SwapReadError = "/proc/meminfo has no SwapTotal/SwapFree"
	}
	if avg, ok := readPSISomeAvg60("/proc/pressure/memory"); ok {
		m.PSISomeAvg60 = avg
	}
	return m
}

// readMeminfo returns /proc/meminfo values in bytes.
func readMeminfo(path string) (map[string]uint64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	values := map[string]uint64{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, rest, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		n, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		if len(fields) > 1 && fields[1] == "kB" {
			n *= 1024
		}
		values[key] = n
	}
	return values, scanner.Err()
}

// readPSISomeAvg60 parses "some avg10=… avg60=… avg300=… total=…".
func readPSISomeAvg60(path string) (float64, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "some ") {
			continue
		}
		for _, field := range strings.Fields(line) {
			if value, ok := strings.CutPrefix(field, "avg60="); ok {
				avg, err := strconv.ParseFloat(value, 64)
				return avg, err == nil
			}
		}
	}
	return 0, false
}
