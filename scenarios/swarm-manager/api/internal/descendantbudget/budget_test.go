package descendantbudget

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func newTestBudget(t *testing.T) *Budget {
	t.Helper()
	b := NewForTest(t, "effort-1")
	b.SetClock(func() time.Time { return time.Date(2026, 9, 13, 19, 0, 0, 0, time.UTC) })
	return b
}

// NewForTest roots a budget under a temp dir for tests.
func NewForTest(t *testing.T, effortID string) *Budget {
	t.Helper()
	budget, err := NewForRoot(t.TempDir(), effortID)
	if err != nil {
		t.Fatalf("NewForRoot: %v", err)
	}
	return budget
}

func limit(depth, active, premium int) Limit {
	return Limit{MaxDepth: depth, MaxActiveDescendants: active, MaxPremiumDescendants: premium}
}

func TestAdmitDirectChild(t *testing.T) {
	b := newTestBudget(t)
	r, err := b.Admit(limit(2, 3, 1), Reservation{AttemptID: "child-1", Depth: 1})
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if r.State != ReservationActive || r.Depth != 1 || r.EffortID != "effort-1" {
		t.Fatalf("unexpected reservation: %+v", r)
	}
}

func TestAdmitRejectsDepthBeyondMaximum(t *testing.T) {
	b := newTestBudget(t)
	_, err := b.Admit(limit(1, 3, 1), Reservation{AttemptID: "deep", Depth: 2})
	if !errors.Is(err, ErrDepthExceeded) {
		t.Fatalf("want ErrDepthExceeded, got %v", err)
	}
}

func TestAdmitRejectsZeroDepth(t *testing.T) {
	b := newTestBudget(t)
	if _, err := b.Admit(limit(2, 3, 1), Reservation{AttemptID: "root", Depth: 0}); err == nil {
		t.Fatal("want error for depth zero")
	}
}

func TestAdmitEnforcesActiveCap(t *testing.T) {
	b := newTestBudget(t)
	for _, id := range []string{"a", "b"} {
		if _, err := b.Admit(limit(2, 2, 0), Reservation{AttemptID: id, Depth: 1}); err != nil {
			t.Fatalf("Admit %s: %v", id, err)
		}
	}
	_, err := b.Admit(limit(2, 2, 0), Reservation{AttemptID: "c", Depth: 1})
	if !errors.Is(err, ErrCapacityExhausted) {
		t.Fatalf("want ErrCapacityExhausted, got %v", err)
	}
}

func TestAdmitEnforcesPremiumCap(t *testing.T) {
	b := newTestBudget(t)
	if _, err := b.Admit(limit(2, 3, 1), Reservation{AttemptID: "p1", Depth: 1, Premium: true}); err != nil {
		t.Fatalf("Admit premium: %v", err)
	}
	_, err := b.Admit(limit(2, 3, 1), Reservation{AttemptID: "p2", Depth: 1, Premium: true})
	if !errors.Is(err, ErrCapacityExhausted) {
		t.Fatalf("want premium exhaustion, got %v", err)
	}
	// A non-premium child is still admitted.
	if _, err := b.Admit(limit(2, 3, 1), Reservation{AttemptID: "e1", Depth: 1}); err != nil {
		t.Fatalf("Admit economical: %v", err)
	}
}

func TestAdmitRejectsPremiumWhenNoneAuthorized(t *testing.T) {
	b := newTestBudget(t)
	_, err := b.Admit(limit(2, 3, 0), Reservation{AttemptID: "p", Depth: 1, Premium: true})
	if !errors.Is(err, ErrCapacityExhausted) {
		t.Fatalf("want ErrCapacityExhausted, got %v", err)
	}
}

func TestAdmitIsIdempotent(t *testing.T) {
	b := newTestBudget(t)
	first, err := b.Admit(limit(2, 3, 1), Reservation{AttemptID: "child-1", Depth: 1})
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	second, err := b.Admit(limit(2, 3, 1), Reservation{AttemptID: "child-1", Depth: 1})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if first.At != second.At {
		t.Fatalf("replay minted a new reservation: %+v vs %+v", first, second)
	}
	totals, err := b.Totals()
	if err != nil {
		t.Fatalf("Totals: %v", err)
	}
	if totals.Active != 1 {
		t.Fatalf("replay consumed a second slot: %+v", totals)
	}
}

func TestAdmitRejectsIdentityReuse(t *testing.T) {
	b := newTestBudget(t)
	if _, err := b.Admit(limit(2, 3, 1), Reservation{AttemptID: "child-1", Depth: 1}); err != nil {
		t.Fatalf("Admit: %v", err)
	}
	_, err := b.Admit(limit(2, 3, 1), Reservation{AttemptID: "child-1", Depth: 2})
	if !errors.Is(err, ErrReservationConflict) {
		t.Fatalf("want ErrReservationConflict, got %v", err)
	}
}

func TestAdmitChildRequiresActiveParentAndExactDepth(t *testing.T) {
	b := newTestBudget(t)
	if _, err := b.Admit(limit(3, 4, 1), Reservation{AttemptID: "parent", Depth: 1}); err != nil {
		t.Fatalf("Admit parent: %v", err)
	}
	if _, err := b.Admit(limit(3, 4, 1), Reservation{AttemptID: "child", ParentAttemptID: "parent", Depth: 2}); err != nil {
		t.Fatalf("Admit child: %v", err)
	}
	if _, err := b.Admit(limit(3, 4, 1), Reservation{AttemptID: "bad-depth", ParentAttemptID: "parent", Depth: 3}); err == nil {
		t.Fatal("want depth mismatch error")
	}
	_, err := b.Admit(limit(3, 4, 1), Reservation{AttemptID: "orphan", ParentAttemptID: "missing", Depth: 2})
	if !errors.Is(err, ErrParentNotActive) {
		t.Fatalf("want ErrParentNotActive, got %v", err)
	}
}

func TestReleaseFreesCapacityAndIsIdempotent(t *testing.T) {
	b := newTestBudget(t)
	if _, err := b.Admit(limit(2, 1, 0), Reservation{AttemptID: "a", Depth: 1}); err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if _, err := b.Admit(limit(2, 1, 0), Reservation{AttemptID: "b", Depth: 1}); !errors.Is(err, ErrCapacityExhausted) {
		t.Fatalf("want capacity exhausted, got %v", err)
	}
	if _, err := b.Release("a"); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if _, err := b.Release("a"); err != nil {
		t.Fatalf("idempotent Release: %v", err)
	}
	if _, err := b.Admit(limit(2, 1, 0), Reservation{AttemptID: "b", Depth: 1}); err != nil {
		t.Fatalf("Admit after release: %v", err)
	}
	if _, err := b.Release("missing"); !errors.Is(err, ErrUnknownReservation) {
		t.Fatalf("want ErrUnknownReservation, got %v", err)
	}
}

func TestBudgetSurvivesReload(t *testing.T) {
	root := t.TempDir()
	b, err := NewForRoot(root, "effort-1")
	if err != nil {
		t.Fatalf("NewForRoot: %v", err)
	}
	if _, err := b.Admit(limit(2, 2, 1), Reservation{AttemptID: "a", Depth: 1, Premium: true}); err != nil {
		t.Fatalf("Admit: %v", err)
	}
	reopened, err := NewForRoot(root, "effort-1")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	totals, err := reopened.Totals()
	if err != nil {
		t.Fatalf("Totals: %v", err)
	}
	if totals.Active != 1 || totals.Premium != 1 || totals.Limit.MaxActiveDescendants != 2 {
		t.Fatalf("reload lost accounting: %+v", totals)
	}
	if _, err := reopened.Admit(limit(2, 2, 1), Reservation{AttemptID: "b", Depth: 1, Premium: true}); !errors.Is(err, ErrCapacityExhausted) {
		t.Fatalf("reload lost premium accounting: %v", err)
	}
}

func TestConcurrentAdmitsRespectCap(t *testing.T) {
	dir := t.TempDir()
	b, err := NewForRoot(dir, "effort-1")
	if err != nil {
		t.Fatalf("NewForRoot: %v", err)
	}
	const cap = 2
	var wg sync.WaitGroup
	var mu sync.Mutex
	admitted := 0
	refused := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := b.Admit(limit(2, cap, 0), Reservation{AttemptID: string(rune('a' + i)), Depth: 1})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				admitted++
			} else {
				refused++
			}
		}(i)
	}
	wg.Wait()
	if admitted != cap || refused != 12-cap {
		t.Fatalf("concurrent admits: admitted=%d refused=%d", admitted, refused)
	}
}

func TestEffortReferenceMustMatch(t *testing.T) {
	b := newTestBudget(t)
	if _, err := b.Admit(limit(2, 3, 1), Reservation{AttemptID: "a", EffortID: "other", Depth: 1}); err == nil {
		t.Fatal("want effort mismatch error")
	}
}

func TestNewForRootRejectsNestedEffortID(t *testing.T) {
	if _, err := NewForRoot(filepath.Join(t.TempDir()), "a/b"); err == nil {
		t.Fatal("want rejection of nested effort id")
	}
}
