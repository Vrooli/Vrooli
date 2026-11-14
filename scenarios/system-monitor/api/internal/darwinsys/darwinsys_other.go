//go:build !darwin

package darwinsys

// HostStatistics64 is unavailable off macOS.
func HostStatistics64() (VMStatistics64, error) { return VMStatistics64{}, ErrUnsupported }

// ProcPIDRusage is unavailable off macOS.
func ProcPIDRusage(int) (RusageInfoV0, error) { return RusageInfoV0{}, ErrUnsupported }
