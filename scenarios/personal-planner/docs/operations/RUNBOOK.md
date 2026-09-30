# Runbook — Personal Planner

## Purpose Of This Document

This document is the operator's guide to running, diagnosing, recovering,
and maintaining Personal Planner. It is design-stage: the only real
running code today is the `/health` endpoints and the worked-example
`notes` domain, so the incident, backup, and maintenance procedures below
describe the **intended** operations for the planned domains and
background jobs, expressed against the standard Vrooli lifecycle. Where a
procedure names a job or domain that is not yet implemented, treat it as
the operating contract to build toward, not a live capability.

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What do I check when a dependency, provider, job, or port misbehaves?
- How do I back up state and restore it safely into a clean workspace?
- What routine maintenance keeps jobs and storage healthy?
- Where do operational defects get filed?

## Start / Stop / Status

Run everything through the lifecycle from the scenario directory. **Never
run the API or UI binaries directly** — the lifecycle owns process
naming, port allocation, health checks, and logs, and running a binary by
hand bypasses all of it.

```bash
make setup     # wraps `vrooli scenario setup`  — build + install deps/CLI
make start     # wraps `vrooli scenario start`  — start API + UI + resources
make status    # wraps `vrooli scenario status` — running surfaces and ports
make logs      # wraps `vrooli scenario logs`   — tail API + UI logs
make stop      # wraps `vrooli scenario stop`   — clean shutdown
make restart   # stop then start (preferred over restarting one surface)
make test      # wraps `vrooli scenario test`   — validation lifecycle
```

Equivalent control-plane forms (`vrooli scenario start|stop|status|test
personal-planner`) are fine. Resolve a live port with
`vrooli scenario port personal-planner API_PORT` (or `UI_PORT`). Both
surfaces expose `/health`.

## Common Incidents

| Symptom | Checks | Response |
|---|---|---|
| **Scenario will not start** | `make status`, `make logs`; is scenario-authenticator running? | scenario-authenticator is **required** (`must_start`). Without an identity provider a workspace cannot be scoped to a subject, so Personal Planner refuses to start by design. Restore the authenticator, then `make restart`. This is expected fail-closed behavior, not a Personal Planner bug. |
| **notification-hub down/degraded** | `make status` for the dependency; outbox age (see Maintenance) | Notices remain **in-app only**; no external channel delivery. The product stays fully usable. Delivery is retried idempotently on recovery and must never produce duplicate human-facing messages. No action needed beyond restoring the hub; do not disable the notifications feature. |
| **External provider outage / rate limit** | integration health panel: last successful refresh, current problem, next retry, affected scope; provider freshness metric | Bounded exponential backoff with jitter runs automatically. Last-known busy intervals are **retained with a freshness warning**; feasibility is marked uncertain. **Never treat a provider failure as newly free time.** No manual action unless the outage persists past the backoff horizon. |
| **Provider reauth required** | `PROVIDER_REAUTH_REQUIRED` surfaced in the health panel; credential owner state | Token expired/revoked. Reconnect through the provider connection flow; stale busy facts are retained meanwhile. Do not paste tokens into logs or configs — credentials live only in the credential owner. |
| **Provider cursor invalidated** | sync failure logs; cursor state for the connection/calendar/query scope | A **scoped full resync** rebuilds a replacement snapshot and is switched in only after success; native tasks/schedules/history are untouched. A moved recurring instance keeps its stable original-instance key. |
| **Stale forecast** | forecast age metric; whether a material input revision occurred | Forecast refresh coalesces on material input revisions and writes an immutable snapshot (no accepted-schedule mutation). If a forecast is older than expected after a relevant edit, confirm the forecast-refresh job is leased and advancing; a stale forecast is a signal, never a reason to auto-change an accepted schedule. |
| **Stuck job lease** | job status: attempts, next retry, last error class, terminal unresolved state; pending outbox age; cursor advancement | Jobs use leases with observable statuses; a crashed worker must **not** strand a permanent active lock. If a lease is held with no progress, the recovery is lease expiry/reclaim, not a manual DB edit. Watch for tight loops on invalid credentials or unsupported source commands — those must back off, not spin. |
| **Port conflict** | `make status`; a previous instance still holding `API_PORT`/`UI_PORT` | `make restart`. The lifecycle reallocates ports; do not pin a port by hand. |
| **API unhealthy** | `/health` on `API_PORT`; SQLite path writable; API logs via `make logs` | `make setup` to rebuild, verify the data directory is writable. On a SQLite write failure the command returns a **failed** state and preserves user input with retry/export — it never claims saved. |
| **UI blank or stale** | `/health` on `UI_PORT`; browser console; `ui/dist` freshness | `make setup` then `make restart` to rebuild the bundle. |
| **CLI talks to an old API** | `personal-planner status`; resolved API base | Reinstall via `make setup` so the CLI picks up the current resolved port and token. |

## Backup / Restore

- **Backup — native versioned JSON export.** Export produces a
  self-describing package: format version, export time, timezone/policy
  metadata, stable IDs, provenance, native records, allocations,
  meaningful revisions, and actuals — enough to interpret and restore. It
  **excludes** credentials, access/session tokens, reusable share
  secrets, and unnecessary private infrastructure IDs. CSV summaries are
  for actuals/estimates; native backup exports carry the selected history
  needed for restoration and label that choice. Downloadable export
  artifacts are authenticated and regenerable (24-hour default retention).
- **Restore — validated native import into a new workspace.** Import
  validates the whole package first (counts, conflicts, warnings, and a
  preview) before writing, defaults to a **new private workspace**, and is
  **idempotent** — reimporting the same package does not duplicate
  objects.
- **Restored state is inactive / needs-review, by design:**
  - Restored **provider connections** require reconnection (no tokens are
    in the export).
  - Restored **share grants** are inactive until re-granted.
  - A restored **running focus session** becomes "needs review," never
    assumed-productive elapsed work.
- **ICS import/export** is a separate manual fallback, not a live
  connection: import previews with duplicate detection, provenance, and
  bounded recurrence expansion, and never fetches URLs embedded in
  descriptions; export never routes private notes through a
  commitment-share path.
- Always **verify a restore before any cutover** (see DEPLOYMENT
  Rollback): a snapshot you have not test-restored is not a backup you can
  rely on.

## Maintenance Tasks

| Task | Cadence | Procedure / What to check |
|---|---|---|
| Cursor health | ongoing | Confirm each provider connection/calendar/query cursor is advancing only after a batch is durably processed; a stalled cursor points at a sync failure or invalidation. |
| Outbox / receipt age | ongoing | Watch **pending outbox age**; records are retained until safely delivered. Growing age means delivery is stuck (often notification-hub or a job lease). Replays return the prior outcome or safely resume — never a duplicate message. |
| Job status review | ongoing | For each job (provider sync, occurrence expansion, forecast refresh, reminder dispatch, source command retry, learning analysis, share maintenance, cleanup): check attempts, next retry, last error class, and terminal unresolved state; ensure none spin on invalid credentials or unsupported source commands. |
| Cleanup / retention | scheduled | Cleanup preserves user-authored history and bounds transient job/log data per the retention defaults in [`../concepts/DATA.md`](../concepts/DATA.md). Source-receipt cleanup must **never** let an old replay resurrect deleted work — durable origin uniqueness/tombstones are kept. |
| Validate tests | before handoff / release | `make test`; the R1 acceptance suite (T01–T36) is the authoritative evidence. |
| Inspect logs | as needed | `make logs`. |

## Escalation

File operational defects as bug observations to **scenario-qa** using the
report-bug skill (`prompt-manager skill read report-bug`) — it is a skill,
not a shell command, and it writes a bug-inbox entry for the
bug-investigator to drain. Do not carry a private host-repair
implementation in this scenario: detection and remediation of host state
belong to the control plane; Personal Planner may observe and report host
state but must not repair it. Append meaningful completed operational work
to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — tiers, release checklist, rollback
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — signals, logs, metrics, alerts
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency contract and failure modes
- [`../concepts/DATA.md`](../concepts/DATA.md) — storage, import/export, retention, deletion
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common first-boot fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
