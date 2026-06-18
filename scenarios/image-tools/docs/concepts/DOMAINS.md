# Domains — Image Tools

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`notes` is a worked example from the template, not product scope.
Replace it after the first real domain is green.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

image-tools is a local-first capability-primitive scenario in Vrooli's
`*-tools` family. It groups its capability surface into the bounded
contexts below. Four are the user-facing operation families
(`ops`, `generation`, `enhancement`, `analysis`); the rest are the
infrastructure domains that make those families reliable, hardware-aware,
and composable (`models`, `backends`, `jobs`, `storage`, `recipes`,
`automation`, `measures`). Every operation across every domain runs fully
headless from the CLI with no UI and no ComfyUI dependency — the UI is an
enhancer, never a gate.

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/image-tools/v1/health/` |
| ops | Deterministic, zero-download, instant image editing. | Operation / transform | No persistent product data (transient inputs/outputs via storage). | API, CLI, UI | OT-P0-001 | `api/internal/ops/`, `api/handlers/ops/`, `cli/domains/ops/`, `ui/src/features/ops/`, `packages/proto/schemas/image-tools/v1/ops/` |
| generation | AI text→image, img2img, inpaint, object removal; P1 outpaint, background-replace. | Operation / AI | No persistent product data; outputs via storage. | API, CLI, UI | OT-P0-002, OT-P1-001 | `api/internal/generation/`, `api/handlers/generation/`, `cli/domains/generation/`, `ui/src/features/generation/`, `packages/proto/schemas/image-tools/v1/generation/` |
| enhancement | AI upscale, background-removal, denoise/deblur; P1 colorize, restore, face-restore, SAM, depth/normal. | Operation / AI | No persistent product data; outputs via storage. | API, CLI, UI | OT-P0-003, OT-P1-002 | `api/internal/enhancement/`, `api/handlers/enhancement/`, `cli/domains/enhancement/`, `ui/src/features/enhancement/`, `packages/proto/schemas/image-tools/v1/enhancement/` |
| analysis | Image→data: OCR, NSFW classify, probe; P1 caption, detection, quality, dedup, embeddings, QR. | Operation / extract | No persistent product data; results returned to caller. | API, CLI, UI | OT-P0-004, OT-P1-003 | `api/internal/analysis/`, `api/handlers/analysis/`, `cli/domains/analysis/`, `ui/src/features/analysis/`, `packages/proto/schemas/image-tools/v1/analysis/` |
| models | Declarative model registry, management, hardware-aware selection. | Registry / query | Model registry state, install/enable state. | API, CLI, UI | OT-P0-006, OT-P0-007, OT-P0-008 | `api/internal/models/`, `api/handlers/models/`, `cli/domains/models/`, `ui/src/features/models/`, `packages/proto/schemas/image-tools/v1/models/` |
| backends | Per-op provider abstraction and fallback ladder. | Abstraction / policy | No product data (in-memory provider registry). | API (internal seam), CLI (introspection) | OT-P0-005, OT-P0-011 | `api/internal/backends/`, `packages/proto/schemas/image-tools/v1/backends/` |
| jobs | Durable server-owned async jobs, GPU-serializing queue, progress/SSE. | Lifecycle / queue | Job records, status, progress. | API, CLI, UI | OT-P0-009 | `api/internal/jobs/`, `api/handlers/jobs/`, `cli/domains/jobs/`, `ui/src/features/jobs/`, `packages/proto/schemas/image-tools/v1/jobs/` |
| storage | api-core storage/blobstore integration and output ownership. | Infrastructure seam | Blob references and ownership metadata. | API (seam), CLI (save-location flag) | OT-P0-010 | `api/internal/storage/`, `packages/proto/schemas/image-tools/v1/storage/` |
| recipes | Saveable, replayable multi-step operation pipelines. | CRUD + replay | Recipe definitions (op-stack graphs). | API, CLI, UI | OT-P1-004 | `api/internal/recipes/`, `api/handlers/recipes/`, `cli/domains/recipes/`, `ui/src/features/recipes/`, `packages/proto/schemas/image-tools/v1/recipes/` |
| automation | Batch, watch-folder, signed webhook callbacks. | Trigger / orchestration | Watch-folder config, callback delivery state. | API, CLI, UI | OT-P1-005, OT-P1-006 | `api/internal/automation/`, `api/handlers/automation/`, `cli/domains/automation/`, `ui/src/features/automation/`, `packages/proto/schemas/image-tools/v1/automation/` |
| measures | Op latency/throughput/queue-wait/fallback-usage observability. | Reporting / telemetry | Measure samples and aggregates. | API, CLI | OT-P0-012 | `api/internal/measures/`, `cli/manifest.json` measure blocks, `packages/proto/schemas/image-tools/v1/measures/` |
| notes | Worked CRUD reference with attachment upload exception. **Template example — remove during implementation.** | CRUD / entity | Notes and attachment metadata. | API, CLI, UI | Template starter only. | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/image-tools/v1/notes/` |

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### ops

- Purpose: deliver the full deterministic editing toolbox — resize/crop/
  rotate/flip/straighten/deskew, format conversion (PNG/JPEG/WebP/AVIF/
  HEIC/GIF/TIFF/BMP, SVG→raster), compress/optimize to a target file
  size, tone/color adjustments (brightness/contrast/exposure/saturation/
  vibrance/hue/white-balance/levels/curves/gamma/shadows-highlights),
  filters/effects (grayscale/sepia/invert/blur/sharpen/vignette/noise/
  posterize/threshold/duotone/LUT), canvas ops (pad/extend/border/fill/
  letterbox), composite/overlay (merge, image+text watermark, text
  overlay, shapes/annotations), thumbnail generation, and metadata
  (EXIF/IPTC/XMP read, GPS-off-by-default strip, ICC convert,
  auto-orient).
- Primary archetype: operation / transform.
- Secondary traits: zero-download, instant, no GPU, no ComfyUI.
- Owns: deterministic image transforms and their parameter contracts.
- Does not own: AI inference, model selection, persistence policy.
- API: `api/handlers/ops/` (Connect-RPC; REST multipart edge for upload).
- CLI: `cli/domains/ops/` — full headless parity.
- UI: `ui/src/features/ops/` (parameter forms; drag-box crop where it
  helps).
- Storage: none persisted; reads inputs and writes outputs via the
  `storage` domain seam.
- Requirements: OT-P0-001.
- Tests: handler, service/transform unit, CLI, UI feature, accessibility.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### generation

- Purpose: AI generation and editing — text-to-image (seed control, N
  variations, size presets), image-to-image, inpaint (mask/box/
  text-guided), and object removal; P1 adds outpaint/generative-expand
  and background replace.
- Primary archetype: operation / AI.
- Secondary traits: async (heavy GPU), provider-abstracted, hardware-aware.
- Owns: generation operation contracts and their op→provider mapping
  requests routed through `backends`.
- Does not own: provider selection policy (`backends`), model registry
  (`models`), queue mechanics (`jobs`).
- API: `api/handlers/generation/` (Connect-RPC; REST multipart for source
  images and masks).
- CLI: `cli/domains/generation/` — full headless parity (mask via file,
  box via coordinates, text via prompt).
- UI: `ui/src/features/generation/` (mask-painting canvas for inpaint/
  object-removal; accessible file-mask fallback).
- Storage: outputs persisted via `storage`; no domain-owned tables.
- Requirements: OT-P0-002, OT-P1-001.
- Tests: handler, provider-seam unit, CLI, UI feature, accessibility,
  job-lifecycle flow.
- Related docs: [`FLOWS.md`](FLOWS.md), [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### enhancement

- Purpose: AI enhancement and restoration — upscale/super-resolution
  (CPU-capable default), background removal (CPU-capable), denoise/deblur/
  unblur; P1 adds colorize, old-photo restore, face restoration, Segment
  Anything (SAM), and depth/normal map generation.
- Primary archetype: operation / AI.
- Secondary traits: CPU-capable defaults, before/after validation in UI.
- Owns: enhancement operation contracts and op→provider requests.
- Does not own: provider policy, model registry, queue.
- API: `api/handlers/enhancement/` (Connect-RPC; REST multipart for
  uploads).
- CLI: `cli/domains/enhancement/` — full headless parity.
- UI: `ui/src/features/enhancement/` (before/after slider; brush touch-up
  for bg-removal with accessible fallback).
- Storage: outputs via `storage`; no domain-owned tables.
- Requirements: OT-P0-003, OT-P1-002.
- Tests: handler, provider-seam unit, CLI, UI feature, accessibility,
  job-lifecycle flow.
- Related docs: [`FLOWS.md`](FLOWS.md), [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### analysis

- Purpose: image→data extraction — OCR/text-extraction, NSFW/safety
  classification (standalone op plus configurable auto-scan of AI output),
  and image info/probe; P1 adds caption/alt-text, object detection/
  tagging, anonymous face detection (count + boxes, no recognition),
  quality assessment, perceptual-hash dedup, semantic embeddings, and
  QR/barcode read.
- Primary archetype: operation / extract.
- Secondary traits: no GPU requirement for core ops; returns structured
  data rather than image bytes.
- Owns: analysis operation contracts and their structured result shapes.
- Does not own: face recognition or identity matching (explicit
  non-goal); document→text (owned by text-tools); model registry.
- API: `api/handlers/analysis/` (Connect-RPC; REST multipart for uploads).
- CLI: `cli/domains/analysis/` — full headless parity.
- UI: `ui/src/features/analysis/` (results panels, alt-text surfacing).
- Storage: results returned to caller; embeddings/probe data not persisted
  unless a consumer requests it.
- Requirements: OT-P0-004, OT-P1-003.
- Tests: handler, classifier/extractor seam unit, CLI, UI feature,
  accessibility.
- Related docs: [`FLOWS.md`](FLOWS.md), [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### models

- Purpose: a schema/proto-validated declarative model registry plus
  management and hardware-aware selection. Each entry carries id/name,
  operation(s) served, backend/provider, variant+quant+size, hardware
  requirements (min VRAM, min RAM, GPU-required, CPU-capable, OS/arch),
  supported input/output dimensions, content-capability labels
  (NSFW-capable, license + commercial-use, base-model lineage,
  known-risks), download source + checksum or local path, enabled flag,
  and default-for-operation marker. A selector picks the best-fit enabled
  model for the probed host, honoring per-op default and user override.
- Primary archetype: registry / query.
- Secondary traits: management lifecycle (install/enable/disable/remove),
  disk-space awareness, checksummed opt-in downloads, custom/fine-tuned
  entries.
- Owns: model registry records, install/enable state, selection logic.
- Does not own: host probing (read from the root `vrooli` CLI host-inventory
  contract over the shared `internal/hostinventory` collector, consumed via the
  `capabilities` seam used by `backends`/selection), the inference itself.
- API: `api/handlers/models/` (Connect-RPC).
- CLI: `cli/domains/models/` — list/search/install/enable/disable/remove.
- UI: `ui/src/features/models/` (Settings: size, hardware-fit indicator,
  license, capability labels at point of use).
- Storage: SQLite registry state (see [`DATA.md`](DATA.md)).
- Requirements: OT-P0-006, OT-P0-007, OT-P0-008.
- Tests: registry repository, selector unit (fit/override cases), handler,
  CLI, UI feature, accessibility.
- Related docs: [`DATA.md`](DATA.md), [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### backends

- Purpose: a per-operation provider interface and the graceful fallback
  ladder. Every operation exposes a provider interface with at minimum one
  standalone-lightweight backend; ComfyUI is an optional second provider;
  BYOK-cloud is a third fallback tier; CPU fallback is guaranteed. The
  system enforces ≥1 standalone backend per operation at registration
  time, and fallback is Local-GPU → Local-CPU → BYOK-cloud with explicit
  user-visible messaging at each transition.
- Primary archetype: abstraction / policy.
- Secondary traits: registration-time invariant enforcement, fallback
  routing, BYOK cost-estimate gating.
- Owns: the provider interface, the per-op provider registry, fallback
  policy, and the ≥1-standalone invariant.
- Does not own: operation semantics (the operation domains), model entries
  (`models`), or queue execution (`jobs`).
- API: internal seam consumed by operation domains; introspection surfaced
  via CLI.
- CLI: backend/provider introspection under operation command groups.
- UI: surfaced indirectly through fallback-tier and hardware messaging.
- Storage: in-memory provider registry; no persistent product data.
- Requirements: OT-P0-005, OT-P0-011.
- Tests: provider-interface unit, ≥1-standalone registration guard,
  fallback-ladder unit.
- Related docs: [`ARCHITECTURE.md`](ARCHITECTURE.md),
  [`FLOWS.md`](FLOWS.md), [`../internal/SEAMS.md`](../internal/SEAMS.md).

### jobs

- Purpose: a durable, server-owned async job queue. Jobs survive client
  disconnect; submission returns a job-id and ETA up front; the CLI uses a
  block-once wait verb; the UI uses SSE for live progress; heavy GPU jobs
  are serialized in a dedicated queue while cheap CPU ops run concurrently.
  The run-lifecycle mirrors the test-genie philosophy: no client polling.
- Primary archetype: lifecycle / queue.
- Secondary traits: GPU serialization, progress streaming, cancellation.
- Owns: job records, status transitions, progress, the GPU-serializing
  queue, SSE progress fan-out.
- Does not own: operation logic, provider selection, model state.
- API: `api/handlers/jobs/` (Connect-RPC submit/status/cancel/wait; SSE
  progress stream).
- CLI: `cli/domains/jobs/` — submit returns job-id+ETA; block-once wait.
- UI: `ui/src/features/jobs/` (progress bars, ETA, cancel controls).
- Storage: SQLite job records (see [`DATA.md`](DATA.md)).
- Requirements: OT-P0-009.
- Tests: queue serialization unit, job-lifecycle flow (Level 5 candidate),
  handler, CLI wait, UI progress feature, accessibility.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### storage

- Purpose: integrate api-core storage/blobstore so image binaries are
  stored outside the repo by default, save location is overridable per
  request, and outputs are user-owned (copyable, movable, deletable
  anywhere).
- Primary archetype: infrastructure seam.
- Secondary traits: overridable save location, output ownership,
  decompression-bomb / large-image safety at the ingestion boundary.
- Owns: the blobstore seam, blob references, output-ownership metadata,
  save-location resolution.
- Does not own: blobstore implementation (api-core), operation logic.
- API: internal seam used by every operation domain.
- CLI: save-location override flag exposed on operation commands.
- UI: surfaced via output download/save targets.
- Storage: blob references and ownership metadata in SQLite; bytes via
  api-core blobstore (see [`DATA.md`](DATA.md)).
- Requirements: OT-P0-010.
- Tests: blobstore seam unit, save-location resolution, ingestion-safety
  guard.
- Related docs: [`DATA.md`](DATA.md), [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### recipes

- Purpose: let users save, name, and replay multi-step operation graphs
  (recipes). The visual op-stack in the UI and the CLI pipeline are a
  unified representation; ComfyUI-style chaining is supported without
  ComfyUI being required.
- Primary archetype: CRUD + replay.
- Secondary traits: op-stack graph as a single shared representation
  across UI and CLI.
- Owns: recipe definitions, validation, and the replay engine.
- Does not own: the operation implementations (it composes them), job
  mechanics (it submits to `jobs`).
- API: `api/handlers/recipes/` (Connect-RPC CRUD + run).
- CLI: `cli/domains/recipes/` — save/list/run recipe pipelines.
- UI: `ui/src/features/recipes/` (visual op-stack with undo/redo over the
  headless op core).
- Storage: SQLite recipe definitions (see [`DATA.md`](DATA.md)).
- Requirements: OT-P1-004.
- Tests: recipe repository, replay-engine unit, handler, CLI, UI feature,
  accessibility.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### automation

- Purpose: batch processing (glob patterns and folder inputs), watch-folder
  automation that triggers configured recipes or single ops on new files
  (configurable debounce and output routing), and optional per-job signed
  webhook callbacks (best-effort POST with retry on completion/failure —
  not an event bus, no pub/sub infrastructure).
- Primary archetype: trigger / orchestration.
- Secondary traits: debounce, output routing, signed best-effort callbacks.
- Owns: watch-folder configuration, batch expansion, callback delivery
  state and retry.
- Does not own: operation logic, recipe definitions (it triggers them),
  the queue (it submits to `jobs`).
- API: `api/handlers/automation/` (Connect-RPC config + batch submit).
- CLI: `cli/domains/automation/` — batch over globs, watch-folder control.
- UI: `ui/src/features/automation/` (watch-folder config, callback URLs).
- Storage: SQLite watch-folder config and callback-delivery state (see
  [`DATA.md`](DATA.md)).
- Requirements: OT-P1-005, OT-P1-006.
- Tests: batch-expansion unit, watch-folder debounce, callback retry/
  signing, handler, CLI, UI feature, accessibility.
- Related docs: [`FLOWS.md`](FLOWS.md), [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### measures

- Purpose: observability for operations — per-model op latency (p50/p95),
  throughput, queue wait time, fallback-tier usage, and VRAM headroom at
  run time, surfaced through `cli/manifest.json` measure blocks enforced by
  the test-genie measures phase.
- Primary archetype: reporting / telemetry.
- Secondary traits: per-model dimensioning, fallback-tier attribution.
- Owns: measure sample collection, aggregation, and manifest measure
  blocks.
- Does not own: the operations being measured, the queue (it observes it).
- API: `api/internal/measures/` exposes aggregates.
- CLI: measure blocks in `cli/manifest.json`; query verbs for aggregates.
- UI: not a primary surface (measures consumed by tooling).
- Storage: SQLite measure samples/aggregates (see [`DATA.md`](DATA.md)).
- Requirements: OT-P0-012.
- Tests: sample-collection unit, aggregation unit, measures-phase contract.
- Related docs: [`DATA.md`](DATA.md),
  [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md).

### notes

- Purpose: demonstrate the expected vertical slice for a real domain.
  **This is the template's worked example and must be removed once the
  real operation domains are green.**
- Primary archetype: CRUD / entity.
- Secondary traits: binary/blob attachment upload, upload workflow.
- Owns: note records, attachment metadata, note validation, note
  service/repository seams, UI note interactions, CLI notes commands.
- Does not own: product scope for a generated scenario.
- API: `api/internal/notes/`, `api/handlers/notes/`.
- CLI: `cli/domains/notes/`.
- UI: `ui/src/features/notes/`, `ui/src/api/notes.ts`.
- Storage: domain-owned SQLite schema in `api/internal/notes/schema.sql`.
- Requirements: template starter only; replace with PRD-specific
  requirements.
- Tests: repository, service, handler, CLI, UI, accessibility, and
  workflow tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Operation | A single named image transform/inference/extraction with a parameter contract. | Operation domains (`ops`/`generation`/`enhancement`/`analysis`). |
| Provider / Backend | A concrete implementation of an operation (standalone, ComfyUI, BYOK, CPU). | `backends`. |
| Model | A registry entry describing a weights artifact and its hardware/capability/licensing facts. | `models`. |
| Job | A durable, server-owned async unit of work with id, ETA, status, and progress. | `jobs`. |
| Recipe | A saved multi-step op-stack graph replayable from UI or CLI. | `recipes`. |
| Headless-completeness | Every operation runs CLI-only with no UI and no ComfyUI. | `ARCHITECTURE.md` tenet, enforced per domain. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| provenance | C2PA content-credentials signing of AI output (OT-P1-007) is opt-in and off by default; folds into operation output handling at P1, not yet its own context. | When provenance gains its own state/policy surface beyond a per-op flag. |
| editor | Light non-destructive UI op-stack layer (OT-P1-008) lives on top of `ops`/`recipes` rather than as a separate bounded context. | If undo/redo/session state grows persistent semantics warranting ownership. |
| imagediff | General-purpose image diff/visual-comparison (OT-P1-009, consumed by test-genie at P2) — likely an operation within `analysis` or a thin domain. | When P1 lands and test-genie adoption (OT-P2-002) requires a stable contract. |
| consumer-embed | Embeddable UI component + stable contract for rich-media-studio (OT-P2-001). | When rich-media-studio composition work begins. |
| looks | The Look/Style Library (OT-P1-012) — data-defined, thumbnail-backed, AI-aware Looks that compile to {prompt + mask/selection + params}; generalizes presets (OT-P1-010) + recipes (OT-P1-004). Likely owns a small `looks` store + a look→request compiler built ON the `recipes` representation. | When the advanced-editing plan reaches Phase 3. |
| selection | Smart-Select → Classify → Contextual Edit (OT-P1-013) — a SAM-backed `segment` op + region classification + a canvas selection→edit compiler. Spans `analysis`/`enhancement` ops + a UI layer; may stay folded into those domains rather than its own context. | When Phase 3 selection work begins; split out only if it grows its own state. |

> **Refined vision (2026-06-18 advanced-editing workshop).** Beyond the table
> above, `enhancement` gains a **Naturalize** op (OT-P1-011, de-plasticize) and
> `generation` gains an **instruction-edit** op + model class (OT-P1-014); a new
> **Functional AI Substrate** target (OT-P0-014) makes real model+backend
> provisioning a precondition for any AI op actually running. See
> [`../internal/DECISIONS.md`](../internal/DECISIONS.md) (2026-06-18 rows).

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.
- Host capability/capacity probing — owned by the platform
  `internal/hostinventory` collector and surfaced by the root `vrooli` CLI
  host-inventory contract; consumed (never reimplemented) here via the
  `capabilities` seam (`packages/vrooli-cli-go`).
- Persistent blob storage implementation — owned by api-core blobstore;
  `storage` is only the consuming seam.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
