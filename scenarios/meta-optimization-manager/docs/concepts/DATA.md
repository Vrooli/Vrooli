# Data

## Purpose Of This Document

The data-ownership and storage map for **meta-optimization-manager**. Because this scenario is a read-mostly aggregator, its owned state is deliberately small: it caches and records its own *outputs*, but it never persists the numerators it computes, and it never owns the denominators (the space docs live with their owner scenarios).

## Storage Overview

- **Engine**: SQLite via `api-core/storage` (per-domain schema, no shared resource).
- **Posture**: minimal owned state. Coverage numerators are computed live and never stored. Denominators are read from owners via the `space --projection` verb, never copied here.
- **What is persisted**: the gaps registry, the trials history + gate registry, the convergence fitness-audit index, and short-TTL coverage snapshots.

## Data Ownership

| Store | Owning domain | What it holds |
|---|---|---|
| `gaps` | focus | Every known gap: projection/cell, status, title, qualitative notes/approaches/follow-ups. |
| `trials_runs` | trials | One row per trial: suite, task, runner, model, success, tokens, wall-time, evaluator, source run id. |
| `trial_gates` | trials | Per Guide-task gate count (for gate-coverage). |
| `convergence_fitness` | convergence | Per-template counts (per-replica cost, drift surfaces, comment-only contracts, add/delete coordinated edits). |
| `reference_health` | convergence | Per gold-star reference verdict (stale-from-template, clean-on-all-tools, stable-days, breadth). |
| `coverage_snapshots` | coverage | Short-TTL cached per-projection coverage + denominator-confidence. |

## Schema Map

Indicative columns (greenfield; finalized at implementation):

- **gaps**: `id, projection, cell, status, title, notes, approaches, follow_ups, created_at, updated_at`
- **trials_runs**: `id, suite, task, runner, model, success, tokens, wall_time_ms, evaluator, source_run_id, created_at`
- **trial_gates**: `task_key, gate_count, updated_at`
- **convergence_fitness**: `template, per_replica_cost, drift_surfaces, comment_only_contracts, coordinated_edits_add, coordinated_edits_delete, captured_at`
- **reference_health**: `reference, template, stale_from_template, clean_on_all_tools, stable_days, breadth, verdict, captured_at`
- **coverage_snapshots**: `projection, coverage_pct, denominator_confidence, computed_at, ttl_seconds`

## Migrations And Compatibility

Per-domain `schema.sql` + `schema.go` registered through `api-core/storage`. Greenfield — no compatibility shims, alias tables, or dual-write migrations; let a schema change fail loudly rather than carry debt.

## Import / Export

- The **gaps registry** is exportable/importable as JSON (so explored-but-unbuilt ideas survive and can seed other tooling).
- **Trials history** is exportable as a time-series (the trend is the value).
- Coverage snapshots are not exported — they are recomputed on demand.

## Retention And Deletion

- **coverage_snapshots**: short TTL; recomputed, not retained.
- **trials_runs**: retained indefinitely (efficiency trend over time is the point).
- **gaps**: retained until resolved or explicitly deferred; resolution is recorded, not deleted.
- **convergence_fitness / reference_health**: refreshed on each scan; prior points retained for the trend.

## Privacy Notes

No end-user PII. Trial runs execute on the project's own code inside `workspace-sandbox`; any code diffs or model outputs from a run remain owned by `agent-manager` / `workspace-sandbox`, not copied here — only the run's metrics (success/tokens/time + a run id) are recorded.

## Cross-References

- [DOMAINS.md](DOMAINS.md) — which domain owns each store.
- [ARCHITECTURE.md](ARCHITECTURE.md) — why the numerator is never persisted.
- [FLOWS.md](FLOWS.md) — when each store is written.
- [../internal/SEAMS.md](../internal/SEAMS.md) — the storage seam and its test double.
