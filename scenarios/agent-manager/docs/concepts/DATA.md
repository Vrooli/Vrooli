# Agent Manager Data Map

This is the durable vocabulary for consumption analytics. The implementation
links are [CODE: scenarios/agent-manager/api/internal/domain/run_configuration.go]
and [CODE: scenarios/agent-manager/api/internal/invocationreadmodel/project.go].

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

`invocation_read_model_runs` is the durable terminal summary for throughput and
all retained cross-run statistics: terminal time (with an explicit fallback
basis), lifecycle status, profile, runner, model, tag, duration, charge, and
token totals. Its companion
`invocation_read_model_run_signals` retains read-call and reread counts using
the same conservative path classifier as `run report`. The run summary,
signals, invocation facts, and watermark are advanced in one SQLite
transaction by the production projection adapter.

### Consumption, charge, yield, billing basis, and workload identity

These are separate facts:

- Consumption is input, output, cache token classes, turns, duration, and tool
  calls. It is emitted even when pricing is unavailable.
- Charge is a nullable micro-USD amount plus an explicit basis (`metered`,
  `subscription`, `local`, `unpriced`, or `unknown`). A nil amount means the
  charge is unpriced; zero is reserved for a known free basis.
- Yield is terminal outcome and successful-completion evidence. Consumption per
  successful completion includes failed attempts in its numerator.
- Billing basis is the run's immutable snapshot of metering, subscription, or
  local execution context.
- Workload identity is the typed kind, key, and instance used to group runs
  without parsing display tags.

A billing snapshot and structured workload reference are stamped on the run at
creation, so later resource-policy or subscription changes cannot silently
reprice or regroup historical runs.

`ProjectRun` decodes metric payloads by shape: usage is folded independently
from charge, and retained duplicate cumulative usage snapshots are ignored.
Historical fused metric JSON is normalized at the event boundary into these
same shapes; no retired domain payload is needed to read it. The architecture
guard `costTrackingRunnerHasChargeSource` fails when a cost-tracking runner is
wired without a charge source. The typed measure registry owns workload
breakdown and efficiency queries.

Per-turn input usage also derives run-level preamble buckets. The injected
instruction estimate is recorded with the run at creation; the fixed portion
uses the minimum observed input per compaction segment minus that injected
estimate. This minimum slightly over-counts because the shortest turn still
contains conversation history, so it is an estimate rather than exact prefix
measurement. Runs without per-turn usage report zero preamble tokens with an
unknown basis.

Token attribution is exposed in three deliberately distinct views; see the
[token attribution reference](../reference/token-attribution.md). `footprint`
answers how large a tool result was, `residency` estimates how much context it
occupied across turns, and `incurred` attributes measured turn usage with an
explicit residual. Invocation facts retain the independent token factors and
their basis instead of storing a derived product as provider truth. Run facts
retain the conservation buckets `preamble_injected`, `preamble_fixed`,
`tool_result_residency`, `assistant_output`, `compaction`, and `unattributed`.
The buckets sum to the run total; any unexplained difference remains visible
in `unattributed_tokens` with a reason.

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
profile, runner, model, workload kind/key, tag prefix, and run status. Cohorts
are bounded and return an explicit truncation flag; callers must never treat a
truncated list as complete.

## Statistics consumer inventory and durable parity boundary

The retained statistics product has three consumers: `agent-manager run stats`
uses `/api/v1/stats/summary`; `ui/src/features/stats/` reads the summary plus
the status, duration, cost, runner, profile, model, tool, error, and
time-series endpoints; `ops` reads the separate typed-event operational
projection. The operational projection is not a candidate for this migration.

The retained statistics compatibility transport is backed exclusively by the
durable read model. Product metadata such as task and profile names can be
joined at read time, but no statistic reopens `run_events` JSON. The typed
measure service is the canonical machine-facing analytics contract; the legacy
summary and drill-down routes remain only for existing product consumers while
they migrate.

Same-snapshot regression coverage proves these question families:

- run volume and status counts;
- terminal run success rate;
- cycle-time statistics;
- cost and token totals;
- runner, profile, and model breakdowns;
- tool usage and per-tool failure rate;
- error-pattern counts; and
- time-series buckets.

The durable read model owns ownership, outcome, retry, help-recovery, repeated
work, file rereads, run success, cycle time, cost, volume, breakdowns, tools,
errors, time series, and cohort questions. `StatsSummary` remains a
compatibility transport, not a second compute path.

The throughput parity fixture covers run volume, terminal success, average
cycle time, total cost, and total tokens from one seeded database snapshot.
It permits only SQLite's documented one-millisecond `julianday` truncation
artifact; it does not permit a different run population or a different value.
