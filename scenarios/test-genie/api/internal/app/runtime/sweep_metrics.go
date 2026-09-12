package runtime

import (
	"log"

	"test-genie/internal/dbexec"
	"test-genie/internal/selfhealthsnapshots"
)

// poolSweepObserver records the bounded advisory sweep's outcome together with
// the only SQLite-pool contention signals that matter to operators. Logging is
// intentionally used here because Test Genie has no metrics exporter; the
// cached status remains the health-path projection and never queries the pool.
func poolSweepObserver(db dbexec.PoolStats, status *selfhealthsnapshots.StatusStore) func(selfhealthsnapshots.SweepStatus) {
	return func(sweep selfhealthsnapshots.SweepStatus) {
		if db != nil {
			stats := db.Stats()
			sweep.PoolWaits = stats.WaitCount
			sweep.PoolWait = stats.WaitDuration
			sweep.PoolOpen = stats.OpenConnections
			sweep.PoolInUse = stats.InUse
		}
		if status != nil {
			status.Record(sweep)
		}
		log.Printf("[test-genie] self-health sweep outcome=%s duration=%s runs=%d deadline=%s pool_waits=%d pool_wait=%s pool_open=%d pool_in_use=%d", sweep.Outcome, sweep.Duration, sweep.RunCount, sweep.Deadline, sweep.PoolWaits, sweep.PoolWait, sweep.PoolOpen, sweep.PoolInUse)
	}
}
