//go:build linux

package collectors

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func collectFragmentation(tracker *counterRateTracker, vmstat map[string]uint64, now time.Time) fragmentationReading {
	r := fragmentationReading{status: "measured", provenance: "/proc/buddyinfo"}
	r.values = make(map[string]interface{})
	raw, err := os.ReadFile("/proc/buddyinfo")
	if err != nil {
		r.status = "failed"
		r.reason = err.Error()
		return r
	}
	parsed := parseBuddyInfo(string(raw))
	for key, value := range parsed.values {
		r.values[key] = value
	}
	return addCompactionRates(r, tracker, vmstat, now)
}

type buddyInfoResult struct {
	values   map[string]interface{}
	minOrder int
}

func parseBuddyInfo(raw string) buddyInfoResult {
	values := make(map[string]interface{})
	histogram := make(map[string]string)
	minOrder := -1
	var totalBytes, lowOrderBytes uint64
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "Node" {
			continue
		}
		counts := make([]uint64, 0, len(fields)-4)
		for _, field := range fields[4:] {
			count, parseErr := strconv.ParseUint(field, 10, 64)
			if parseErr != nil {
				counts = nil
				break
			}
			counts = append(counts, count)
		}
		if len(counts) == 0 {
			continue
		}
		maxOrder := -1
		for order, count := range counts {
			if count > 0 {
				maxOrder = order
			}
			bytes := count * (uint64(1) << order) * 4096
			totalBytes += bytes
			if order < 4 {
				lowOrderBytes += bytes
			}
		}
		// Zones with no free blocks at any order contribute no evidence to the
		// whole-host index; do not let their sentinel -1 erase a measured zone.
		if maxOrder >= 0 && (minOrder < 0 || maxOrder < minOrder) {
			minOrder = maxOrder
		}
		zone := fields[3]
		values["buddyinfo_"+zone] = counts
		histogram[zone] = fmt.Sprint(counts)
	}
	if len(histogram) > 0 {
		values["buddyinfo"] = histogram
	}
	if minOrder >= 0 {
		values["fragmentation_max_free_order"] = minOrder
		if totalBytes > 0 {
			values["fragmentation_low_order_share"] = float64(lowOrderBytes) / float64(totalBytes)
		}
	}
	return buddyInfoResult{values: values, minOrder: minOrder}
}

func addCompactionRates(r fragmentationReading, tracker *counterRateTracker, vmstat map[string]uint64, now time.Time) fragmentationReading {
	for _, name := range []string{"compact_stall", "compact_fail", "compact_success", "thp_fault_fallback", "thp_fault_alloc"} {
		if total, ok := vmstat[name]; ok {
			for key, value := range counterRateValues(tracker, name, total, now) {
				r.values[key] = value
			}
		} else {
			r.values[name+"_rate_status"] = "unsupported"
			r.values[name+"_rate_reason"] = "counter absent from /proc/vmstat"
		}
	}
	if fail, failOK := vmstat["compact_fail"]; failOK {
		if success, successOK := vmstat["compact_success"]; successOK {
			failRate, failMeasured := tracker.observe("compact_fail_ratio_fail", fail, now)
			successRate, successMeasured := tracker.observe("compact_fail_ratio_success", success, now)
			if failMeasured && successMeasured && failRate+successRate > 0 {
				r.values["compaction_failure_ratio"] = failRate / (failRate + successRate)
				r.values["compaction_failure_ratio_status"] = "measured"
			} else {
				r.values["compaction_failure_ratio_status"] = "not_yet_sampled"
			}
		}
	}
	return r
}
