# Technique Substrate — Operation / Technique / Model

> Phase-0 design note for the *operation-technique-model substrate + capability-derivation*
> refactor. This is the contract the rest of that plan executes against. It defines the three
> layers, who owns each, the glossary that prevents synonym drift, and the boundary decisions
> (W3 doc/code reconciliation, recipes↔looks). No behavior changes from this note alone.

## 1. The three layers

The AI execution surface conflates two orthogonal facts on one field (`Model.operations[]`):
*what the weights are* and *which pipelines can run on them*. This refactor separates them into
three named layers, each with one owner.

```
 Operation   what the user asks for           (text_to_image, inpaint, upscale, …)
     │        — a verb with an I/O contract
     ▼
 Technique   a named pipeline that maps a      (txt2img, img2img, inpaint, edit-via-img2img, …)
     │        model architecture → one op       — "how", declared not coded, proven (Ready) or not
     ▼
 Model       concrete weights + architecture   (sdxl-1.0 [arch sdxl], instruct-pix2pix [arch ip2p])
              — "what", plus the runtime family the sidecar loads
```

A model's **effective op set** is derived, not hand-listed:

```
effectiveOps(model) = declaredOps(model)  ∪  derivable(model.architecture)
                       (native)                (derived, each carries a quality caveat)
```

A base SDXL checkpoint declaring only `text_to_image` therefore *derives* `image_to_image`,
`inpaint`, `outpaint`, and `edit-via-img2img` from its `architecture: sdxl` — through proven
techniques — with zero per-model code.

## 2. Ownership (boundary-of-responsibility)

| Concern | Owner (package) | One-sentence responsibility |
|---|---|---|
| **Operation vocabulary** | `internal/operations` (NEW, Phase 1) | The single declarative table of every op `{name, category, summary, requires_image, requires_mask, prompt_driven}`. Everyone *reads* it; no one re-declares it. |
| **Technique table** | `internal/technique` (NEW, Phase 2) | The flat declarative table of named techniques `{name, op, pipeline_class, ready, caveat, build}`; the arg-builders that today are the three `buildDiffusers*` funcs become rows here. |
| **Architecture → technique table** | `internal/technique` (data, Phase 3) | Maps each model `architecture` → the techniques it can run + the op each yields; Go SSOT mirrored in Python (parity test). |
| **Model registry + derivation** | `internal/models` (Phase 3) | Owns `Model` (incl. the new `architecture` field) and `effectiveOps = declared ∪ derivable`; `selector` ranks over the derived set, tagging native/derived. |
| **Resolution** | `internal/resolver` (NEW, Phase 4) | Turns `(operation, model, host)` into one inspectable `Resolution` value object; the single home for the op→technique→model→params decision; `--explain` returns it without executing. |
| **Compound-op substrate** | `internal/looks` (Phase 5) | The *only* multi-step format; outpaint / hi-res-fix / edit-via-img2img are seeded Looks that compile to ordered {operation + mask/selection + params} steps. |
| **Backend process adapter** | `internal/ai` (exec/availability/host-tool) + `internal/backends` (ladder) | Runs a chosen technique's args on the Local-GPU→Local-CPU→BYOK ladder; knows nothing about *which* technique to pick. |
| **Safety weight** | `internal/safety` | `OpWeight` is keyed on the **operation**, invariant to native-vs-derived (a derived inpaint is still HIGH). |

**The screaming-architecture outcome:** the package list itself reads
`operations · technique · resolver · models · looks · ai(engine) · backends · safety` — the
Operation/Technique/Model spine is visible in `ls api/internal/`, not buried in one 44KB file.

## 3. Glossary (concept-vocabulary-unification — one name per concept)

| Term | Exact definition | NOT |
|---|---|---|
| **Operation** | A user-facing verb with an I/O contract (`text_to_image`, `inpaint`). The vocabulary SSOT entry. | a pipeline; a model |
| **Technique** | A named, proven pipeline mapping an architecture → one operation (`img2img`, `inpaint`, `edit-via-img2img`). | the diffusers class; the backend process |
| **Pipeline** | The runtime object a technique drives (a diffusers `*Pipeline` instance, a sd.cpp invocation). | the technique name |
| **Pipeline-class** | The concrete diffusers class string a family loads (`StableDiffusionXLInpaintPipeline`). | the family name |
| **Backend** | The *process/runtime* that executes (diffusers sidecar, stable-diffusion.cpp, iopaint, onnxruntime, rembg, realesrgan). | the technique; the model |
| **Provider** | The Go object wrapping a backend on the selection ladder (Local-GPU/Local-CPU/BYOK tier). | the backend binary |
| **Family** | A registered diffusers execution adapter (`instruct-pix2pix`, `qwen-image-edit-plus`); one per pipeline architecture the sidecar can load. SSOT = `families.go`. | the architecture (a family is the *adapter*, architecture is the *weights' lineage*) |
| **Architecture** | A model's weight lineage/topology (`sdxl`, `sd15`, `flux`, `instruct-pix2pix`, `qwen-image-edit`) — what determines which techniques are *derivable*. | the family adapter; the backend |
| **Look** | A saved/built-in compound recipe that compiles to ordered {operation + mask/selection + params} steps. The single multi-step substrate. | a recipe (recipes is the superseded name — see §5) |
| **Operation category** | `generation` · `enhancement` · `restoration` · `analysis` — the taxonomy bucket of an operation. | a technique |

### Operation taxonomy (the 28-op vocabulary, bucketed)

The vocabulary SSOT carries a `category` per op so each consumer filters to the ops it owns
*without* a hand-maintained second list:

- **generation (7):** `text_to_image`, `image_to_image`, `edit_instruct`, `inpaint`, `outpaint`,
  `object_removal`, `background_replace`
- **enhancement (7):** `upscale`, `background_removal`, `denoise`, `naturalize`, `colorize`,
  `depth_map`, `normal_map`
- **restoration (3):** `deblur`, `face_restore`, `old_photo_restore` — forward-declared in the
  vocabulary and provider-served, but **not yet wired into the `ai` engine's runner set**; kept
  out of the `ai` discovery catalog so Phase 1 changes no behavior. (Wiring them is out of scope.)
- **analysis (11):** `segment`, `ocr`, `nsfw_classify`, `caption`, `object_detection`, `tagging`,
  `face_detection`, `quality_assessment`, `duplicate_detect`, `embedding`, `qr_barcode_read` —
  owned by `internal/analysis`.

**Why a category drives `ai`'s catalog:** `internal/ai` owns exactly the ops it builds runners for
(`BuildRunners` ⇒ generation ∪ enhancement = 14). Deriving `ai`'s catalog as
`operations.ByCategory(generation, enhancement)` reproduces today's 14 ops exactly while deleting
the duplicated `Op` table — the membership rule is now *data*, not a second hand-list.

## 4. Boundary decision — W3: generation/enhancement packages

DOMAINS.md declares `api/internal/generation/`, `api/internal/enhancement/`, and
`api/internal/recipes/` packages; **none exist**. Reality: `api/internal/ai/` holds the single
generation+enhancement execution engine (`engine.go`, `providers.go`, `catalog.go`, …), and
`api/internal/looks/` is the compound substrate.

**Decision: `ai/` is ONE bounded context — the model-backed AI execution engine — not two.**
`generation` and `enhancement` are *operation categories*, not separate packages. The engine
(`probe → select model → select backend → execute → persist`), the provider exec layer, and the op
catalog are genuinely shared across both categories; splitting them would duplicate the engine to
satisfy a doc taxonomy. The Operation/Technique/Model spine is expressed by the **new** packages
(`operations`, `technique`, `resolver`) layered above `ai`, not by fracturing the engine.

→ DOMAINS.md is reconciled to reality: the `generation` + `enhancement` rows collapse into one
`ai` (generation+enhancement) domain pointing at `api/internal/ai/`, `api/handlers/ai/`,
`cli/domains/ai/`, `ui/src/features/workspace/`. (Edits land incrementally — op-family naming in
Phase 1, recipes/looks in Phase 5, the full documentation-health pass in Phase 7.)

## 5. Boundary decision — recipes ↔ looks

DOMAINS.md carries both a `recipes` domain (`api/internal/recipes/`, never built) and a `looks`
domain (`api/internal/looks/`, the implementation). Decision 102 already states Looks *generalize*
presets + recipes.

**Decision: Looks is the single compound/multi-step substrate; `recipes` is removed from the domain
inventory** (not aliased — greenfield: no second pipeline format survives). The `recipes` row and
section in DOMAINS.md are deleted in Phase 5; the glossary `Recipe` term redirects to `Look`.

## 6. What this note commits the plan to

- One vocabulary SSOT (`internal/operations`); `models` + `ai` read it; `operations_vocabulary` in
  the seed JSON and the `Op` table in `ai/catalog.go` are **deleted** (Phase 1).
- One technique table (`internal/technique`); the three `buildDiffusers*` funcs become rows;
  `providers.go` splits backend-process from technique dispatch (Phase 2).
- `Model.architecture` + an architecture→technique table (Go SSOT + Python mirror + parity test);
  `ServesOperation` literal check replaced by derived `effectiveOps` (Phase 3).
- `internal/resolver` + `Resolution` + `ExplainResolution`/`--explain`; `safety.OpWeight`
  operation-keyed invariant (Phase 4).
- Compound ops as seeded Looks; `recipes` removed (Phase 5).
- Picker capability matrix (native/via-workflow/unsupported) + caveat banner (Phase 6).
- "enabled ⇒ runnable" extended: a derived op is offered only when its technique is `Ready` and the
  pipeline class is load-smoked (Phases 3–4).
