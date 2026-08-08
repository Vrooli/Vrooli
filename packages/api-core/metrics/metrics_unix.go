//go:build unix

package metrics

import (
	"runtime"
	"syscall"
)

// rusageSample is a normalized snapshot of process resource usage. CPU times are
// in milliseconds; peak RSS is in bytes regardless of the platform's native
// getrusage units. ok is false when the platform cannot sample rusage.
type rusageSample struct {
	cpuUserMs   int64
	cpuSysMs    int64
	maxRSSBytes int64
	ok          bool
}

// sampleRusage reads both the provider process and its reaped children. The
// child counters are cumulative, just like self counters, so the collector's
// window delta includes subprocesses such as go test and scanners. ru_maxrss
// remains the provider process lifetime high-water mark: it is kilobytes on
// Linux but bytes on the BSDs/Darwin, so it is normalized to bytes by GOOS.
func sampleRusage() rusageSample {
	var self, children syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &self); err != nil {
		return rusageSample{}
	}
	if err := syscall.Getrusage(syscall.RUSAGE_CHILDREN, &children); err != nil {
		return rusageSample{}
	}
	maxRSS := int64(self.Maxrss)
	if runtime.GOOS == "linux" {
		maxRSS *= 1024
	}
	return rusageSample{
		cpuUserMs:   timevalMs(int64(self.Utime.Sec), int64(self.Utime.Usec)) + timevalMs(int64(children.Utime.Sec), int64(children.Utime.Usec)),
		cpuSysMs:    timevalMs(int64(self.Stime.Sec), int64(self.Stime.Usec)) + timevalMs(int64(children.Stime.Sec), int64(children.Stime.Usec)),
		maxRSSBytes: maxRSS,
		ok:          true,
	}
}

func timevalMs(sec, usec int64) int64 {
	return sec*1000 + usec/1000
}
