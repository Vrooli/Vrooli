# Data — Asset Studio

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

**This scenario stores bytes.** Produced artifacts — images, video files,
composited output, character sheets — go through the template's BlobStore seam
with the filesystem implementation. Metadata, provenance, and every gate's
state live in SQLite; no artifact bytes ever enter a proto payload.

That is the deliberate fork with `content-desk`, which has no blob seam at all
and stores only references. Together the two hold exactly one copy of every
artifact, in the scenario that produced it. A change that put image bytes in
`content-desk`, or asset metadata authority in `content-desk`, is a defect in
the layering rather than a storage decision.

## Data Ownership

Each domain owns its own tables and is the source of truth for its data.
The `health` domain owns no product data — it only probes configured
database reachability.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Identity record | identities | SQLite | `api/internal/identities/schema.sql` | Permanent once referenced. | Kind (`character`/`scene`/`product`), descriptive traits, status. Unreferenced records are freely editable and deletable. |
| Identity block | identities | SQLite | `api/internal/identities/schema.sql` | **Never deleted once referenced.** | The frozen subset a model must reproduce. Immutable after an accepted asset binds it; a change is a new version. |
| Identity version | identities | SQLite | `api/internal/identities/schema.sql` | Permanent. | Version chain per record. A superseded version stays resolvable so old provenance keeps resolving. |
| Reference image link | identities | SQLite | `api/internal/identities/schema.sql` | Follows its version. | Points at a character sheet or reference asset in the asset library. The conformance judgement is made against it. |
| Conditioning artifact reference | identities | SQLite | `api/internal/identities/schema.sql` | **Follows its version; frozen with the block.** | Kind (`adapter` / `reference-set` / `look`), locator, and version of the artifact that actually reproduces the identity at render time (D-017). Part of the identity block, so it is immutable once released-referenced. Bytes for adapters and reference sets live in the asset library or in image-tools, never duplicated here. |
| Conditioning preparation cost | renders | SQLite | `api/internal/renders/schema.sql` | Permanent. | Cost of training or preparing a conditioning artifact. Attributed to the **identity version**, not to each render that uses it — a different cost shape from per-render spend (`ASSET-P1-012`). |
| Credential-claims field | identities | SQLite | `api/internal/identities/schema.sql` | Follows its record. | Required and required-empty on persona-depicting records. Encodes the AI-UGC ban structurally rather than as a review note. |
| Import key | identities | SQLite | `api/internal/identities/schema.sql` | Permanent. | `hash(source_path, normalized_content)`, unique-indexed. The whole idempotency mechanism; no cursor, no watermark. |
| Spec | specs | SQLite | `api/internal/specs/schema.sql` | Permanent once rendered. | Template reference, look reference, frames, and campaign reference. |
| Frame | specs | SQLite | `api/internal/specs/schema.sql` | Follows its spec. | One per output image or shot. Multi-frame specs (P1) share bindings across frames. |
| Identity binding | specs | SQLite | `api/internal/specs/schema.sql` | Follows its spec. | Binds an identity **version**, never a record id. This is what makes resolution reproducible. |
| Resolved payload | specs | SQLite | `api/internal/specs/schema.sql` | Follows its spec version. | The model-facing document. Stored rather than recomputed so provenance is exact even if resolution logic later changes. |
| Render job | renders | SQLite | `api/internal/renders/schema.sql` | Permanent. | Status, backend, timing, cancellation state. |
| Render attempt | renders | SQLite | `api/internal/renders/schema.sql` | Permanent. | One per submission, including failures. A failed attempt is evidence, not noise. |
| Provenance record | renders | SQLite | `api/internal/renders/schema.sql` | **Never deleted.** | Spec version, bound identity versions, backend, model, seed, resolved parameters. Cannot be backfilled. |
| Cost record | renders | SQLite | `api/internal/renders/schema.sql` | Permanent. | Estimated and actual, per attempt. A failed attempt still records what it consumed. |
| Asset record | assets | SQLite | `api/internal/assets/schema.sql` | Permanent. | Dimensions, format, required alt text, AI-generated flag, disclosure requirement, release state. One job may produce several candidate assets (D-018); selection promotes one and discards the rest. |
| Refinement link | assets | SQLite | `api/internal/assets/schema.sql` | Permanent. | Parent artifact, mask, and the image-tools operation that produced a refined artifact (D-019). Regeneration replays parent-then-edit; the refined artifact carries its own conformance verdict and never inherits the parent's. |
| Asset bytes | assets | Filesystem BlobStore | BlobStore implementation in the assets module | Same lifecycle as the record. | Opaque bytes outside proto payloads. Local by default. |
| Derived variant | assets | SQLite + BlobStore | `api/internal/assets/schema.sql` | Follows its parent asset. | Aspect-ratio and format derivations produced through image-tools, linked to the parent. |
| Conformance verdict | conformance | SQLite | `api/internal/conformance/schema.sql` | **Never deleted.** | Frame, identity version, reference image, verdict, judging actor and kind, timestamp. A re-judgement is a new verdict; history is kept. |
| Policy check result | conformance | SQLite | `api/internal/conformance/schema.sql` | Permanent. | Credential-claims, disclosure, and likeness checks with their outcomes. |
| Automated score | conformance | SQLite | `api/internal/conformance/schema.sql` | Follows its verdict. | P1. Advisory only — stored beside the operator verdict, never in place of it. |

**Retention summary.** Provenance records and conformance verdicts are never
deleted, because both are audit surfaces for something already published.
Identity blocks become permanent the moment an accepted asset binds them.
Everything else follows its parent. Asset bytes are the only large object, and
deleting them is a product decision that must leave the metadata and provenance
intact — a deleted artifact should still be explicable.

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| identities, identity_blocks, identity_versions, reference_images, import_keys | identities | `api/internal/identities/schema.sql` | identities repository/service; read by specs, renders, conformance |
| specs, frames, bindings, resolved_payloads | specs | `api/internal/specs/schema.sql` | specs resolution; read by renders |
| render_jobs, render_attempts, provenance, costs | renders | `api/internal/renders/schema.sql` | renders dispatch and reporting; read by assets |
| assets, asset_variants | assets | `api/internal/assets/schema.sql` | assets repository; read by conformance and the reference surface |
| conformance_verdicts, policy_checks, conformance_scores | conformance | `api/internal/conformance/schema.sql` | conformance policy engine; read by the release gate |
| Blob namespace `assets/` | assets | BlobStore implementation | assets module only |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

### Reproducibility contract

Provenance is complete when a released artifact can be re-rendered from it
alone — no ambient state, no "current version" lookup, no configuration read at
render time that was not recorded. `ASSET-P1-010` exists to *test* that
property rather than to add a feature: if regeneration needs anything the
provenance record does not hold, the record is missing something and the gap is
a defect in `renders`.

The reverse never holds. An asset cannot reconstruct its spec or its identity
versions from its bytes. Any change that would let produced output become
authoritative for a declaration is a layering defect.

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

Two columns must never be loosened by a later migration: the required-empty
`credential_claims` field on persona-depicting identity records, and the
non-null provenance reference on a released asset. Both encode a gate; making
either nullable disables the gate without removing it, which is worse than
removing it because the surface still looks safe.

## Import / Export

### Inventory

| Path | Format | Owner | Status |
|---|---|---|---|
| `docs/marketing/catalogs/rich-media/characters/*.json` | JSON per character | identities | `ASSET-P0-003` — recurring, idempotent |
| `docs/marketing/catalogs/rich-media/scenes/*.json` | JSON per scene | identities | `ASSET-P0-003` — recurring, idempotent |
| `docs/marketing/catalogs/rich-media/products/*.json` | JSON per product | identities | `ASSET-P0-003` — recurring, idempotent |
| `docs/marketing/catalogs/rich-media/assets/character-sheets/*` | Image files | identities | Linked as reference images on import |
| `docs/marketing/catalogs/rich-media/templates/*.template.json` | JSON prompt templates | specs | Referenced and validated against; never copied |
| Export | — | — | None planned. The asset reference surface and `--json` CLI output serve consumers. |

### Import is one-directional

The catalogue is operator-curated marketing canon that moves by accepted
decision. This scenario reads it and never writes back. The scenario is
authoritative for the *operational* identity record — validated, versioned,
render-bound — while the catalogue stays authoritative for which personas exist
and why. That is the same split `content-desk` draws between the post-type
registry it owns and the strategic post-type canon it reads.

A consequence worth stating: after import, the catalogue and the registry can
disagree. The registry wins for rendering; the catalogue wins for strategy. A
sweep re-import reconciles the first without touching the second.

### Idempotency

Re-import must never duplicate. The mechanism is a content-addressed key,
unique-indexed:

```
import_key = hash(source_path, normalized_content)
```

Re-running an import is a no-op for anything unchanged. This is the diff, and
it needs no watermark, cursor, or state file — which matters because every one
of these sources is a file a human or an agent may rewrite or reorder at any
time.

**Deliberately not positional.** File offsets and ordinal position are not
stable keys; the first reformat would re-import an entire catalogue as new.

**What a changed hash actually does (D-016).** Detecting change and creating a
version are two different questions, and conflating them produces churn:

| Head version state | Import behaviour | Why |
|---|---|---|
| Unreferenced (nothing released from it) | **Update the head in place.** | Immutability only protects versions a released artifact depends on. An unreferenced head can absorb edits freely, so routine authoring produces no versions at all. |
| Referenced (a released asset binds it) | **Create a new version.** | The edit is a new declaration. Prior renders keep resolving to what they actually used, and no already-released artifact silently changes meaning. |
| Unchanged hash | **No-op.** | This is the diff. |

The version chain therefore records *what was published from*, not every
keystroke. That is deliberate: it is a provenance record, not an edit log. If a
full edit history is ever wanted, it is a separate feature and should be built
as one rather than obtained by removing this rule.

"Referenced" here means **released** rather than merely rendered — see D-015.
Iterating on an identity by rendering test frames against it freezes nothing.

### Validation on import

An item that fails its kind's schema **aborts that item and reports it**. It is
never imported partially and never skipped silently. A source whose shape has
changed must surface as a failure rather than as an empty diff that reads as
"nothing new" — the same rule `vrooli-memory` applies to harness adapters, for
the same reason.

The catalogue has never been validated against a schema, so the first import is
expected to surface real shape problems in files that currently look fine.

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Identity record, block, version | None once referenced. Unreferenced records may be deleted. | Permanent after first accepted binding. | None — immutability is the scenario's central invariant. |
| Import key | None. | Permanent. | None. |
| Spec, frame, binding, resolved payload | None once rendered. | Permanent. | None. |
| Render job, attempt, provenance, cost | None. | Permanent. | None — provenance is an audit surface for published material. |
| Asset record and metadata | Operator delete of an unreleased asset. | Permanent once released. | Deleting a released asset's *bytes* while retaining its record is unspecified; decide before any storage-pressure work. |
| Asset bytes | Operator delete, or artifact-pruning policy. | Follows the record until a pruning policy exists. | No pruning policy. Generation volume is unknown until the loop runs. |
| Conformance verdict, policy check | None. | Permanent, with history. | None. |
| Automated score | Follows its verdict. | Advisory. | Scoring model unvalidated (`ASSET-P1-005`). |

No deletion path reaches a provenance record or a conformance verdict. If an
artifact is removed, the account of how it was made and who accepted it remains.

## Privacy Notes

This scenario stores generated depictions of fictional persona-actors, not
personal data about real people — and that boundary is enforced rather than
assumed. Real-person likeness is a policy check in `conformance`, and
`credential_claims` is required-empty on every persona record.

Reference images and character sheets are generated artifacts, not photographs
of identifiable people. If that ever ceases to be true — a real person's
likeness licensed for use, an operator's own image as a reference — this
document and [`../internal/SECURITY.md`](../internal/SECURITY.md) must be
updated before the first such record is stored.

No credential, token, or account handle is stored here in any form. Publishing
identity belongs to the scheduler; this scenario never learns which account an
artifact will be posted from.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
