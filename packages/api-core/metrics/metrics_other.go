//go:build !unix && !windows

package metrics

// rusageSample mirrors the unix shape so the platform-agnostic collector code
// compiles unchanged. On non-unix platforms rusage is not sampled, so ok is
// always false and CPU/RSS are reported UNAVAILABLE rather than a misleading 0.
type rusageSample struct {
	cpuUserMs   int64
	cpuSysMs    int64
	maxRSSBytes int64
	ok          bool
}

func sampleRusage() rusageSample {
	return rusageSample{}
}
