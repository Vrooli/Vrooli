# Agent Manager Data Map

## Durable invocation analytics

`run_events` is the append-only operational source for an agent run. It is
retention-bounded and therefore is not the historical analytical source of
truth for tool friction.

`invocation_read_model_facts` is the durable, one-row-per-tool-invocation
projection. It stores the occurrence time and basis, tool identity, ownership,
outcome, retry linkage, help-recovery flag, fingerprint, availability,
classifier version, and the run dimensions used by corpus filters (profile,
runner, model, tag, and status). Its indexes support time, ownership, outcome,
executable, and fingerprint predicates.

`invocation_read_model_watermarks` stores one row per run. The last event id,
event time, classifier version, and projection time are committed atomically
with the corresponding replacement facts. A missing watermark means the run
has not yet been projected.

`invocation_read_model_runs` is the durable terminal summary for throughput:
terminal time (with an explicit fallback basis), lifecycle status, profile,
runner, model, tag, duration, cost, and token totals. Its companion
`invocation_read_model_run_signals` retains read-call and reread counts using
the same conservative path classifier as `run report`. The run summary,
signals, invocation facts, and watermark are advanced in one SQLite
transaction by the production projection adapter.

`run_findings` is already durable investigation evidence. Finding-recurrence
measures window it by `created_at` (not run terminal time), group fingerprints
within that filtered corpus, and then apply run dimensions through the durable
terminal summary.

`investigation_invocation_facts` is a legacy investigation cache retained for
safe forward migration and compatibility. Startup migration copies recoverable
rows into the durable read model without recomputing their classifier version.
When the source event is unavailable, the migrated row uses its run end time
and explicitly records a derived time basis.

## Retention and replay

The durable facts outlive `run_events`. Replay is therefore only possible for
runs whose source events remain retained. A replayable run is rebuilt from the
versioned fold. An unreplayable run keeps its stored facts and classifier
version, and the replay result reports that status rather than implying a full
historical rebuild.

## Query contract

The invocation analytics filter is shared by aggregate, cohort, and metrics
queries. It supports `[from,to)`, ownership, outcome, executable, fingerprint,
profile, runner, model, tag prefix, and run status. Cohorts are bounded and
return an explicit truncation flag; callers must never treat a truncated list
as complete.

## Statistics consumer inventory and parity boundary

The retained statistics product has three consumers: `agent-manager run stats`
uses `/api/v1/stats/summary`; `ui/src/features/stats/` reads the summary plus
the status, duration, cost, runner, profile, model, tool, error, and
time-series endpoints; `ops` reads the separate typed-event operational
projection. The operational projection is not a candidate for this migration.

Before the summary transport is removed, parity must be demonstrated at one
database snapshot for these question families:

- run volume and status counts;
- terminal run success rate;
- cycle-time statistics;
- cost and token totals;
- runner, profile, and model breakdowns;
- tool usage and per-tool failure rate;
- error-pattern counts; and
- time-series buckets.

The durable read model already owns ownership, outcome, retry, help-recovery,
repeated work, file rereads, run success, cycle time, cost, volume, and cohort
questions. Legacy breakdown, error-pattern, and time-series questions remain
explicitly unverified rather than being silently approximated. `StatsSummary`
remains available until each question has a typed measure and a same-snapshot
parity result.

The throughput parity fixture covers run volume, terminal success, average
cycle time, total cost, and total tokens from one seeded database snapshot.
It permits only SQLite's documented one-millisecond `julianday` truncation
artifact; it does not permit a different run population or a different value.
