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
		handle, openErr := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
		if openErr == nil {
			var creation, exit, kernel, user windows.Filetime
			timeErr := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user)
			counters := processMemoryCounters{CB: uint32(unsafe.Sizeof(processMemoryCounters{}))}
			ret, _, _ := getProcessMemoryInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&counters)), uintptr(unsafe.Sizeof(counters)))
			_ = windows.CloseHandle(handle)
			if timeErr == nil && ret != 0 {
				sample.utime = uint64(user.Nanoseconds() / 1e7)
				sample.stime = uint64(kernel.Nanoseconds() / 1e7)
				sample.RSSKB = int64(counters.WorkingSetSize / 1024)
				sample.SwapKB = int64(counters.PagefileUsage / 1024)
				sample.MajorFaults = uint64(counters.PageFaultCount)
				sample.MetricsStatus = "measured"
				sample.MetricsReason = "Windows GetProcessTimes/GetProcessMemoryInfo"
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
