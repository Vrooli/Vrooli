# Data — Architecture Cartographer

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

Cartographer stores all data in local SQLite via `modernc.org/sqlite`.
The lifecycle sets `SQLITE_PATH` through `.vrooli/service.json`; the
API applies schemas on startup through `api-core/database`. **No
external storage resource is added in v1.** This is a deliberate
choice — the cartographer must remain runnable against a single
scenario directory with no infrastructure prerequisites beyond the
two language-graph dependency scenarios.

Data falls into five categories:

1. **Graph snapshots** — keyed by `(target_scenario, content_hash)`,
   immutable once written. Cached so repeated extracts on unchanged
   code are free.
2. **Manifest cache** — parsed manifest forms; invalidated by
   manifest file mtime + content hash.
3. **Conflict records** — typed Conflict envelopes; lifecycle from
   `detected` → `resolved` → applied-and-committed.
4. **Apply history** — every `arch-cart apply <domain>` invocation,
   the operations it executed, build-status baseline and post, force
   notes if any.
5. **Analytics event log** — append-only event stream covering
   detection, resolution, auto-placement, override, and build deltas
   (see `analytics` domain in [`DOMAINS.md`](DOMAINS.md)).

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Graph snapshots | graph | SQLite `graph_snapshots` table + JSON blob column | `api/internal/graph/schema.sql` | Indefinite by default; per-scenario reset via `arch-cart graph clear`. | Keyed by `(scenario, content_hash)`; many snapshots per scenario over time. |
| Manifest cache | manifest | SQLite `manifest_cache` table | `api/internal/manifest/schema.sql` | Invalidated on manifest file mtime/hash change. | Cache only — manifest source of truth is the target scenario's manifest file. |
| Conflict records | conflicts | SQLite `conflicts` table | `api/internal/conflicts/schema.sql` | Retained per scenario migration; cleared when migration is finalized. | Conflict envelope shape stable across versions. |
| Resolution log | conflicts | SQLite `conflict_resolutions` table | `api/internal/conflicts/schema.sql` | Same lifecycle as conflict records. | Includes resolution method, agent note, --force note. |
| Apply history | apply | SQLite `apply_runs` and `apply_operations` tables | `api/internal/apply/schema.sql` | Indefinite; growth bounded by migration cadence. | Each row records a `(scenario, domain, operations, baseline_status, post_status)`. |
| Analytics event log | analytics | SQLite `events` append-only table | `api/internal/analytics/schema.sql` | Indefinite. Volume small (hundreds of rows per migration). | Source data for `arch-cart history` and `arch-cart stats`. |
| Auto-placement verdicts | analytics (via signals) | SQLite `placements` table | `api/internal/analytics/schema.sql` | Same lifecycle as analytics events. | Includes signal scores, aggregator verdict, override flag if agent overrode. |
| Notes | notes (template placeholder) | SQLite | `api/internal/notes/schema.sql` | Removed when notes domain is removed per Gate 7. | Will be deleted; not product scope. |

## Schema Map

| Table / Object | Owner | Defined In | Used By |
|---|---|---|---|
| `graph_snapshots` | graph | `api/internal/graph/schema.sql` | graph extract/show, conflict detection |
| `manifest_cache` | manifest | `api/internal/manifest/schema.sql` | manifest validate/show, conflict detection |
| `conflicts` | conflicts | `api/internal/conflicts/schema.sql` | conflict list/show/assign/resolve, apply |
| `conflict_resolutions` | conflicts | `api/internal/conflicts/schema.sql` | conflict resolve, analytics |
| `apply_runs` | apply | `api/internal/apply/schema.sql` | apply, history |
| `apply_operations` | apply | `api/internal/apply/schema.sql` | apply, history |
| `events` | analytics | `api/internal/analytics/schema.sql` | history, stats, calibrate |
| `placements` | analytics | `api/internal/analytics/schema.sql` | signals explain, calibrate |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

## Migrations And Compatibility

Domain schema files use `CREATE TABLE IF NOT EXISTS` and live beside
the code that interprets them. For schema evolution:

- **Additive changes** (new columns with defaults, new tables, new
  indexes) are landed via `ALTER TABLE ... ADD COLUMN` migrations in
  the same `schema.sql` file, guarded by `pragma_table_info` checks.
- **Destructive changes** (column drops, renames, retypes, data
  backfills) require a scenario-specific migration plan in this doc
  and a corresponding [`../internal/DECISIONS.md`](../internal/DECISIONS.md)
  entry recording the tradeoff. Backfills must be idempotent.
- The **Conflict envelope shape stays stable across versions**. Adding
  detectors does not change the envelope — new fields are forbidden
  on `Conflict`; new optional fields on `Fix` are permitted.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| `arch-cart graph export <scenario>` | JSON (graph snapshot) | graph | Planned for P0 — used to ship a graph as a baseline artifact in PRs. |
| `arch-cart conflicts export <scenario>` | JSON | conflicts | Planned for P0 — used by CI to report drift. |
| `arch-cart history export` | JSON | analytics | Planned for P1 — feeds cross-scenario calibration analyses. |
| Manifest import | JSON / YAML | manifest | Read from target scenario; cartographer does not own manifest files. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Graph snapshots | `arch-cart graph clear <scenario>` or content hash superseded | Last N snapshots per scenario (default N=10, configurable in `.vrooli/service.json`). | Auto-eviction policy to be implemented with snapshot persistence (OT-P1-004). |
| Conflict records | `arch-cart migrate finalize <scenario>` | Until migration is finalized; finalization is explicit. | Long-running migrations could accumulate; mitigated by per-domain apply landing conflicts incrementally. |
| Apply history | None automatic | Append-only audit trail. | Considered durable history; do not delete. |
| Analytics event log | None automatic | Append-only audit trail. Volume small. | None — analytics is intentionally retain-all. |
| Manifest cache | Manifest file mtime/hash change, or `arch-cart manifest clear` | Stale cache invalidated on next access. | None. |

## Privacy Notes

Cartographer reads source code from target scenarios. Source code can
contain credentials, internal hostnames, customer identifiers, or
other sensitive material if a scenario has been carelessly committed.

- **Never log source code at info level.** Graph extraction extracts
  structural information (file paths, imports, symbol names) but not
  full source text. Source bodies fetched via `arch-cart conflict
  show` are streamed to the requesting agent and not persisted in
  analytics.
- **Analytics never stores source bodies** — only structural
  identifiers (file paths, symbol names, conflict types, severity,
  scores).
- **`arch-cart history export` strips evidence values** that contain
  arbitrary text (the `Evidence` field on a verdict may contain
  symbol names but never source bodies); a redaction pass is required
  before sharing exports outside the scenario maintainers.

If the cartographer is ever pointed at a scenario containing
regulated data (PHI, PII at scale, payment data), update this
document and [`../internal/SECURITY.md`](../internal/SECURITY.md)
before any analysis is run and confirm the privacy guardrails above
are appropriate for that data classification.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
