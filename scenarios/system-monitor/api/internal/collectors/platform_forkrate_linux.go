//go:build linux

package collectors

import (
	"os"
	"strconv"
	"strings"
)

// readForkRate reads the cumulative process-creation counter from /proc/stat.
// This is a single file read with no subprocess, which matters: a collector that
// forked to measure the fork rate would pollute the very signal it reports.
func readForkRate() forkRateReading {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return forkRateUnsupported("read /proc/stat: " + err.Error())
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[0] != "processes" {
			continue
		}
		total, parseErr := strconv.ParseUint(fields[1], 10, 64)
		if parseErr != nil {
			return forkRateUnsupported("parse /proc/stat processes counter: " + parseErr.Error())
		}
		return forkRateReading{total: total, supported: true, provenance: "linux /proc/stat"}
	}
	return forkRateUnsupported("/proc/stat exposes no processes counter")
}
