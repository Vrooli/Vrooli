// Package retention enforces declared storage ceilings so Vrooli-created data
// carries an upper bound by construction rather than by each author remembering
// to write a cleanup loop.
//
// A component — scenario, resource, tool, or safeguard — declares budgets in
// its manifest:
//
//	"retention": {
//	  "budgets": {
//	    "system_events": {
//	      "target": {
//	        "kind": "sqlite_table",
//	        "database": "autoheal.sqlite",
//	        "table": "system_events",
//	        "time_column": "occurred_at"
//	      },
//	      "max_age": "30d",
//	      "max_bytes": "2GiB",
//	      "pruner": "builtin",
//	      "rationale": "Host event ingest; volume is host-driven, so the byte ceiling is the real bound."
//	    }
//	  }
//	}
//
// # An age bound alone is not a bound
//
// This package exists because a correctly configured 30-day retention policy
// freed nothing while its database grew to 453 GB — 41% of the host disk. Every
// row was 17 days old, so a 30-day horizon deleted none of them. Retention
// expressed only in time is a storage promise proportional to an ingest rate the
// component usually does not control. A budget must therefore declare at least
// one bound, and a budget declaring only MaxAge is reported as unbounded in
// size.
//
// The engine enforces whichever bound binds first and names it in
// [Result.BoundBy]. A [BoundBytes] result is a finding about the producer, not a
// routine success: it means data is being generated faster than the declared
// horizon allows, and reporting it is what keeps retention from silently hiding
// the defect it compensates for.
//
// # Retention needs its own database connection
//
// A component must NOT hand [ScenarioConfig.OpenDatabase] the same *sql.DB it
// serves requests from. Open a second handle.
//
// This is the sharpest edge in the package and it has already drawn blood.
// Autoheal shared its serving handle, whose pool was capped at one connection,
// reasoning that a second handle would contend for the write lock. It would —
// and that contention is the cheap outcome. What sharing produced instead was a
// queue: a cycle issues thousands of statements, each took the only connection
// the API had, and a correct bounded prune presented to the rest of the process
// as a database that had stopped answering. Its health probe timed out, its
// supervisor concluded the process was dead, the restart aborted the cycle, and
// RunOnStart began it again from zero. The cycle could never finish.
//
// In WAL mode readers proceed throughout and every batch is its own short
// transaction, so writer contention between two handles resolves in
// milliseconds. Milliseconds of contention is what a second handle buys instead
// of minutes of starvation.
//
// Two related expectations follow, for whatever watches the component:
//
//   - A health probe's latency budget is not a liveness test. A database busy
//     with maintenance is reachable, and reporting it as a failed dependency
//     invites a restart that can only make things worse.
//   - A cycle is preemptible but not instant. [SQLiteTableConfig.MaxDuration]
//     bounds it and [SQLiteTableConfig.BatchPause] leaves gaps for the serving
//     path, but a supervisor that restarts on slowness will still interrupt
//     work that was about to finish.
//
// # A budget bounds only the table it names
//
// Nothing bounds the tables nobody declared. Autoheal's three budgets were each
// correct while incident_observations grew unattended in the same file, and the
// manifest looked complete the whole time. [Manager.AuditUnbudgetedTables]
// answers the question a manifest cannot ask about itself, and the manager logs
// its findings once at startup.
//
// # Prune and rebuild have different disk profiles
//
// [SQLiteTablePruner.Prune] deletes in batches and needs no headroom.
// [SQLiteTablePruner.RebuildToBudget] keeps the surviving rows twice until it
// swaps, because the original's index on the time column is what makes the copy
// cheap and it cannot be dropped early. A rebuild therefore peaks at roughly two
// ceilings above where it started, which is why it is an operator command rather
// than a scheduled one, and why it refuses with [ErrInsufficientSpace] rather
// than discovering the shortfall part-way through a write.
//
// # Ordering
//
// Pruning runs before compaction, never the reverse. Compaction writes a
// complete new copy of the result, so its cost is the size AFTER pruning.
// Compacting first would have needed roughly 453 GB of free space against
// 226 GB available and would have failed part-way through a write.
//
// # Seams
//
// [Pruner] is the seam a component implements when it owns a domain selection
// rule that no generic age rule can express — for example "keep the newest N
// snapshots per scenario", where a generic age rule would delete the only
// snapshot of a stable scenario while keeping twenty of a noisy one. The
// framework owns whether a target is within budget; the component owns which
// items die. Components with no such rule declare pruner "builtin" and write no
// Go code at all.
//
// Target paths resolve through the api-core/storage class roots, so a shadow
// variant prunes its own data and never live's.
//
// Reference documentation: docs/reference/storage-retention.md.
package retention
