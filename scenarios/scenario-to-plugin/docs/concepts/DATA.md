# Data — Scenario to Plugin

This document is the canonical map of what this scenario persists, which
domain owns each shape, where the schema of record lives, and how long
data is kept.

The governing rule for this scenario: **the database stores records and
references; the capture store stores bytes.** No artifact, signature,
SBOM, or rehearsal log is ever inlined into a table, a proto payload, or
an emitted verdict.

## Purpose Of This Document

Use this document to answer:

- What does this scenario persist, and which domain owns it?
- Which file is the source of truth for each schema?
- What is retained, for how long, and on what trigger is it deleted?
- What leaves this scenario, and in what form?

## Storage Overview

Storage is embedded SQLite through `modernc.org/sqlite`. The lifecycle
sets `SQLITE_PATH` through `.vrooli/service.json`, and the API applies
schemas on startup through `api-core/database`.

A second store — the **capture store**, a scenario-owned filesystem
directory — holds opaque bytes: composed artifact trees, signatures,
provenance attestations, SBOMs, and rehearsal logs. Every capture-store
object is addressed by content digest, and the database holds the digest
rather than the object.

This split is not an optimization. `deployment-manager` stores evidence
references and never artifact bytes, so a ramp that inlined bytes into
its verdicts would violate the governance-plane boundary. Keeping the
split inside this scenario means the emitted verdict is a projection of
how data is already stored, not a special case at the boundary.

No external storage resource is declared. See
[`INTEGRATIONS.md`](INTEGRATIONS.md) for why, and revisit that decision
there before editing `.vrooli/service.json`.

## Data Ownership

Each domain owns its own tables and is the source of truth for its data.
The `health` domain owns no product data — it only probes configured
database reachability.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Declaration snapshots | declaration | SQLite | `api/internal/declaration/schema.sql` | Superseded snapshots pruned after 90 days; any snapshot referenced by a publication is retained indefinitely | Immutable per source commit. A package must be explainable after the declaration changes. |
| Readiness evaluations | declaration | SQLite | `api/internal/declaration/schema.sql` | Latest per scenario retained; history pruned after 30 days | Derived and recomputable; safe to prune. |
| Packages | composition | SQLite | `api/internal/composition/schema.sql` | Retained while any publication references them; otherwise pruned after 90 days | One row per scenario per source commit per build. |
| Package components | composition | SQLite | `api/internal/composition/schema.sql` | Same lifecycle as parent package | Component-level status supports independent validation reporting. |
| Artifact trees | composition | Capture store (digest-addressed) | Capture store index | Retained while referenced by a package or publication | Bytes never enter SQLite or a proto payload. |
| Conformance runs | conformance | SQLite | `api/internal/conformance/schema.sql` | Retained with the package; never pruned ahead of it | A signature's meaning depends on the conformance record that permitted it. |
| Findings | conformance | SQLite | `api/internal/conformance/schema.sql` | Same lifecycle as the run | Carries file, offset, and rule id. Redacted before display. |
| Manifest pins | conformance | SQLite | `api/internal/conformance/schema.sql` | Retained with the run | The `cli-manifest` revision a drift check compared against. |
| Attestations | attestation | SQLite (references) + capture store (bytes) | `api/internal/attestation/schema.sql` | Retained indefinitely once published | A published artifact's trust chain must remain explainable after revocation. |
| Scan verdicts | attestation | SQLite | `api/internal/attestation/schema.sql` | Retained with the attestation | Scanner name, version, and verdict; findings summarized, never raw. |
| Rehearsal runs | rehearsal | SQLite | `api/internal/rehearsal/schema.sql` | Retained with the package; pruned with it | Includes gate dispositions and journey state transitions. |
| Rehearsal logs | rehearsal | Capture store | Capture store index | 90 days, or with the package if referenced by a publication | Redacted at capture time, not at read time. |
| Publications | distribution | SQLite | `api/internal/distribution/schema.sql` | Retained indefinitely | The revocation fan-out is derived from this table; losing it loses the kill switch. |
| Channel outcomes | distribution | SQLite | `api/internal/distribution/schema.sql` | Retained with the publication | One row per channel per attempt, including failures. |
| Revocations | distribution | SQLite | `api/internal/distribution/schema.sql` | Retained indefinitely | Includes partial-revocation state and the channels still carrying the artifact. |
| Install attributions | distribution | SQLite | `api/internal/distribution/schema.sql` | 400 days rolling | Counts and referrer origins only. See Privacy Notes. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `declarations`, `readiness_evaluations` | declaration | `api/internal/declaration/schema.sql` | declaration repository/service/handlers |
| `packages`, `package_components` | composition | `api/internal/composition/schema.sql` | composition repository/service/handlers |
| `conformance_runs`, `findings`, `manifest_pins` | conformance | `api/internal/conformance/schema.sql` | conformance repository/service/handlers |
| `attestations`, `scan_verdicts`, `artifact_digests` | attestation | `api/internal/attestation/schema.sql` | attestation repository/service/handlers |
| `rehearsals`, `journeys`, `gate_results`, `evidence_refs` | rehearsal | `api/internal/rehearsal/schema.sql` | rehearsal repository/service/handlers |
| `publications`, `channel_outcomes`, `revocations`, `install_attributions` | distribution | `api/internal/distribution/schema.sql` | distribution repository/service/handlers |
| Capture-store objects | composition, attestation, rehearsal | Capture store index | The producing domain only; readers resolve by digest |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

No domain reads another domain's tables. Cross-domain reads go through
the owning domain's service, which is what keeps the pipeline chain
enforceable rather than advisory.

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

The attachment BlobStore seam is the reference implementation for this
scenario's capture store: opaque bytes behind a seam, metadata in
SQLite, nothing binary inside a proto payload.
<!-- EXAMPLE-DOMAIN:notes END -->

## Migrations And Compatibility

- Schemas are applied on startup by `api-core/database`. Each domain's
  schema file is idempotent and additive.
- Published artifacts are immutable. A schema change may never alter the
  interpretation of an existing `publications` or `attestations` row,
  because those rows explain artifacts that already exist on machines
  this scenario does not control.
- The emitted plugin manifest carries the Agent Plugins schema version it
  was built against. When that specification revises, composition emits
  the new version behind a declaration opt-in rather than silently
  changing what existing declarations produce.
- `manifest_pins` exists so a drift result stays interpretable after the
  wrapped CLI moves. Never backfill or normalize it.

## Import / Export

- **Export (product output):** the composed plugin tree, its signature,
  provenance attestation, and SBOM. This is the scenario's actual
  deliverable and leaves through `distribution` only.
- **Export (governance):** `TargetVerdict` messages to
  `deployment-manager`, carrying evidence references and dispositions.
  Never bytes.
- **Export (channel evidence):** per-plugin install and referrer counts
  to `offer-desk`. Aggregates only.
- **Import:** the target scenario's plugin declaration and its pinned
  `cli-manifest`. Both are read-only inputs; this scenario never writes
  into another scenario's tree.
- There is no bulk import path, and that is deliberate: every package
  must be traceable to a declaration at a source commit.

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Readiness evaluations | Age | Latest retained per scenario; history pruned at 30 days | None. |
| Unpublished packages and artifacts | Age, with no referencing publication | Pruned at 90 days | Prune job is not yet implemented; see `PROBLEMS.md`. |
| Rehearsal logs | Age, with no referencing publication | Pruned at 90 days | Same prune job. |
| Conformance runs and attestations | Never independently | Retained as long as the package they explain | None. |
| Publications and revocations | Never | Retained indefinitely | Losing this table loses the revocation fan-out, so it is excluded from any prune. |
| Install attributions | Age | 400-day rolling window | Window is chosen to allow year-over-year comparison; not yet enforced. |

A revoked publication is **not** deleted. Revocation records that an
artifact was withdrawn; deleting the row would erase the fact that it was
ever published, which is the opposite of what an incident response needs.

## Privacy Notes

- This scenario stores no end-user personal data. Its subjects are
  scenarios, packages, and channels.
- **Credential literals must never be persisted, logged, or attested.**
  `secrets-manager` holds credentials; this scenario holds references.
  `PLG-ATTEST-NO-SECRETS` fails a package before publication if a literal
  reaches an artifact, an SBOM, or an attestation — the check runs before
  anything leaves the host, because a published attestation is
  permanently retrievable and cannot be recalled.
- Rehearsal command output is redacted **at capture time**, not at read
  time. Redacting on read would mean the unredacted bytes existed on disk.
- Install attribution stores counts and referrer origins. It does not
  store IP addresses, user identifiers, or machine fingerprints, and the
  ramp must not add them: the channel's question is "did this capability
  get used", not "who used it".
- The operator identity attached to a publish or revoke action is
  retained, because release actions must be attributable.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — which domain owns each shape
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — why no shared resource is declared
- [`FLOWS.md`](FLOWS.md) — when each record is written
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — redaction and credential handling
- [`../reference/configuration.md`](../reference/configuration.md) — `SQLITE_PATH` and capture-store configuration
