package validation

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

const EnvScannerMaxProcs = "SECURITY_HEALTH_SCANNER_MAX_PROCS"

func defaultScannerMaxProcs(cpu int) int {
	if cpu < 1 {
		cpu = 1
	}
	value := cpu / 4
	if value < 2 {
		value = 2
	}
	if value > cpu {
		value = cpu
	}
	return value
}

func resolveScannerMaxProcs(raw string, cpu int) int {
	defaultValue := defaultScannerMaxProcs(cpu)
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 1 || value > maxInt(cpu, 1) {
		return defaultValue
	}
	return value
}

func scannerMaxProcs() int {
	return resolveScannerMaxProcs(os.Getenv(EnvScannerMaxProcs), runtime.NumCPU())
}

func scannerEnvironment() map[string]string {
	return map[string]string{"GOMAXPROCS": strconv.Itoa(scannerMaxProcs())}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
