package procsampler

import (
	"testing"
	"time"
)

func TestReadActivityRatesRejectReusedPIDsAndCounterResets(t *testing.T) {
	base := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	point := func(pid int, start, requested, storage uint64, at time.Time) readActivityPoint {
		return readActivityPoint{process: ProcessSample{PID: pid, Comm: "worker", StartTimeTicks: start}, requested: requested, storage: storage, observedAt: at}
	}
	first := readActivitySnapshot{observedAt: base, enumerated: 3, points: map[int]readActivityPoint{1: point(1, 1, 100, 10, base), 2: point(2, 2, 500, 20, base), 3: point(3, 3, 500, 20, base)}}
	last := readActivitySnapshot{observedAt: base.Add(2 * time.Second), enumerated: 3, points: map[int]readActivityPoint{1: point(1, 1, 300, 50, base.Add(2*time.Second)), 2: point(2, 9, 900, 30, base.Add(2*time.Second)), 3: point(3, 3, 10, 2, base.Add(2*time.Second))}}
	got := summarizeReadActivity(first, last, 20)
	if got.Status != "partial" || got.Matched != 1 || got.Unmatched != 2 || len(got.Rows) != 1 {
		t.Fatalf("coverage=%+v", got)
	}
	if row := got.Rows[0]; row.PID != 1 || row.ReturnedBytesPerSecond != 100 || row.StorageBytesPerSecond != 20 || row.Owner != OwnerUnknown {
		t.Fatalf("row=%+v", row)
	}
}

func TestReadActivityZeroDoesNotInventAttribution(t *testing.T) {
	got := summarizeReadActivity(readActivitySnapshot{points: map[int]readActivityPoint{}}, readActivitySnapshot{points: map[int]readActivityPoint{}}, 20)
	if got.Status != "not_yet_sampled" || len(got.Rows) != 0 || got.Reason == "" {
		t.Fatalf("result=%+v", got)
	}
}

func TestReadActivityFailureWhenNoReadableCounters(t *testing.T) {
	got := summarizeReadActivity(readActivitySnapshot{points: map[int]readActivityPoint{}, unreadable: 1, permissionDenied: 1}, readActivitySnapshot{points: map[int]readActivityPoint{}}, 20)
	if got.Status != "failed" || got.PermissionDenied != 1 || got.Reason == "" {
		t.Fatalf("result=%+v", got)
	}
}

func TestReadActivityRanksStorageSeparatelyAndReportsTruncation(t *testing.T) {
	start := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	mk := func(pid int, read, storage uint64, when time.Time) readActivityPoint {
		return readActivityPoint{process: ProcessSample{PID: pid, StartTimeTicks: uint64(pid)}, requested: read, storage: storage, observedAt: when}
	}
	first := readActivitySnapshot{observedAt: start, points: map[int]readActivityPoint{1: mk(1, 0, 0, start), 2: mk(2, 0, 0, start)}}
	last := readActivitySnapshot{observedAt: start.Add(time.Second), points: map[int]readActivityPoint{1: mk(1, 100, 1, start.Add(time.Second)), 2: mk(2, 50, 30, start.Add(time.Second))}}
	got := summarizeReadActivity(first, last, 1)
	if !got.RowsTruncated || got.Active != 2 || len(got.Rows) != 1 || len(got.StorageRows) != 1 || got.Rows[0].PID != 1 || got.StorageRows[0].PID != 2 {
		t.Fatalf("rankings = %+v", got)
	}
}

func TestReadActivityFlagsObservedChildExitWithoutClaimingCause(t *testing.T) {
	start := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	parent := ProcessSample{PID: 1, StartTimeTicks: 5}
	child := ProcessSample{PID: 2, PPID: 1, StartTimeTicks: 6}
	first := readActivitySnapshot{observedAt: start, points: map[int]readActivityPoint{1: {process: parent, observedAt: start}, 2: {process: child, observedAt: start}}}
	last := readActivitySnapshot{observedAt: start.Add(time.Second), points: map[int]readActivityPoint{1: {process: parent, requested: 100, observedAt: start.Add(time.Second)}}}
	got := summarizeReadActivity(first, last, 20)
	if got.Status != "partial" || len(got.Rows) != 1 || got.Rows[0].ObservedChildExits != 1 {
		t.Fatalf("exit evidence = %+v", got)
	}
}
