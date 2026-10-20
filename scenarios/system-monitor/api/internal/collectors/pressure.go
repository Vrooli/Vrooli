package collectors

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// PressureCollector reads the kernel's low-cost pressure evidence. It does
// not scan processes or fork commands; unsupported hosts surface an explicit
// degraded value so recovery consumers never confuse unavailable with clear.
//
// Every platform reports through the paging seam (platform_paging_*.go); the
// collector owns counter-to-rate conversion through the shared rate tracker
// and derives the platform-neutral memory-pressure reading.
type PressureCollector struct {
	BaseCollector
	rates          *counterRateTracker
	memoryPressure memoryPressureConfig
	platform       string
	readFile       func(string) ([]byte, error)
	readPaging     func(map[string]uint64) pagingReading
	now            func() time.Time
}

func NewPressureCollector() *PressureCollector {
	return &PressureCollector{
		BaseCollector: NewBaseCollector("pressure", 10*time.Second),
		rates:         newCounterRateTracker(),
		platform:      runtime.GOOS,
		readFile:      os.ReadFile,
		readPaging:    readPlatformPaging,
		now:           time.Now,
	}
}

// SetMemoryPressureThresholds applies live level boundaries from settings.
// Non-positive values fall back to the documented defaults.
func (c *PressureCollector) SetMemoryPressureThresholds(thresholds MemoryPressureThresholds) {
	c.memoryPressure.set(thresholds)
}

func (c *PressureCollector) Collect(context.Context) (*MetricData, error) {
	now := c.now()
	values := map[string]interface{}{"available": false}
	signals := map[string]PressureSignal{}
	var paging pagingReading
	if c.platform == "linux" {
		c.collectLinuxCPUPSI(values, now)
		const memoryPSIProvenance = "linux /proc/pressure/memory"
		if raw, err := c.readFile("/proc/pressure/memory"); err == nil {
			values["available"] = true
			memory := parsePSI(string(raw))
			values["memory"] = memory
			// Flatten the primary PSI values as numeric time-series keys while
			// retaining the complete structured payload for current inspection.
			for class, metrics := range memory {
				for window, value := range metrics {
					values["memory_psi_"+class+"_"+window] = value
				}
			}
			for name, signal := range psiSignals(memory, memoryPSIProvenance) {
				signals[name] = signal
			}
		} else {
			values["degraded_reason"] = "memory PSI unavailable"
			if os.IsNotExist(err) {
				for name, signal := range psiUnavailableSignals(SignalUnsupported, "memory PSI is unavailable because the Linux PSI interface is absent", memoryPSIProvenance) {
					signals[name] = signal
				}
			} else {
				for name, signal := range psiUnavailableSignals(SignalFailed, "read /proc/pressure/memory: "+err.Error(), memoryPSIProvenance) {
					signals[name] = signal
				}
			}
		}
		if raw, err := c.readFile("/proc/vmstat"); err == nil {
			vmstat := parseVMStat(string(raw))
			paging = c.readPaging(vmstat)
			values["vmstat_status"] = "measured"
			values["vmstat_provenance"] = paging.provenance
			c.applyPaging(values, paging, now)
			if total, ok := vmstat["oom_kill"]; ok {
				values["oom_kill_count"] = total
			}
			if total, ok := vmstat["oom"]; ok {
				values["oom_count"] = total
			}
			if in, inOK := vmstat["pswpin"]; inOK {
				if out, outOK := vmstat["pswpout"]; outOK {
					inRate, inMeasured := c.rates.observe("swap_traffic_pswpin", in, now)
					outRate, outMeasured := c.rates.observe("swap_traffic_pswpout", out, now)
					if inMeasured && outMeasured {
						values["swap_traffic_pages_per_second"] = inRate + outRate
						values["swap_traffic_rate_status"] = "measured"
					} else {
						values["swap_traffic_rate_status"] = "not_yet_sampled"
					}
				}
			}
			for key, value := range collectFragmentation(c.rates, vmstat, now).payload() {
				values[key] = value
			}
		} else {
			values["vmstat_status"] = "failed"
			values["vmstat_reason"] = "read /proc/vmstat: " + err.Error()
			paging = pagingReading{provenance: "/proc/vmstat", reason: "read /proc/vmstat: " + err.Error()}
			c.applyPaging(values, paging, now)
		}
	} else {
		values["cpu_psi_status"] = "unsupported"
		values["cpu_psi_reason"] = "CPU pressure stall information is a Linux PSI interface"
		values["cpu_psi_provenance"] = "platform pressure backend"
		for name, signal := range psiUnavailableSignals(SignalUnsupported, "pressure stall information is a Linux PSI interface", c.platform+" memory-pressure backend") {
			signals[name] = signal
		}
		paging = c.readPaging(nil)
		values["degraded_reason"] = "PSI is only available on Linux"
		c.applyPaging(values, paging, now)
		values["fragmentation_status"] = "unsupported"
		values["fragmentation_reason"] = "buddy-allocator fragmentation is a Linux concept"
	}
	for name, signal := range pagingSignals(values, paging) {
		signals[name] = signal
	}
	reading := buildMemoryPressureReading(c.platform, signals, c.memoryPressure.get(), now)
	values["memory_pressure"] = reading
	values["memory_pressure_level"] = reading.Level
	if code, ok := memoryPressureLevelCode(reading.Level); ok {
		// Numeric projection for history. Unknown has no code so it is
		// absent from the series rather than plotted as a healthy zero.
		values["memory_pressure_level_code"] = code
	}
	return &MetricData{CollectorName: c.GetName(), Timestamp: now, Type: "pressure", Values: values}, nil
}

func memoryPressureLevelCode(level string) (float64, bool) {
	switch level {
	case MemoryPressureNormal:
		return 0, true
	case MemoryPressureElevated:
		return 1, true
	case MemoryPressureCritical:
		return 2, true
	default:
		return 0, false
	}
}

// applyPaging converts the seam's cumulative counters into rates and
// republishes them under the stable cross-platform aliases.
func (c *PressureCollector) applyPaging(values map[string]interface{}, paging pagingReading, now time.Time) {
	values["paging_provenance"] = paging.provenance
	if paging.supported {
		values["paging_status"] = "measured"
	} else {
		values["paging_status"] = "unsupported"
		values["paging_reason"] = paging.reason
	}
	for _, name := range paging.rateCounters {
		if total, present := paging.counters[name]; present {
			for key, value := range counterRateValues(c.rates, name, total, now) {
				values[key] = value
			}
			continue
		}
		values[name+"_rate_status"] = "unsupported"
		if paging.supported {
			values[name+"_rate_reason"] = "counter absent from " + paging.provenance
		} else {
			values[name+"_rate_reason"] = paging.reason
		}
	}
	for native, alias := range paging.aliases {
		key := alias + "_per_second"
		if rate, ok := values[native+"_per_second"].(float64); ok {
			values[key] = rate
		}
		if status, ok := values[native+"_rate_status"].(string); ok {
			values[key+"_status"] = status
			if reason, ok := values[native+"_rate_reason"].(string); ok {
				values[key+"_reason"] = reason
			}
		}
	}
	for alias, reason := range paging.unsupportedAliases {
		key := alias + "_per_second"
		if _, set := values[key+"_status"]; set {
			continue
		}
		values[key+"_status"] = "unsupported"
		values[key+"_reason"] = reason
	}
	// Every consumer alias carries a state, so an absent backend is never
	// read as "not yet sampled" or as a measured zero.
	for _, alias := range pagingAliases {
		key := alias + "_per_second"
		if _, set := values[key+"_status"]; set {
			continue
		}
		reason := paging.reason
		if reason == "" {
			reason = "counter is not provided by " + paging.provenance
		}
		values[key+"_status"] = "unsupported"
		values[key+"_reason"] = reason
	}
}

// pagingAliases are the stable cross-platform paging rate names.
var pagingAliases = []string{"swap_in", "swap_out", "major_faults", "page_faults", "page_ins", "page_reads", "compressions", "decompressions"}

func (c *PressureCollector) collectLinuxCPUPSI(values map[string]interface{}, now time.Time) {
	raw, err := c.readFile("/proc/pressure/cpu")
	if err != nil {
		values["cpu_available"] = false
		if os.IsNotExist(err) {
			values["cpu_psi_status"] = "unsupported"
			values["cpu_psi_reason"] = "CPU pressure stall information is unavailable because the Linux PSI interface is absent"
		} else {
			values["cpu_psi_status"] = "failed"
			values["cpu_psi_reason"] = "read /proc/pressure/cpu: " + err.Error()
		}
		values["cpu_psi_provenance"] = "linux /proc/pressure/cpu"
		return
	}
	values["cpu_available"] = true
	cpu := parsePSI(string(raw))
	values["cpu_psi"] = cpu
	// Keep state per PSI signal. A kernel may expose the interface while
	// omitting one of the stall classes, which must not be presented as a
	// measured zero.
	for _, class := range []string{"some", "full"} {
		if _, present := cpu[class]; !present {
			for _, window := range []string{"avg10", "avg60", "avg300"} {
				key := "cpu_psi_" + class + "_" + window
				values[key+"_status"] = "unsupported"
				values[key+"_reason"] = "CPU PSI class is absent from /proc/pressure/cpu"
			}
		}
	}
	for class, metrics := range cpu {
		for window, value := range metrics {
			values["cpu_psi_"+class+"_"+window] = value
			values["cpu_psi_"+class+"_"+window+"_status"] = "measured"
		}
	}
	values["cpu_psi_status"] = "measured"
	values["cpu_psi_provenance"] = "linux /proc/pressure/cpu"
	if total, ok := cpu["some"]["total"]; ok {
		for key, value := range counterRateValues(c.rates, "cpu_psi_some_total", uint64(total), now) {
			values["cpu_psi_some_total_"+key] = value
		}
	}
}

func parsePSI(raw string) map[string]map[string]float64 {
	out := map[string]map[string]float64{}
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		metrics := map[string]float64{}
		for _, field := range fields[1:] {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) != 2 {
				continue
			}
			if value, err := strconv.ParseFloat(parts[1], 64); err == nil {
				metrics[parts[0]] = value
			}
		}
		out[fields[0]] = metrics
	}
	return out
}

func parseVMStat(raw string) map[string]uint64 {
	values := make(map[string]uint64)
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err == nil {
			values[fields[0]] = value
		}
	}
	return values
}
