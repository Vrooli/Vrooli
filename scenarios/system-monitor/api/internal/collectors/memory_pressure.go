package collectors

// DOC: docs/concepts/ARCHITECTURE.md#memory-pressure

import (
	"fmt"
	"sync"
	"time"
)

// Memory-pressure levels. Unknown means no supported signal could produce a
// verdict; it is never a synonym for normal.
const (
	MemoryPressureNormal   = "normal"
	MemoryPressureElevated = "elevated"
	MemoryPressureCritical = "critical"
	MemoryPressureUnknown  = "unknown"
)

// Signal states mirror models.MetricState so the projection is lossless.
const (
	SignalMeasured      = "measured"
	SignalUnsupported   = "unsupported"
	SignalFailed        = "failed"
	SignalNotYetSampled = "not_yet_sampled"
)

// Stable memory-pressure signal names. Every reading carries every name; a
// platform that cannot measure one reports it unsupported with a reason.
const (
	SignalPSISomeAvg10           = "psi_some_avg10"
	SignalPSISomeAvg60           = "psi_some_avg60"
	SignalPSISomeAvg300          = "psi_some_avg300"
	SignalPSISomeTotal           = "psi_some_total_us"
	SignalPSIFullAvg10           = "psi_full_avg10"
	SignalPSIFullAvg60           = "psi_full_avg60"
	SignalPSIFullAvg300          = "psi_full_avg300"
	SignalPSIFullTotal           = "psi_full_total_us"
	SignalSwapInBytesPerSecond   = "swap_in_bytes_per_second"
	SignalSwapOutBytesPerSecond  = "swap_out_bytes_per_second"
	SignalMajorFaultsPerSecond   = "major_faults_per_second"
	SignalPageInsPerSecond       = "page_ins_per_second"
	SignalPageReadsPerSecond     = "page_reads_per_second"
	SignalCompressionsPerSecond  = "compressions_per_second"
	SignalDecompressionsPerSec   = "decompressions_per_second"
	SignalPagingFileUsagePercent = "paging_file_usage_percent"
	SignalCommitChargePercent    = "commit_charge_percent"
	SignalNativePressureLevel    = "native_pressure_level"
)

// MemoryPressureSignalNames is the complete signal contract, in display order.
var MemoryPressureSignalNames = []string{
	SignalPSISomeAvg10, SignalPSISomeAvg60, SignalPSISomeAvg300, SignalPSISomeTotal,
	SignalPSIFullAvg10, SignalPSIFullAvg60, SignalPSIFullAvg300, SignalPSIFullTotal,
	SignalSwapInBytesPerSecond, SignalSwapOutBytesPerSecond, SignalMajorFaultsPerSecond,
	SignalPageInsPerSecond, SignalPageReadsPerSecond, SignalCompressionsPerSecond,
	SignalDecompressionsPerSec, SignalPagingFileUsagePercent, SignalCommitChargePercent,
	SignalNativePressureLevel,
}

// PressureSignal is one raw memory-pressure observation with honest state.
type PressureSignal struct {
	Value      float64 `json:"value"`
	Status     string  `json:"status"`
	Reason     string  `json:"reason,omitempty"`
	Provenance string  `json:"provenance"`
	Units      string  `json:"units,omitempty"`
}

// Measured reports whether Value is a real observation.
func (s PressureSignal) Measured() bool { return s.Status == SignalMeasured }

func measuredSignal(value float64, provenance, units string) PressureSignal {
	return PressureSignal{Value: value, Status: SignalMeasured, Provenance: provenance, Units: units}
}

func unsupportedSignal(reason, provenance, units string) PressureSignal {
	return PressureSignal{Status: SignalUnsupported, Reason: reason, Provenance: provenance, Units: units}
}

// MemoryPressureThresholds are the Linux level boundaries (and the swap-in
// boundaries that may raise, never lower, a macOS native level).
type MemoryPressureThresholds struct {
	SomeAvg60ElevatedPercent     float64 `json:"some_avg60_elevated_percent"`
	FullAvg60CriticalPercent     float64 `json:"full_avg60_critical_percent"`
	SwapInElevatedBytesPerSecond float64 `json:"swap_in_elevated_bytes_per_second"`
	SwapInCriticalBytesPerSecond float64 `json:"swap_in_critical_bytes_per_second"`
}

const mebibyte = 1024 * 1024

// DefaultMemoryPressureThresholds returns the documented defaults: elevated
// at PSI some avg60 >= 10% or swap-in >= 10 MiB/s; critical at PSI full
// avg60 >= 5% or swap-in >= 50 MiB/s.
func DefaultMemoryPressureThresholds() MemoryPressureThresholds {
	return MemoryPressureThresholds{
		SomeAvg60ElevatedPercent:     10,
		FullAvg60CriticalPercent:     5,
		SwapInElevatedBytesPerSecond: 10 * mebibyte,
		SwapInCriticalBytesPerSecond: 50 * mebibyte,
	}
}

// withDefaults fills non-positive boundaries with the defaults.
func (t MemoryPressureThresholds) withDefaults() MemoryPressureThresholds {
	d := DefaultMemoryPressureThresholds()
	if t.SomeAvg60ElevatedPercent <= 0 {
		t.SomeAvg60ElevatedPercent = d.SomeAvg60ElevatedPercent
	}
	if t.FullAvg60CriticalPercent <= 0 {
		t.FullAvg60CriticalPercent = d.FullAvg60CriticalPercent
	}
	if t.SwapInElevatedBytesPerSecond <= 0 {
		t.SwapInElevatedBytesPerSecond = d.SwapInElevatedBytesPerSecond
	}
	if t.SwapInCriticalBytesPerSecond <= 0 {
		t.SwapInCriticalBytesPerSecond = d.SwapInCriticalBytesPerSecond
	}
	return t
}

// MemoryPressureReading is the platform-neutral verdict plus raw signals.
type MemoryPressureReading struct {
	Level       string                    `json:"level"`
	LevelSource string                    `json:"level_source"`
	LevelReason string                    `json:"level_reason"`
	Platform    string                    `json:"platform"`
	Signals     map[string]PressureSignal `json:"signals"`
	ObservedAt  time.Time                 `json:"observed_at"`
}

// LevelRank orders levels for comparison. Unknown ranks below normal so it
// never satisfies an "at least elevated" condition.
func LevelRank(level string) int {
	switch level {
	case MemoryPressureCritical:
		return 3
	case MemoryPressureElevated:
		return 2
	case MemoryPressureNormal:
		return 1
	default:
		return 0
	}
}

// completeSignals ensures every contract signal is present, filling gaps
// with an explicit unsupported state for this platform.
func completeSignals(platform string, signals map[string]PressureSignal) map[string]PressureSignal {
	if signals == nil {
		signals = make(map[string]PressureSignal, len(MemoryPressureSignalNames))
	}
	for _, name := range MemoryPressureSignalNames {
		if _, ok := signals[name]; !ok {
			signals[name] = unsupportedSignal(fmt.Sprintf("%s is not measured on %s", name, platform), platform+" memory-pressure backend", signalUnits(name))
		}
	}
	return signals
}

func signalUnits(name string) string {
	switch name {
	case SignalPSISomeAvg10, SignalPSISomeAvg60, SignalPSISomeAvg300, SignalPSIFullAvg10, SignalPSIFullAvg60, SignalPSIFullAvg300,
		SignalPagingFileUsagePercent, SignalCommitChargePercent:
		return "percent"
	case SignalPSISomeTotal, SignalPSIFullTotal:
		return "microseconds"
	case SignalSwapInBytesPerSecond, SignalSwapOutBytesPerSecond:
		return "bytes per second"
	case SignalNativePressureLevel:
		return "native level"
	default:
		return "per second"
	}
}

// DeriveMemoryPressureLevel maps platform signals to a level.
//
//   - linux: PSI is the primary evidence. critical when full avg60 >= the
//     critical boundary or swap-in >= its critical rate; elevated when some
//     avg60 >= the elevated boundary or swap-in >= its elevated rate. With no
//     PSI, swap-in alone decides; with neither, the level is unknown.
//   - darwin: kern.memorystatus_vm_pressure_level is authoritative (1 normal,
//     2 warn -> elevated, 4 critical). The swap-in boundaries may raise it,
//     never lower it. Without the native level, swap-in alone decides.
//   - windows: the LowMemoryResourceNotification state is the only native
//     verdict (signaled -> critical, otherwise normal). Windows exposes no
//     native elevated band, and Pages Input/sec is not used because it counts
//     memory-mapped file reads as well as pagefile reads.
func DeriveMemoryPressureLevel(platform string, signals map[string]PressureSignal, thresholds MemoryPressureThresholds) (level, source, reason string) {
	th := thresholds.withDefaults()
	swapLevel, swapReason, swapMeasured := swapInLevel(signals[SignalSwapInBytesPerSecond], th)
	switch platform {
	case "linux":
		some, full := signals[SignalPSISomeAvg60], signals[SignalPSIFullAvg60]
		if !some.Measured() && !full.Measured() {
			if !swapMeasured {
				return MemoryPressureUnknown, "linux psi + swap-in rate", "neither PSI nor the swap-in rate is available"
			}
			return swapLevel, "linux swap-in rate (PSI unavailable)", swapReason
		}
		level, reason = MemoryPressureNormal, "PSI and swap-in rate are below the elevated boundaries"
		if full.Measured() && full.Value >= th.FullAvg60CriticalPercent {
			return MemoryPressureCritical, "linux psi + swap-in rate", fmt.Sprintf("memory PSI full avg60 %.2f%% >= %.2f%%", full.Value, th.FullAvg60CriticalPercent)
		}
		if swapLevel == MemoryPressureCritical {
			return MemoryPressureCritical, "linux psi + swap-in rate", swapReason
		}
		if some.Measured() && some.Value >= th.SomeAvg60ElevatedPercent {
			return MemoryPressureElevated, "linux psi + swap-in rate", fmt.Sprintf("memory PSI some avg60 %.2f%% >= %.2f%%", some.Value, th.SomeAvg60ElevatedPercent)
		}
		if swapLevel == MemoryPressureElevated {
			return MemoryPressureElevated, "linux psi + swap-in rate", swapReason
		}
		return level, "linux psi + swap-in rate", reason
	case "darwin":
		native := signals[SignalNativePressureLevel]
		if !native.Measured() {
			if !swapMeasured {
				return MemoryPressureUnknown, "macos kern.memorystatus_vm_pressure_level", "native pressure level and swap-in rate are unavailable"
			}
			return swapLevel, "macos swap-in rate (native level unavailable)", swapReason
		}
		switch int(native.Value) {
		case 1:
			level, reason = MemoryPressureNormal, "kern.memorystatus_vm_pressure_level=1 (normal)"
		case 2:
			level, reason = MemoryPressureElevated, "kern.memorystatus_vm_pressure_level=2 (warn)"
		case 4:
			level, reason = MemoryPressureCritical, "kern.memorystatus_vm_pressure_level=4 (critical)"
		default:
			return MemoryPressureUnknown, "macos kern.memorystatus_vm_pressure_level", fmt.Sprintf("unrecognized native pressure level %d", int(native.Value))
		}
		if swapMeasured && LevelRank(swapLevel) > LevelRank(level) {
			return swapLevel, "macos native level raised by swap-in rate", swapReason
		}
		return level, "macos kern.memorystatus_vm_pressure_level", reason
	case "windows":
		native := signals[SignalNativePressureLevel]
		if !native.Measured() {
			return MemoryPressureUnknown, "windows LowMemoryResourceNotification", "low-memory resource notification is unavailable"
		}
		if native.Value >= 1 {
			return MemoryPressureCritical, "windows LowMemoryResourceNotification", "the low-memory resource notification is signaled"
		}
		return MemoryPressureNormal, "windows LowMemoryResourceNotification", "the low-memory resource notification is not signaled"
	default:
		return MemoryPressureUnknown, platform + " memory-pressure backend", "memory pressure has no backend on " + platform
	}
}

func swapInLevel(signal PressureSignal, th MemoryPressureThresholds) (level, reason string, measured bool) {
	if !signal.Measured() {
		return MemoryPressureUnknown, "swap-in rate unavailable", false
	}
	mib := signal.Value / mebibyte
	switch {
	case signal.Value >= th.SwapInCriticalBytesPerSecond:
		return MemoryPressureCritical, fmt.Sprintf("swap-in %.1f MiB/s >= %.1f MiB/s", mib, th.SwapInCriticalBytesPerSecond/mebibyte), true
	case signal.Value >= th.SwapInElevatedBytesPerSecond:
		return MemoryPressureElevated, fmt.Sprintf("swap-in %.1f MiB/s >= %.1f MiB/s", mib, th.SwapInElevatedBytesPerSecond/mebibyte), true
	default:
		return MemoryPressureNormal, fmt.Sprintf("swap-in %.1f MiB/s is below %.1f MiB/s", mib, th.SwapInElevatedBytesPerSecond/mebibyte), true
	}
}

// buildMemoryPressureReading completes the signal set and derives the level.
func buildMemoryPressureReading(platform string, signals map[string]PressureSignal, thresholds MemoryPressureThresholds, now time.Time) MemoryPressureReading {
	signals = completeSignals(platform, signals)
	level, source, reason := DeriveMemoryPressureLevel(platform, signals, thresholds)
	return MemoryPressureReading{Level: level, LevelSource: source, LevelReason: reason, Platform: platform, Signals: signals, ObservedAt: now}
}

// psiSignals converts a parsed /proc/pressure/memory payload. A missing
// stall class is unsupported, never zero.
func psiSignals(psi map[string]map[string]float64, provenance string) map[string]PressureSignal {
	out := make(map[string]PressureSignal, 8)
	for _, class := range []string{"some", "full"} {
		metrics, present := psi[class]
		for _, window := range []string{"avg10", "avg60", "avg300"} {
			name := "psi_" + class + "_" + window
			if value, ok := metrics[window]; present && ok {
				out[name] = measuredSignal(value, provenance, "percent")
			} else {
				out[name] = unsupportedSignal("memory PSI "+class+" "+window+" is absent from /proc/pressure/memory", provenance, "percent")
			}
		}
		name := "psi_" + class + "_total_us"
		if value, ok := metrics["total"]; present && ok {
			out[name] = measuredSignal(value, provenance, "microseconds")
		} else {
			out[name] = unsupportedSignal("memory PSI "+class+" total is absent from /proc/pressure/memory", provenance, "microseconds")
		}
	}
	return out
}

func psiUnavailableSignals(status, reason, provenance string) map[string]PressureSignal {
	out := make(map[string]PressureSignal, 8)
	for _, name := range []string{SignalPSISomeAvg10, SignalPSISomeAvg60, SignalPSISomeAvg300, SignalPSISomeTotal, SignalPSIFullAvg10, SignalPSIFullAvg60, SignalPSIFullAvg300, SignalPSIFullTotal} {
		out[name] = PressureSignal{Status: status, Reason: reason, Provenance: provenance, Units: signalUnits(name)}
	}
	return out
}

// pagingSignals projects the paging seam's rates into pressure signals. The
// rate keys come from the pressure collector's alias pass so every platform
// shares first-sample semantics.
func pagingSignals(values map[string]interface{}, paging pagingReading) map[string]PressureSignal {
	out := make(map[string]PressureSignal, 8)
	pageSize := float64(paging.pageSize)
	for alias, signal := range map[string]struct {
		name  string
		bytes bool
	}{
		"swap_in":        {SignalSwapInBytesPerSecond, true},
		"swap_out":       {SignalSwapOutBytesPerSecond, true},
		"major_faults":   {SignalMajorFaultsPerSecond, false},
		"page_ins":       {SignalPageInsPerSecond, false},
		"page_reads":     {SignalPageReadsPerSecond, false},
		"compressions":   {SignalCompressionsPerSecond, false},
		"decompressions": {SignalDecompressionsPerSec, false},
	} {
		key := alias + "_per_second"
		units := signalUnits(signal.name)
		status, _ := values[key+"_status"].(string)
		reason, _ := values[key+"_reason"].(string)
		provenance := paging.provenance
		if rate, ok := values[key].(float64); ok && status == SignalMeasured {
			if signal.bytes {
				if pageSize <= 0 {
					out[signal.name] = unsupportedSignal("page size is unknown, so page counts cannot be converted to bytes", provenance, units)
					continue
				}
				rate *= pageSize
			}
			out[signal.name] = measuredSignal(rate, provenance, units)
			continue
		}
		switch status {
		case SignalNotYetSampled:
			out[signal.name] = PressureSignal{Status: SignalNotYetSampled, Reason: "rate needs a second counter sample", Provenance: provenance, Units: units}
		case "":
			// The alias was never produced: the platform has no such counter.
		default:
			out[signal.name] = PressureSignal{Status: status, Reason: reason, Provenance: provenance, Units: units}
		}
	}
	for name, gauge := range paging.gauges {
		out[name] = gauge
	}
	return out
}

// memoryPressureConfig holds the live thresholds pushed from settings.
type memoryPressureConfig struct {
	mu         sync.RWMutex
	thresholds MemoryPressureThresholds
}

func (c *memoryPressureConfig) get() MemoryPressureThresholds {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.thresholds.withDefaults()
}

func (c *memoryPressureConfig) set(t MemoryPressureThresholds) {
	c.mu.Lock()
	c.thresholds = t
	c.mu.Unlock()
}
