# Data — Channel Manager

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
The lifecycle sets `SQLITE_PATH` through `.vrooli/service.json`, and
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

**No credential storage.** This scenario reads credentials from the `vault`
resource at execution time and persists only a vault path. There is no encrypted
column, no keyring, and no "temporary" cache. A schema change introducing a
credential-shaped column is a defect, not a new feature — and
`CHANMGR-P0-002` asserts it in the suite rather than trusting review.

**No blob storage.** Evidence attached to a manually completed action is a URL or a
short text note. If screenshot bytes are ever needed they belong behind a BlobStore
seam, added deliberately with a decision behind it.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Platform descriptor | platforms | JSON file, seeded to SQLite | `data/platforms/<platform>.json` | Versioned in git. | Formats, limits, cadence ceilings, media constraints, disclosure rules, executor availability. The file is authoritative; the table is a cache. |
| Warming program | warming | JSON file, seeded to SQLite | `data/warming-programs/<platform>/<id>.json` | Versioned in git. | Preconditions, session policy, phases, gates, graduation, maintenance, portfolio constraints, provenance. |
| Program observation | warming | SQLite | `api/internal/warming/schema.sql` | **Never deleted.** | Append-only outcomes of real program runs. The path from speculative defaults to measured ones; a revision must never erase the evidence that motivated it. |
| Identity | identities | SQLite | `api/internal/identities/schema.sql` | Until retired; retirement is a status, not a delete. | Platform, handle, purpose tag, persona ref, environment ref, vault path, lane grants, lifecycle status. |
| Environment attestation | identities | SQLite | `api/internal/identities/schema.sql` | Follows its identity. | Which preconditions were attested, by whom, when. Recorded, not verified — see `ARCHITECTURE.md` § Intentional Deviations. |
| Lane grant | identities | SQLite | `api/internal/identities/schema.sql` | Until revoked or the identity is quarantined. | The output of graduation and the input to the eligibility query. |
| Warming plan and rolls | warming | SQLite | `api/internal/warming/schema.sql` | Follows its identity. | Generated schedule with each resolved count and time plus the seed that produced them. |
| Phase and gate state | warming | SQLite | `api/internal/warming/schema.sql` | Follows its identity. | Current phase, gate evaluations and their measurements, graduation or quarantine outcome. |
| Queued action | queue | SQLite | `api/internal/queue/schema.sql` | Retained after execution. | Identity, action kind, parameters, session, window, executor, status. |
| Session | queue | SQLite | `api/internal/queue/schema.sql` | Follows its actions. | Grouping with start, duration, and the gap constraint it satisfied. |
| Action record | queue | SQLite | `api/internal/queue/schema.sql` | **Never deleted.** | What actually happened: executor, timestamps, outcome, evidence reference. Identical shape across all three executors. |
| Release record | queue | SQLite | `api/internal/queue/schema.sql` | **Never deleted.** | Draft ref, idempotency key, post id, URL. The idempotency key is uniquely indexed — that index *is* the retry guarantee. |
| Metric observation | signals | SQLite | `api/internal/signals/schema.sql` | **Never deleted.** | Per-identity reach, impressions, engagement, follower delta, audience geo share, with the time observed and how it was obtained. |
| Baseline | signals | SQLite | `api/internal/signals/schema.sql` | Rebuildable. | Rolling per-metric baseline. Safe to drop and recompute from observations. |
| Flag | signals | SQLite | `api/internal/signals/schema.sql` | Until resolved by an operator. | The evidence that raised it and the pause it caused. Never a verdict — see `DOMAINS.md` § signals. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| identities, environments, attestations, lane_grants | identities | `api/internal/identities/schema.sql` | identities repository/service; read by warming, queue, signals |
| platform_descriptors | platforms | `api/internal/platforms/schema.sql` | seeded from `data/platforms/`; read by queue for ceilings and by warming for validation |
| warming_programs, plans, plan_rolls, phase_state, gate_evaluations, observations | warming | `api/internal/warming/schema.sql` | warming service and its scheduled pass; read by queue for phase guards |
| queued_actions, sessions, action_records, release_records | queue | `api/internal/queue/schema.sql` | queue service, executors, and the release handoff |
| metric_observations, baselines, flags | signals | `api/internal/signals/schema.sql` | signals service; read by warming for gates and by queue for pause state |
| `data/platforms/*.json` | platforms | scenario `data/` | boot seeding and descriptor validation |
| `data/warming-programs/**/*.json` | warming | scenario `data/` | boot seeding and descriptor validation |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

### Rebuild contract

`baselines` are a cache and can be recomputed from `metric_observations` at any
time. Nothing else here is rebuildable: action records, release records,
observations, and metric observations are the evidence of what happened to a real
account, and there is no second copy anywhere.

Descriptors run the other way — the JSON file is authoritative and the seeded table
is disposable. Reseeding replaces rows rather than duplicating them, so an edited
descriptor is applied by reseeding and never by an UPDATE against the cache.

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
| Descriptor seeding | JSON → SQLite | platforms, warming | Recurring and idempotent. Reseeding replaces cached rows; it never duplicates. |
| Predecessor migration | — | — | **None.** `social-media-scheduler` was retired 2026-07-28 without migration: it had not compiled since 2025-09-08 and held no production data. |
| Export | — | — | None planned. `--json` on the read verbs is the export surface. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Identity | None. Retirement is a status. | Permanent, so that action history stays attributable. | None. |
| Environment attestation | None. Re-attestation appends. | Permanent, with history. | Attestations are unverifiable by construction; see `ARCHITECTURE.md`. |
| Action record, release record | None. | Permanent — the evidence of what was done as this identity. | None. |
| Metric observation | None. | Permanent; baselines depend on the series. | None. |
| Program observation | None. | Permanent, append-only. | None. |
| Baseline | Recompute. | Rebuildable cache. | None. |
| Flag | Operator resolves it. | Until resolved. | Margin and run-length thresholds are unvalidated (`CHANMGR-P0-017`). |
| Queued action | Cancelled by quarantine or pause. | Cancellation is a status, not a delete. | None. |
| Credential | n/a | **Never stored here.** Lives in `vault`. | None. |

Nothing in this scenario deletes the record of an action taken as a real account.
If a retention requirement ever demands it, that is a change to the scenario's
accountability posture and belongs in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md), not in a migration.

## Privacy Notes

This scenario stores **account handles**, which marketing canon deliberately keeps
out of `docs/marketing/` and routes here, and it references credentials held in
`vault`. It also stores audience-composition metrics that are aggregate figures
reported by the platform, never individual audience-member data.

Two rules follow. Handles and vault paths must not appear in log output at default
verbosity. And no endpoint returns a credential value under any circumstance —
including a debug or admin path, of which there are none.

See [`../internal/SECURITY.md`](../internal/SECURITY.md) for the full posture.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
