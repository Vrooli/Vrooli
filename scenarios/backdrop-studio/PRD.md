# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Own the production path from an art-direction intent to a released, layout-fit **backdrop** — the ambient imagery that sits behind or beside the foreground content on a landing page, sign-up screen, app-store listing, or promotional surface. The permanent capability is a **classified style catalog** plus a **generation ladder** that prefers deterministic code over model inference, an **output surface registry** that binds every render to a declared pixel geometry, and a **legibility gate** that refuses to release an image its own foreground cannot survive. It is the difference between commissioning a nice picture and being able to produce a consistent, brand-locked, provably readable one for every surface the portfolio ships.
- **Primary users/verticals**: The operator designing or refreshing a landing page; the marketing-crew producer agent commissioning promotional surfaces; `landing-page-business-suite`, which consumes released backdrops by reference; `scenario-to-android` and `scenario-to-ios`, whose store-listing asset targets have no producer today; and — as a bundled product — any subscriber who wants distinctive imagery for their own pages and app listings without hiring a designer.
- **Deployment surfaces**: UI (the studio workbench — catalog, composer, contact sheet, placement preview, release), CLI (the agent-facing catalog, compose, render, and gate verbs), and API (Connect-RPC). Ships in the business bundle and as a desktop app via `scenario-to-desktop`.
- **Value promise**: Makes distinctive imagery **systematic instead of commissioned**. One style record yields a correct image per brand, per surface, per composition; the treatment pass makes brand cohesion mechanical rather than a matter of art-directing every asset; the surface registry makes externally mandated geometry — a Play feature graphic, an App Store screenshot — a data lookup rather than a thing someone remembers wrong; and because the majority of styles are procedural, most production runs cost nothing, work offline, and reproduce exactly from a seed.

### Why this is not part of `asset-studio`

`asset-studio` exists so that *the same character renders the same way in six weeks* — its unit of value is reproducible identity and its release gate is an operator confirming a frame depicts the subject it claims to. Backdrop Studio has no subject to conform to; its unit of value is **fitness for a layout** and its gate is measured contrast under overlaid type. The two share mechanics and must not share domains. Where they meet is deliberate and narrow: **every render that invokes a model or costs money is released through `asset-studio`**, so there is exactly one system of record for spend, provenance, and disclosure. See `docs/internal/DECISIONS.md` D-001.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Classified style catalog | The system shall persist a style as a record carrying its five-axis classification — role, subject, treatment, lineage, and placement — and shall refuse to write a style whose axis values are not in the declared enums
- [ ] OT-P0-002 | Strategy declaration | The system shall require every style to declare exactly one generation strategy from `procedural`, `procedural-treated`, `guided`, or `synthesized`, and shall refuse a style whose declared strategy does not match the fields it carries
- [ ] OT-P0-003 | Axis query surface | When given an axis filter, the system shall return every style matching it, so that a caller selects a style by classification rather than by remembering its identifier
- [ ] OT-P0-004 | Delegated image execution | The system shall perform every raster operation and every model inference through `image-tools`, and shall contain no raster implementation and no model-provider configuration of its own
- [ ] OT-P0-005 | Palette slot binding | The system shall resolve `$brand.*` treatment parameters through `brand-manager` at render time, and shall refuse to render a style holding a slot that does not resolve, naming the unresolved slot
- [ ] OT-P0-006 | Inspectable render plan | Before executing anything, the system shall resolve a style and a brief into an explicit plan naming the strategy, the ordered operations, the resolved parameters, and the expected execution path, and shall return that plan without executing it when asked
- [ ] OT-P0-007 | Procedural determinism | While a style declares a strategy of `procedural` or `procedural-treated`, the system shall produce byte-identical output for the same style version, seed, and resolved palette
- [ ] OT-P0-008 | Composition scaffold | Where a style declares the `guided` strategy, the system shall render its scaffold preset into a conditioning image carrying the horizon, focal mass, framing geometry, and reserved-region voids, and shall submit that image as the conditioning input for generation
- [ ] OT-P0-009 | Candidate sets | The system shall allow one render request to produce several candidates and shall require an explicit selection before a candidate proceeds to release
- [ ] OT-P0-010 | Reserved regions | The system shall hold reserved regions on every style as proportional rectangles, each declaring whether foreground content **overlays** it (text, which must stay legible) or **occludes** it (a device frame or card, behind which image detail is wasted), so that a composition decision is data rather than a judgement repeated per asset
- [ ] OT-P0-011 | Worst-pixel contrast measurement | The system shall report the contrast ratio of the **least legible pixel** within each of a candidate's overlay regions against that region's intended text color, and shall never substitute a mean or median for that minimum
- [ ] OT-P0-012 | Legibility gate | The system shall refuse to release a candidate whose worst-pixel contrast is below its declared threshold, and shall state the measured value and the threshold when it refuses
- [ ] OT-P0-013 | Disclosure derived from strategy | The system shall derive an asset's AI-generated flag from its style's declared strategy, shall reject any attempt to set that flag directly, and shall not mark procedurally produced output as AI-generated
- [ ] OT-P0-014 | Release through `asset-studio` | While a render invokes a model, the system shall release its selected candidate through `asset-studio` so that provenance, cost, and disclosure are recorded in one system of record rather than duplicated here
- [ ] OT-P0-015 | Backdrop reference surface | The system shall expose released backdrops by stable identifier with their surface, placement, reserved regions, measured contrast, and disclosure state, so that a consuming scenario references a backdrop without copying its bytes
- [ ] OT-P0-016 | Execution-path transparency | The system shall record and report, for every render, whether the work ran on local hardware or was routed onward by `image-tools`, so that a degraded result is attributable rather than mysterious
- [ ] OT-P0-017 | Commercial-use gate | If a resolved plan depends on a conditioning adapter whose licensing forbids commercial use, then the system shall refuse to release the resulting asset and shall name the adapter and its restriction
- [ ] OT-P0-018 | Operator workbench | The system shall provide a UI to browse the catalog by axis, compose a brief against a style, watch a render, judge candidates against the legibility gate, and release a backdrop, and shall state the specific cause whenever release is unavailable
- [ ] OT-P0-019 | Style version immutability | The system shall refuse to mutate a style version that any released backdrop references, and shall require a new version instead
- [ ] OT-P0-020 | Render lifecycle contract | The system shall persist each render as a job with an explicit status and shall reject any transition the lifecycle contract does not declare
- [ ] OT-P0-021 | Alt text as a release precondition | The system shall refuse to release a backdrop that carries no operator-authored alt text, so that an inaccessible asset cannot reach a page through the API path any more than through the workbench
- [ ] OT-P0-022 | Output surface registry | The system shall persist an output surface as a record declaring its pixel geometry, its permitted placements, and the authority its geometry derives from, so that a target such as a Play feature graphic or an App Store screenshot is a data lookup rather than a remembered number
- [ ] OT-P0-023 | Surface geometry conformance | The system shall refuse to release a backdrop whose produced dimensions do not match its declared surface, and shall state the expected and the actual geometry when it refuses
- [ ] OT-P0-024 | Dual-view candidate judgement | The system shall present every candidate both standalone and composed into at least one in-context mockup of its target surface, because an ambient image judged alone is judged against the wrong question

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Remix with recorded lineage | When an operator forks a style and mutates one axis, the system should persist the new style with a reference to its parent, so that a catalog accumulates a traceable family rather than unrelated entries
- [ ] OT-P1-002 | Contact sheet sweep | The system should render a grid across a chosen pair of axes in one request, so that a style family is evaluated as a set rather than one image at a time
- [ ] OT-P1-003 | Placement preview | The system should render a selected candidate into each placement its style declares, at both a desktop and a mobile viewport, so that layout fitness is judged before release rather than after integration
- [ ] OT-P1-004 | Scrim solving | Where a candidate fails the legibility gate, the system should compute the minimum scrim opacity that would pass it and should offer that scrim as an amendment to the style
- [ ] OT-P1-005 | Brand kit adoption | When given a target scenario, the system should resolve its design tokens into palette bindings, so that a backdrop composed for that scenario inherits its palette without hand-entry
- [ ] OT-P1-006 | Surface variants | The system should derive sized variants of a released backdrop for the surfaces a page actually needs — hero, split panel, open-graph card, and mobile crop — preserving the reserved regions per variant
- [ ] OT-P1-007 | Style pack import and export | The system should exchange a set of styles as a portable pack, so that a catalog is shareable between installations and sellable as a product
- [ ] OT-P1-008 | Prompt and parameter disclosure | The system should display and copy the full resolved plan for any candidate, so that a result is auditable and reproducible by the operator who did not run it
- [ ] OT-P1-009 | Brand-themed mockup chrome | The system should draw the foreground of a product mockup — type scale, control shapes, spacing, neutrals — from the target scenario's design tokens, so that the preview reads as plausibly belonging to that product rather than as generic placeholder furniture
- [ ] OT-P1-010 | Store surface pack | The system should ship the app-store surfaces as declared records — Play feature graphic, Play phone and tablet screenshots, App Store screenshots at each required device class — each carrying the geometry its store mandates and a reference to the published requirement it derives from
- [ ] OT-P1-011 | Device-frame placements | The system should compose a supplied application screenshot into a device frame over a backdrop, in the arrangements a store listing actually uses — device alone, caption above device, caption below device, and caption only — reserving the frame's footprint as an occluding region
- [ ] OT-P1-012 | Platform-accurate store mockup | The system should preview a store asset inside a facsimile of the store listing it is destined for rather than inside brand-themed chrome, because the judgement being made is how the asset reads against that store's furniture and its neighbours

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Live backdrop export | The system may export a procedural style as runtime code — CSS, canvas, or shader — so that a page renders the backdrop live instead of shipping a raster
- [ ] OT-P2-002 | Motion backdrops | The system may produce short ambient loops for styles whose treatment survives animation
- [ ] OT-P2-003 | Conversion feedback loop | The system may accept conversion outcomes for released backdrops and may rank catalog styles by measured performance
- [ ] OT-P2-004 | Community style packs | The system may consume third-party style packs under a declared licensing contract

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**: Go API (Connect-RPC), Go CLI, React + TypeScript + Vite UI — the standard `react-vite` scenario shape.
- **Data + storage expectations**: SQLite, in-process. Styles, briefs, render jobs, candidates, legibility verdicts, and released references are durable. **Image bytes are not stored here** — rendered blobs live behind `image-tools` and released assets behind `asset-studio`'s blob seam. This scenario persists metadata and references only.
- **Integration strategy**: Compose, never re-implement. `image-tools` for every raster and model operation; `brand-manager` for palette and contrast authority; `asset-studio` for provenance, cost, disclosure, and release of model-backed work. `ai-gateway` is reached **transitively through `image-tools`**, never called directly — `image-tools` already registers the gateway as a provider alongside its local backends and selects between them from a probed host-capability inventory, so the local-first ladder is inherited rather than rebuilt.
- **Non-goals / guardrails**:
  - No raster implementation. A treatment that does not exist in `image-tools` is a change request against `image-tools`, recorded in `docs/internal/PROBLEMS.md`.
  - No model names, provider URLs, or credentials. Styles name an `ai-gateway` **role** and **profile**; the concrete model is resolved downstream.
  - No second provenance store. Model-backed releases go through `asset-studio` or they do not happen.
  - No general-purpose image editor. Backdrop Studio produces ambient imagery for layouts; focal and evidential imagery are out of scope.
  - No page building. It produces the backdrop and its composition contract; the page belongs to the consuming scenario.
  - **No application screenshot capture.** `scenario-to-android` and `scenario-to-ios` already capture screenshots from journey evidence. Backdrop Studio supplies the backdrop, the device-frame composition, and the surface geometry; the screenshot itself is an input it receives, never an artifact it produces. The same rule as everywhere else — it composes, it does not re-implement.
  - No store submission. Producing a listing asset is not publishing one; the store relationship belongs to the deployment scenarios.

## 🤝 Dependencies & Launch Plan

- **Required resources**: none beyond the in-process SQLite default.
- **Scenario dependencies**:
  | Scenario | Used for | Status |
  |---|---|---|
  | `image-tools` | Treatment operations, model inference, conditioning adapters, host-capability routing | **Blocking** — treatment ops do not exist yet (see PROBLEMS BS-P-001) |
  | `brand-manager` | Palette slot resolution, contrast authority | Available; binding surface unverified |
  | `asset-studio` | Provenance, cost, disclosure, release for model-backed renders | Available; conformance verdict is identity-coupled (see PROBLEMS BS-P-002) |
  | `ai-gateway` | Typed inference — reached transitively via `image-tools` | Available; no direct dependency |
  | `landing-page-business-suite` | First consumer of released backdrops | Available; hero currently hardcoded |
  | `scenario-to-android` | Consumer — supplies screenshots, consumes store surfaces and backdrops | Available; its `scenario-to-android OT-P1-007` store listing asset target has no producer today |
  | `scenario-to-ios` | Consumer — supplies screenshots, consumes store surfaces and backdrops | Available; its `scenario-to-ios OT-P1-007` App Store listing asset target has no producer today |
- **Operational risks**:
  - **The treatment layer does not exist.** `image-tools` ships no halftone, dither, duotone, posterize, grain, or scrim operation. Every style in every strategy terminates in a treatment pass, so this is on the critical path and is not a Backdrop Studio change.
  - **`asset-studio` has never been exercised.** It is built but unvalidated in production use; its conformance verdict binds a non-null identity version, which a backdrop does not have.
  - **Model availability is not guaranteed.** `guided` and `synthesized` styles depend on a capable local host or a configured gateway route. The catalog must remain useful when neither is present, which is why the procedural lanes are P0 and the model-backed lanes are the ones that degrade.
- **Launch sequencing**:
  1. Land treatment operations in `image-tools` (upstream; unblocks everything).
  2. Catalog and scaffold domains with the procedural lanes end-to-end — a usable product with zero model dependency.
  3. Legibility gate and release surface; first consumer integration in `landing-page-business-suite`.
  4. `guided` and `synthesized` lanes, with `asset-studio` release and disclosure.
  5. Studio UX depth — contact sheet, placement preview, remix.

## 🎨 UX & Branding

- **Look & feel**: The workbench is a **specimen sheet**, not a gallery. Imagery is the content, so the interface around it stays achromatic and quiet — the only saturated color on screen should be inside the artwork. Dense, aligned grids; hairline rules; parameters set in a monospace face beside every specimen so a result is always legible as a recipe rather than an accident. Full light and dark support is mandatory, and the canvas frame must not borrow its ground from the theme — a specimen is judged against a declared surface or it is not judged at all.
- **Accessibility**: WCAG 2.2 AA across the workbench. Two obligations beyond the usual: every candidate carries operator-editable alt text before release, and the legibility gate's verdict is conveyed by an explicit numeric ratio and a text label, never by color alone — a contrast tool that fails colorblind users would be self-refuting.
- **Voice & messaging**: Precise and unhedged. Name treatments by their real terms — halftone, error diffusion, duotone — and teach the vocabulary through use rather than simplifying it away; the words are part of what the product sells. A refusal always states the measured value, the threshold, and the amendment that would pass.
- **Branding hooks**: The scenario consumes brand identity rather than declaring one. Palette bindings resolve through `brand-manager`, so the workbench's own specimens re-render in whichever brand is active.

## 📎 Appendix

- `docs/concepts/ARCHITECTURE.md` — the layer boundary and the delegation contract.
- `docs/concepts/DOMAINS.md` — the eight bounded contexts and their build order.
- `docs/reference/taxonomy.md` — the five axes, their enums, and the axis-value to operation-name mapping.
- `docs/reference/surfaces.md` — the output surface registry and the geometry authorities it derives from.
- `docs/internal/DECISIONS.md` — durable decisions, including the `asset-studio` split and the treatment-terminates-every-lane invariant.
- `docs/internal/PROBLEMS.md` — upstream changes this scenario needs in other scenarios.
- `docs/business/MONETIZATION.md` — free / metered / gated placement per capability.
