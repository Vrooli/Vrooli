# Studio rebuild — handoff

Status: slice landed in the working tree, **not committed**; the served scenario was not restarted.
Direction: A + C's system lens + B's surface, per `PLAN.md`.

## What changed

- **Shell**: far-left icon rail (Design / Components / Assets / Evidence / Settings, brand mark slot),
  top bar with breadcrumb, multi-design tabs (open / close / switch / duplicate / new), undo / redo /
  history popover with jump, device segmented control, share (copy URL) and present.
- **Left pane (286 / 262 / 320 sheet)**: Layers (semantic tree, type + RCL/custom badge, slot marker,
  collapse, up/down/duplicate/remove, add-layer palette, template shelf, collapsed page context),
  System (theme tokens with hex values and a live inline editor → primitives with use counts and
  "used by N frames" → patterns → pages), Assets (file picker, reference fidelity, empty state).
- **Canvas**: the real `TemplateRenderer` in `DeviceFrame` at 1180×780 / 834×900 / 390×844, one fit
  zoom above 420 px, per-frame fit below it, frame labels with device + size + zoom, selection ring,
  click-to-select, guides, zoom ±/Fit, add/remove frame. Overflow is measured per frame.
- **Inspector (316 / 292)**: Inspect (NodeInspector props, system path, tokens consumed, frame
  summary), Checks (composed claims + measured overflow / touch targets / state gaps), AI (clearly
  labelled preview, scope chips, disabled Propose, no model call).
- **Status bar**: draft / saving / saved · revision N / save failed, open count, RCL count share,
  issue count, zoom.
- **History**: one named command stack per design over immutable document snapshots, committed by
  spec edits, theme edits, frame add/remove and template application; consecutive prop/token edits
  merge into one step; undo keeps the selected layer when it still exists.
- **Persistence**: `designs.ts` list/get/create/save; seeds three frames from `DEFAULT_TEMPLATE` when
  empty; 700 ms autosave; revision + relative time in the bar.

## File map

| Path | Role |
|---|---|
| `ui/src/workbench/pages/DesignStudioPage.tsx` | composition: state wiring, claims, deep links, present, sheets |
| `ui/src/workbench/studio/useStudio.ts` | designs, tabs, history, autosave, assets, mutations |
| `studio/model.ts` | frame metrics, seeding/rekeying, doc signature, visible rows, addable palette |
| `studio/history.ts` | pure command stack (push/undo/redo/jump/merge) |
| `studio/rcl.ts` | count-based RCL share, primitive/pattern usage |
| `studio/checks.ts` | state audit, button estimates, layout + claim verdicts, DOM audit types |
| `studio/useFrameAudit.ts` | per-frame overflow/button measurement; canvas width |
| `studio/StudioRail/TopBar/Layers/System/Assets/Canvas/Inspector/StatusBar.tsx` | panes |
| `studio/NodeInspector.tsx` | property editor moved out of the page |
| `studio/studio.css` | studio chrome, token-only |
| `ui/src/workbench/pages.css` | deleted the replaced `.wb-studio*` block (DEBT #6) |

## Still stubbed / deliberate limits

- **Assets are in-memory** for this slice (per design, lost on reload); labels say *reference*.
- **AI never calls a model**; the disabled button and copy explain the intended diff flow.
- **RCL share is count-based** (`RCL` primitives / all layers); painted-area share is not built.
- **Page context is not bound**: the selector updates the URL (query or `/design/:scenario/:page`)
  and shows a note that it does not change the frames. The route claim was removed. Binding needs a
  page field on `DesignDocument` (API stream).
- Theme type/radius come from the default template; only `TemplatePalette` tokens are design-owned.
- Under 420 px of canvas width each frame fits individually rather than sharing one zoom; the page
  itself never overflows at 390.
- The Aperture phone frame has 11 px of real horizontal overflow (4 px content overhang, avatar row);
  the check reports it as a fail rather than hiding it.

## Evidence

- `pnpm test`: **524/524 pass** (104 files; 25 new studio tests cover history, coverage, models,
  checks).
- `pnpm exec tsc --noEmit`: **clean** for the whole UI.
- `pnpm exec eslint src/workbench/studio src/workbench/pages/DesignStudioPage.tsx`: clean.
- Screenshots (Vite dev on 23920, fresh localStorage): `/tmp/opencode/studio-1440.png`,
  `studio-390.png`, `studio-system.png`, `studio-checks.png`, `studio-assets.png`,
  `studio-present-390.png`.
- Browser console: 0 errors / 0 exceptions at both sizes. The only network log entries are 404s for
  API routes the standalone Vite server does not host; `designs.ts` falls back to localStorage.
- Interaction pass: select layer → edit text → autosave `Saved · revision 2` → undo restores the
  text and keeps selection; theme token edit recolours all frames; checks report overflow `fail`,
  targets `warn`, evidence `unmeasured`; device switch; history jump; asset import; AI panel;
  present at 390 renders 342 px wide (zoom 0.29, no page overflow).

## Debt items (audit) addressed

- **#2** primary action now requires a primary button; capture evidence is always `unmeasured`
  without a capture; the API-page route claim was removed. Verdicts come from `studio/checks.ts`.
- **#3** no silent mismatch: the context selector is explicitly labelled as unbound and the claim is
  gone; binding is the first follow-up.
- **#7** `useParams` feeds scenario/page; `designPath()` is used by the context selectors; deep links
  at `/design/:scenario/:page` render three frames with the path values.
- **#10** present frame gets `max-width: 100%` + computed `zoom`; verified at 390.
- **#6** deleted the dead studio CSS block from `pages.css` (kept `.wb-count`, restored the
  `.wb-inspector` mobile rule).

## Follow-ups

1. API: add `pageId`/scenario ownership to `DesignDocument`; bind the context selector, then restore
   a route claim based on the design.
2. Persist reference assets with the design (currently `useStudio` state).
3. Feed measured contrast from `instruments.ts` into the Checks panel.
4. Consolidate the four verdict renderers (DEBT #12) behind one primitive.
5. Frames: `DeviceFrame` still exposes the page as `role="img"` (DEBT #8).
6. Fix the Aperture phone overhang in `templates.css`, or keep it as the check's first real finding.
7. Workbench `t()` keys still live only in `defaultValue` (DEBT #1, platform).
