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
		samples = append(samples, ProcessSample{
			PID:     int(entry.ProcessID),
			PPID:    int(entry.ParentProcessID),
			Comm:    windows.UTF16ToString(entry.ExeFile[:]),
			Threads: int(entry.Threads),
		})
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
