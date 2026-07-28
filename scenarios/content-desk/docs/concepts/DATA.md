# Data — Content Desk

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

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Campaign | campaigns | SQLite | `api/internal/campaigns/schema.sql` | Retained after close as historical record. | Theme, audiences, channels, hypotheses, linked SKUs, status, evidence refs. |
| Artifact slot | campaigns | SQLite | `api/internal/campaigns/schema.sql` | Follows its campaign. | Declared channel/format slots and their occupancy. The budget is a hard cap. |
| Draft | artifacts | SQLite | `api/internal/artifacts/schema.sql` | Retained; abandoned drafts are marked, not deleted. | Body, hook, type, lane, audience, channel, status, approval attribution. |
| Revision | artifacts | SQLite | `api/internal/artifacts/schema.sql` | Follows its draft. | Prior bodies, so an edit history survives review. Records whether a revision came from the operator or a commissioned agent. |
| Image attachment | artifacts | SQLite | `api/internal/artifacts/schema.sql` | Follows its draft. | **Reference and metadata only, never bytes** (D-018): an `image-tools` asset id, resolved path, role (`banner` or `inline`), declared aspect ratio, alt text, and position. Bytes live in image-tools. |
| Claim | claims | SQLite | `api/internal/claims/schema.sql` | **Never deleted.** | Assertion text, kind, verification state, search date for novelty claims. Shared across drafts. |
| Evidence | claims | SQLite | `api/internal/claims/schema.sql` | Follows its claim. | Either a citation or a re-runnable check with command, expected result, last run, last observed result. |
| Citation | claims | SQLite | `api/internal/claims/schema.sql` | Follows the draft. | The many-to-many join between drafts and claims. Deleting a draft never deletes a claim. |
| Post type | posttypes | SQLite | `api/internal/posttypes/schema.sql` | Seeded at boot; reseed overwrites. | Medium, paired-skill ref, required fields, activation criteria, failure-mode set. |
| Review run | review | SQLite | `api/internal/review/schema.sql` | Retained as history. | Per-failure-mode verdicts with evidence, plus challenge and resolution state. |
| Publish record | ledger | SQLite | `api/internal/ledger/schema.sql` | **Never deleted.** | Draft, channel, URL, platform post id, series, prior post, published timestamp. The draft reference is **nullable by design** — see D-012. |
| Coverage snapshot | ledger | SQLite | `api/internal/ledger/schema.sql` | Rebuildable. | Derived from publish records; safe to drop and recompute. |
| Subject mention | ledger | SQLite | `api/internal/ledger/schema.sql` | Follows its publish record. | Which subjects a published post introduced, per audience. |
| Narrated item | ledger | SQLite | `api/internal/ledger/schema.sql` | Follows its publish record. | What a post said about a subject, so later posts advance rather than repeat. |
| Import key | ledger | SQLite | `api/internal/ledger/schema.sql` | Permanent. | Content-addressed hash with a unique index; the mechanism behind idempotent import. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| campaigns, artifact_slots | campaigns | `api/internal/campaigns/schema.sql` | campaigns repository/service; read by artifacts for slot admission |
| drafts, revisions, requests, image_attachments | artifacts | `api/internal/artifacts/schema.sql` | artifacts repository/service; read by claims, review, and ledger |
| claims, evidence, citations | claims | `api/internal/claims/schema.sql` | claims policy engine; read by artifacts for the approval gate |
| post_types, activation_criteria, failure_modes | posttypes | `api/internal/posttypes/schema.sql` | posttypes validation; read by artifacts and review |
| review_runs, verdicts, challenges | review | `api/internal/review/schema.sql` | review service; read by artifacts for the approval gate |
| publish_records, coverage, subject_mentions, narrated_items, import_keys | ledger | `api/internal/ledger/schema.sql` | ledger reporting; written on publish and import |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships the `notes` domain as a worked CRUD slice with a
binary attachment-upload exception, showing how a real domain owns its
tables, metadata, and opaque blob bytes. Copy its shape, then remove it.

Its Data Ownership rows:

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Notes | notes | SQLite | `api/internal/notes/schema.sql` | Until deleted by future product behavior | Template reference data; remove with notes domain. |
| Attachment metadata | notes | SQLite | `api/internal/notes/schema.sql` | Until parent note or attachment is deleted by future product behavior | Metadata only; bytes are stored through BlobStore. |
| Attachment bytes | notes | Filesystem BlobStore by default | BlobStore implementation in notes handler module | Same lifecycle as metadata | Opaque bytes stay outside proto payloads. |

Its Schema Map row:

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| notes tables | notes | `api/internal/notes/schema.sql` | notes repository/service/handlers |

Its Retention And Deletion row:

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Template notes data | Domain removal or future product delete behavior | Local development data only | Real scenarios must define product-specific deletion semantics. |
<!-- EXAMPLE-DOMAIN:notes END -->

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
| `marketing-crew/shared/*.jsonl` — publish log, campaign drafts, audience scans, mentions, improvements | JSONL | ledger | `CONTENTD-P0-013` — recurring, idempotent |
| `docs/marketing/strategy/CAMPAIGNS.md` — campaign entries and their artifact slots | Markdown | campaigns | `CONTENTD-P0-013` — one-time seed; canon stays operator-curated |
| `docs/marketing/catalogs/post-types/**` — type registry seed | Markdown | posttypes | `CONTENTD-P0-008` — seeded at boot from canon |
| Export | — | — | None planned. The API and `--json` CLI output are the export surface. |

### Idempotency

Re-import must never duplicate. The mechanism is a **content-addressed import
key**, unique-indexed:

```
import_key = hash(source_file, normalized_item)
```

Re-running an import is a no-op for anything unchanged; that *is* the diff, and
it needs no watermark, cursor, or state file.

**Deliberately not positional.** Line offsets are not stable keys. Every one of
these sources is a file a human or an agent rewrites and reorders at will, and
the first reflow would re-import an entire file as new.

**Accepted consequence.** An item edited in place hashes differently and imports
as a new record rather than updating the old one. Under an append-oriented
ledger that is correct — the edit is a new assertion. Fuzzy-matching edits back
onto existing records is a meaningfully harder system whose failure mode is
silent false merges.

### Imported history has no draft

Every imported publish record predates this scenario, so no draft produced it.
The `ledger` domain is also built before `artifacts` exists, which would
otherwise make the import unbuildable. The publish record's draft reference is
therefore **nullable by design** (D-012), and the two cases stay
distinguishable:

| Record shape | Meaning |
|---|---|
| Draft reference present | The post went through the campaign, claim, and review gates. |
| Draft reference absent | Imported history. No gate ever ran on it. |

Reporting must never present the second case as gated. Synthesising a
retrospective draft to fill the column would put a false approval trail in the
audit surface, which is the opposite of what the ledger is for.

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Claim, evidence | None. | Permanent. | None — a claim outlives every draft citing it, which is what makes reuse and contamination reporting possible. |
| Publish record, subject mention, narrated item | None. | Permanent. | None — publish history is the audit surface. |
| Draft, revision | None. Abandonment is a status, not a delete. | Permanent. | None. |
| Campaign, artifact slot | None. Closing is a status. | Permanent. | None. |
| Review run, verdict | None. | Permanent, with history. | None. |
| Coverage snapshot | Recompute. | Rebuildable cache. | None — derived entirely from publish records. |
| Post type registry | Reseed from canon. | Latest seed only. | None — canon is the source of truth. |
| Import key | None. | Permanent. | Deleting one would cause a re-import to duplicate. |

**Retention summary.** Only coverage snapshots and the post-type seed are ever
destroyed, and both are rebuildable. Nothing that records what was asserted or
what shipped is deleted. If a retention requirement ever demands deleting a
publish record, that is a change to the scenario's audit guarantee and belongs
in [`../internal/DECISIONS.md`](../internal/DECISIONS.md), not in a migration.

**Credentials.** This scenario stores none, in any table, in any form. Account
identity and platform credentials belong to the scheduler and the `vault`
resource. A schema change that introduces a token column is a defect.

## Privacy Notes

Generated template data is local development data. If a scenario stores
personal, regulated, customer, financial, or sensitive business data,
update this document and [`../internal/SECURITY.md`](../internal/SECURITY.md)
before implementation expands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
