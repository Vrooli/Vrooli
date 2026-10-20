//go:build windows

package procsampler

import (
	"context"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsSampler struct {
	delta *cpuDeltaTracker
	now   func() time.Time
}

type processMemoryCounters struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

var getProcessMemoryInfo = windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo")

func NewSampler() Sampler {
	return &windowsSampler{delta: newCPUDeltaTracker(), now: time.Now}
}

func (s *windowsSampler) Sample(ctx context.Context) ([]ProcessSample, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)

	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(handle, &entry); err != nil {
		return nil, err
	}
	samples := make([]ProcessSample, 0)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sample := ProcessSample{
			PID:           int(entry.ProcessID),
			PPID:          int(entry.ParentProcessID),
			Comm:          windows.UTF16ToString(entry.ExeFile[:]),
			Threads:       int(entry.Threads),
			MetricsStatus: "unsupported",
			MetricsReason: "GetProcessMemoryInfo/GetProcessTimes unavailable",
		}
		sample.FaultCounter = "windows PageFaultCount (hard and soft faults)"
		sample.FaultStatus = FaultUnsupported
		sample.FaultReason = "OpenProcess/GetProcessMemoryInfo was refused (protected or exited process)"
		sample.SwapStatus = "unsupported"
		// proc is the per-process handle; the snapshot handle above must stay
		// untouched because Process32Next walks it.
		proc, openErr := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
		if openErr == nil {
			var creation, exit, kernel, user windows.Filetime
			timeErr := windows.GetProcessTimes(proc, &creation, &exit, &kernel, &user)
			counters := processMemoryCounters{CB: uint32(unsafe.Sizeof(processMemoryCounters{}))}
			ret, _, _ := getProcessMemoryInfo.Call(uintptr(proc), uintptr(unsafe.Pointer(&counters)), uintptr(unsafe.Sizeof(counters)))
			_ = windows.CloseHandle(proc)
			if timeErr == nil && ret != 0 {
				applyWindowsMemoryCounters(&sample, windowsProcessCounters{
					UserTime100ns:   uint64(user.Nanoseconds() / 100),
					KernelTime100ns: uint64(kernel.Nanoseconds() / 100),
					WorkingSetSize:  uint64(counters.WorkingSetSize),
					PagefileUsage:   uint64(counters.PagefileUsage),
					PageFaultCount:  counters.PageFaultCount,
				})
			}
		}
		samples = append(samples, sample)
		if err := windows.Process32Next(handle, &entry); err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				break
			}
			return nil, err
		}
	}
	s.delta.apply(samples, s.now())
	sortByCPUDesc(samples)
	return samples, nil
}
