# Data — Device Control

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

The template default is embedded SQLite through `modernc.org/sqlite`.
The database path is resolved from the scenario id by `api-core/storage`, and
the API applies schemas on startup through `api-core/database`.

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

Each domain owns its own tables and is the source of truth for its data.
The `health` domain owns no product data — it only probes configured
database reachability. As you build real domains, add a row per data
shape they persist: name it, name the owning domain, the storage backend,
the schema file that is the source of truth, the retention rule, and any
remarks. Keep blob/opaque bytes outside proto payloads, behind a seam
such as BlobStore.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Device records | devices | SQLite | `api/internal/devices/schema.sql` | Until the device leaves the bridge fleet. | Mirrors bridge fleet identity; bridge remains the source of truth for identity and trust. |
| Capability snapshots | devices | SQLite | `api/internal/devices/schema.sql` | Rolling window per device; latest snapshot always retained. | Records what was *probed*, never what the device kind usually has. |
| Authentication profiles | auth | SQLite | `api/internal/auth/auth.go` | Metadata retained for audit; authority-held values are deleted or rotated separately. | Stores device binding, method, policy, status, and credential-authority reference only; never a secret value. |
| Strategy registrations | strategies | SQLite | `api/internal/strategies/schema.sql` | Until the strategy is unregistered. | Declaration only; the verified tier comes from a conformance result. |
| Conformance results | strategies | SQLite | `api/internal/strategies/schema.sql` | Latest per strategy plus history for trend. | The authority for what a strategy can actually do. |
| Lease records | sessions | SQLite | `api/internal/sessions/schema.sql` | Retained after expiry for audit. | Expired leases are never deleted eagerly — they are the audit trail for who held a device when. |
| Verb audit | sessions | SQLite | `api/internal/sessions/schema.sql` | Long-lived; governed retention. | Actor, device, lease, verb, outcome. Append-only. |
| Saved flow definitions | flows | SQLite | `api/internal/flows/library.go` | Until deleted by the owner; versioned. | The durable library stores validated flow revisions and promotion receipts. Includes declared capability requirements so a flow can be checked before a run. |
| Ordinary device-flow results, run chapters, and replay exports | flows/control | Service memory; owner-selected runs in routed SQLite | `api/internal/control/execution.go`, `api/internal/flows/retained_runs.sql` | Process-local by default; new explicit records expire at the owner-selected deadline. | Retention is opt-in: full flow targets, arguments and result messages may be private. Clipboard steps are refused. Retention does not copy artifact bytes. Existing process-local runs are not retroactively persisted. |
| Desktop flow runs | flows | SQLite | `api/internal/flows/desktop_runs.sql`, `desktop_runs.go` | Retained in the desktop run store; governed retention is not specified here. | This persistence path is separate from ordinary Android flow execution. |
| Capture artifacts | flows | Routed filesystem, producer-owned | `api/internal/control/flow_helpers.go` | Governed by artifact/run retention; bytes remain outside SQLite. | **Bytes never enter the database or a consumer payload.** Consumers receive `common/v1` `EvidenceRef` only. Artifact IDs resolve to files after process restart, while artifact-kind metadata is currently held in memory. |
| Visual anchors | flows | Filesystem + SQLite metadata | `api/internal/flows/schema.sql` | Until the owning flow or anchor is deleted. | Reference images with stable identity for the deterministic middle rung. |
| Agent run records | agent | Service memory for the current process | `api/internal/control/agent.go` | Until the scenario restarts. | Includes typed chapters and executable planned steps. Promotion places a deterministic run/export in the ordinary process-local device-flow maps; a separately saved validated flow revision is durable in SQLite. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| Device records, capability snapshots | devices | `api/internal/devices/schema.sql` | devices repository/service/handlers; inventory probe |
| Strategy registrations, conformance results | strategies | `api/internal/strategies/schema.sql` | strategies repository/service/handlers; `strategy verify` |
| Lease records, verb audit | sessions | `api/internal/sessions/schema.sql` | sessions repository/service/handlers; lease enforcement, kill switch |
| Saved flow definitions, desktop runs, anchor metadata | flows | `api/internal/flows/library.go`, `desktop_runs.sql`, and anchor schema bootstrap | flow library and desktop-run repositories; flow handlers |
| Ordinary device-flow run results and replay exports | flows/control | `Service.runs`, `Service.flowRuns`, `Service.runDevices`, and `api/internal/flows/retained_runs.sql` | Process-local unless explicitly retained in the flows repository. |
| Agent run records, promotion provenance | agent | `api/internal/control/agent.go` | agent service/handlers; run-to-flow promotion |
| Capture artifacts, anchor images | flows | Filesystem, producer-owned; referenced by checksum | Never a table. Consumers receive `common/v1` `EvidenceRef` only. |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

## Migrations And Compatibility

The generated template uses idempotent schema bootstrap. Domain schema
files should use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| `GET /api/v1/flows/{id}/export` | JSON replayable flow and rung metadata | control | Available for a process-local or explicitly retained run. |
| `GET /api/v1/flows/runs` and `GET /api/v1/flows/retained` | IDs and current active-flow count | control | Point-in-time IDs of cached completed entries and retained records; preflight/auth failures can return without entering the cache. It does not block new admission or account for all process state. It is not a safe-restart proof. |
| `POST/GET/DELETE /api/v1/flows/{id}/retained` | Full run result and flow | flows | POST requires `acknowledge_sensitive: true` and a future RFC3339 `expires_at`; CLI `flow retain` requires `--confirm-sensitive --expires-at <timestamp>`. Capture returns the exact deadline; list and get expose it. At or after expiry, the next get or list removes the database row; no background task runs, so an idle expired row can remain until accessed. Existing records without a deadline remain unchanged and show `expires_at: null`. Delete and expiry remove the durable row, not current process cache, artifact files, backups, or SQLite WAL remnants. |
| Saved-flow library list/get | JSON versioned flow definitions and validation receipt | flows | Durable SQLite-backed owner API. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Capture artifacts (screen frames, recordings) | Run retention expiry, or explicit owner deletion. | Governed retention; runs referenced by a delivery-ramp release are pinned and exempt until the release is retired. | **High-sensitivity owner data.** Authorized local capture is unredacted by default; evidence records whether an optional configured transform ran and that semantic content classification was not performed. External sharing and publication follow separate approval and classification rules. |
| Verb audit | Never deleted by product behavior. | Long-lived, governed retention. | Audit is the accountability record for a capability that can drive a personal device; truncating it silently would defeat its purpose. |
| Lease records | Retained after expiry. | Governed retention alongside audit. | None; expiry releases the device but keeps the record. |
| Capability snapshots | Superseded by the next probe; history trimmed to a rolling window. | Latest snapshot per device always retained. | None. |
| Authentication profiles | Profile revocation; delete or rotate the authority-held credential separately. | Metadata remains for audit and troubleshooting. | Credential values are never owned by device-control and must not be copied into SQLite. |
| Device records | Device leaves the bridge fleet. | Removed with the fleet member. | Bridge owns the lifecycle; this scenario must not resurrect a removed device from a stale snapshot. |
| Saved flow definitions | Owner deletion. | Versioned definitions remain in SQLite until deleted. | None. |
| Ordinary device-flow results and replay exports | Restart for the process-local copy; owner deletion for an explicitly retained record. | Only explicitly retained runs survive replacement; newly selected records expire on access at the owner-selected deadline. | Expiry and deletion do not clear the current process maps, artifact files, SQLite backups or WAL copies; idle expired rows remain until accessed. An inventory read is not a drain or restart fence; audit rows and artifact files alone cannot reconstruct a run. |
| Desktop flow runs | Governed run retention. | Persisted through `SQLiteDesktopRuns`. | Retention policy is not specified here yet. |

An external recording stop can return HTTP 409 `recording_stop_uncertain` with
`recovered_recording` when video was processed through the normal evidence path
but the recorder reported a separate failure, such as restoration failure. The
owner preserves the uncertain status and recovered reference in process memory
and refuses automatic stop replay. A repeat request requires a valid device lease. The evidence file uses the existing capture
policy; that policy is not a general secret detector. The status and artifact-kind
metadata do not survive restart, and the remote encoder's termination is not
proven by a local ADB wrapper exit. A failed stop with no recoverable video also
remains uncertain; do not treat it as successful cleanup.

## Privacy Notes

## Desktop session data

Desktop sessions retain only bounded metadata: session id, device/display,
controller lease epoch, readiness, transport mode, timestamps, and terminal
outcome. Video frames, pointer/keyboard events, clipboard text, WebRTC
credentials, and native permission material are transient and are never
written to the scenario database or logs. Evidence records may state that a
clipboard/input/capture assertion passed without copying its contents.

Generated template data is local development data. If a scenario stores
personal, regulated, customer, financial, or sensitive business data,
update this document and [`../internal/SECURITY.md`](../internal/SECURITY.md)
before implementation expands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
