# Capability Model — axes, vocabulary, and hot-loading

This document defines **what a capability is** in Image Tools and how capabilities
are classified. It replaces the earlier two-bucket framing ("deterministic" vs
"model-backed"), which conflated four independent facts and had already started
to misdescribe shipped operations.

Read this before adding an operation. The substrate that *implements* these
concepts is [`../internal/CAPABILITY-MANIFEST.md`](../internal/CAPABILITY-MANIFEST.md);
the composition layer is [`COMPOSITION.md`](COMPOSITION.md).

## Why the old split was wrong

The old taxonomy put every operation in one of two buckets:

| Old bucket | Claim | Counter-example in the shipped catalog |
|---|---|---|
| "deterministic" | no model, instant, always available | `naturalize` is a `builtin` model in the AI catalog; fixed-seed diffusion inference is also deterministic |
| "model-backed" | needs weights, async, GPU | `qr_barcode_read`, `duplicate_detect`, `quality_assessment` are pure Go and synchronous |

It also filed operations by *application* (`generation` / `enhancement` /
`restoration` / `analysis`) while using those same labels to decide *execution
ownership*. That is why `background_removal` sits under `enhancement` but is
derived from segmentation, `depth_map` sits under `enhancement` but is analysis,
and `segment` sits under `analysis` but runs on the AI job lane.

One label cannot carry both facts. The model below separates them.

## The four axes

A capability is described by four **orthogonal** axes. Every field is data; none
of them is inferred from a category name.

### Axis 1 — Mechanism (`compute` · `infer` · `fetch`)

What actually happens when the operation runs.

| Mechanism | Meaning | Examples |
|---|---|---|
| `compute` | Closed-form code over pixels/bytes. No learned weights. Deterministic given the same input and parameters. | `resize`, `crop`, `adjust`, `dither_*`, `halftone`, `vectorize`, `metadata`, `duplicate_detect`, `qr_barcode_read`, `naturalize` |
| `infer` | Learned weights on a runtime (ONNX, diffusers, sd.cpp, llama.cpp, tesseract, a model-backed CLI). | `inpaint`, `upscale`, `caption`, `segment`, `face_swap`, `garment_replace` |
| `fetch` | A remote capability behind a credential and a policy role. | `openrouter` text-to-image / edit |

Why these words: they name the mechanism, not a quality judgement. `compute` is
the arithmetic path; `infer` is the model path; `fetch` is the network path. They
are verb-symmetric and unambiguous in a package list (`internal/compute`,
`internal/infer`).

Alternatives considered and rejected: *deterministic/learned* (determinism is a
property of a run, not a capability — fixed-seed inference is deterministic);
*algorithmic/model-backed* (accurate but one bucket still means "everything else");
*local/remote* (remote is an orthogonal requirement, not a mechanism).

### Axis 2 — Application (`family`)

What the user is trying to achieve. This is presentation and discovery only; it
must not decide execution ownership.

| Family | Intent | Examples |
|---|---|---|
| `generate` | Create pixels from a prompt or reference | `text_to_image`, `face_swap` |
| `edit` | Change part of an existing image | `inpaint`, `object_removal`, `garment_replace` |
| `enhance` | Improve fidelity / realism | `upscale`, `denoise`, `naturalize`, `colorize` |
| `restore` | Repair degradation | `deblur`, `face_restore`, `old_photo_restore` |
| `extract` | Produce structured data or a derived map from an image | `ocr`, `caption`, `segment`, `depth_map`, `object_detection`, `embedding` |
| `compose` | Combine multiple inputs into one raster | `overlay`, `composite`, `parallax`, `canvas` |

`background_removal` is `extract` (it is a segmentation result used as an alpha
channel), even though the product surfaces it under Enhance. `depth_map` is
`extract`. `segment` is `extract` that happens to run on the AI job lane.

### Axis 3 — Requirement (`requires`)

What the environment must provide before the operation can run. This — not the
family — drives readiness, badges, ETA, and the provider ladder.

| Requirement | Meaning | Consequence |
|---|---|---|
| `none` | Nothing beyond the binary | always available, synchronous |
| `process` | A host tool or linked library (tesseract, a Go/CGO lib) | availability probe on the program; still CPU |
| `weights` | A downloaded artifact set | install gate, disk admission, RAM reservation |
| `network` | A credential and a remote endpoint | BYOK policy, cost disclosure, opt-in |

These are ordered: `network` ⊃ `weights` ⊃ `process` ⊃ `none` for the purposes of
messaging ("install X" / "enable BYOK"), but they are independent facts in the
manifest.

### Axis 4 — Harm surface (`consent_weight`)

How much an operation can change *who a person is*: `none` · `low` · `high`.

This axis is **policy input, never an availability gate**. It decides whether a
configured control (consent affirmation, elevated provenance, forced scan)
applies. On a permissive local install it changes nothing. See
[`../internal/SECURITY.md`](../internal/SECURITY.md) and
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) (2026-07-04).

A fifth axis — **composition shape** (inputs/outputs, ports) — lives in
[`COMPOSITION.md`](COMPOSITION.md).

## Vocabulary rules

1. **One name per concept.** Operation (a verb), technique (a pipeline),
   model (weights), backend (a runtime), provider (the ladder object), Look (a
   saved compound workflow). See
   [`../internal/TECHNIQUE-SUBSTRATE.md`](../internal/TECHNIQUE-SUBSTRATE.md).
2. **An operation is declared once.** The capability manifest set is the vocabulary
   SSOT. Consumers filter; they never re-declare.
3. **Adding an operation is a data change plus, at most, one adapter.** See the
   hot-loading ladder below.
4. **Capability ≠ policy.** Policy decides *how a deployment exposes* a
   capability. It never decides whether the capability exists.

## Hot-loading ladder

The target is that most growth requires no Go change. Realistically:

| Tier | Change | Required work |
|---|---|---|
| 1 | New weights for an **existing family** (another SDXL/ONNX checkpoint) | **manifest/data only** |
| 2 | New operation in an **existing mechanism + family** (another ONNX extractor, another inpaint-style editor) | **manifest only** (params schema + provider preference) |
| 3 | New **backend runtime** (a new inference engine) | one adapter (argv/JSON contract + probes), then manifests |
| 4 | New **semantics** (a new modality or I/O shape, e.g. image→mesh) | one manifest + one adapter |

Tiers 1–2 are the "hot load a new model or tool" goal. Tiers 3–4 are normal
engineering and should stay explicit — pretending otherwise would mean shipping a
plugin sandbox with an unbounded trust surface, which is a separate decision.

The manifest substrate is what makes tiers 1–2 data-only: today an operation
touches seven hand-synced surfaces (see the manifest doc's problem statement).

## Current catalog through this model

| Mechanism | Count | Notes |
|---|---|---|
| `compute` | 38 deterministic ops + `naturalize`/`normal_map`/`quality_assessment`/`duplicate_detect`/`qr_barcode_read` | all synchronous, no weights |
| `infer` | the remaining model-backed ops (generation, enhancement, restoration, extraction) | durable jobs; CPU floor + GPU/quality tiers |
| `fetch` | OpenRouter text-to-image / image-to-image / instruct edit | BYOK, last tier |

**32 vocabulary operations** are model-backed/analysed and **38** are
deterministic, for **70 total** user-facing operations. The registry carries 62
models (30 enabled) across 18 registered backends.

## The three axes, as implemented

The classification above is the manifest, and the old single `Category` field is
gone. `api/internal/capability/manifests.seed.json` carries:

| Field | Answers | Values |
|---|---|---|
| `family` | what does it do? | generate · edit · enhance · restore · extract · compose |
| `mechanism` | what executes the work? | compute · infer · fetch |
| `engine` | which runtime runs it? | ai · analysis · compute · unwired |

`engine` is membership, not taxonomy: the AI job lane, the structured analysis
service, and the in-process compute table each read the manifests that name them,
so no package keeps a second list of what it serves. `unwired` is an honest
fourth state — declared so models can reference it, with no runner yet.

Two further fields are deliberately *not* part of the classification:

- `panel` (`create` / `enhance` / `analyze`, or absent) is where a person finds
  the operation in the workspace. It is presentation, and it is separate from
  `family` precisely because the old `category` conflated them.
- `presentation` (icon, i18n keys) is copy.

The conformance test `TestEnginesAgreeWithManifests` proves both directions: every
manifest that names an engine is listed by that engine's catalog, and every
operation an engine can run is declared. A runner cannot exist for an undeclared
operation, and a manifest cannot promise a runner that does not exist.

## Cross-references

- [`../internal/CAPABILITY-MANIFEST.md`](../internal/CAPABILITY-MANIFEST.md) — the substrate that implements these axes
- [`COMPOSITION.md`](COMPOSITION.md) — ports, workflows, cross-scenario composition
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and seams
- [`../internal/MATURATION-PLAN.md`](../internal/MATURATION-PLAN.md) — debt reduction and phasing
