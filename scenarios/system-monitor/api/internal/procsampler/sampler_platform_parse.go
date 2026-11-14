package procsampler

import (
	"errors"
	"syscall"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/darwinsys"
)

// windowsProcessCounters is the raw GetProcessTimes/GetProcessMemoryInfo
// reading for one process (PROCESS_MEMORY_COUNTERS fields).
type windowsProcessCounters struct {
	UserTime100ns   uint64
	KernelTime100ns uint64
	WorkingSetSize  uint64
	PagefileUsage   uint64
	PageFaultCount  uint32
}

// applyWindowsMemoryCounters maps PROCESS_MEMORY_COUNTERS onto a sample.
// PageFaultCount counts hard and soft faults together, so its rate is an
// upper bound on paging rather than a major-fault rate. PagefileUsage is the
// process's private commit charge, the closest Windows analogue of swapped
// memory.
func applyWindowsMemoryCounters(sample *ProcessSample, counters windowsProcessCounters) {
	// Ticks are kept in the sampler's 1/100 s unit (clockTicksPerSec).
	sample.utime = counters.UserTime100ns / 100000
	sample.stime = counters.KernelTime100ns / 100000
	sample.RSSKB = int64(counters.WorkingSetSize / 1024)
	sample.SwapKB = int64(counters.PagefileUsage / 1024)
	sample.SwapStatus = "measured"
	sample.MajorFaults = uint64(counters.PageFaultCount)
	sample.FaultCounter = "windows PageFaultCount (hard and soft faults)"
	sample.FaultStatus = FaultNotYetSampled
	sample.FaultReason = ""
	sample.MetricsStatus = "measured"
	sample.MetricsReason = "Windows GetProcessTimes/GetProcessMemoryInfo"
}

// applyDarwinRusage maps proc_pid_rusage(RUSAGE_INFO_V0) onto a sample.
// ri_pageins counts pages the process read from disk (file-backed or swap),
// which is the macOS analogue of a major fault. Processes owned by another
// user return EPERM without privileges; their fault rate is unsupported, not
// zero. macOS keeps no per-process swap accounting.
func applyDarwinRusage(sample *ProcessSample, info darwinsys.RusageInfoV0, err error) {
	sample.FaultCounter = "macos proc_pid_rusage ri_pageins"
	sample.SwapStatus = "unsupported"
	if err != nil {
		sample.FaultStatus = FaultUnsupported
		if errors.Is(err, syscall.EPERM) {
			sample.FaultReason = "proc_pid_rusage denied for a process owned by another user"
		} else {
			sample.FaultReason = "proc_pid_rusage: " + err.Error()
		}
		return
	}
	sample.MajorFaults = info.PageIns
	sample.FaultStatus = FaultNotYetSampled
	if info.ResidentSize > 0 {
		sample.RSSKB = int64(info.ResidentSize / 1024)
	}
}
