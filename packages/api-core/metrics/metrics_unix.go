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

// sampleRusage reads getrusage(RUSAGE_SELF). ru_maxrss is kilobytes on Linux but
// bytes on the BSDs/Darwin, so it is normalized to bytes by GOOS.
func sampleRusage() rusageSample {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return rusageSample{}
	}
	maxRSS := int64(ru.Maxrss)
	if runtime.GOOS == "linux" {
		maxRSS *= 1024
	}
	return rusageSample{
		cpuUserMs:   timevalMs(int64(ru.Utime.Sec), int64(ru.Utime.Usec)),
		cpuSysMs:    timevalMs(int64(ru.Stime.Sec), int64(ru.Stime.Usec)),
		maxRSSBytes: maxRSS,
		ok:          true,
	}
}

func timevalMs(sec, usec int64) int64 {
	return sec*1000 + usec/1000
}
