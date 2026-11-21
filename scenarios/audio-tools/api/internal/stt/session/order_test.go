package session

import (
	"fmt"
	"testing"
)

// TestCommittedSegmentsAreSnapshottedInSampleOrder is the regression for
// map-iteration ordering in reconnect/recovery replay. Both Snapshot and
// PersistedState promise canonical sample order; iterating the committed map
// directly returned a different order on each call.
func TestCommittedSegmentsAreSnapshottedInSampleOrder(t *testing.T) {
	const count = 128
	ledger := newLedger(t, 1<<20)
	for i := 0; i < count; i++ {
		// Insert in reverse sample order so insertion order cannot mask a
		// missing sort.
		start := int64((count - 1 - i) * 100)
		segment := Segment{
			ID:          fmt.Sprintf("segment-%03d", count-1-i),
			Text:        fmt.Sprintf("text-%03d", count-1-i),
			StartSample: start,
			EndSample:   start + 100,
		}
		if _, err := ledger.Commit(segment); err != nil {
			t.Fatal(err)
		}
	}

	assertSampleOrder := func(name string, segments []Segment) {
		t.Helper()
		if len(segments) != count {
			t.Fatalf("%s returned %d segments, want %d", name, len(segments), count)
		}
		for i := 1; i < len(segments); i++ {
			if segments[i].StartSample < segments[i-1].StartSample {
				t.Fatalf("%s is not in canonical sample order at %d: %d after %d",
					name, i, segments[i].StartSample, segments[i-1].StartSample)
			}
		}
	}

	assertSampleOrder("Snapshot", ledger.Snapshot().Committed)
	assertSampleOrder("PersistedState", ledger.PersistedState().Committed)
	// Two snapshots must agree, not merely be individually ordered.
	first := ledger.Snapshot().Committed
	second := ledger.Snapshot().Committed
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("snapshot order changed between calls at %d: %#v vs %#v", i, first[i], second[i])
		}
	}
}

// TestCommittedSegmentsWithoutSampleCursorsKeepCommitOrder covers strategies
// that do not populate sample cursors (OverlapAgree): there is no sample order
// to sort by, so the ledger must retain the order the segments were committed
// rather than inventing lexical order from generated IDs.
func TestCommittedSegmentsWithoutSampleCursorsKeepCommitOrder(t *testing.T) {
	const count = 128
	ledger := newLedger(t, 1<<20)
	want := make([]string, 0, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("session:1:%d:0:0", i)
		want = append(want, id)
		if _, err := ledger.Commit(Segment{ID: id, Text: id}); err != nil {
			t.Fatal(err)
		}
	}

	assertOrder := func(name string, segments []Segment) {
		t.Helper()
		if len(segments) != count {
			t.Fatalf("%s returned %d segments, want %d", name, len(segments), count)
		}
		for i, segment := range segments {
			if segment.ID != want[i] {
				t.Fatalf("%s order at %d = %q, want commit order %q", name, i, segment.ID, want[i])
			}
		}
	}

	assertOrder("Snapshot", ledger.Snapshot().Committed)
	assertOrder("PersistedState", ledger.PersistedState().Committed)
}
