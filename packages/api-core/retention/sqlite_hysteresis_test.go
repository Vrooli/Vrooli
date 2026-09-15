package retention

import (
	"context"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// A byte-bound prune that stops the instant it is under the ceiling leaves the
// target one row below its limit, so the producer puts it back over within one
// interval and the next cycle pays the whole cost again. That is the difference
// between retention as an occasional correction and retention as permanent
// background load — and on autoheal it was the second: health_results sat at
// 1.96 GiB of a 2 GiB ceiling and re-entered the delete loop every cycle.
//
// The contract is that a cycle which does work leaves headroom behind it.
func TestByteBoundPruneLeavesHeadroomBelowTheCeiling(t *testing.T) {
	db, path := newFixtureDB(t, 2000, time.Minute)
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) {
		cfg.BatchSize = 50
		cfg.ReclaimPercent = 80
	})

	before, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	// A ceiling under what the table currently holds, so the byte bound is what
	// does the work and the age bound stays out of it.
	ceiling := before.Bytes * 3 / 4
	budget := Budget{Name: "events", MaxBytes: ceiling}

	if _, err := pruner.Prune(context.Background(), budget); err != nil {
		t.Fatalf("prune: %v", err)
	}

	after, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("measure after: %v", err)
	}
	if after.Bytes > ceiling {
		t.Fatalf("prune left the table over its ceiling: %d > %d", after.Bytes, ceiling)
	}

	// The point of the reclaim target: not merely under, but far enough under
	// that the producer has runway before the next cycle has anything to do.
	lowWater := ceiling / 100 * 80
	if after.Bytes > lowWater {
		t.Errorf("prune stopped at the ceiling instead of the low-water mark: %d bytes, want <= %d (ceiling %d)",
			after.Bytes, lowWater, ceiling)
	}
}

// The companion to the test above, and the one that states the actual goal: a
// second cycle over an already-reduced target must be a no-op. If it deletes so
// much as one batch, the duty-cycling did not happen and the loop is still hot.
func TestSecondCycleOverAReducedTargetDoesNothing(t *testing.T) {
	db, path := newFixtureDB(t, 2000, time.Minute)
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) {
		cfg.BatchSize = 50
	})

	before, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	budget := Budget{Name: "events", MaxBytes: before.Bytes * 3 / 4}

	first, err := pruner.Prune(context.Background(), budget)
	if err != nil {
		t.Fatalf("first prune: %v", err)
	}
	if first.Deleted == 0 {
		t.Fatal("first prune deleted nothing, so the fixture never exercised the byte bound")
	}

	second, err := pruner.Prune(context.Background(), budget)
	if err != nil {
		t.Fatalf("second prune: %v", err)
	}
	if second.Deleted != 0 {
		t.Errorf("second cycle deleted %d rows over an already-reduced target; retention is running hot every cycle",
			second.Deleted)
	}
	if second.BoundBy == BoundBytes {
		t.Error("second cycle reported BoundBytes, which raises a finding on every cycle for a target that is within budget")
	}
}

// The reclaim target is a bound on how much history retention may discard on its
// own initiative. A config that could set it near zero would let a byte ceiling
// empty a table the budget's author asked to keep 30 days of.
func TestReclaimPercentIsRangeChecked(t *testing.T) {
	db, path := newFixtureDB(t, 10, time.Minute)
	for _, pct := range []int{-1, 10, 49, 101, 1000} {
		_, err := NewSQLiteTablePruner(SQLiteTableConfig{
			DB:             db,
			Path:           path,
			Table:          "system_events",
			TimeColumn:     "occurred_at",
			ReclaimPercent: pct,
		})
		if err == nil {
			t.Errorf("ReclaimPercent %d was accepted; it must be rejected", pct)
		}
	}
	// The boundaries themselves are legal: 100 means "prune to the ceiling",
	// which is the old behaviour and stays available.
	for _, pct := range []int{reclaimFloorPercent, 90, 100} {
		if _, err := NewSQLiteTablePruner(SQLiteTableConfig{
			DB:             db,
			Path:           path,
			Table:          "system_events",
			TimeColumn:     "occurred_at",
			ReclaimPercent: pct,
		}); err != nil {
			t.Errorf("ReclaimPercent %d was rejected: %v", pct, err)
		}
	}
}

// A prune loop with no gaps in it holds the database for every instant of a
// multi-minute cycle. Nothing else can reach it, which is how a health probe
// with a sub-second budget concludes the database is dead while retention is
// working correctly.
//
// This asserts the gaps exist, by timing a cycle whose batch count is known.
func TestBatchPauseYieldsBetweenBatches(t *testing.T) {
	db, path := newFixtureDB(t, 600, time.Minute)
	const pause = 20 * time.Millisecond
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) {
		cfg.BatchSize = 100
		cfg.BatchPause = pause
	})

	// Every row but the newest is past the horizon (the cutoff comparison is
	// strict), so this is five full batches of 100 and a partial sixth.
	budget := Budget{Name: "events", MaxAge: time.Minute}

	start := time.Now()
	result, err := pruner.Prune(context.Background(), budget)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	elapsed := time.Since(start)

	if result.Deleted != 599 {
		t.Fatalf("deleted %d rows, want 599", result.Deleted)
	}
	// Five gaps between six batches; the partial last batch skips its pause.
	if min := 4 * pause; elapsed < min {
		t.Errorf("cycle took %s with no room for inter-batch pauses; want at least %s", elapsed, min)
	}
}

// The pause must not become a way for a cancelled cycle to keep running: a
// shutdown that arrives during a pause has to be observed there, not one batch
// later.
func TestBatchPauseObservesCancellation(t *testing.T) {
	db, path := newFixtureDB(t, 5000, time.Minute)
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) {
		cfg.BatchSize = 10
		cfg.BatchPause = time.Hour
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	result, _ := pruner.Prune(ctx, Budget{Name: "events", MaxAge: time.Minute})
	if elapsed := time.Since(start); elapsed > 30*time.Second {
		t.Fatalf("cancellation was not observed during the pause: took %s", elapsed)
	}
	if !result.Incomplete {
		t.Error("a cancelled cycle must report Incomplete so the caller knows the target may still be over budget")
	}
}

// A rebuild holds the surviving rows twice until the swap, so it needs real
// headroom on the filesystem. Discovering that mid-write is how a tool meant to
// clear a disk fills it instead; the guard turns it into a refusal that names
// the number and points at the alternative.
func TestRebuildRefusesWhenFreeSpaceCannotHoldThePeak(t *testing.T) {
	db, path := newFixtureDB(t, 500, time.Minute)
	const ceiling = 8 << 20
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) {
		// Comfortably more than the ceiling, but nowhere near the two copies
		// plus headroom that a rebuild actually peaks at.
		cfg.FreeSpace = func(string) (int64, error) { return ceiling + (1 << 20), nil }
	})

	before := rowCount(t, db)
	_, err := pruner.RebuildToBudget(context.Background(), Budget{Name: "events", MaxBytes: ceiling})
	if !errors.Is(err, ErrInsufficientSpace) {
		t.Fatalf("RebuildToBudget error = %v, want ErrInsufficientSpace", err)
	}
	if after := rowCount(t, db); after != before {
		t.Errorf("a refused rebuild changed the table: %d rows, want %d", after, before)
	}
}

// The guard must not stand in the way of a rebuild the disk can actually hold,
// or the operator path it protects becomes unusable.
func TestRebuildProceedsWhenFreeSpaceCoversThePeak(t *testing.T) {
	db, path := newFixtureDB(t, 500, time.Minute)
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) {
		cfg.FreeSpace = func(string) (int64, error) { return 1 << 40, nil }
	})
	if _, err := pruner.RebuildToBudget(context.Background(), Budget{Name: "events", MaxBytes: 8 << 20}); err != nil {
		t.Fatalf("RebuildToBudget with ample free space: %v", err)
	}
}
