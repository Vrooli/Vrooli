//go:build windows

package hostpressure

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type (
	windowsFiletime       struct{ LowDateTime, HighDateTime uint32 }
	windowsMemoryStatusEx struct {
		Length                                             uint32
		MemoryLoad                                         uint32
		TotalPhys, AvailPhys, TotalPageFile, AvailPageFile uint64
		TotalVirtual, AvailVirtual, AvailExtendedVirtual   uint64
	}
)

var (
	windowsGetSystemTimes       = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetSystemTimes")
	windowsGlobalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")
)

func collect(ctx context.Context, opts Options) PressureSnapshot {
	s := PressureSnapshot{CapturedAt: time.Now()}
	if opts.Now != nil {
		s.CapturedAt = opts.Now()
	}
	s.CPUPressure = NewUnread("windows:cpu-pressure", "Windows has no PSI equivalent")
	s.Load1 = NewUnread("windows:load", "load average is not a Windows kernel metric")
	s.ForkRate = NewUnread("windows:fork-rate", "fork counter is intentionally unsupported on windows")
	s.ForkCounter = NewUnread("windows:fork-counter", "fork counter is intentionally unsupported on windows")
	var memory windowsMemoryStatusEx
	memory.Length = uint32(unsafe.Sizeof(memory))
	if ret, _, callErr := windowsGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memory))); ret != 0 {
		s.MemoryTotal = NewRead(float64(memory.TotalPhys), "Windows GlobalMemoryStatusEx")
		s.MemoryAvail = NewRead(float64(memory.AvailPhys), "Windows GlobalMemoryStatusEx")
		s.SwapTotal = NewRead(float64(memory.TotalPageFile), "Windows GlobalMemoryStatusEx")
		s.SwapUsed = NewRead(float64(memory.TotalPageFile-memory.AvailPageFile), "Windows GlobalMemoryStatusEx")
	} else {
		reason := fmt.Sprintf("GlobalMemoryStatusEx: %v", callErr)
		s.MemoryTotal = NewUnread("windows:GlobalMemoryStatusEx", reason)
		s.MemoryAvail = NewUnread("windows:GlobalMemoryStatusEx", reason)
		s.SwapTotal = NewUnread("windows:GlobalMemoryStatusEx", reason)
		s.SwapUsed = NewUnread("windows:GlobalMemoryStatusEx", reason)
	}
	count, err := windowsProcessCount(ctx)
	if err == nil {
		s.ProcessCount = NewRead(float64(count), "Windows Toolhelp32Snapshot")
	} else {
		s.ProcessCount = NewUnread("windows Toolhelp32Snapshot", err.Error())
	}
	return s
}

func windowsProcessCount(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(handle)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(handle, &entry); err != nil {
		return 0, err
	}
	count := 0
	for {
		count++
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if err := windows.Process32Next(handle, &entry); err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				return count, nil
			}
			return 0, err
		}
	}
}
