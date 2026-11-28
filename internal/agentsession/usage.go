package agentsession

import (
	"errors"
	"fmt"
	"strings"

	platformgo "github.com/vrooli/platform-go"
)

// Usage statuses. A figure that could not be read is never reported as 0.
const (
	// UsageMeasured is a sum over every member process.
	UsageMeasured = "measured"
	// UsagePartial is a process sum with one or more unreadable members; it
	// is incomplete and must not be treated as a cgroup total.
	UsagePartial = "partial"
	// UsageUnsupported is a platform with no per-process reader.
	UsageUnsupported = "unsupported"
	// UsageUnavailable is a session whose processes could not be listed
	// (scope not held, scope unreadable, or no member readable).
	UsageUnavailable = "unavailable"
)

// ScopeUsage is what a session's processes hold. MemoryBytes and SwapBytes
// are nil unless their status is measured or partial.
type ScopeUsage struct {
	// MemoryBytes is cgroup charged RAM when available, including file cache
	// and some kernel charges; the fallback is a per-process VmRSS sum that
	// can double-count shared pages.
	MemoryBytes  *int64
	MemoryStatus string
	MemorySource string
	// SwapBytes is cgroup swap charge when available; the fallback is a sum
	// of member VmSwap readings.
	SwapBytes  *int64
	SwapStatus string
	SwapSource string
	// Detail says why a status is not measured.
	Detail string
	// Processes is how many members were read.
	Processes int
}

const (
	UsageSourceCgroupV2 = "cgroup-v2"
	UsageSourceProcSum  = "proc-sum"
)

// unknownUsage is a usage nothing could be read for.
func unknownUsage(status, detail string) ScopeUsage {
	return ScopeUsage{MemoryStatus: status, SwapStatus: status, Detail: detail}
}

// MeasureUsage sums the resident and swapped memory of a session's member
// processes. It is the one implementation behind `vrooli agent list`'s
// figures and the reap plan's Memory/Swap columns and audit rows.
func MeasureUsage(host HostFacts, members []int, membersErr error) ScopeUsage {
	switch {
	case host == nil:
		return unknownUsage(UsageUnavailable, "no host reader")
	case errors.Is(membersErr, platformgo.ErrUnsupported):
		return unknownUsage(UsageUnsupported, "this platform cannot list a session's processes")
	case membersErr != nil:
		return unknownUsage(UsageUnavailable, "session processes unreadable: "+membersErr.Error())
	case len(members) == 0:
		return unknownUsage(UsageUnavailable, "the session's scope is not held or holds no readable process")
	}
	var rss, swap int64
	read, unreadable := 0, 0
	for _, pid := range members {
		pRSS, pSwap, err := host.ProcessMemory(pid)
		if errors.Is(err, platformgo.ErrUnsupported) {
			return unknownUsage(UsageUnsupported, "this platform has no per-process memory reader")
		}
		if err != nil {
			unreadable++
			continue
		}
		rss += pRSS
		swap += pSwap
		read++
	}
	if read == 0 {
		return unknownUsage(UsageUnavailable, fmt.Sprintf("none of %d member process(es) could be read", len(members)))
	}
	status, detail := UsageMeasured, ""
	if unreadable > 0 {
		status, detail = UsagePartial, fmt.Sprintf("process sum incomplete: %d of %d member process(es) unreadable", unreadable, len(members))
	}
	return ScopeUsage{MemoryBytes: &rss, MemoryStatus: status, MemorySource: UsageSourceProcSum, SwapBytes: &swap, SwapStatus: status, SwapSource: UsageSourceProcSum, Detail: detail, Processes: read}
}

// MeasureScopeUsage prefers cgroup v2 charged RAM (including file cache and
// some kernel charges) and swap. It falls back independently to process
// VmRSS/VmSwap sums when a cgroup counter is unavailable; those sums can
// double-count shared pages and are labeled accordingly.
func MeasureScopeUsage(host HostFacts, scopeUnit string, members []int, membersErr error) ScopeUsage {
	usage := MeasureUsage(host, members, membersErr)
	if host == nil || scopeUnit == "" {
		return usage
	}
	occupancy, err := host.ScopeOccupancy(scopeUnit)
	if err != nil {
		appendUsageDetail(&usage, "cgroup occupancy unavailable: "+err.Error())
		return usage
	}
	if occupancy.MemoryBytes >= 0 {
		value := occupancy.MemoryBytes
		usage.MemoryBytes = &value
		usage.MemoryStatus = UsageMeasured
		usage.MemorySource = UsageSourceCgroupV2
	} else if usage.MemoryBytes == nil {
		usage.MemoryStatus = UsageUnavailable
		appendUsageDetail(&usage, "cgroup memory.current is unavailable")
	}
	if occupancy.SwapBytes >= 0 {
		value := occupancy.SwapBytes
		usage.SwapBytes = &value
		usage.SwapStatus = UsageMeasured
		usage.SwapSource = UsageSourceCgroupV2
	} else if usage.SwapBytes == nil {
		usage.SwapStatus = UsageUnavailable
		appendUsageDetail(&usage, "cgroup memory.swap.current is unavailable")
	}
	return usage
}

func appendUsageDetail(usage *ScopeUsage, detail string) {
	if usage.Detail == "" {
		usage.Detail = detail
		return
	}
	usage.Detail = strings.TrimSpace(usage.Detail + "; " + detail)
}

// orUnknown renders an optional figure for an audit row, where -1 means
// unknown (a 0 would claim the session held nothing).
func orUnknown(v *int64) int64 {
	if v == nil {
		return -1
	}
	return *v
}
