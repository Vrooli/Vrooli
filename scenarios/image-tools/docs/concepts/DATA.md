# Data — Image Tools

## Purpose Of This Document

This is the source map for data that Image Tools owns. Domain schemas live
beside the Go code that interprets them; runtime artifacts and image bytes use
their owning storage contracts instead of being copied into SQLite.

## Storage Overview

Image Tools uses these storage classes:

- **Scenario metadata:** embedded SQLite, resolved from the scenario id by
  `api-core/storage` and initialized through `api-core/database`. The module
  registry applies each domain's schema at API boot.
- **Image and result bytes:** `api-core` blobstore behind the
  `internal/storage` seam. Job payloads and protobuf messages carry references,
  not image bytes. Explicit local output paths are user-owned and outside the
  scenario cleanup boundary.
- **Installed model and adapter assets:** files under their managed roots, with
  content identity and lifecycle state recorded in the corresponding catalog.
  Model and adapter weights are not generic cache-cleanup targets.

The outside-repository roots and cleanup boundaries are declared in
`.vrooli/service.json`. Managed job-result cleanup is a separate governed
owner operation; it cannot delete inputs, active model assets, or arbitrary
local output paths.

## Schema Map

## Data Ownership

| Data | Owner | Schema | Source of truth and lifecycle |
|---|---|---|---|
| Durable job records, request identity, payload references, progress and terminal metadata | `jobs` | `api/internal/jobs/schema.sql` | Server-owned job state; retained across client disconnects. Recovery fails orphaned nonterminal jobs rather than resuming ambiguous work. |
| Model enable overrides, install records, explicit lifecycle state, build-operation intent, custom catalog entries and per-operation default pins | `models` | `api/internal/models/schema.sql` | `registry.seed.json` is the immutable seed baseline. SQLite contains only operator overrides and add-only custom entries. Weights remain files on the managed model root. |
| Adapter enable overrides, install records and custom entries | `adapters` | `api/internal/adapters/schema.sql` | `adapters.seed.json` is the seed baseline; custom entries cannot shadow seeded ids. Weights remain files under the adapter root. |
| User-created Looks | `looks` | `api/internal/looks/schema.sql` | Built-in Looks come from `internal/looks/seed.go`; the table holds custom proto-JSON Looks and thumbnail references. Looks are the sole compound-workflow substrate. |
| Consent affirmations for high-consent operations on public deployments | `safety` | `api/internal/safety/schema.sql` | Append-only audit rows record the operation, effective weight, deployment tier and affirmation time. Local personal-use tier does not write this log. |
| Prepared-photograph metadata and idempotent review-operation identities | `sceneprep` | `api/internal/sceneprep/schema.sql` | Immutable records refer to pixels and geometry stored as blobs. A repeated operation id resolves to the same prepared child. |
| Per-operation measurements and structured terminal job traces | `measures` | `api/internal/measures/schema.sql` | Finalized jobs record duration, queue wait, result state and available model/backend/tier facts. Aggregates are computed on read. |

`api/internal/database/system.sql` is intentionally empty; tables belong to a
domain schema. There is no `recipes`, `automation`, or `storage` SQLite
domain/table today. The prior recipe graph is superseded by Looks; no second
pipeline format should be added. Storage-reference metadata, where needed, is
owned by the `api-core` storage contract rather than a local Image Tools table.

## Storage Details

| Data | Owner | Location / contract | Deletion boundary |
|---|---|---|---|
| Uploaded inputs and operation outputs | `storage` seam; `api-core/blobstore` implementation | Opaque blob handles; consumers do not depend on a concrete object-store engine | Inputs and referenced outputs follow the api-core storage owner and explicit user operations. The generic Image Tools result-cleanup provider is limited to eligible terminal outputs. |
| Installed model files | `models` | Managed model root; checksum and lifecycle fingerprint recorded in SQLite | Removed only by the model lifecycle owner after model-use readers are released. |
| Installed adapter files | `adapters` | Managed adapter root; checksum and install record in SQLite | Removed only by adapter lifecycle owner after reader leases are released. |
| Python environment and backend runtime files | lifecycle / sidecar setup | Rebuildable scenario runtime roots | Rebuilt by managed setup; active runtime use is protected from cleanup. |
| Quality datasets and evaluator artifacts | `quality` | Versioned local manifests and receipt directories | Evaluation fixtures retain source/license, reference, model and output identity in their manifest. These files are research artifacts, not user job state. |

## Migrations And Compatibility

- Domain SQL remains beside its interpreting package and is applied through the
  module schema registry. Additive tables are preferred; do not recreate a
  deployed database to change a catalog or lifecycle contract.
- Seed model and adapter catalogs are source-controlled baselines. User state
  is overlaid at read time; custom entries are add-only and validated against
  the same operation/backend vocabulary before persistence.
- Image bytes are not placed in protobuf fields or SQLite JSON. Persist an
  opaque blob reference and keep byte ownership with the blobstore contract.
- Keep model lifecycle hashes separate: the artifact hash identifies installed
  bytes, while runtime hash identifies the execution environment used for smoke.
  A stale smoke result cannot make changed weights Ready.

## Import / Export

| Surface | Contract | Current behavior |
|---|---|---|
| Image input | Multipart upload or a retained blob reference | Stored behind `internal/storage`; AI and deterministic operations receive a reference/path only at their execution boundary. |
| Image output | Job result reference or an explicit local output path | Managed results remain governed job artifacts; a local output path stays user-owned. |
| Model import | Inspected repository/local-source metadata with explicit commercial-rights attestation | The model manager validates metadata and rights declaration before catalog registration or weight acquisition. An import is not an independent legal review. |
| Adapter import | Inspected adapter metadata and explicit rights declarations | The adapter manager validates before persistence or download. |
| Quality evaluation | Versioned manifest linking operation, candidate outputs, references, budgets and provenance | `image-tools quality evaluate` records metrics and judge evidence; it is not a user-image import surface. |

## Retention And Deletion

Image bytes can contain personal and location information. Do not infer a
universal upload TTL from a handler or UI session: the api-core blobstore and
declared owner policy govern retention. The job-result cleanup provider applies
only to explicitly eligible, age-qualified terminal output artifacts; it never
walks arbitrary paths. Local outputs, installed weights and active model or
adapter reads are outside that provider's authority.

## Privacy Notes

Model or adapter API keys are not stored in catalog rows. Cost and execution
facts belong in structured job metadata/measurements; secrets remain with the
platform credential owner. Image Tools performs anonymous face detection only;
it does not store identity matches or facial identity records. Consent logging
for high-consent-weight transformations is enabled by deployment policy, not by
fixture content.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data owners and domain boundaries
- [`FLOWS.md`](FLOWS.md) — state transitions and cleanup ordering
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — platform storage and host boundaries
