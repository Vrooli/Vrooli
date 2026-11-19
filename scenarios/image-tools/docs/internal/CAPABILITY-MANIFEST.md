# Capability Manifest — the target operation substrate

> **Status: implemented; one polish item remains.**
> - **Phase 0 (debt removal):** done — dead modules deleted, perceptual-hash
>   duplication extracted, orphan tables dropped through a versioned migration,
>   and the requirements ledger re-derived. See `MATURATION-PLAN.md`.
> - **Phase 1 (manifest for existing ops):** done. `api/internal/capability`
>   declares **71 operations** (23 AI, 5 analysis, 38 compute, 5 forward-declared)
>   and is load-bearing: consent weights, discovery, the UI catalog, the AI and
>   deterministic CLI tables, the deterministic parameter forms, and the wire
>   binding are all generated from it (`make capability-gen`) and drift-gated.
> - **Phase 2 (params contract):** done. `AIParams.op_params` + `params_schema`
>   replace the flat fields and hand validators; `internal/wireops` generates the
>   deterministic bridge from `OpParams`. **Remaining:** generate typed Go/TS
>   param structs from the schema so the last hop into a backend request stops
>   being `map[string]string`.
> - **Phases 3–4:** done — typed ports + `scenario` on a Look step, and a
>   versioned `common.v1` media-operation contract published over one RPC.
>
> This document describes the implemented substrate. Anything still open is named
> as open; nothing below is deferred silently.

## The problem, measured (pre-manifest baseline)

This table is the **baseline that motivated the substrate**, kept so the removed
tax is reviewable. The “current size” column describes the pre-manifest state; the
Generated-surfaces table below is the delivery. Adding a model-backed operation
now is one manifest row (plus a provider/technique row for a new backend).

| # | Surface | File | Current size |
|---|---|---|---|
| 1 | Operation vocabulary | `api/internal/capability/manifests.seed.json` (was `internal/operations`, deleted) | 71 records |
| 2 | Proto params | `packages/proto/schemas/image-tools/v1/ai/ai.proto` (`AIParams`) | flat struct, one field group per op |
| 3 | Typed→untyped bridge | `api/internal/wireops` (generated) | derived from `OpParams`; AI uses `op_params` + schema |
| 4 | Per-op validators | `api/handlers/ai/params.go` (`validate*`) | one function per param-bearing op, each also rejecting *other* ops' params |
| 5 | Technique + provider rows | `api/internal/ai/provider_catalog.go` | 35 technique rows across 10 provider specs |
| 6 | CLI commands | `cli/domains/ai/register.go` + `ops/register.go` | **generated** from the manifests (AI) and from `OpParams` (deterministic); no per-op command or param builder |
| 7 | UI catalogs + i18n | `ui/src/features/workspace/*Catalog.ts` + 3 locales | was 21 of 32 hand-covered; now generated |

Three consequences:

1. **Silent incompleteness.** Surfaces 6–7 are not conformance-tested against the
   vocabulary, so an op can exist in Go, be reachable by API, and render as a raw
   name in the UI with every gate green. (Surfaces 1–5 *are* guarded.)
2. **Type safety dies at the backend boundary.** Params are typed in proto,
   flattened to `map[string]string` by `paramsMap`, and read as untyped strings by
   every `ArgBuilder`. The one place a typo becomes a silent wrong render is the
   only place there is no type.
3. **Provider quirks are code, selected by string comparison.** `RegisterProviders`
   branches on `s.name == "onnxruntime" / "sam2-transformers" /
   "stable-diffusion.cpp"` to attach probes and warm workers.

The vocabulary itself is already single-source and category-filtered — that part
works and is worth preserving. Everything *below* it is the problem.

## The record

A **capability manifest** is one declarative record per operation, checked into
the repo next to the model and adapter seeds, loaded at boot the same way
`registry.seed.json` is.

```json
{
  "id": "screenshot_cleanup",
  "registry": "operations",
  "summary": "Remove UI chrome, borders and non-content surroundings from a screenshot",
  "family": "restore",
  "mechanism": "infer",
  "io": {
    "inputs": [
      {"name": "image", "kind": "image", "required": true},
      {"name": "region", "kind": "box", "required": false}
    ],
    "output": {"kind": "image"}
  },
  "requires": {"process": true, "weights": true, "network": false},
  "backends": ["iopaint", "mask-fill"],
  "consent_weight": "low"
}
```

**Deterministic records are shorter by construction.** A compute record omits
`mechanism`, `requires`, `backends`, `consent_weight`, and — when it is the engine
default — `io` (one image in, one image out; the four operations that differ
declare their ports): the registry
already means compute, in this process, no backend, no environment requirement,
and no identity impact, so the loader makes those explicit before validation and
a deterministic record cannot declare a contradiction. The model-backed registry
declares all four, and `consent_weight` is **required** there — a host that
enforces consent must never inherit "none" from an omission.

**Shipped shape.** The fields above are what
`api/internal/capability/manifests.seed.json` carries today. Three additions are
now in place:

| Field | Status |
|---|---|
| `params_schema` | present on model-backed operations; validated by `internal/paramschema` and carried by `AIParams.op_params` |
| `presentation` | present where the UI needs it: icon + label/desc keys for every surfaced op, and `fields` for the deterministic parameter form (generated + cross-checked against `OpParams`) |
| technique binding | deliberately **not** in the manifest: the provider owns its technique set, and duplicating the names here would create a second source that can drift |

### Field ownership

| Field | Owner | Notes |
|---|---|---|
| `id`, `summary` | its operation table | the only thing every other surface needs |
| `engine` | substrate | `ai` / `analysis` / `compute` / `unwired`; the single membership fact every runner projection reads |
| `family` | product | application axis only — never execution |
| `mechanism` | substrate | `compute` / `infer` / `fetch` (see [`CAPABILITY-MODEL.md`](../concepts/CAPABILITY-MODEL.md)) |
| `io` | substrate + composition | port kinds feed [`COMPOSITION.md`](../concepts/COMPOSITION.md) |
| `requires` | capabilities seam | drives readiness/badges/ETA |
| `backends` | resolver | ordered preference; the provider still owns technique dispatch |
| `consent_weight` | safety | **authoritative today** — `internal/safety` reads it |
| `cli` | CLI | `{group, verb, flags}`; the AI submit command table is **generated** from it |
| `params_schema` | handler validation | **landed:** JSON Schema subset validated by `internal/paramschema`; carried on the wire by `AIParams.op_params`. The non-standard `requiresTrue` and `withinFrame` keywords exist so relational rules stay in the contract instead of a per-op Go validator |
| `presentation` | UI | icon + label/desc keys; `fields` declares the deterministic parameter form, which the generator cross-checks against the wire |

The record does **not** own weights, checksums, licenses, or architecture — those
stay in `internal/models/registry.seed.json`, which already validates them. The
manifest references a model; it does not duplicate model facts.

## Generated surfaces

| Surface | Generated from | Gate |
|---|---|---|
| UI capability catalog | `presentation` + `id` | `TestGeneratedUICatalogIsUpToDate` |
| Deterministic parameter forms | `presentation.fields`, cross-checked against `OpParams` | `TestGeneratedOpSpecsIsUpToDate` + `TestEveryDeclaredFormFieldExistsOnTheWire` |
| AI CLI command table | `id` + `cli` | `TestGeneratedCLICommandsUpToDate` |
| Deterministic CLI command table | `OpParams` descriptor | `TestGeneratedOpsCLICommandsUpToDate` |
| Deterministic wire binding | `OpParams` descriptor | `TestGeneratedBindingsAreUpToDate` |
| `docs/reference/backends.md` host-tool matrix | `providers` | `make backends-doc` |
| Operation discovery (`ListAIOperations`, `ListOperations`) | manifest set | `TestEnginesAgreeWithManifests` |
| Consent-weight table | `consent_weight` | `TestConsentWeightsCoverEveryAIOp` |

The **typed param structs (Go + TS)** row is the one not yet generated; it is
tracked in `MATURATION-PLAN.md` D3 and `CAPABILITY-MANIFEST.md` Phase 2.

`json-schema-to-go`/`json-schema-to-typescript` style codegen is already the
pattern used by `ui/scripts/gen-strings.mjs` and `cmd/gen-endpoints`; this is the
same governance model, not a new one.

## Params: `Struct` + schema instead of a growing message

Today `AIParams` grows a field group per operation (`segmentation`,
`scene_geometry`, `image_to_object`, `background_repair`, `inpaint`). Every
addition is a proto change plus a `paramsMap` branch plus a validator that must
reject the params on every *other* op.

Target:

```proto
message AIParams {
  // Shared controls every op may honor.
  string prompt = 1;
  string negative_prompt = 2;
  int64 seed = 3;
  string model_override = 4;
  bool allow_byok = 5;
  // ... shared knobs ...

  // Operation-specific parameters, validated against the operation's
  // params_schema from the capability manifest.
  google.protobuf.Struct params = 20;
  string params_schema_version = 21;
}
```

Why this is still proto-first:

- `Struct` **is** a proto type; the wire shape is generated, not hand-written.
- The *contract* moves to the manifest's JSON Schema, which is stricter than a
  proto message for per-op validation (ranges, enums, defaults, `additionalProperties`).
- The repo already validates REST payloads against JSON Schema shapes via
  `module.Schema`; this reuses that pattern.
- Validation and typed decoding are **generated**, so type safety returns at the
  technique boundary instead of being lost to `map[string]string`.

`paramsMap` and the `validate*` family are deleted in Phase 2. Until then, new
operations should add a schema-shaped field only if the phase gate slips;
otherwise use the shared knobs.

## Provider data-ization

`providerSpecs()` is already close to data — the remaining code-shaped parts are
the per-provider attachments:

```go
if s.name == "onnxruntime" && run == nil { p.warm = newWarmSidecarRunner() }
if s.name == "sam2-transformers" && run == nil { p.warm = newWarmSidecarRunner() }
if s.name == "stable-diffusion.cpp" { p.gpuProbe = probeStableDiffusionGPU; p.progressScan = ... }
```

Target spec fields: `warm_sidecar`, `gpu_probe` (`torch`|`onnx`|`sd`|`none`),
`progress_scan` (`sd`|`none`), `imports`, `host_tool`. Then registering a backend
is a data change, and `RegisterProviders` becomes a loop with no name matching.

Also fold the two hand-appended `HostToolBindings` entries (llama.cpp, tesseract)
back into the spec table so there is one source.

## Conformance tests (the migration's safety net)

| Test | Guards | Status |
|---|---|---|
| `TestOpVocabularyConformance` | every model/provider/Look/analysis reference resolves to the vocabulary | exists |
| `TestConsentWeightsCoverEveryAIOp` | every AI op has a declared consent weight | exists |
| `TestBackendsDocHostToolMatrixUpToDate` | generated backends doc matches provider data | exists |
| `TestManifestsCoverVocabulary` | every vocabulary op has exactly one manifest | **landed** |
| `TestManifestBackendsExistInModelSeed` | every declared backend is backed by a seed model for that op | **landed** |
| `TestManifestRequirementClassMatchesMechanism` | compute/infer/fetch agree with the requirement facts | **landed** |
| `TestManifestIOMatchesVocabularyContract` | declared ports match `requires_image`/`requires_mask`/`prompt_driven` | **landed** |
| `TestManifestConsentWeightIsAuthoritative` | `internal/safety` reads the manifest, not a second table | **landed** |
| `TestSeedOrderIsStable` | each registry's seed order is stable | **landed** |
| `TestDeterministicManifestsAreInProcess` | deterministic ops declare no backend/requirement | **landed** |
| `TestGeneratedUICatalogIsUpToDate` | the committed TS catalog matches the seed | **landed** |
| `TestGeneratedCLICommandsUpToDate` | the committed CLI table matches the seed | **landed** |
| `TestUICatalogCoversManifest` | generated catalogs cover every manifest op | Phase 1b |
| `TestCLISurfaceCoversManifest` | every manifest op has a CLI command | Phase 1b |
| `TestOpParamsValidateAndThread` | schema-valid `op_params` reach the backend request | **landed** |
| `TestOpParamsRejectedWhenNoSchema` | an operation without a schema refuses extra params | **landed** |
| `TestManifestParamsRoundTrip` | the remaining typed param messages migrate to schemas | Phase 2b |

The landed tests already close the most dangerous hole: a backend typo, a missing
manifest, and a divergent consent table now fail the build instead of surfacing as
a silent degradation at runtime. The remaining two holes are the CLI/UI surfaces.

## Migration phases

Each phase ships independently and leaves the tree green.

### Phase 0 — debt removal (done)

Delete what the manifest will replace or what is already dead. No behavior change.

- [x] Extract the duplicated perceptual hash into `internal/percepthash`
- [x] Reconcile the requirements ledger (47 false `failing` re-derived; zero dead refs)
- [x] Remove the stale `docs/manifest.json` template description
- [x] Drop the live-but-undeclared `attachments` and `notes` tables (versioned migration)
- [x] Split `api/internal/ai` by concern: `aigateway` and `capacity` extracted; the admission→execution core stays together (see MATURATION-PLAN D5)
- [x] Fold `HostToolBindings` hand-appends into the provider spec table

**Gate:** `make check` green; no behavior diff; duplication count (`dupl -t 60`)
lower than the 2026-07-04 baseline.

### Phase 1 — manifest for existing operations, no new behavior

- [x] Define the manifest type + loader in `internal/capability`
- [x] Write manifests for all 71 operations (23 AI + 5 analysis + 38 compute + 5 unwired)
- [x] Make the manifest set load-bearing (consent weights now read from it)
- [x] Add the coverage / backend / requirement / I/O / weight conformance tests
- [x] Generate the UI capability catalog + drift gate (`make capability-gen`)
- [x] Generate the AI CLI submit command table + drift gate
- [x] Prove headless coverage: the generated table adds `deblur`, `face-restore`, `old-photo-restore`, which had no CLI verb before
- [x] Add `presentation` (icon + i18n key) and generate the label catalogs
- [x] Generate the deterministic CLI group from the `OpParams` descriptor
- [x] Generate the deterministic parameter forms from `presentation.fields`, cross-checked against `OpParams`

**Gate (1b):** generated CLI/UI behavior is indistinguishable from today;
`TestUICatalogCoversManifest` and `TestCLISurfaceCoversManifest` pass.

### Phase 2 — params contract

- [x] Add `AIParams.op_params` (`google.protobuf.Struct`) as the forward path
- [x] Compile and validate `params_schema` at manifest load (`internal/paramschema`)
- [x] Thread validated `op_params` into the backend request, typed fields winning on conflict
- [x] Prove it on a real gap: `object_removal` now exposes `grow_px`/`halo_threshold`
- [x] Migrate `inpaint` and `image_to_object`; delete their hand validators and `paramsMap` branches
- [x] Express the inpaint relational rule as schema `requiresTrue`
- [x] Migrate `background_repair`, `segmentation`, `scene_geometry` (shape-preserving)
- [ ] **Remaining:** generate Go/TS param types from schemas so the technique boundary is typed (today the wire is typed and the schema validates; the last hop into a backend request is still `map[string]string`)

**Gate (2b):** the typed sub-messages are gone; a new op adds knobs with **zero**
`.proto` edits; CLI `--op-params` and the UI use the generated contract.

### Phase 3 — composition (**delivered**)

See [`COMPOSITION.md`](../concepts/COMPOSITION.md). `LookStep` carries typed
ports and a scenario; the lane is validated; `screenshot-cleanup` proves the
image→image chain headlessly.

### Phase 4 — cross-scenario catalog (**delivered**)

`packages/proto/schemas/common/v1/media_operations.proto` is the versioned
contract, published by Image Tools over one RPC. Peer adoption is a consumer
decision, not an Image Tools dependency.

## What stays code

Being explicit prevents a false promise:

- **New mechanism/family semantics** (tier 4). A new modality or a genuinely new
  inference contract needs an adapter.
- **Model-family quirks.** A pipeline that cannot accept a LoRA, or that needs
  different kwargs, is an adapter, not data.
- **Policy decisions.** Consent weights are data, but *which* controls a
  deployment runs is an operator choice, not a manifest property.
- **Capacity/admission.** RAM/GPU leasing is platform behavior.

## Anti-goals

- Do **not** build a general plugin/exec sandbox. Tiers 3–4 stay reviewed code.
- Do **not** move model facts (weights, checksums, licenses) into the manifest.
  The model registry owns them and validates them.
- Do **not** generate the *implementation* of an operation. A manifest says what
  an op is and which provider runs it; it does not contain the pixel or inference
  code.
- Do **not** keep two sources during migration longer than the phase gate.

## Measuring progress

| Metric | Baseline (2026-07-04) | Target |
|---|---|---|
| Hand-synced surfaces per new op | 7 | 1 (manifest) |
| Files touched to add a model-backed op | manifest only for knobs + 1 adapter at most | 1–2 |
| Operations with a manifest | 71/71 | 71/71 |
| Generated UI catalog drift | not guarded | build-failing |
| Duplicated LOC (`dupl -t 60`, api) | ~762 of 40,780 (1.9%) | < 1% (not re-measured since the generator deletions) |
| Unguarded surfaces | CLI, UI catalogs | 0 |
| Ops with a declared consent weight | 32/32 (guarded) | all ops |
| Backend-name typos reaching runtime | possible | impossible (seed-checked) |

## Cross-references

- [`../concepts/CAPABILITY-MODEL.md`](../concepts/CAPABILITY-MODEL.md) — the axes this substrate implements
- [`../concepts/COMPOSITION.md`](../concepts/COMPOSITION.md) — ports and workflows
- [`TECHNIQUE-SUBSTRATE.md`](TECHNIQUE-SUBSTRATE.md) — operation/technique/model spine
- [`MATURATION-PLAN.md`](MATURATION-PLAN.md) — cleanup, boundaries, and sequencing
