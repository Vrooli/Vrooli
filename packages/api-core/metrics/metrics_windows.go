//go:build windows

package metrics

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows exposes process CPU and working-set measurements through kernel32
// and psapi rather than getrusage. The memory value has the same semantics as
// unix ru_maxrss: it is a process-lifetime high-water mark.
type rusageSample struct {
	cpuUserMs   int64
	cpuSysMs    int64
	maxRSSBytes int64
	ok          bool
}

type processMemoryCountersEx struct {
	cb                         uint32
	pageFaultCount             uint32
	peakWorkingSetSize         uintptr
	workingSetSize             uintptr
	quotaPeakPagedPoolUsage    uintptr
	quotaPagedPoolUsage        uintptr
	quotaPeakNonPagedPoolUsage uintptr
	quotaNonPagedPoolUsage     uintptr
	pagefileUsage              uintptr
	peakPagefileUsage          uintptr
	privateUsage               uintptr
}

var getProcessMemoryInfo = windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo")

func sampleRusage() rusageSample {
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(windows.CurrentProcess(), &creation, &exit, &kernel, &user); err != nil {
		return rusageSample{}
	}
	counters := processMemoryCountersEx{cb: uint32(unsafe.Sizeof(processMemoryCountersEx{}))}
	ret, _, _ := getProcessMemoryInfo.Call(
		uintptr(windows.CurrentProcess()),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(unsafe.Sizeof(counters)),
	)
	if ret == 0 {
		return rusageSample{}
	}
	return rusageSample{
		cpuUserMs:   user.Nanoseconds() / int64(1e6),
		cpuSysMs:    kernel.Nanoseconds() / int64(1e6),
		maxRSSBytes: int64(counters.peakWorkingSetSize),
		ok:          true,
	}
}
