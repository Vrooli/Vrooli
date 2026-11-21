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
[token attribution reference](../reference/operational-stats.md#token-attribution). `footprint`
answers how large a tool result was, `residency` estimates how much context it
occupied across turns, and `incurred` attributes measured turn usage with an
explicit residual. Invocation facts retain the independent token factors and
their basis instead of storing a derived product as provider truth. Run facts
retain the conservation buckets `preamble_injected`, `preamble_fixed`,
`tool_result_residency`, `assistant_output`, `compaction`, and `unattributed`.
The buckets sum to the run total; any unexplained difference remains visible
in `unattributed_tokens` with a reason.

`run_findings` is durable investigation evidence: the recommendations an
investigation produced, the subject runs they implicate, and the intake topic
each was routed to. It carries no review state (the operator-decision and
effectiveness-measurement columns are removed). Finding-recurrence
measures window it by `created_at` (not run terminal time), group fingerprints
within that filtered corpus, and then apply run dimensions through the durable
terminal summary. `friction_problem_status` holds what the operator did with a
recurring friction problem (`reported` with its scenario-qa bug id, or
`dismissed`); a problem with no row is open.

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

External transcripts are captured by the Transcript Indexer
(`api/internal/transcriptindex`, see
[USAGE-INTELLIGENCE.md](USAGE-INTELLIGENCE.md#t-1-capture)). Its
`transcript_files` row per file (keyed by device:inode) and its
`transcript_usage` rows (one per provider usage report, with raw cumulative
counters and per-turn non-text counts) are durable summaries: they are never
evicted with `run_events`. An imported run whose events were evicted is
projected and accounted from those rows, and an explicit replay rebuilds its
events from the source file. The full events and search documents of indexed
runs are a recall layer bounded by `ImportedCorpusMaxBytes`, measured on
those rows alone.

For an imported run, `running` and `complete` reflect the indexer's activity
window, not an authoritative native goal state. An active import has no
`ended_at`; an idle import can use the latest timestamp actually observed in
its transcript, which is not proof of native completion. That observation is
stored with the file checkpoint across restarts. Missing, unreadable, changed
or incompletely indexed sources do not establish an idle transition in the
idle sweep; inspect the native owner and capture coverage when their state
matters. The imported-run status and transcript checkpoint commit in one
owner transaction, preceded by a durable capture intent. An interrupted or
uncertain capture requires reconciliation; no automatic repair command is
currently provided. Event writes and downstream projections remain separate
effects. Recovery cannot establish source identity from file metadata and a
short head hash alone. Run-owned usage capture uses the same conditional intent
fence. Recall eviction excludes unresolved checkpoints and atomically checks
the checkpoint and event stream before removing recall; per-run sequence
watermarks and the global retention generation preserve cursor semantics.
Detected transcript replacement or parser-version
reimport stages a bounded snapshot and promotes the new checkpoint, usage,
events, run state and configured SQLite quota observations together, fenced
against concurrent changes to the old evidence. Existing quota observations
are immutable provider history and remain even if their source is replaced;
unrelated same-scope observations are never deleted. Failure before promotion
retains prior evidence. A missing required quota owner fails replacement closed;
only an explicitly disabled quota sink skips observation capture. Size,
timestamps and a matching second source read
detect some changes but do not prove an atomic snapshot against a concurrent
writer. Post-commit search and projection notifications have a durable pending
marker and are retried. Do not treat these records as proof of complete capture.

For a source-replacement guard, inspect `GET /api/v1/runs/{run-id}/transcript-reconcile`
to obtain its exact file key and reason. After inspecting the retained evidence
and source, an authorized human operator may explicitly retry that one target with
`POST /api/v1/runs/{run-id}/transcript-reconcile` and JSON `{"fileKey":"..."}`.
This operation does not clear uncertain event-recording outcomes. A failed or
ambiguous response requires checking the owner before another attempt.
Legacy snapshot imports can be staged and relinked only when their exact run
provenance and private path still match, its old event stream is empty, and no
other transcript or usage rows are associated with the run. The owner transaction replaces the event stream,
run pointer and source checkpoint together and records a cleanup obligation.
The private transcript is retained: `GET` reports the pending cleanup, including
after restart, but no identity-safe cleanup operation is currently provided.
The POST retry applies only to a guarded source, not this cleanup obligation.
The obligation survives run deletion; discovery currently requires the run ID.
Existing files and usage requiring cross-source reconciliation remain guarded.

An explicit recall rebuild stages at most 32 MiB of checkpointed source and
32 MiB of parsed event data in memory, then publishes only if the retained event
stream remains empty and the original checkpoint still matches in the owner
transaction that commits both events and the new checkpoint. Read, parse, and
pre-commit failures do not publish staged events. A failed post-commit search
or projection notification leaves a durable retry marker. Size and
modification-time checks and a comparison of
the first up to 4 KiB with the checkpoint fingerprint detect some changes, but
do not prove that the snapshot matches historical source bytes. A missing or
invalid checkpoint fingerprint requires reconciliation.
Replacing a nonempty retained stream is not supported by this rebuild path.

Conversation-search state has a different retention class. Canonical messages
remain in `run_events` under the policy above. `conversation_search_catalog`,
its external-content FTS table, generation/checkpoint rows, and the variant-scoped Qdrant
collection are regenerable projections and never become a historical source of
truth. `conversation_search_telemetry` contains categorical request/outcome
facts only. It retains at most 30 days and 100,000 newest rows; a read samples at
most 10,000 newest rows and reports truncation. Stable hit IDs can correlate a
selection with one returned result, but query text, regex, snippet text,
message content, and raw paths have no columns.

## Query contract

The invocation analytics filter is shared by aggregate, cohort, and metrics
queries. It supports `[from,to)`, ownership, outcome, executable, fingerprint,
profile, runner, model, workload kind/key, tag prefix, and run status. Cohorts
are bounded and return an explicit truncation flag; callers must never treat a
truncated list as complete.

## Statistics consumer inventory and durable parity boundary

The typed Measures service (`agent_manager.v1.measures.MeasuresService`, Connect)
is the canonical analytics contract. The current consumers are:

- `ui/src/features/usage/` reads the Measures procedures through
  `measuresClient` for every analytics panel. It does not read the REST
  `/api/v1/stats/summary` or drill-down routes. It reads
  `/api/v1/pricing/compare` for model cost comparison.
- `agent-manager measures get <id>` executes any measure the API declares
  through the measure registry (`measures list` prints them); there is no
  per-measure CLI verb and no `run stats`.
- `agent-manager measures get fallback-insights` reads the retained fallback and
  policy-candidate events through `FallbackInsights`; it is not part of the
  durable read model.
- The pricing model picker ranks recent and popular models from
  `invocation_read_model_runs` through the narrowed `StatsRepository`.

Every `MeasuresService` procedure, and `/measures/execute`, is an agent read.
The admission rule is derived from the generated service descriptor
(`handlers/authorization_routes.go`), so a new measure needs no allowlist edit.

The REST statistics summary and drill-down routes (`/api/v1/stats/summary`,
`status-distribution`, `success-rate`, `duration`, `cost`, `runners`,
`profiles`, `models`, `models/runs`, `tools`, `tools/models`, `tools/runs`,
`errors`, `time-series`) were deleted under Usage Intelligence decision D4.
No statistic reopens `run_events` JSON.

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
errors, time series, and cohort questions. `StatsSummary` is a transport
over that read model, not a second compute path, until the REST layer is
removed.

The throughput parity fixture covers run volume, terminal success, average
cycle time, total cost, and total tokens from one seeded database snapshot.
It permits only SQLite's documented one-millisecond `julianday` truncation
artifact; it does not permit a different run population or a different value.
