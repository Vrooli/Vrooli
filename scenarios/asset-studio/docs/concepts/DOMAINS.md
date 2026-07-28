# Domains — Asset Studio

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The scaffold also ships one clearly fenced worked example domain (never
product scope) as a copyable reference; `template-manager detemplate
<scenario>` removes every fenced example once the real domains are green.

**Status: designed, not implemented.** The domain map below is the output of
the design work recorded in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).
Only `health` exists in code today. Each row states the boundary the
implementation is expected to honour.

The load-bearing split is between **declaration** and **production**.
`identities` and `specs` declare what should exist; `renders` produces it and
`assets` holds the result. A declaration is editable until something depends on
it and immutable afterwards, because provenance that points at a mutable input
is not provenance. If a change would let a produced artifact silently acquire
different inputs than the ones it recorded, it belongs on the declaration side
as a new version instead.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/asset-studio/v1/shared/health.proto` |
| identities | Own the character, scene, and product registry, its per-kind schemas, versioning, and canon import. | The thing that must look the same every time is the thing worth modelling; everything downstream binds to a version of it. | Identity records, identity blocks, versions, reference image links, import keys. | crud | policy | Identity, IdentityBlock, Version, ReferenceSheet | `api/internal/identities/` |
| specs | Own the asset specification, its frames, its identity bindings, and prompt resolution. | The declarative input to a render — reproducible because it binds versions, not names. | Specs, frames, bindings, resolved payloads, template references. | crud | service | Spec, Frame, Binding, ResolvedPayload | `api/internal/specs/` |
| renders | Own the render job, backend dispatch, provenance capture, and cost accounting. | Turning a declaration into bytes is a long-running external call with retries, partial failure, and real money attached. | Jobs, attempts, provenance records, cost records. | service | scheduler | RenderJob, Attempt, Provenance, Cost | `api/internal/renders/` |
| assets | Own the produced artifact library, derived variants, disclosure state, and the reference surface. | One copy of every artifact, one place where its disclosure state is true, and one identifier other scenarios cite. | Asset records, blob references, variants, alt text, disclosure flags, release state. | crud | integration | Asset, Variant, Disclosure, AssetRef | `api/internal/assets/` |
| conformance | Own identity-conformance verdicts, the AI-UGC policy checks, and the release gate. | Without a check that the render honoured the identity, the registry is decorative. | Verdicts, reference bindings, policy check results. | policy | service | Verdict, ConformanceCheck, PolicyCheck | `api/internal/conformance/` |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for your own domains, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/asset-studio/v1/notes/` |

- Purpose: demonstrate the expected vertical slice for a real domain.
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
<!-- EXAMPLE-DOMAIN:notes END -->

## Domain Details

### identities

- Purpose: hold every reusable thing that must look the same across renders.
- Primary archetype: CRUD, with a versioning policy.
- Secondary traits: schema validation per kind; idempotent canon import.
- Owns: identity records (`character`, `scene`, `product`); the frozen identity
  block; descriptive traits; reference image links and character sheets;
  version chains; content-addressed import keys; the required-empty
  `credential_claims` field on persona-depicting records.
- Does not own: what a persona is *for* — persona strategy, audience fit, and
  the AI-UGC stance stay operator-curated marketing canon. This domain holds
  the data those decisions describe, never the decisions.
- Invariant: an identity block referenced by an accepted asset is immutable.
  A change is a new version; the prior version stays resolvable forever.
- Invariant: the kind set is closed (`character`, `scene`, `product`) and
  validation is per kind. An unrecognised kind is a hard error at write, never
  a permissive default — a record with no schema is a record nothing can check.
- Degradation rule: a canon source that fails schema validation aborts *that
  item* and reports it. A vendor or an author changing shape must surface as a
  failure, not as an empty import that looks like "nothing new".
- Requirements: `ASSET-P0-001`, `ASSET-P0-002`, `ASSET-P0-003`,
  `ASSET-P1-007`.

### specs

- Purpose: declare what to produce, in a form that resolves the same way twice.
- Primary archetype: CRUD with a pure resolution function.
- Secondary traits: template contract validation; multi-frame composition (P1);
  the capture spec kind (P1).
- Owns: spec records, their frames, identity bindings **by version**, the
  referenced prompt template, look references, and the resolved model-facing
  payload.
- Does not own: prompt template *content* — the templates are marketing canon
  under `docs/marketing/catalogs/rich-media/templates/`, referenced here and
  validated against, never authored here.
- Invariant: resolution is a pure function of spec, bound identity versions,
  and template. No ambient state, no clock, no "current" version lookup. This
  is what makes `ASSET-P1-010` regeneration possible at all.
- Invariant: a spec binds identity *versions*. Creating a newer identity
  version never changes an existing spec's resolved payload.
- Requirements: `ASSET-P0-004`, `ASSET-P0-005`, `ASSET-P1-001`,
  `ASSET-P1-003`, `ASSET-P1-009`.

### renders

- Purpose: execute a spec against a backend and record exactly what happened.
- Primary archetype: service with a modelled job lifecycle.
- Secondary traits: backend selection; retry and cancellation; cost capture.
- Owns: render jobs and their attempts; backend dispatch through the inference
  seam; the provenance record (spec version, bound identity versions, backend,
  model, seed, resolved parameters); estimated and actual cost.
- Does not own: the model. Every inference call routes through ai-gateway,
  which owns model policy, routing, and capacity. This domain never speaks a
  vendor protocol. It also does not drive a browser — capture dispatches to
  browser-automation-studio through a seam.
- Invariant: a job that fails mid-render writes no asset. There is no partial
  artifact, and the failure is recorded with whatever it consumed.
- Invariant: provenance is captured at completion or the job fails. It cannot
  be backfilled — an artifact produced without it can never acquire it.
- Requirements: `ASSET-P0-006`, `ASSET-P0-007`, `ASSET-P0-008`,
  `ASSET-P1-002`, `ASSET-P1-004`, `ASSET-P1-006`, `ASSET-P1-010`.

### assets

- Purpose: hold the produced artifacts and be the one place they are cited from.
- Primary archetype: CRUD.
- Secondary traits: integration surface for consuming scenarios; derived
  variants through image-tools.
- Owns: asset records; blob references behind the BlobStore seam; derived
  variants by aspect ratio and format; dimensions; required alt text;
  AI-generated and disclosure-requirement flags; release state.
- Does not own: image processing itself. Resize, crop, and format conversion go
  to image-tools; this domain stores what came back.
- Invariant: **this scenario stores bytes, and `content-desk` stores none.**
  That is the deliberate fork between the two scenarios. A consumer receives a
  reference, never a copy.
- Invariant: a generatively produced asset is marked AI-generated at creation
  by the producing path, not by the caller. Disclosure state travels with the
  reference so a platform label never depends on a later step remembering.
- Requirements: `ASSET-P0-009`, `ASSET-P0-012`, `ASSET-P0-014`.

### conformance

- Purpose: decide whether a produced frame is actually the identity it claims.
- Primary archetype: policy / rules.
- Secondary traits: operator adjudication; automated scoring as advice (P1).
- Owns: conformance verdicts per frame with the reference they were judged
  against; the AI-UGC policy checks (credential claims empty, disclosure
  present, no real-person likeness); the release gate that consumes both.
- Does not own: rendering, or the identity record itself. It reads both and
  writes only verdicts.
- Invariant: **an unchecked frame is not a passing frame.** Release is refused
  on an *unresolved* verdict, not only on a failing one. The gate fails closed.
- Invariant: a verdict comes from a human operator. Automated scoring (P1)
  arrives as a recommendation and never satisfies the gate on its own, because
  the scoring model is unvalidated and an automated pass on a mis-scored frame
  is precisely the silent failure the gate exists to prevent.
- Highest-consequence error in the scenario: a frame that drifts from its
  identity being released, because every later artifact anchors to it and the
  drift compounds silently across a campaign.
- Requirements: `ASSET-P0-010`, `ASSET-P0-011`, `ASSET-P0-013`,
  `ASSET-P1-005`.

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

## Build Order

Domains that read a domain that does not exist yet must wait for it. The chain
here is linear and the first slice runs its whole length:

```
identities ──▶ specs ──▶ renders ──▶ assets ──▶ conformance
```

`conformance` reads `assets` (the frame) and `identities` (the reference), so it
is last. `assets` reads `renders`. Nothing reads backwards. The first vertical
slice is one still image walked end to end through all five — not each domain
built to completion in turn.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Identity block | The frozen subset of an identity record that a model must reproduce exactly. | `identities`. |
| Provenance | The complete set of inputs that produced an artifact. | `renders`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `voice` — persona speech bound to an identity | A persona's voice should be as versioned as its face, but the host has no hardware audio sink today and no persona account exists to publish voiced content to. | `ASSET-P2-001`: once audio-tools has a validated path on this host and a persona account is live. |
| `experiments` — labelled variants compared by outcome | Generating variants with no measurement source produces taste dressed as evidence. | `ASSET-P2-002`: once publish telemetry exists in `content-desk`. |

## Explicitly Not Domains (Decided)

| Considered | Verdict | Reason |
|---|---|---|
| A `capture` domain separate from `specs`/`renders` | **Rejected.** | Capture and generation are different *sources* with an identical downstream — same job model, same provenance, same library. Making capture its own domain would duplicate all three. It is a spec kind and a backend (`ASSET-P1-003`). |
| An `editor` domain for timeline video editing | **Rejected.** | Compositing is named, ordered slots (`ASSET-P1-004`). Frame-level editing is a different product; answering that request would turn this scenario into a video application and it would never finish. |
| A `models` domain wrapping generation vendors | **Rejected.** | ai-gateway owns model policy, routing, and capacity fleet-wide. A local model registry would be a second, drifting source of truth for something already owned. |
| A `styles` domain for look recipes | **Rejected.** | image-tools already owns look recipes. Duplicating them here would create a second place visual consistency is defined — the exact failure this scenario exists to prevent, applied to itself (`ASSET-P1-009`). |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
