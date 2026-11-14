package agentsession

import (
	"errors"
	"fmt"

	platformgo "github.com/vrooli/platform-go"
)

// Usage statuses. A figure that could not be read is never reported as 0.
const (
	// UsageMeasured is a sum over every member process.
	UsageMeasured = "measured"
	// UsagePartial is a sum over the members that could be read; some
	// exited or were unreadable mid-read, so the figure is a lower bound.
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
	// MemoryBytes is the sum of the members' resident set sizes (VmRSS on
	// Linux). Shared pages are counted once per process, so the sum can
	// exceed the scope's own memory charge.
	MemoryBytes  *int64
	MemoryStatus string
	// SwapBytes is the sum of the members' swapped-out memory (VmSwap).
	SwapBytes  *int64
	SwapStatus string
	// Detail says why a status is not measured.
	Detail string
	// Processes is how many members were read.
	Processes int
}

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
		status, detail = UsagePartial, fmt.Sprintf("%d of %d member process(es) unreadable; figures are a lower bound", unreadable, len(members))
	}
	return ScopeUsage{MemoryBytes: &rss, MemoryStatus: status, SwapBytes: &swap, SwapStatus: status, Detail: detail, Processes: read}
}

// orUnknown renders an optional figure for an audit row, where -1 means
// unknown (a 0 would claim the session held nothing).
func orUnknown(v *int64) int64 {
	if v == nil {
		return -1
	}
	return *v
}
