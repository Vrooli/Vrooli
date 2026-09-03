# Test Genie Configuration Levers

This is the operator-facing configuration reference for Test Genie. These levers are the intended control surface for tuning behavior without editing code.

## Core runtime

These are normally provided by the Vrooli lifecycle system:

| Variable | Scope | Default | Purpose |
|----------|-------|---------|---------|
| `API_PORT` | API runtime | lifecycle | Port the Test Genie API listens on |
| `VROOLI_STORAGE_ROOT` | API runtime | unset | Redirects every storage class root, isolating a run. Scenario-agnostic: each scenario still resolves its own path beneath it. There is deliberately no per-database path variable — a generic one is inherited by every child process and redirects siblings. |
| `SCENARIO_DATA_DIR` | API runtime | lifecycle | Default root for embedded persistent state |
| `SCENARIOS_ROOT` | API runtime | inferred from cwd | Root directory for scenario discovery |
| `VROOLI_ROOT` | API + CLI | environment | Repo root for docs, scenario lookup, and path resolution |

## High-value Test Genie levers

| Variable | Scope | Default | Purpose |
|----------|-------|---------|---------|
| `TEST_GENIE_EXECUTION_TIMEOUT` | CLI `execute` | `900` seconds | Blocking timeout for synchronous suite execution |
| `TEST_GENIE_MAX_CONCURRENT_RUNS` | Run manager | `4` | GLOBAL hard ceiling on suites executing at once across ALL scenarios, shared by manually-started runs and the background fleet sweep. Requests beyond the effective cap are admitted as `queued` and promoted FIFO as slots free. Floor 1. |
| `TEST_GENIE_MIN_CONCURRENT_RUNS` | Run manager | `2` | Adaptive-concurrency floor. The measured effective cap never falls below this value or rises above the max ceiling. Floor 1; if configured above the ceiling, both use the ceiling. |
| `TEST_GENIE_MAX_QUEUED_RUNS` | Run manager | `16` | Hard global cap on durable queued runs. Once full, a new divergent scenario request receives retryable saturation rather than adding another queued record. Floor 1. |
| `TEST_GENIE_MAX_QUEUED_RUNS_PER_CALLER` | Run manager | `4` | Fair-share cap on queued runs for one caller identity. Requests without `X-Vrooli-Caller` share the conservative anonymous bucket. Floor 1. |
| `TEST_GENIE_MAX_QUEUED_RUNS_PER_RESERVATION` | Run manager | `12` | Queue slice for a declared collection reservation. Reserved members use this bucket instead of the per-caller limit; reservations expire after all declared members terminate or one hour. Floor 1. |
| `TEST_GENIE_MAX_PREVIEW_RUNS` | Run manager | `2` | Non-blocking admission cap on expensive plan previews. Requests beyond it receive retryable saturation before preview work or queued-record allocation. Floor 1. |
| `TEST_GENIE_MAX_PREVIEW_RUNS_PER_CALLER` | Run manager | `1` | Fair-share cap on concurrent expensive previews for one caller identity. Floor 1. |
| `TEST_GENIE_MAX_RUNS_PER_SCENARIO` | Run manager | `1` | Per-scenario in-progress cap. `1` is a correctness invariant (one live instance per scenario); raising it is documented-unsafe until per-run isolation lands. |
| `TEST_GENIE_EVENT_REPLAY_MAX_EVENTS` | Run manager | `512` | Maximum compact events retained for a reconnecting follower. Older detail is read from durable evidence, not replay memory. Floor 1. |
| `TEST_GENIE_EVENT_REPLAY_MAX_BYTES` | Run manager | `1048576` | Maximum approximate bytes retained in the compact event replay tail. Floor 1. |
| `FollowRun.after_sequence` | Runs API | `0` | Resume strictly after a received event sequence. A non-zero cursor older than the bounded tail is rejected; fetch durable evidence/detail instead of retrying the stream without bounds. |
| Pin lease TTL | Runs API | `30 days` | `PinRun` grants a finite evidence-retention lease. Renew it by pinning the same run and owner again; `UnpinRun` revokes it. This is intentionally not an indefinite index pin. |
| `test-genie execute --retain-for-evidence` | CLI/API run admission | off | Grants a server-owned 30-day retention lease before execution so calibration evidence survives asynchronous cleanup. Pair with `--retention-reason` for the audit explanation. |
| `TEST_GENIE_SELFHEALTH_SWEEP_DISABLED` | API runtime | `false` | Disables advisory self-health snapshots. It does not affect lifecycle health or test execution. |
| `TEST_GENIE_SELFHEALTH_SWEEP_INTERVAL` | API runtime | `1h` | Minimum interval between advisory self-health sweeps. |
| `TEST_GENIE_SELFHEALTH_SWEEP_START_DELAY` | API runtime | `30s` | Minimum delay after the HTTP listener is live before the first advisory sweep. |
| `TEST_GENIE_SELFHEALTH_SWEEP_START_JITTER` | API runtime | `30s` | Randomized extra startup delay that prevents synchronized analytical work after restarts. |
| `TEST_GENIE_SELFHEALTH_SWEEP_TIMEOUT` | API runtime | `20s` | Per-sweep deadline. A timed-out advisory sweep is deferred and must not hold the runtime SQLite pool indefinitely. |
| `TEST_GENIE_PLAYBOOKS_RETAIN` | Workflow compatibility | `0` | Keep temporary isolated Postgres/Redis/SQLite resources alive after legacy seed/debug paths |
| `TEST_GENIE_SKIP_PLAYBOOKS` | Workflow compatibility | unset | Hard-disable workflow execution through the legacy playbooks alias |
| `TEST_GENIE_DOCS_DIR` | Docs handlers | scenario default | Override docs directory served by the API |

## Admission diagnosis

`GET /api/v1/admission` reports current running, queued, and preview occupancy,
their configured limits, adaptive effective concurrency, process-lifetime rejection/coalescing counters, and
the cumulative `phaseAdmissionsTotal` and `fallbackAdmissionsTotal` counters.
`suiteEnvelope` contains only scenario/preset reservations with at least five
timestamped reliable runs. `shadowWouldAdmit`, `shadowWouldDefer`, and
`shadowErrors` summarize the non-authoritative suite-level capacity probe.
The fallback rate is `fallbackAdmissionsTotal / phaseAdmissionsTotal`; use it
to evaluate whether the measured fallback reservation needs review.
Use it before raising a limit: a full preview gate points to planning pressure,
while a full queue points to execution throughput. Saturation responses are
retryable; clients should back off rather than repeatedly submitting work.
The Connect saturation reply also carries an `AdmissionSaturation` detail with
the limit kind, current occupancy, configured limit, FIFO position, and a
`retry_after_seconds` hint. The existing prose remains for compatibility with
older clients.
Queued `GetRunStatus` responses expose `queue_position` and
`estimated_queue_wait_seconds` separately from execution ETA. When no running
suite has a usable duration estimate, the queue wait is explicitly unknown
instead of being reported as zero.

Trusted gateways may set `X-Vrooli-Caller` on REST or Connect requests to make
the per-caller limits fair. It is an admission label, never written into run
artifacts or returned by the status endpoint. Missing identity is deliberately
limited as `anonymous` rather than treated as unrestricted.

## Lifecycle health boundary

`/health` is the canonical lifecycle endpoint. It uses the shared api-core
response schema and returns HTTP `200` for `healthy` or `degraded`, and HTTP
`503` for `unhealthy`. Test Genie probes SQLite through a dedicated lifecycle
connection with a bounded read-only schema check; normal service work continues
to use its intentionally single-connection runtime pool. Self-health snapshots
are advisory work: they start only after the listener is live, have a delayed
and jittered first run, and are never invoked by a health request.

The latest advisory sweep is exposed as an optional health dependency. A failed
or timed-out sweep produces `degraded` while keeping HTTP 200/readiness true;
a failed primary-store probe remains `unhealthy` with HTTP 503.

Each completed advisory sweep emits one structured log line with its outcome,
duration, input run cardinality, configured deadline, and SQLite pool
`waits`, cumulative `wait`, `open`, and `in_use` values. Rising waits or wait
duration while `in_use=1` identifies contention on the intentionally single
runtime pool; investigate foreground work or defer analytics rather than
raising the pool limit. Health itself uses its separate lifecycle connection,
so an advisory timeout is visible as degraded state rather than a blocked
health request.

There is intentionally no separate `/live` or `/ready` endpoint. The canonical
endpoint is already bounded, externally enforceable, and distinguishes a
genuinely unavailable primary store from ordinary background analytics.

## Offline evidence cutover

Use `test-genie evidence-cutover plan` and `apply` only while Test Genie is
stopped and its SQLite WAL has been checkpointed. Both commands require the
scenario directory, coverage archive destination, database path, and database
archive destination. `plan` rejects queued/in-progress indexed runs and emits
`required_free_bytes`, the minimum combined archive capacity; reserve extra
space for the replacement database on its live volume. `apply` additionally
requires `--confirm ARCHIVE_TEST_GENIE_EVIDENCE`. The operation archives the
old evidence tree and SQLite store, creates a fresh canonical SQLite store,
verifies integrity, and writes receipts.

Operator procedure: record the expected outage window, stop Test Genie, wait
for no active runs, checkpoint/remove WAL and SHM sidecars, copy the complete
scenario evidence and SQLite store into an isolated rehearsal directory, run
`plan`, compare its digests and free-space requirement to the copied inventory,
then run the confirmed `apply`. Verify archive readability, replacement
integrity, row counts, receipts, and rollback before requesting a separately
approved live maintenance window. Never run this procedure against
`scenarios/test-genie/data` during rehearsal.

For an active incident only, set `TEST_GENIE_PROFILING_ENABLED=1` and a strong
`TEST_GENIE_PROFILING_TOKEN`. A token-authenticated `POST /api/v1/admission/profile?kind=heap`
returns a heap profile; `kind=cpu&seconds=1..30` captures a bounded CPU profile.
Send `X-Test-Genie-Run-ID` to correlate the download with a run; it is echoed
only in the response header. Keep profiling disabled normally and never expose
the token through client-side configuration.

## Workflow Compatibility

These levers are retained for legacy playbooks seed/debug compatibility. New
phase selection should use `workflow`.

### Debug retained isolation

```bash
TEST_GENIE_PLAYBOOKS_RETAIN=1 test-genie execute my-scenario --phases workflow
```

When retention is enabled, legacy seed compatibility leaves temporary isolated
resources alive and prints inspection commands in the observations.

### Skip workflow entirely

```bash
TEST_GENIE_SKIP_PLAYBOOKS=1 test-genie execute my-scenario --phases workflow
```

This is intended for debugging and constrained environments. It is not a
substitute for fixing broken workflow assets or provider availability.

## CLI timeout tuning

Long suites can exceed the default synchronous timeout. Extend it when running comprehensive or slow playbook-heavy suites:

```bash
TEST_GENIE_EXECUTION_TIMEOUT=1800 test-genie execute my-scenario --preset comprehensive
```

## Playbooks registry metadata

Some behavior is expressed in `bas/registry.json` rather than environment variables:

| Field | Purpose |
|-------|---------|
| `metadata.execution_mode` | Marks a registry as observer-only when every playbook is read-only |
| `playbooks[].reset` | Declares whether the next workflow needs fresh seed state |
| `playbooks[].requirements` | Requirement IDs attached to the workflow |

The registry should be generated via `test-genie registry build`, not hand-maintained.

## Dependencies Phase

The dependencies phase delegates to Scenario Dependency Analyzer:

```bash
scenario-dependency-analyzer health <scenario> --json
```

Dependency readiness, runtime dependency policy, graph drift, governance, release-age policy, and degraded Security Health dependency-index status are configured and interpreted by SDA. Test Genie does not read `.vrooli/testing.json` dependency knobs for this phase.

Common remediation:

| Finding | Typical command |
|---------|-----------------|
| `dependency.readiness.*` | Run the remediation reported by `scenario-dependency-analyzer health <scenario> --json` |
| `dependency.runtime.*` | Start or restart the reported resource/scenario dependency |
| `dependency.graph.*` | Update `.vrooli/service.json` or remove stale dependency usage |
| `dependency.governance.*` | Use SDA governance verbs (never hand-edit the JSON): `scenario-dependency-analyzer deps approved explain <ecosystem>/<pkg>`, then `approve-observed --apply` / `widen-range` / `deny-vulnerable` |
| `dependency.release_age.*` | Add/raise pnpm `minimumReleaseAge` or record an approved exclusion |
| `dependency.security.*` | Check `security-health deps status --json` |

## Advanced infrastructure levers

Test Genie also participates in broader Vrooli infrastructure concerns:

- `AGENT_MANAGER_ENABLED`
- `AGENT_MANAGER_PROFILE_KEY`
- `CONTAINMENT_*`

Those are advanced deployment controls rather than primary Test Genie workflow levers. Use them when you are changing agent-execution or containment policy, not when debugging normal suite execution.

### Realtime note

Agent run streaming comes from `agent-manager`, not from a native Test Genie WebSocket endpoint. Test Genie exposes `/api/v1/agents/ws-url` so the UI can discover that external socket when agent workflows are enabled. Because that channel is optional infrastructure, the scenario's core integration phase should not treat it as a required runtime surface.
