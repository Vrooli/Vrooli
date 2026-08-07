# Data — Channel Manager

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

The template default is embedded SQLite through `modernc.org/sqlite`.
The lifecycle sets `SQLITE_PATH` through `.vrooli/service.json`, and
the API applies schemas on startup through `api-core/database`.

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

`channelmanager` owns one coherent in-memory state projection. SQLite persists
that projection as one JSON document in `channel_manager_state`; it is not a
collection of per-domain tables. Platform and warming descriptors are loaded from
versioned JSON files and are deliberately excluded from that runtime record.

**No credential storage.** This scenario reads credentials from the credential
authority at execution time and persists only an authority reference. There is no encrypted
column, no keyring, and no "temporary" cache. A schema change introducing a
credential-shaped column is a defect, not a new feature — and
`CHANMGR-P0-002` asserts it in the suite rather than trusting review.

**No blob storage.** Evidence attached to a manually completed action is a URL or a
short text note. If screenshot bytes are ever needed they belong behind a BlobStore
seam, added deliberately with a decision behind it.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Platform descriptor | descriptor files | JSON | `data/platforms/*.json` | Versioned in git. | Formats, post types, preview-fit rules, local cadence policy, disclosure rules, retry policy, and an official-source provenance record. Runtime-resolved limits are intentionally not guessed. |
| Warming program | descriptor files | JSON | `data/warming-programs/*.json` | Versioned in git. | Preconditions, sessions, phases, gates, graduation, and provenance. |
| Operator state | channelmanager | SQLite JSON document | `api/internal/channelmanager/sqlite.go` | See state fields below. | The single `channel_manager_state` row is a durable projection, not a cache. |
| Account activity ledger | channelmanager | `State.ActivityEvents` in the SQLite JSON document | `api/internal/channelmanager/ledger.go` | Permanent. | Append-only, chronological and redacted facts for lifecycle, queueing, BAS dispatch, manual verification, releases, and metrics. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `channel_manager_state` | channelmanager | `api/internal/channelmanager/sqlite.go` | `Store.Load` / `Store.Save` |
| `data/platforms/*.json` | platforms | scenario `data/` | boot seeding and descriptor validation |
| `data/warming-programs/*.json` | warming | scenario `data/` | boot loading and descriptor validation |

### Rebuild contract

The current runtime projection is restored as a whole. Individual field semantics
still matter: actions, releases, observations, metric samples, and program outcomes
are retained because they are account evidence; identity retirement is a status,
not a delete. Descriptor files are immutable inputs to each service start.

## Migrations And Compatibility

This is greenfield state. Changes must preserve the documented state contract
without introducing in-process relocation, backfill, or compatibility logic.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Descriptor loading | JSON → service | platforms, warming | Immediate JSON files are validated at boot. |
| Export | — | — | None planned. `--json` on the read verbs is the export surface. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Identity | None. Retirement is a status. | Permanent, so that action history stays attributable. | None. |
| Environment attestation | None. Re-attestation appends. | Permanent, with history. | Attestations are unverifiable by construction; see `ARCHITECTURE.md`. |
| Action record, release record, activity ledger event | None. | Permanent — the evidence of what was done as this identity. | The ledger stores only redacted artifact references; raw BAS artifacts remain BAS-owned. |
| Metric observation | None. | Permanent; later analyses use the series. | None. |
| Program observation | None. | Permanent, append-only. | None. |
| Flag | Operator resolves it. | Until resolved. | Margin and run-length thresholds are unvalidated (`CHANMGR-P0-017`). |
| Queued action | Cancelled by quarantine or pause. | Cancellation is a status, not a delete. | None. |
| Credential | n/a | **Never stored here.** Lives in the credential authority. | None. |

Nothing in this scenario deletes the record of an action taken as a real account.
If a retention requirement ever demands it, that is a change to the scenario's
accountability posture and belongs in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md), not in an implicit
storage rewrite.

## Privacy Notes

This scenario stores **account handles**, which marketing canon deliberately keeps
out of `docs/marketing/` and routes here, and it references credentials held in
the credential authority. It also stores audience-composition metrics that are aggregate figures
reported by the platform, never individual audience-member data.

Two rules follow. Handles and authority references must not appear in log output at default
verbosity. And no endpoint returns a credential value under any circumstance —
including a debug or admin path, of which there are none.

See [`../internal/SECURITY.md`](../internal/SECURITY.md) for the full posture.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
