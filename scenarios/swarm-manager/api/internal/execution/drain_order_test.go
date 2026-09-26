package execution

import (
	"testing"
)

func rec(kind, name, createdAt string) Record {
	return Record{BacklogKind: kind, BacklogName: name, Status: StatusPending, CreatedAt: createdAt}
}

func order(pending []Record) []string {
	out := make([]string, len(pending))
	for i, r := range pending {
		out[i] = r.BacklogKind + "/" + r.BacklogName
	}
	return out
}

func TestSortPendingForDrain_NilPrioritiesIsFIFO(t *testing.T) {
	pending := []Record{
		rec("execute", "c", "2026-07-01T03:00:00Z"),
		rec("execute", "a", "2026-07-01T01:00:00Z"),
		rec("execute", "b", "2026-07-01T02:00:00Z"),
	}
	sortPendingForDrain(pending, nil)
	got := order(pending)
	want := []string{"execute/a", "execute/b", "execute/c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("FIFO order = %v, want %v", got, want)
		}
	}
}

func TestSortPendingForDrain_GoaledBeforeUngoaledThenPriority(t *testing.T) {
	pending := []Record{
		rec("execute", "ungoaled-old", "2026-07-01T00:00:00Z"),
		rec("execute", "goal-low", "2026-07-01T05:00:00Z"),
		rec("execute", "goal-high", "2026-07-01T06:00:00Z"),
	}
	priorities := map[string]int{
		"execute/goal-low":  2,
		"execute/goal-high": 9,
	}
	sortPendingForDrain(pending, priorities)
	got := order(pending)
	// Higher-priority goal first, then lower-priority goal, then ungoaled —
	// even though the ungoaled item is the oldest by CreatedAt.
	want := []string{"execute/goal-high", "execute/goal-low", "execute/ungoaled-old"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("goal-priority order = %v, want %v", got, want)
		}
	}
}

func TestSortPendingForDrain_SameGoalPriorityFallsBackToFIFO(t *testing.T) {
	pending := []Record{
		rec("execute", "newer", "2026-07-01T09:00:00Z"),
		rec("execute", "older", "2026-07-01T08:00:00Z"),
	}
	priorities := map[string]int{"execute/newer": 5, "execute/older": 5}
	sortPendingForDrain(pending, priorities)
	got := order(pending)
	if got[0] != "execute/older" || got[1] != "execute/newer" {
		t.Fatalf("tie should break FIFO, got %v", got)
	}
}
