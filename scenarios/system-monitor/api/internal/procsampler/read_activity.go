package procsampler

import (
	"context"
	"sort"
	"time"
)

// ReadActivity reports rates for processes observed at both ends of one short
// on-demand window. It cannot observe a process that exits between samples.
type ReadActivity struct {
	Status           string            `json:"status"`
	Reason           string            `json:"reason,omitempty"`
	Platform         string            `json:"platform"`
	WindowSeconds    float64           `json:"window_seconds,omitempty"`
	FirstObservedAt  *time.Time        `json:"first_observed_at,omitempty"`
	LastObservedAt   *time.Time        `json:"last_observed_at,omitempty"`
	FirstEnumerated  int               `json:"first_enumerated,omitempty"`
	LastEnumerated   int               `json:"last_enumerated,omitempty"`
	Unreadable       int               `json:"unreadable_observations,omitempty"`
	PermissionDenied int               `json:"permission_denied_observations,omitempty"`
	Disappeared      int               `json:"disappeared_observations,omitempty"`
	Unmatched        int               `json:"unmatched,omitempty"`
	Truncated        bool              `json:"truncated,omitempty"`
	Matched          int               `json:"matched,omitempty"`
	Active           int               `json:"active,omitempty"`
	RowsTruncated    bool              `json:"rows_truncated,omitempty"`
	Rows             []ReadActivityRow `json:"rows,omitempty"`
	StorageRows      []ReadActivityRow `json:"storage_rows,omitempty"`
	Semantics        string            `json:"semantics"`
}

type ReadActivityRow struct {
	PID                    int     `json:"pid"`
	Name                   string  `json:"name"`
	Owner                  string  `json:"owner"`
	ReturnedBytesPerSecond float64 `json:"returned_bytes_per_second"`
	StorageBytesPerSecond  float64 `json:"storage_bytes_per_second"`
	IntervalSeconds        float64 `json:"interval_seconds"`
	ObservedChildExits     int     `json:"observed_child_exits,omitempty"`
}

type readActivityPoint struct {
	process            ProcessSample
	requested, storage uint64
	observedAt         time.Time
}

type readActivitySnapshot struct {
	points                        map[int]readActivityPoint
	enumerated, unreadable        int
	permissionDenied, disappeared int
	truncated                     bool
	observedAt                    time.Time
}

// summarizeReadActivity rejects PID reuse, resets and non-positive intervals.
// It reports bounded top rows, while retaining coverage counts for the caller.
func summarizeReadActivity(first, last readActivitySnapshot, top int) ReadActivity {
	result := ReadActivity{Status: "measured", Platform: "linux", FirstObservedAt: &first.observedAt, LastObservedAt: &last.observedAt,
		WindowSeconds: last.observedAt.Sub(first.observedAt).Seconds(), FirstEnumerated: first.enumerated, LastEnumerated: last.enumerated,
		Unreadable: first.unreadable + last.unreadable, PermissionDenied: first.permissionDenied + last.permissionDenied, Disappeared: first.disappeared + last.disappeared, Truncated: first.truncated || last.truncated,
		Semantics: "Rows rank bytes returned from successful read-like calls, including cache hits; storage_rows rank bytes fetched from the storage layer (accurate for block-backed filesystems). Linux counters include waited-for children and may overlap across processes; a reaped child's lifetime I/O can appear as a brief spike. Observed child exits are incomplete and do not prove reaping. Owner attribution uses ancestry and location heuristics. Processes absent at either sample are not ranked; these rates do not identify kernel slab allocations."}
	var samples []ProcessSample
	for _, point := range last.points {
		samples = append(samples, point.process)
	}
	NewAttributor(nil).Attribute(context.Background(), samples)
	owners := make(map[int]string, len(samples))
	for _, sample := range samples {
		owners[sample.PID] = sample.Owner
	}
	childExits := make(map[int]int)
	for pid, previous := range first.points {
		current, exists := last.points[pid]
		if !exists || current.process.StartTimeTicks != previous.process.StartTimeTicks {
			parentBefore, hadParent := first.points[previous.process.PPID]
			parentAfter, hasParent := last.points[previous.process.PPID]
			if hadParent && hasParent && parentBefore.process.StartTimeTicks != 0 && parentBefore.process.StartTimeTicks == parentAfter.process.StartTimeTicks {
				childExits[previous.process.PPID]++
			}
		}
	}
	for pid, current := range last.points {
		previous, ok := first.points[pid]
		seconds := current.observedAt.Sub(previous.observedAt).Seconds()
		if !ok || current.process.StartTimeTicks == 0 || current.process.StartTimeTicks != previous.process.StartTimeTicks || seconds <= 0 || current.requested < previous.requested || current.storage < previous.storage {
			result.Unmatched++
			continue
		}
		result.Matched++
		if current.requested == previous.requested && current.storage == previous.storage {
			continue
		}
		result.Rows = append(result.Rows, ReadActivityRow{PID: pid, Name: current.process.Comm, Owner: owners[pid], ReturnedBytesPerSecond: float64(current.requested-previous.requested) / seconds, StorageBytesPerSecond: float64(current.storage-previous.storage) / seconds, IntervalSeconds: seconds, ObservedChildExits: childExits[pid]})
	}
	for pid := range first.points {
		if _, ok := last.points[pid]; !ok {
			result.Unmatched++
		}
	}
	sort.Slice(result.Rows, func(i, j int) bool {
		if result.Rows[i].ReturnedBytesPerSecond == result.Rows[j].ReturnedBytesPerSecond {
			return result.Rows[i].PID < result.Rows[j].PID
		}
		return result.Rows[i].ReturnedBytesPerSecond > result.Rows[j].ReturnedBytesPerSecond
	})
	result.Active = len(result.Rows)
	result.StorageRows = append([]ReadActivityRow(nil), result.Rows...)
	sort.Slice(result.StorageRows, func(i, j int) bool {
		if result.StorageRows[i].StorageBytesPerSecond == result.StorageRows[j].StorageBytesPerSecond {
			return result.StorageRows[i].PID < result.StorageRows[j].PID
		}
		return result.StorageRows[i].StorageBytesPerSecond > result.StorageRows[j].StorageBytesPerSecond
	})
	if top >= 0 && len(result.Rows) > top {
		result.RowsTruncated = true
		result.Rows = result.Rows[:top]
		result.StorageRows = result.StorageRows[:top]
	}
	if result.Matched == 0 && result.Unreadable > 0 {
		result.Status, result.Reason = "failed", "no comparable process counters were available; inspect permission and disappearance counts"
	} else if result.Matched == 0 {
		result.Status, result.Reason = "not_yet_sampled", "no process was matched across both observations"
	} else if result.Unreadable > 0 || result.Unmatched > 0 || result.Truncated {
		result.Status, result.Reason = "partial", "some processes were inaccessible, changed identity, exited, or exceeded the bound"
	}
	return result
}
