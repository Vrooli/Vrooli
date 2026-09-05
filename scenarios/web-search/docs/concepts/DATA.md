# Data — Web Search

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

> **Scaffold status (2026-06-09):** The schema below is the *intended*
> data model from `PRD.md` / requirements (OT-P0-005, OT-P1-006). It is
> not yet implemented; the only on-disk schema is the template `notes`
> example, which will be removed.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

web-search persists durable data in its **own embedded SQLite** database
(`${SCENARIO_DATA_DIR}/web-search.db`, applied on startup
via `api-core/database`) for findings/briefs metadata, and in a **Qdrant
collection** (`web-search-findings`) for the semantic index, written
through the `aisearch-go` package (the cli-health adoption pattern).

Two deliberate non-storage decisions:

- **No durable web corpus.** Live web results are a passthrough; the only
  live-web persistence is the ephemeral TTL **cache** (eviction by TTL,
  not a corpus). External pages are never stored wholesale.
- **No writes into knowledge-observatory.** The learnings store is owned
  exclusively by web-search. It is a peer corpus, not a contribution to
  KO's curated documentation. (See [`../internal/DECISIONS.md`](../internal/DECISIONS.md).)

External storage resources (Qdrant) are introduced because the findings
domain genuinely needs semantic recall — documented in
[`INTEGRATIONS.md`](INTEGRATIONS.md).

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Findings | findings | SQLite (`findings` table) | `api/internal/findings/schema.sql` (planned) | Never hard-deleted; superseded rows archived in place | Atomic cited claim; the indexed unit. |
| Findings semantic index | findings | Qdrant (`web-search-findings`) | Derived from SQLite via aisearch-go reconcile | Rebuilt on drift; mirrors active+disputed findings | Default query filter excludes `superseded`. |
| Briefs | findings (produced by research) | SQLite (`briefs` table) | `api/internal/findings/schema.sql` (planned) | Retained with their findings for provenance | One research run; holds many findings. |
| Citations | findings | SQLite (`finding_citations` table or embedded JSON) | same | Same lifecycle as parent finding | URL + title + retrieved-at per source. |
| Finding audit log | findings | SQLite (`finding_audit` table) | same | Append-only; retained for auditability | what/why/which-brief on every mutation. |
| Live-web result cache | livesearch | SQLite or in-memory (TTL'd) | n/a (derived from SearXNG) | TTL eviction only | Dampens repeated external queries. |
| Budget governor state | livesearch | in-memory token bucket | n/a | Per-window, not durable | Caps external request rate. |
| Provider descriptors | federation | `.vrooli/search.json` (file) | the file itself | Versioned with the repo | web-search.live + web-search.learnings. |
| Usage/effectiveness telemetry *(P2)* | curation | SQLite (planned) | n/a | TBD | Deferred (OT-P2-001). |

## Schema Map

| Table/File/Object | Owner | Defined In (planned) | Used By |
|---|---|---|---|
| `findings` | findings | `api/internal/findings/schema.sql` | findings repository/service/handlers; research capture |
| `briefs` | findings | `api/internal/findings/schema.sql` | research run orchestration; findings provenance |
| `finding_citations` | findings | `api/internal/findings/schema.sql` | finding read/projection |
| `finding_audit` | findings | `api/internal/findings/schema.sql` | every finding mutation |
| `web-search-findings` (Qdrant) | findings | aisearch-go collection spec | semantic recall |
| live cache | livesearch | `api/internal/livesearch/` | L0/L1 query path |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot |

### Finding (intended shape)

| Field | Type | Notes |
|---|---|---|
| `id` | text (uuid) | Stable id. |
| `claim` | text | The cited claim (the embedded/indexed body). |
| `citations` | json/rel | One or more `{url, title, retrieved_at}`. |
| `retrieval_date` | timestamp | When the supporting evidence was gathered. |
| `confidence` | real [0,1] | From distillation; drives gating + decay. |
| `status` | enum | `active` / `disputed` / `superseded`. |
| `brief_id` | text (fk) | Originating research run. |
| `query` | text | The query/sub-question that produced it. |
| `superseded_by` | text (fk, nullable) | Set when archived by a newer finding. |
| `dispute_note` | text (nullable) | Why flagged; surfaced with the warning. |
| `created_at` / `updated_at` | timestamp | Standard. |

## Migrations And Compatibility

Schema bootstrap is idempotent (`CREATE TABLE IF NOT EXISTS`), applied on
startup. **Existing-table column changes are always one-shot migrations —
never recreate the DB or treat learnings data as disposable.** (Vrooli
SQLite policy: migrate, never recreate; `ADD COLUMN IF NOT EXISTS` is
Postgres-only, so guard SQLite column adds in code.) Record any column
drop/rename/backfill plan here and the tradeoff in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Manual finding add | CLI/API input | findings | Shipped (OT-P0-006) — operator-authored finding + citation via `findings add`. |
| Findings export | JSONL of finding + citations + audit (proposed) | findings | DEFERRED by decision (2026-06-10, [`DECISIONS.md`](../internal/DECISIONS.md)). Seam: `Repository.LoadIndexable` already yields every non-superseded finding; the audit table is append-only. A future P2 export composes those two reads — no schema change needed. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Findings | Never hard-deleted | Supersede → archived in place (recoverable, auditable) | Long-term: P2 GC may prune permanently-decayed/superseded rows. |
| Superseded findings | Reconcile / explicit supersede | Excluded from default search; retrievable via `--include-archived` | — |
| Live cache | TTL expiry | Ephemeral | — |
| Audit log | Never | Append-only | Growth managed by P2 GC. |

## Privacy Notes

Findings are distilled from public web content but carry source URLs and
retrieval dates. They are **unvetted external knowledge** — surfaced with
provenance and (when conflicting) dispute warnings, never presented as
curated fact. Query strings sent to SearXNG should follow the privacy
posture in [`../internal/SECURITY.md`](../internal/SECURITY.md) (SearXNG
itself is privacy-respecting and local). Do not persist user PII in
findings or briefs.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
