# Design studio — implementation plan

Status: **active** · Direction chosen: **A + C's system lens + B's surface**

This file is the working contract for the studio rebuild. Read it before changing anything in
`ui/src/workbench/pages/DesignStudioPage.tsx`, `ui/src/workbench/component/InContext*`,
`ui/src/workbench/studio/`, or the designs API.

Design reference (outside the repo, not committed):
`~/design-studies/rcl-studio-options/index.html`

## 1. Outcome

An agent or operator can open the studio, start from a blank or imported design, compose a
professional page at desktop/tablet/phone, see how much of it resolves to RCL, run the checks with
canvas instruments, and export/keep it — without hand-holding.

The eight-screen study is the quality bar: if the studio cannot produce work at that level, it is
not done.

## 2. Chosen information architecture

- **Far-left rail**: Design, Components, Assets, Evidence, Settings.
- **Top bar**: breadcrumb, design tabs (multiple designs open), undo/redo/history, device, share,
  present, brand mark.
- **Left pane** (docked, 286px) with three tabs:
  - **Layers** — composition tree (primitives, patterns, slots, references), RCL badges.
  - **System** — tokens → primitives → patterns → pages, with "used by" counts and edits that
    preview affected designs.
  - **Assets** — imports and uploads, tied to the design, draggable.
- **Canvas** — frames at one zoom; browser/iPad/iPhone frames with status bars and safe areas.
- **Right inspector** (316px) — Inspect / Checks / AI.
- **Status bar** — draft state, RCL coverage (layers + painted area), check counts.

## 3. Object model

```
Design   { id, name, updatedAt, frames[], themeId, assets[], revision }
Frame    { id, name, device: desktop|tablet|phone, width, height, spec: TemplateNode }
Layer    = TemplateNode (existing spec model) + role + rcl provenance
Theme    = the existing TemplatePalette + type/space/radius tokens
Asset    { id, designId, kind: capture|image|reference, fidelity, src, meta }
Run      = the existing experience claims/evidence, evaluated at a revision
```

The existing `Template`/`TemplateNode` spec in `ui/src/workbench/templates/spec.ts` is the layer
model. Do not invent a second tree.

### Typed slots (required for in-context)

Add `role?: "hero" | "metric" | "chart" | "list" | "action" | "nav" | "text"` to the `slot` node
props (and `slotHint` stays human copy). A slot also declares `accepts: string[]` of asset kinds
(`metric`, `chart`, `table`, `button`…). Auto-placement scores candidates by role match, size fit,
and page position, and returns the chosen slot plus a human-readable reason.

### Designs client (owned by the API stream, consumed by the studio)

Create `ui/src/workbench/designs.ts` with exactly this surface:

```ts
export type DesignSummary = { id: string; name: string; updatedAt: string; frameCount: number };
export type DesignDocument = {
  id: string; name: string; updatedAt: string;
  frames: Array<{ id: string; name: string; device: "desktop" | "tablet" | "phone"; spec: unknown }>;
  theme: unknown;
  revision: number;
};

export function listDesigns(): Promise<DesignSummary[]>;
export function getDesign(id: string): Promise<DesignDocument>;
export function saveDesign(doc: DesignDocument): Promise<DesignDocument>;
export function createDesign(input: { name: string }): Promise<DesignDocument>;
export function deleteDesign(id: string): Promise<void>;
```

Until the API lands, the studio may import a local fallback with the same signatures, but the
import path must be one module so it can be swapped in one place.

## 4. Workstreams and file ownership

Parallel work is split by files. Do not edit a file owned by another stream.

| Stream | Owns | Must not touch |
|---|---|---|
| **Frames (me)** | `ui/src/workbench/component/DeviceFrame.tsx`, `ui/src/workbench/component/device-frame.css` | studio page, in-context page |
| **Studio** | `ui/src/workbench/pages/DesignStudioPage.tsx`, `ui/src/workbench/studio/**`, `ui/src/workbench/studio/studio.css` | API, `InContextStage.tsx`, `ComponentPage.tsx` |
| **In context** | `ui/src/workbench/component/InContextStage.tsx`, `FixtureFrame.tsx`, new `ui/src/workbench/component/incontext.css`, the In-context branch of `ComponentPage.tsx` | `DesignStudioPage.tsx`, API |
| **API** | `api/**`, `ui/src/workbench/designs.ts` | studio page, in-context page |
| **Debt** | `docs/studio/DEBT.md` and low-risk fixes anywhere, one file at a time | features under construction |

Shared files may only be edited by their owner. If a change is needed elsewhere, write it in the
handoff note instead of editing.

## 5. Debt rules (non-negotiable)

1. No new hardcoded colour, radius, spacing or duration in components. Use the token layer
   (`app-tokens.css` / CSS variables) or the RCL primitives.
2. No new user-visible string outside `t()` with a `defaultValue`; never edit locale files by hand.
3. No second implementation of a primitive. If a drawer footer, action bar, tab bar or device frame
   exists, use it; if it is wrong, fix it at the owner.
4. Every new pure function gets a unit test. Every new API route gets a handler test.
5. Delete what you replace. No dead code left behind, no commented-out blocks.
6. Run the gates before handoff: `pnpm test`, `pnpm lint`, `pnpm type-check` (or the scenario
   `make check`), and the relevant screenshot at 390 + 1440.

## 6. Phases and acceptance

**P1 · Frames + shell** — DeviceFrame primitive; docked three-pane shell; design tabs; frames at one
zoom; layers/system/assets tabs; inspector; status bar.
*Done when:* one design opens with three frames, switching tabs never loses selection, and the page
renders at 390/834/1180 from one spec.

**P2 · Object model + persistence** — designs CRUD with autosave; revisions; brand from Brand
Manager (name, mark, accent) with a neutral fallback; undo/redo over every mutation.
*Done when:* reload restores the exact design and selection; 50 mixed operations undo cleanly.

**P3 · In context** — real frames, typed slots, auto-placement with a visible reason, slot picker,
instruments (guides, blur, contrast, targets, overflow, type ramp) driving the checks.
*Done when:* a metric component auto-lands in a KPI slot at all three devices and no check is
"pass" without evidence.

**P4 · Provenance** — RCL coverage by layers and by painted area; extractability score for custom
regions; import HTML capture or image with a region picker and fidelity labels.
*Done when:* importing the eight-screen study reports ~90% custom and flags the tab bar and pill as
extractable.

**P5 · System lens + AI** — token/primitive/pattern pages chain; "used by"; affected-design preview;
AI proposes a patch with a visual diff and one-step undo.

**P6 · Flow** — multiple frames per design arranged as a flow; connect, annotate, present.

## 7. Validation

- Unit: `pnpm test`, `go test ./...` in `api`.
- Gates: `pnpm build` (runs experience/pins/tokens/strings/usage-contract checks) and `make check`.
- Served: screenshot `/design` and a component's in-context view at 1440×900 and 390×844, plus one
  interaction pass (tabs, undo, device switch, slot change).
- Evidence goes in the handoff note, not just in chat.

## Progress log

- 2026-11-19 · **API stream done.** Designs CRUD (`/api/designs`, SQLite `studio_designs`) and
  `GET /api/brand` (Brand Manager Connect surface, `assigned:false` fallback). `designs.ts` now
  calls the API with the contract unchanged; proto generation remains clean. Handoff:
  `HANDOFF-api.md`.
- 2026-11-19 · DeviceFrame primitive + tests landed (coordinator). Shell brand now reads
  `useBrand()` and shows an honest "Brand not assigned" fallback.
- 2026-11-19 · **Studio stream done.** Docked Layers/System/Assets, design tabs, command-stack
  history, autosave, device canvas, inspector, status bar. 25 new studio tests; suite 524/524;
  `tsc` clean. Handoff: `HANDOFF-studio.md`.
- 2026-11-19 · **Integration fixes (coordinator).** Removed the studio's second navigation rail
  (WorkbenchShell already owns global nav) — the canvas gained 56 px. Fixed template responsiveness:
  `container-type: inline-size` was declared on an unused `.tpl-host`, so every container query was
  dead and phone frames rendered the desktop grid. It now lives on `.tpl`; the phone frame reflows
  and its overflow finding clears. DeviceFrame a11y fixed (DEBT #8).

## Next wave (not in this slice)

1. **Imported styles**: HTML import maps structure, headings and copy; CSS is deliberately not
   interpreted. A future version could map a small allowlist of layout hints or offer a captured
   reference image beside the draft. Remote assets stay out.
2. **Extraction across history**: candidates are computed for one frame and compare shape, not
   meaning. Comparing across designs (and ignoring the three device copies) is the next honest step.
3. **AI beyond one turn**: the protocol is one prompt → one bounded patch. Model selection belongs
   to the gateway's role catalog; streaming, a conversation, and image input are deliberately out.
4. **`t()` duplication**: every workbench call keeps both a key and an inline `defaultValue`. The
   catalog is authoritative; a codemod can drop the defaults once translations start.
5. **Offline creation**: a design created while the API is unreachable has a browser id and is
   retained as a local draft; there is no promote-to-server flow yet.
6. **Live route preview**: a bound page opens its design, but the studio does not render the live
   page route next to the composition.

- 2026-11-19 · **Wave 2 (maturity).**
  - **String catalog rebuilt from usage.** 980 keys generated by `ui/scripts/sync-catalog.mjs`; the
    `strings:no-unused-keys` rule is satisfied, `pnpm lint` is **0 problems** (was 649 errors), and
    `strings:catalog:check` runs in `pnpm build`. `ja`/`ar` are now incremental with English fallback
    (partial-translation test) instead of carrying 520 dead keys; `pluralDefault` and its hardcoded
    English plural rules were deleted in favour of i18next CLDR plurals.
  - **Autosave data loss fixed.** The old effect skipped a tick whenever a save was in flight and
    only saved the "last edited" design, so switching tabs before the debounce could strand edits.
    `studio/saveQueue.ts` serialises per design, coalesces follow-ups, stops on failure, and the
    server-owned revision is adopted after each save. Optimistic concurrency returns `409` with the
    current document; the studio pauses and offers "Load the other version" / "Keep mine".
  - **Provenance by painted area.** `useFrameAudit` measures leaf rects through a ResizeObserver
    (fonts/images/reflow) and reports layer share **and** painted-area share.
  - **Design ↔ page binding.** `DesignDocument.scenario/page` round-trips; `/design/:scenario/:page`
    opens the bound design or creates a blank one and binds it. Removed the old effects that
    silently picked the first page and created designs on a bare `/design` load.
  - **Assets are durable.** Server table + byte route + ETag, 8 MiB/image limits, cascade delete,
    cross-design gallery. Import → reference → place as an `image` layer is a complete path.
  - **Robustness/quality.** Per-frame error boundary; unknown layer types render a labelled
    placeholder instead of vanishing; locale-aware relative dates; delete-design affordance with
    two-step confirm; honest "local draft" / "N designs could not be loaded" status; assets and
    design loads clean up requests on unmount.
  - **Live verification caught a real crash.** The asset Tests tab called `.toDate()` on a Connect
    JSON ISO string (`createdAt`); `lib/timestamp.ts` now normalises every timestamp shape and the
    tab renders 24 report rows with no console errors.
  - **Gates.** UI 110 files / 583 tests; `tsc --noEmit` clean; `pnpm lint` clean; `pnpm build` green;
    Go tests show only the three known pre-existing library/fixture failures. Live checks:
    `/design` (1 design, no side effects), `/design/react-component-library/coverage` (creates a
    blank bound design), asset upload → place → image layer, 409 conflict envelope, 390 no overflow.

- 2026-11-19 · **Wave 3 (AI, capture, shelf, sync).**
  - **AI assistance is real.** `POST /api/designs/{id}/assist` builds a bounded, scoped context,
    dispatches the existing `extract.structured` role through `designinference` (durable
    idempotency, local-only, confidential), and returns a validated **inert** patch. The studio
    applies it as one command (`AI · <summary>`) or discards it. Every proposal carries the
    revision and a document signature, so a proposal made before an edit is refused with "ask
    again". Live: `ollama` / `qwen3.5:9b`, 2.6 s, schema-validated, applied and undone.
    The gateway's schema subset is narrower than JSON Schema (no type arrays, no `maxProperties`,
    boolean-only `additionalProperties`); `HANDOFF-ai.md` records it.
  - **Region capture.** An uploaded image can be cropped with a pointer marquee, keyboard nudge and
    numeric fields; the crop is uploaded as a `captured` asset and can be placed as an `image`
    layer. Fidelity is visible on every card.
  - **Template shelf is durable.** `studio_templates` + `/api/templates` CRUD, a synchronous
    browser cache for first paint, a one-time cache→server migration, a serialized per-template
    write queue and an honest `rcl:workbench-templates-error` event. Selecting a template no longer
    destroys the composition; an explicit **Apply template** button does (undoable).
  - **Cross-tab freshness.** A dependency-free channel (`BroadcastChannel` with a `window` fallback)
    announces saves, creates, deletes and asset changes. An idle tab reloads a newer revision; a
    tab with local work gets the existing conflict banner. Bursts coalesce at 250 ms, focus and
    visibility refetch, and every listener/timer is removed on unmount.
  - **A failed save is no longer "saved".** `putDesign` returns where the write landed. A
    browser-only write is marked **Local draft**, is not broadcast as a server revision, is treated
    as dirty for cross-tab reconciliation, and retries on `online`. Verified by blocking PUT in one
    tab: it showed Local draft, then the conflict banner, then adopted the other tab's copy.
  - **Accessibility/debt.** `Field` now associates its label with the control (39 callsites);
    Cmd/Ctrl+Z and Ctrl+Y drive the command stack outside text fields; relative dates and numbers
    follow the locale; the template save route bug (POST to `/{id}` → 405) was caught live and
    fixed with regression tests.
  - **Gates.** UI **117 files / 685 tests**, `pnpm lint` 0, `tsc --noEmit` clean, `pnpm build`
    green. Go retains only the three known pre-existing failures.

- 2026-11-19 · **Wave 4 (import, extraction, chrome).**
  - **HTML import.** A detached `DOMParser` document is sanitised (scripts/styles/iframes/forms and
    every attribute removed; images skipped with a warning) and mapped to the studio's layer model:
    headings, paragraphs, links, buttons, lists, tables and containers, with deterministic ids and a
    region picker (whole document or any element subtree). Caps: 2 MiB, 500 source elements, 200
    layers, depth 12. Import creates an unbound design, so it cannot overwrite a page binding.
    Containers are labelled by source role + their own heading ("Section · Why now"), so the layer
    tree reads like the page. Live: the sample imported as 26 layers / 26 regions with one skipped
    image; RCL 4% from the mapped button.
  - **Extraction candidates.** A deterministic bottom-up shape signature (type + prop keys +
    children, ignoring ids/copy/placement) finds repeated custom structures in the frame, excludes
    RCL primitives/pages/slots, requires 3 layers × 2 occurrences, dedups nested candidates
    largest-first, and ranks by `(occurrences-1) × layers` (max 5). The System pane shows the
    example, usage and cost and selects the first example. Live: the imported page's two cards were
    found as "Group · Signals + Every change gets a trail." ×2.
  - **Design-owned chrome.** `studio_designs.options_json` (idempotent migration) stores validated
    `radius` (0–48) and `display` (`sans|serif`); create/save/get round-trip it, old clients keep
    working, and the client's `seedInput`, `duplicateInput`, `docSignature` and `frameTemplate` all
    honour it. The System pane edits it as one undoable command; the frame's `--t-radius` and
    `--t-display` update immediately. Fixed the controls' fallback to match `frameTemplate` (16 /
    serif) so the UI can never show a value the canvas is not rendering.
  - **AI race.** A proposal now carries its `designId`; switching tabs aborts the request and
    clears the result, and apply refuses a proposal made for another design even when the signatures
    match. Staleness remains the document signature, not the revision, so an autosave between
    propose and apply does not invalidate a valid patch.
  - **Layout/debt.** The extraction row no longer overflows the 245 px pane (`min-width:0` on the
    list/item); imported container labels replaced `Stack · html 9 stack` noise; Ctrl/Cmd+Z is
    verified live on an AI patch.
  - **Gates.** UI **123 files / 750 tests**, `pnpm lint` 0, `tsc --noEmit` clean, `pnpm build`
    green; Go only the three known pre-existing failures. Live: full and region imports, extraction
    select, chrome 24 px/serif persisted across reload, 390 no overflow, zero console errors.
