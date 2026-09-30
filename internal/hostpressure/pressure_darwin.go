//go:build darwin

package hostpressure

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// macOS does not expose Linux PSI or a host-wide fork counter. Those fields
// remain explicitly unread. The remaining values use native sysctl state and
// kern.proc.all, so the watchdog never shells out to platform commands.
func collect(_ context.Context, opts Options) PressureSnapshot {
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	s := PressureSnapshot{CapturedAt: now()}
	s.CPUPressure = NewUnread("darwin:cpu-pressure", "macOS has no PSI equivalent")
	s.ForkRate = NewUnread("darwin:fork-rate", "fork counter is intentionally unsupported on darwin")
	s.ForkCounter = NewUnread("darwin:fork-counter", "fork counter is intentionally unsupported on darwin")
	if raw, err := unix.SysctlRaw("vm.loadavg"); err == nil && len(raw) >= 16 {
		fscale := int32(binary.LittleEndian.Uint32(raw[12:16]))
		if fscale > 0 {
			s.Load1 = NewRead(float64(int32(binary.LittleEndian.Uint32(raw[0:4])))/float64(fscale), "darwin sysctl vm.loadavg")
		}
	}
	if s.Load1.State != Read {
		s.Load1 = NewUnread("darwin sysctl vm.loadavg", "load average is unavailable")
	}
	if total, err := unix.SysctlUint64("hw.memsize"); err == nil {
		s.MemoryTotal = NewRead(float64(total), "darwin sysctl hw.memsize")
	} else {
		s.MemoryTotal = NewUnread("darwin sysctl hw.memsize", err.Error())
	}
	s.MemoryAvail = NewUnread("darwin host_statistics64", "host_statistics64 available-memory binding is not exposed by this build")
	if raw, err := unix.SysctlRaw("vm.swapusage"); err == nil && len(raw) >= 24 {
		total := binary.LittleEndian.Uint64(raw[0:8])
		free := binary.LittleEndian.Uint64(raw[16:24])
		s.SwapTotal = NewRead(float64(total), "darwin sysctl vm.swapusage")
		s.SwapUsed = NewRead(float64(total-free), "darwin sysctl vm.swapusage")
	} else {
		reason := "vm.swapusage is unavailable"
		if err != nil {
			reason = fmt.Sprintf("vm.swapusage: %v", err)
		}
		s.SwapTotal = NewUnread("darwin sysctl vm.swapusage", reason)
		s.SwapUsed = NewUnread("darwin sysctl vm.swapusage", reason)
	}
	if processes, err := unix.SysctlKinfoProcSlice("kern.proc.all"); err == nil {
		s.ProcessCount = NewRead(float64(len(processes)), "darwin sysctl kern.proc.all")
	} else {
		s.ProcessCount = NewUnread("darwin sysctl kern.proc.all", err.Error())
	}
	return s
}
