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
type PressureCollector struct {
	BaseCollector
	rates *counterRateTracker
}

func NewPressureCollector() *PressureCollector {
	return &PressureCollector{BaseCollector: NewBaseCollector("pressure", 10*time.Second), rates: newCounterRateTracker()}
}

func (c *PressureCollector) Collect(context.Context) (*MetricData, error) {
	values := map[string]interface{}{"available": false}
	if runtime.GOOS == "linux" {
		if raw, err := os.ReadFile("/proc/pressure/memory"); err == nil {
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
		} else {
			values["degraded_reason"] = "memory PSI unavailable"
		}
		if raw, err := os.ReadFile("/proc/vmstat"); err == nil {
			now := time.Now()
			vmstat := parseVMStat(string(raw))
			paging := readPlatformPaging(vmstat)
			values["vmstat_status"] = "measured"
			values["vmstat_provenance"] = paging.provenance
			if !paging.supported {
				values["paging_status"] = "unsupported"
				values["paging_reason"] = paging.reason
			} else {
				values["paging_status"] = "measured"
			}
			for _, name := range []string{"pswpin", "pswpout", "pgmajfault", "pgfault", "workingset_refault_file", "workingset_refault_anon", "allocstall_movable", "allocstall_normal", "oom_kill"} {
				if total, present := paging.counters[name]; present {
					for key, value := range counterRateValues(c.rates, name, total, now) {
						values[key] = value
					}
				} else {
					values[name+"_rate_status"] = "unsupported"
					values[name+"_rate_reason"] = "counter absent from /proc/vmstat"
				}
			}
			// Stable cross-platform aliases. Linux exposes the source counter
			// names above; Darwin/Windows backends map their native counters to
			// these same consumer-facing series when available.
			for source, alias := range map[string]string{
				"pswpin_per_second":     "swap_in_per_second",
				"pswpout_per_second":    "swap_out_per_second",
				"pgmajfault_per_second": "major_faults_per_second",
				"pgfault_per_second":    "page_faults_per_second",
			} {
				if rate, ok := values[source].(float64); ok {
					values[alias] = rate
				}
				if status, ok := values[strings.TrimSuffix(source, "_per_second")+"_rate_status"].(string); ok {
					values[alias+"_status"] = status
				}
			}
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
		}
	} else {
		paging := readPlatformPaging(nil)
		values["degraded_reason"] = "PSI is only available on Linux"
		values["paging_status"] = map[bool]string{true: "measured", false: "unsupported"}[paging.supported]
		values["paging_reason"] = paging.reason
		values["paging_provenance"] = paging.provenance
		values["fragmentation_status"] = "unsupported"
		values["fragmentation_reason"] = "buddy-allocator fragmentation is a Linux concept"
	}
	return &MetricData{CollectorName: c.GetName(), Timestamp: time.Now(), Type: "pressure", Values: values}, nil
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
