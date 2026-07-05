# Data — Experience Manager

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
| Experience specs (`experience/` per target scenario) | `spec` (planned) | In-repo JSON files in each **target** scenario | The target scenario's `experience/` folder, validated by `.vrooli/schemas/scenario-experience-spec.schema.json` | Versioned with the target repo tree; never stored here | This scenario reads/writes them via studio + validation; it is not the storage owner. Intent sections are stable; `bindings` sections are the volatile selector SSOT. |
| Generated BAS case stubs | `autofix` | In-repo JSON files in each target scenario's `bas/cases/experience-spec/` | Derived from active page specs, with `metadata.labels.spec_entry_id` linking back to the page id | Versioned with the target repo tree | workflow-health owns cataloging and execution; experience-manager only derives stubs and checks spec↔case references. |
| Wireframe and variant review artifacts | `render` | Local coverage artifacts | `api/internal/render/` output | Disposable per local run | HTML artifacts under `coverage/wireframes/<scenario>/`; variant promotion writes only the selected `experience/` JSON, not the review artifact. |
| Studio authoring sessions | `studio` | SQLite | `api/internal/authoring/schema.sql` | Explicit discard; apply leaves the session available for audit/resume until discarded. | Resumable form state + diff previews before any file write. |
| Attestation ledger | `attest` | SQLite, append-only | `api/internal/attestation/schema.sql` | Append-only; expiry is a validity attribute, not deletion | Manual-tier claim attestations with expiry timestamps; expired attestations emit `experience.attestation_expired`. |
| Fleet sweep | `fleet` | None | `api/internal/fleet/` | Compute-on-read; no cache to prune | Reads the scenario tree and parser depths live for worst-first debt visibility. |
| Capture evidence references | `reconcile` | SQLite + artifact refs | `api/internal/reconcile/schema.sql` | Per validation run; pruning policy deferred until evidence volume is known | Screenshot + a11y-tree pairs per claim check. Labeled (a11y reliability, verdict), never aesthetics-filtered — see DECISIONS on the training-data byproduct. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `authoring_sessions` | `studio` | `api/internal/authoring/schema.sql` | Studio API/CLI session lifecycle |
| `authoring_pages` | `studio` | `api/internal/authoring/schema.sql` | Studio page-form drafts, preview, and apply |
| `manual_attestations` | `attest` | `api/internal/attestation/schema.sql` | Manual-tier evidence freshness checks |
| `reconcile_evidence` | `reconcile` | `api/internal/reconcile/schema.sql` | Reconciliation checks and future evidence surfaces |
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
| None yet. | n/a | n/a | Add when product requirements include import/export. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Studio authoring sessions | `author discard <session>` or future stale-session pruning. | Retained until explicit discard; stale-session pruning is deferred. | No automatic stale-session sweep yet. |
| Manual attestations | None. | Append-only; superseding evidence appends a new row with a later expiry. | No pruning by design for v1 auditability. |
| Reconcile evidence | Future evidence pruning job or explicit scenario reset. | Retained until pruning lands; rows are labeled by scenario/page/state/claim/capture. | Latest-N pruning is still deferred. |

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
