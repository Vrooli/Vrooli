package effectiveness

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/maturity-go/dimensions"
)

const standards = dimensions.Dimension("standards")

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestExpectedEfficacyColdStartEqualsPrior(t *testing.T) {
	s := Stat{} // n = 0
	for _, prior := range []float64{0, 0.5, -2.3, 10} {
		if got := s.ExpectedEfficacyPerToken(prior, DefaultShrinkageK); !approx(got, prior) {
			t.Fatalf("n=0 must equal prior %v, got %v", prior, got)
		}
	}
}

func TestObservedEfficacyPerToken(t *testing.T) {
	// 6 net findings over 2000 tokens → 6 / (2 + ε=1) = 2.0 per-1k-tokens.
	s := Stat{ClosedCount: 8, IntroducedCount: 2, TotalRuns: 4, TotalTokens: 2000}
	if got := s.ObservedEfficacyPerToken(); !approx(got, 2.0) {
		t.Fatalf("expected 2.0, got %v", got)
	}

	// Unknown cost (0 tokens) falls back to net per run: 6/4 = 1.5 — not free.
	su := Stat{ClosedCount: 8, IntroducedCount: 2, TotalRuns: 4, TotalTokens: 0}
	if got := su.ObservedEfficacyPerToken(); !approx(got, 1.5) {
		t.Fatalf("expected 1.5 per-run fallback, got %v", got)
	}

	// A net-negative skill yields negative efficacy (deprioritized in selection).
	sn := Stat{ClosedCount: 1, IntroducedCount: 5, TotalRuns: 2, TotalTokens: 1000}
	if got := sn.ObservedEfficacyPerToken(); got >= 0 {
		t.Fatalf("expected negative efficacy, got %v", got)
	}
}

func TestExpectedEfficacyShrinksTowardPrior(t *testing.T) { // [REQ:EM-P1-002]
	// With n=k the blend is exactly halfway between observed and prior.
	s := Stat{ClosedCount: 4, IntroducedCount: 0, TotalRuns: 3, TotalTokens: 0} // observed = 4/3
	prior := 0.0
	want := 0.5 * s.ObservedEfficacyPerToken()
	if got := s.ExpectedEfficacyPerToken(prior, 3); !approx(got, want) {
		t.Fatalf("expected %v at n=k, got %v", want, got)
	}
}

func TestMemoryStoreRecordRoundTrip(t *testing.T) {
	fixed := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	m := NewMemoryStore().WithClock(func() time.Time { return fixed })

	m.Record(CreditEvent{
		SkillID:               "lint-fix",
		TargetDimension:       standards,
		ClosedByDimension:     map[dimensions.Dimension]int{standards: 3},
		IntroducedByDimension: map[dimensions.Dimension]int{"tests": 1},
		Tokens:                1500,
	})

	got, ok, err := m.Get("lint-fix", standards)
	if err != nil || !ok {
		t.Fatalf("expected standards row, ok=%v err=%v", ok, err)
	}
	if got.ClosedCount != 3 || got.TotalRuns != 1 || got.TotalTokens != 1500 {
		t.Fatalf("unexpected target stat: %+v", got)
	}
	if !got.LastRunAt.Equal(fixed) {
		t.Fatalf("expected last_run_at %v, got %v", fixed, got.LastRunAt)
	}

	// Collateral debt landed on the tests dimension without a run/token count.
	tdebt, ok, _ := m.Get("lint-fix", "tests")
	if !ok || tdebt.IntroducedCount != 1 || tdebt.TotalRuns != 0 || tdebt.TotalTokens != 0 {
		t.Fatalf("expected collateral tests debt with zero runs, got %+v (ok=%v)", tdebt, ok)
	}
}

func TestMemoryStoreIncrementsAccumulate(t *testing.T) {
	m := NewMemoryStore()
	for i := 0; i < 3; i++ {
		m.Record(CreditEvent{
			SkillID:           "refactor",
			TargetDimension:   standards,
			ClosedByDimension: map[dimensions.Dimension]int{standards: 2},
			Tokens:            1000,
		})
	}
	got, _, _ := m.Get("refactor", standards)
	if got.ClosedCount != 6 || got.TotalRuns != 3 || got.TotalTokens != 3000 {
		t.Fatalf("expected accumulated 6/3/3000, got %+v", got)
	}
}

func TestMemoryStoreConcurrentRecord(t *testing.T) {
	m := NewMemoryStore()
	const n = 200
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			m.Record(CreditEvent{
				SkillID:           "refactor",
				TargetDimension:   standards,
				ClosedByDimension: map[dimensions.Dimension]int{standards: 1},
				Tokens:            10,
			})
		}()
	}
	wg.Wait()

	got, _, _ := m.Get("refactor", standards)
	if got.TotalRuns != n || got.ClosedCount != n || got.TotalTokens != n*10 {
		t.Fatalf("concurrent increments lost: %+v", got)
	}
}

func TestMemoryStoreBulkAndList(t *testing.T) {
	m := NewMemoryStore()
	m.Record(CreditEvent{SkillID: "a", TargetDimension: standards, ClosedByDimension: map[dimensions.Dimension]int{standards: 1}})
	m.Record(CreditEvent{SkillID: "b", TargetDimension: standards, ClosedByDimension: map[dimensions.Dimension]int{standards: 1}})
	m.Record(CreditEvent{SkillID: "a", TargetDimension: "tests", ClosedByDimension: map[dimensions.Dimension]int{"tests": 1}})

	bulk, _ := m.Bulk(standards)
	if len(bulk) != 2 {
		t.Fatalf("expected 2 skills in standards bulk, got %d", len(bulk))
	}

	all, _ := m.List("", "")
	if len(all) != 3 {
		t.Fatalf("expected 3 rows total, got %d", len(all))
	}
	bySkill, _ := m.List("a", "")
	if len(bySkill) != 2 {
		t.Fatalf("expected 2 rows for skill a, got %d", len(bySkill))
	}
}
