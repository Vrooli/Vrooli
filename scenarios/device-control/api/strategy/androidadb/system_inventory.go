package androidadb

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

const (
	maxSystemProbeBytes  = 256 * 1024
	maxInstalledPackages = 2000
)

var packageLine = regexp.MustCompile(`^package:([A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)*)$`)

// InspectSystemInventory reads parsed system facts with fixed commands and
// hard output bounds. It never returns raw procfs, package-manager, or df text.
func (a *Adapter) InspectSystemInventory(ctx context.Context) (strategy.SystemInventory, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result := strategy.SystemInventory{Unavailable: map[string]string{}}
	probe := func(field string, args ...string) string {
		var out []byte
		var err error
		if bounded, ok := a.runner.(boundedRunner); ok {
			out, err = bounded.RunBounded(ctx, "adb", maxSystemProbeBytes, a.args(args...)...)
		} else {
			out, err = a.runner.Run(ctx, "adb", a.args(args...)...)
		}
		if err != nil || len(out) > maxSystemProbeBytes {
			result.Unavailable[field] = "bounded read-only Android probe failed"
			return ""
		}
		return string(out)
	}

	if cores := cpuCoreCount(probe("cpu_logical_cores", "shell", "cat", "/proc/cpuinfo")); cores > 0 {
		result.CPULogicalCores = uint32(cores)
	} else {
		result.Unavailable["cpu_logical_cores"] = "CPU core count was not readable"
	}
	if load1, load5, load15, ok := loadAverage(probe("cpu_load", "shell", "cat", "/proc/loadavg")); ok {
		result.Load1, result.Load5, result.Load15 = &load1, &load5, &load15
	} else {
		result.Unavailable["cpu_load"] = "CPU load averages were not readable"
	}
	mem := probe("memory", "shell", "cat", "/proc/meminfo")
	result.MemoryTotalBytes, result.MemoryAvailableBytes = memoryFacts(mem)
	if result.MemoryTotalBytes == nil {
		result.Unavailable["memory_total_bytes"] = "MemTotal was not readable"
	}
	if result.MemoryAvailableBytes == nil {
		result.Unavailable["memory_available_bytes"] = "MemAvailable was not readable"
	}
	result.DataTotalBytes, result.DataAvailableBytes = dataStorageFacts(probe("data_storage", "shell", "df", "-k", "/data"))
	if result.DataTotalBytes == nil || result.DataAvailableBytes == nil {
		result.Unavailable["data_storage"] = "data filesystem size was not readable"
	}
	packages := probe("installed_packages", "shell", "pm", "list", "packages")
	result.InstalledPackages, result.PackagesTruncated = parseInstalledPackages(packages)
	if packages == "" {
		result.Unavailable["installed_packages"] = "installed package inventory was not readable"
	}
	result.ObservedAt = time.Now().UTC()
	if len(result.Unavailable) == 0 {
		result.Unavailable = nil
	}
	if result.CPULogicalCores == 0 && result.MemoryTotalBytes == nil && result.DataTotalBytes == nil && len(result.InstalledPackages) == 0 {
		return result, fmt.Errorf("Android system inventory could not establish any requested facts")
	}
	return result, nil
}

func cpuCoreCount(output string) int {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "processor") && strings.Contains(line, ":") {
			count++
			if count == 256 {
				return count
			}
		}
	}
	return count
}

func loadAverage(output string) (float64, float64, float64, bool) {
	fields := strings.Fields(output)
	if len(fields) < 3 {
		return 0, 0, 0, false
	}
	values := [3]float64{}
	for i := range values {
		value, err := strconv.ParseFloat(fields[i], 64)
		if err != nil || value < 0 || value > 1e6 {
			return 0, 0, 0, false
		}
		values[i] = value
	}
	return values[0], values[1], values[2], true
}

func memoryFacts(output string) (*uint64, *uint64) {
	var total, available *uint64
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value := kibibytes(fields[1])
		switch strings.TrimSuffix(fields[0], ":") {
		case "MemTotal":
			total = value
		case "MemAvailable":
			available = value
		}
	}
	return total, available
}

func dataStorageFacts(output string) (*uint64, *uint64) {
	var total, available *uint64
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 6 {
			total, available = kibibytes(fields[1]), kibibytes(fields[3])
		}
	}
	return total, available
}

func parseInstalledPackages(output string) ([]string, bool) {
	packages := make([]string, 0, 64)
	seen := map[string]bool{}
	truncated := false
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		match := packageLine.FindStringSubmatch(line)
		if len(match) != 3 || len(match[1]) > 255 || seen[match[1]] {
			continue
		}
		if len(packages) == maxInstalledPackages {
			truncated = true
			break
		}
		seen[match[1]] = true
		packages = append(packages, match[1])
	}
	return packages, truncated
}
