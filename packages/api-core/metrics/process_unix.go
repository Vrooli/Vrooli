//go:build unix

package metrics

import (
	"os"
	"runtime"
	"syscall"
)

// processPeakRSSBytes converts a waited child process's ru_maxrss to bytes.
// Linux reports kilobytes; BSD and Darwin report bytes.
func processPeakRSSBytes(state *os.ProcessState) (int64, bool) {
	if state == nil {
		return 0, false
	}
	rusage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok || rusage == nil {
		return 0, false
	}
	peak := rssNativeBytes(int64(rusage.Maxrss))
	return peak, peak > 0
}

// rssNativeBytes normalizes the platform-specific ru_maxrss unit exactly once
// for all Unix RSS measurements.
func rssNativeBytes(native int64) int64 {
	peak := native
	if runtime.GOOS == "linux" {
		peak *= 1024
	}
	return peak
}
