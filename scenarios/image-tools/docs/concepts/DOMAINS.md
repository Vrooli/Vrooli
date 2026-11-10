# Domains — Image Tools

## Purpose Of This Document

This is the implementation map for Image Tools' product and infrastructure
owners. Update it when a public capability, ownership boundary or source path
changes. The mission's operation list is the target; this file records current
owners and surfaces and does not claim every target is qualified end to end.

Image Tools separates distinct ideas in its model-backed path: an **operation** is
a user request, a **technique** is a declared pipeline, a **model** is a set of
weights, and a **backend/provider** is the process or route that executes it.
`internal/operations`, `internal/technique`, and `internal/resolver` form the
shared operation/technique/model spine. They are not another operation catalog
or product feature family. See
[`../internal/TECHNIQUE-SUBSTRATE.md`](../internal/TECHNIQUE-SUBSTRATE.md).

## Domain Inventory

| Domain | Primary Archetype | Source Paths | Responsibility | Data / effects | Surfaces |
|---|---|---|---|---|---|
| `ops` | Deterministic processing | `api/internal/ops` | Deterministic image transforms that do not require model inference. | Reads input references and writes output through `storage`; no domain table. | API, CLI, workspace UI. |
| `ai` | Model-backed processing | `api/internal/ai` | Model-backed image generation, editing, enhancement, restoration and image-producing model operations. | Durable jobs; model/backend execution and output references. | API, CLI, workspace UI. |
| `analysis` | Image analysis | `api/internal/analysis` | Image-to-structured-data operations such as OCR, classification, probe, quality heuristics and extraction. | Structured result or durable job record; image bytes remain in storage. | API, CLI, workspace/analysis UI. |
| `jobs` | Durable scheduler | `api/internal/jobs` | Durable lifecycle, priority, shared admission, progress, cancellation and cleanup. | `jobs` table and terminal result metadata. | API, CLI, job activity UI. |
| `models` | Model lifecycle | `api/internal/models` | Seed/custom model catalog, rights checks, install, smoke, readiness, stale detection and defaults. | Seed registry plus model-state/install/lifecycle/build/custom/default tables and managed files. | API, CLI, settings/picker UI. |
| `adapters` | Adapter lifecycle | `api/internal/adapters` | LoRA, ControlNet and IP-Adapter catalog, compatibility, import and lifecycle. | Seed/custom adapter metadata, install/enable state and managed files. | API, CLI, AI request controls. |
| `backends` | Provider resolution | `api/internal/backends` | Provider availability and execution-tier resolution for a selected model/operation. | In-memory provider registry; shared platform capacity is an external owner. | API internals, CLI inspection, UI readiness/fallback messages. |
| `storage` | Storage boundary | `api/internal/storage` | Image/blob reference seam and output ownership boundaries. | Uses api-core blobstore; it does not own a local SQLite table or blob implementation. | API internals, CLI output options, workspace UI. |
| `looks` | Compound workflow | `api/internal/looks` | Saved transformation definitions compiled to ordered operation steps. | Built-in seed and custom `look` table. Looks are the only compound-workflow format. | API, CLI, Looks/workspace UI. |
| `selection` | Interactive selection | `api/internal/selection` | Smart selection, mask output and contextual edit request compilation. | Uses normal operation jobs and storage references. | API, CLI, selection UI. |
| `measures` | Performance telemetry | `api/internal/measures` | Operation timing, queue-wait and model/backend trace collection and aggregation. | `op_measure` and `job_trace` tables. | API internals and CLI measure surface. |
| `quality` | Quality evaluation | `api/internal/quality` | Reference-based image quality evaluation, judge evidence and regression gates. | Versioned manifests and evaluation artifacts; no model catalog ownership. | API, CLI; used by model-quality research and promotion. |
| `preparation` / `sceneprep` | Product domain | `api/internal/`sceneprep` | Durable photographic preparation and review metadata. | `prepared_photograph` and idempotent review-operation tables; image/geometry bytes remain in blobs. | API, CLI, preparation UI. |
| `safety` | Consent policy | `api/internal/safety` | Operation/adapter consent policy and public-deployment affirmation records. | Policy code plus `consent_log` where deployment policy requires it. | API, CLI, safety UI. |
| `diff` | Image comparison | `api/internal/diff` | Pixel/perceptual comparison of image pairs. | Reads image references and returns a comparison artifact/result. | API, CLI, comparison UI. |
| `health` | Service health | `api/internal/health` | API and dependency readiness reporting. | No product data. | API, CLI status, health UI. |

The user-facing `ai` catalog is an operation-category view over the vocabulary;
there are no separate `generation/` or `enhancement/` API packages. `analysis`
owns structured extraction. The effective operation set for a model combines
its declared operations with only the architecture-derived techniques that are
Ready. The selector and resolver enforce that boundary.

## Shared Concepts

Operations, techniques, models and providers are distinct concepts. Their shared
contracts live in `internal/operations`, `internal/technique` and `internal/resolver`.

## Domain Details

The table above lists current owners and their persistence or side effects.
There is no single domain package that owns every capability; model work uses
the shared operation, technique, resolver, provider and job boundaries.

## Deferred Domains

The mission's unqualified end-to-end capabilities remain work in progress.
Their open routes and evidence live in the research checkpoints and are not
presented here as completed domains.

## Non-Domains

Explicit non-domains and superseded names:

- `internal/recipes`, `automation`, and `notes` are not implemented product
  domains. The old Recipes concept is subsumed by Looks; no second graph format,
  schema or replay pipeline should be introduced.
- `internal/operations`, `internal/technique`, and `internal/resolver` are
  shared technical layers, not parallel user-facing domains.
- `internal/database`, `internal/modules`, `internal/module`, `internal/server`
  and `internal/testutil` are infrastructure. Host hardware detection and
  remediation belong to the platform control plane; Image Tools consumes its
  supported inventory/capacity contracts.
- Blob persistence is implemented by api-core. `internal/storage` is the
  Image Tools consumer seam, not a second object store.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — service composition and boundaries
- [`FLOWS.md`](FLOWS.md) — lifecycle and ordering rules
- [`DATA.md`](DATA.md) — persisted data and retention boundaries
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — platform and scenario dependencies
- [`../internal/TECHNIQUE-SUBSTRATE.md`](../internal/TECHNIQUE-SUBSTRATE.md) — operation/technique/model design
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — injectable side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md) — validation scope

### Layered 2.5D nearby views

The deterministic `composite` operation supports a bounded pinhole-camera view
of prepared, full-canvas color/alpha layers at explicit relative distances.
Optional depth estimation, segmentation and inpainting prepare those layers;
this is not automatic scene reconstruction. Video orchestration and calibrated
live-screen replacement remain Device Studio's responsibilities. See
[`../reference/layered-photography.md`](../reference/layered-photography.md)
for the measured limits and safe-use contract.
