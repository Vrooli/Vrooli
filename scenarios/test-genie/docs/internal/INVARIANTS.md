# Execution Evidence Invariants

## Canonical ownership

- A completed run has exactly one detailed normalized findings document at
  `coverage/runs/<run-id>/findings.json`.
- `coverage/latest/manifest.json` identifies the latest run; it never contains
  a copied findings payload.
- Phase-result JSON is a compact projection: status, counts, bounded metadata,
  and a reference to the canonical findings document. It must not embed
  `findings` or `observations` arrays.
- `evidence-manifest.json` is the versioned digest-checked index for a run's
  detailed evidence. Summaries and transports may carry references, never a
  second detailed payload.

## Live memory

- Event replay is diagnostic state, not durable evidence. It is bounded by
  `TEST_GENIE_EVENT_REPLAY_MAX_EVENTS` and
  `TEST_GENIE_EVENT_REPLAY_MAX_BYTES`; event messages are capped at 4 KiB.
- Every replay event has a monotonic per-run sequence. `FollowRun` resumes
  strictly after `after_sequence`; an evicted non-zero cursor fails explicitly
  so the caller recovers from durable evidence rather than receiving a
  misleading partial replay.
- A run event cannot carry a terminal `SuiteExecutionResult` pointer.
- A disconnected follower must never cause a run to retain more history or
  block execution. Older/terminal detail is recovered from durable evidence.

## Error and recovery semantics

- `unsupported execution evidence schema` means an operator must use a
  compatible reader or perform the planned cutover; it is never silently
  interpreted as an empty pass.
- `corrupt execution evidence` means the digest/reference contract is broken;
  callers surface degraded evidence rather than inventing zero values.
- `evidence artifact exceeds configured budget` preserves the run and records
  a bounded failure outcome; partial files are removed before publication.

## Retention boundary

- `RetentionService` is the sole owner of run artifact, run-log, and compact
  SQLite execution-history deletion. It writes a tombstone before deletion and
  reconciles an interrupted deletion before deciding new eligibility.
- A retention lease is stored at `coverage/pin-leases.json`, names an owner,
  and has a mandatory expiry. The compatibility `PinRun` RPC grants a 30-day
  lease; callers renew it if they still need the evidence.
- Legacy index pins are inert historical metadata and do not protect a run
  from retention. No runtime code reads or writes them.
- Compact execution-history rows are deliberately read as summaries and are
  removed with the run when lifecycle policy selects that run. The archival
  cutover remains an explicit operator action; it must not be silently run
  against existing evidence.
- `suite_execution_phases` is the sole queryable phase-history projection. It
  contains only status, timing, classification, runnability, finding counts,
  and a metrics-presence bit. Runtime readers never parse a historical
  `suite_executions.phases` JSON document; any such legacy column is archive
  input for the offline rebuild, not a compatibility source.
- An offline cutover always inventories and confirms the coverage tree and
  SQLite file together. It rejects active SQLite WAL/SHM sidecars, archives the
  old database intact, verifies archive and replacement integrity, and restores
  the database if the evidence-tree operation fails. A live data window still
  requires separate operator approval.

## Operational health boundary

- `/health` is the only canonical lifecycle contract. It is rendered by
  `packages/api-core/health` and follows its status/readiness/HTTP semantics.
- Lifecycle health uses a dedicated bounded SQLite probe connection. It must
  never borrow the single runtime SQLite pool used by execution, retention, or
  advisory aggregation.
- Self-health snapshots are advisory. They begin only after the HTTP listener
  is live, have a delayed and jittered first run, receive a process-owned
  cancellation context, and each sweep has a deadline.
- A health request must not aggregate execution history, hydrate evidence, or
  cause a self-health sweep. Background analytics cannot delay lifecycle health
  beyond the endpoint's bounded primary-store probe.
- The most recent advisory sweep is cached in process memory with its outcome,
  duration, run cardinality, and error. A failed or timed-out sweep is an
  optional `self_health_sweep` dependency and therefore degrades `/health`; it
  never changes a healthy primary store into an unavailable service.
