package collectors

import (
	"fmt"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/darwinsys"
)

const (
	darwinVMProvenance       = "macos host_statistics64(HOST_VM_INFO64)"
	darwinPressureProvenance = "macos sysctl kern.memorystatus_vm_pressure_level"
)

// darwinPagingReading maps host_statistics64 counters and the native pressure
// level onto the paging seam. swapins/swapouts count pages moved between the
// compressor and the swap files; pageins counts pages read from disk for any
// backing (file or swap). macOS keeps no host-wide major-fault counter.
func darwinPagingReading(stats darwinsys.VMStatistics64, statsErr error, nativeLevel uint32, levelErr error, pageSize uint64) pagingReading {
	gauges := map[string]PressureSignal{
		SignalPagingFileUsagePercent: unsupportedSignal("paging-file usage is a Windows counter; macOS swap usage is reported by vm.swapusage", darwinVMProvenance, "percent"),
		SignalCommitChargePercent:    unsupportedSignal("commit charge is a Windows counter", darwinVMProvenance, "percent"),
	}
	if levelErr != nil {
		gauges[SignalNativePressureLevel] = PressureSignal{Status: SignalFailed, Reason: "read kern.memorystatus_vm_pressure_level: " + levelErr.Error(), Provenance: darwinPressureProvenance, Units: "native level"}
	} else {
		gauges[SignalNativePressureLevel] = measuredSignal(float64(nativeLevel), darwinPressureProvenance, "native level")
	}
	reading := pagingReading{
		rateCounters: []string{"swapins", "swapouts", "pageins", "pageouts", "faults", "compressions", "decompressions"},
		aliases:      map[string]string{"swapins": "swap_in", "swapouts": "swap_out", "pageins": "page_ins", "faults": "page_faults", "compressions": "compressions", "decompressions": "decompressions"},
		unsupportedAliases: map[string]string{
			"major_faults": "macOS keeps no host-wide major-fault counter; pageins counts pages read from disk",
			"page_reads":   "page read operations are a Windows counter",
		},
		gauges:     gauges,
		pageSize:   pageSize,
		provenance: darwinVMProvenance,
	}
	if statsErr != nil {
		reading.reason = fmt.Sprintf("host_statistics64: %v", statsErr)
		return reading
	}
	reading.supported = true
	reading.counters = map[string]uint64{
		"swapins":        stats.Swapins,
		"swapouts":       stats.Swapouts,
		"pageins":        stats.Pageins,
		"pageouts":       stats.Pageouts,
		"faults":         stats.Faults,
		"compressions":   stats.Compressions,
		"decompressions": stats.Decompressions,
	}
	return reading
}
