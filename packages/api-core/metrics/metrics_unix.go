//go:build unix

package metrics

import "syscall"

// rusageSample is a normalized snapshot of process resource usage. CPU times are
// in milliseconds; peak RSS is in bytes regardless of the platform's native
// getrusage units. ok is false when the platform cannot sample rusage.
type rusageSample struct {
	cpuUserMs   int64
	cpuSysMs    int64
	maxRSSBytes int64
	ok          bool
}

// sampleRusage reads both the provider process and its reaped children.
//
// CPU: the child counters are cumulative, just like self counters, so the
// collector's window delta includes subprocesses such as go test and scanners.
//
// Peak RSS: taken as the MAXIMUM of the provider's own high-water mark and the
// largest reaped child's. This used to read self only, which made the figure
// meaningless for the phases that matter: a provider that shells out to go test
// or a security scanner is a thin wrapper, and reporting the wrapper's memory
// understated the phase by an order of magnitude — the unit phase claimed 48 MB.
// Capacity admission sizes its RAM reservation from that number, so it
// under-reserved for the two heaviest phases and over-committed the host.
//
// MAX rather than a sum, deliberately: ru_maxrss for RUSAGE_CHILDREN is already
// the peak over all reaped children, not their total, and parent and children do
// not necessarily peak at the same moment. The max is the honest answer to "how
// large did the biggest process in this tree get"; summing would invent a
// concurrent peak that may never have existed.
//
// ru_maxrss is kilobytes on Linux but bytes on the BSDs/Darwin, so it is
// normalized to bytes by GOOS.
func sampleRusage() rusageSample {
	var self, children syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &self); err != nil {
		return rusageSample{}
	}
	if err := syscall.Getrusage(syscall.RUSAGE_CHILDREN, &children); err != nil {
		return rusageSample{}
	}
	maxRSS := rssNativeBytes(int64(self.Maxrss))
	if childRSS := rssNativeBytes(int64(children.Maxrss)); childRSS > maxRSS {
		maxRSS = childRSS
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
