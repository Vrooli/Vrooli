//go:build !(aix || android || darwin || dragonfly || freebsd || hurd || illumos || linux || netbsd || openbsd || solaris)

package census

func deviceCoverage(_ string, measured int64, complete bool) ScanCoverage {
	return ScanCoverage{MeasuredBytes: measured, Complete: complete}
}
