// Package schedule is the shared, durable, capacity-aware deferred-work
// scheduler.
//
// The capacity broker (package capacity and the control plane's internal
// capacity engine) answers one question: may this operation run right now? It
// does not remember work that should run later, and it deliberately owns no
// queue, batching, or fairness policy. This package supplies that missing half:
// durable jobs that wait for a window, a budget, or free capacity, grouped by
// model affinity so a local model is loaded once instead of once per job, and
// admitted through the broker so there is still exactly one admission policy in
// the system.
//
// # Layering
//
//	schedule.Job ──> Store (SQLite: jobs, runs, model dwell)
//	                    │
//	                    ▼
//	              Scheduler.Tick
//	                    │  selection + grouping + fair share
//	                    ▼
//	              Admission ──> packages/capacity.Broker (RAM/CPU/VRAM policy)
//	                    │
//	                    ▼
//	              Executor (embeddings, annotations, any batch work)
//
// The scheduler never re-implements admission. A denied or queued verdict is a
// deferral with the broker's own reason preserved, retried with backoff. The
// scheduler always asks for capacity without reclaim: deferred work must not
// evict a resident interactive model to make progress.
//
// # Selection algorithm
//
// One tick admits one batch at a time and continues while the gate, budget and
// capacity allow it:
//
//  1. Ready jobs are those in state queued or deferred with NotBefore <= now and
//     a deadline that has not passed. Overdue deadlines fail before selection.
//  2. Ready jobs are grouped by (Kind, Owner, ResourceClass, ModelAffinity,
//     BatchKey). Kind is part of the key so an embedding job and an annotation
//     job that happen to name the same model are never executed as one batch;
//     owner and resource class are part of it because one capacity claim can
//     only be attributed to one owner and one resource dimension.
//  3. A group whose model is still resident inside Policy.ModelDwell is
//     preferred, because running it costs no model load.
//  4. Groups are ordered by: priority tier (interactive, service, batch), then
//     hot model, then value density (sum of Value divided by sum of estimated
//     seconds), then age (oldest NotBefore first), then fair-share deficit
//     (owners furthest below their share first).
//  5. Up to MaxBatch jobs are taken from the chosen group, trimmed again so the
//     batch fits the remaining seconds budget.
//  6. A group whose owner is already at MaxSharePerOwner yields to a remaining
//     group that is still below its share. When every remaining group is at its
//     share the gate stands down, so the queue cannot deadlock.
//  7. Admission is called once for the batch with the aggregate estimate. A
//     non-grant verdict defers every job in the batch with the verdict reason
//     and exponential backoff, and ends the tick: the host is full.
//  8. Each leased job is executed, checkpointed, completed with its actual cost,
//     and its admission lease released.
//
// # Modes
//
//   - off: nothing runs, including urgent work.
//   - passive: only explicit interactive-priority jobs run; no background fill,
//     no window/idle/budget gating, because an operator asked for that work now.
//   - idle: background fill only when the host observation is idle and the
//     current time is inside a configured window.
//   - balanced: background fill inside windows, within budgets.
//   - eager: any capacity the broker admits, ignoring windows and idle gating.
//     Explicit budget caps are still enforced; they are safety limits, not
//     scheduling preferences.
//
// # Durability
//
// Jobs, execution attempts, modelled dwell, and every run's estimated and actual
// cost live in a dedicated SQLite database (never the broker's claim ledger).
// A running job whose lease expires - process crash, host reboot, lost renewer -
// is requeued by the next tick, preserving its attempt count and its checkpoint.
// Executors that accept a Checkpointer can persist progress mid-run and resume
// from Job.Checkpoint after a crash.
//
// # Status
//
// Status reports queue depth by kind and model, the oldest due job, counts per
// deferral reason, remaining budget in the rolling hour, the window and idle
// gate, and the last run's actual versus estimated cost.
package schedule
