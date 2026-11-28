# Handoff — In context (P3)

Status: **shipped**. Scope: `InContextStage`, `FixtureFrame`, `InContextChecks`, `placement`,
`instruments`, `contextChecks`, `viewports`, `incontext.css`, the In-context branch of
`ComponentPage`, slot typing in `spec.ts`/`library.ts`, plus debt #4 in `PreviewPage`.

## Contracts

- `TemplateNode` gained optional `role?: SlotRole` (`hero|metric|chart|list|action|nav|text`) and
  `accepts?: string[]`. Both are top-level node fields so `props` stays `string|number|boolean`.
  `slotNode()` builds a typed slot; `slotsOf()`, `slotRole()`, `slotAccepts()` read them. Legacy
  slots with neither field still render and are "open" (usable, unproven).
- `InContextStageProps`: adds `slotChoice: string | null`, `onSlotChoice`, `onReport(report)`,
  `states?: {total,covered,status}`. Removed `onOverflowChange`/`onPlacementChange` (the report
  replaces them). `Device` is now `DeviceKind` from `viewports.ts`.
- `InContextReport` (exported): `rows: ContextCheckRow[]`, `passed`, `scored`, `decision`,
  `device`, `viewport`, `status`, `instruments`, `storyId`, `hasStory`.
- `FixtureFrame`: adds `theme?: {id, tokens, resolved}`; posts `rcl-theme-apply` and
  `rcl-resolved-theme` so the specimen uses the page's surface/ink/accent instead of white.
- `viewports.ts` is the one metrics table: phone 390×844, tablet 834×1112, desktop 1180×760,
  plus 320×640; `REFERENCE_WIDTHS = 390/834/1180`; safe insets 54/24/0 top.

## Placement score (pure, deterministic)

`placeComponent({spec, asset, viewportWidth, preferredSlotId})` scores every slot:
role **0–45** (direct role = 45, declared `accepts` = 34, role family = 22, undeclared = 10,
incompatible = 6) + size **0–25** (component preferred width ÷ `estimateSlotBox` width) +
position **0–15** (earlier slots win). Normalised to 0–100, ties by document order then id.
`estimateSlotBox` walks the spec splitting width at rows/grids — no DOM, so tests and screenshot
agree. `TemplateRenderer` takes `activeSlotId`: the chosen slot receives the live component and
every other slot keeps its authored fallback, so no slot is rewritten into a spacer. `manual` is
true whenever a slot is pinned, so the reset always exists.

## Computed vs unmeasured

Computed: slot role/accepts, overflow at 390/834/1180 (hidden page-only probes), type inside the
slot wrapper, touch targets and text contrast from the preview harness probe (the component's own
controls and first readable text), safe area (first content offset + frame inset), and state
coverage from experience evidence whose verdict is actually `pass` (a skipped capture is not
coverage). `experience` absent ⇒ states is unmeasured, never pass. Squint is a human instrument:
it is listed but not scored.

Page health and component fit are separate: overflow and safe area describe the page, while the
slot, control sizes and contrast describe the component in that page.

Instruments → rows: Guides→safe area, Overflow→overflow (red outlines), Targets→touch (amber),
Contrast→text contrast (probe badge), Blur→squint. Off/unrun = `unmeasured`.

## Debt fixed

- #5: 8 s probe/readiness timeout, HTTP status check before mounting, `onError`, unreachable
  message; error renders in the slot and turns the harness chip red. `allow-same-origin` kept and
  documented: the harness is served by the API origin and resolves its modules/tokens there; an
  opaque origin fails every same-origin fetch. The parent still never reads its DOM.
- #4: `PreviewPage` now uses `DeviceFrame` + `viewports.ts` (no `.wb-device*` chrome, one metrics
  table).
- #12: my renderer uses `pass|fail|warn|unmeasured` only (squint is unmeasured, not "info");
  consolidation of the four renderers remains a follow-up.

## Evidence

`pnpm test` → **104 files / 524 tests pass** (new: placement 14, instruments 8, contextChecks 7,
FixtureFrame 3). `pnpm exec tsc --noEmit` clean. `eslint` clean on every owned file.
Shots at 1440×900 (and 390×844): `/tmp/opencode/ctx-phone.png`, `ctx-tablet.png`, `ctx-390.png`,
`ctx-pinned.png`, `ctx-blur.png`, `ctx-error.png`, `preview-desktop.png`, `fixtures-1440.png`.
Servers used Vite 23922 behind a temp proxy (`/tmp/opencode/vite.dev-proxy.config.ts`) because
`vite.config.ts` has no dev proxy; the production server on 23906 was not rebuilt or restarted.

## Outside scope / follow-ups

- `DeviceFrame.tsx` fails the string lint (`"thumb zone"`, line 120) — frames stream.
- A concurrent edit removed `.wb-comp-layout`'s 980 px collapse from `pages.css`; the In-context
  branch now scopes `.wb-comp-layout.is-context` in `incontext.css`. Prefer restoring the shared
  rule.
- Every authored template declares one slot, so the picker usually lists one candidate. The
  scorer and neutraliser already handle multi-slot pages (local/studio templates).
- Dev proxy belongs in `vite.config.ts` so `pnpm dev` stops 404ing the API.

## Quality pass (2026-11-19, afternoon)

Status: **shipped**. Scope: the files listed at the top of this note, no others.

**Type changes.** `ContextFacts.overflow` rows carry an optional `scrollWidth` (a pass requires it)
and `slot.role` is now `string | undefined`; `FixtureFrame` gains an optional `onRetry`. No
`InContextStageProps` change.

**Passes are now earned.** `contextChecks.ts` only returns `pass` when the backing reading is
present and plausible. Slot: pass needs a declared role *and* declared `accepts`; an open slot
warns ("unproven"). Overflow: every reference width needs a real `scrollWidth > 0`; the overflow is
recomputed from `scrollWidth`, so a supplied boolean cannot claim a pass. Touch: a scan that finds
zero targets is unmeasured, and `small > total` is unmeasured. Contrast: a ratio needs both colours
readable and `1 ≤ ratio ≤ 21`. Type: placeholder `—` readings are unmeasured. States: pass needs
`covered === total`, `total > 0`, and `fresh`. Safe area: pass needs a real height and offset.
Squint: always `unmeasured`, scored false. The reference widths now come from `viewports.ts`.

**Instrument fixes.** `effectiveBackground` walks through transparent ancestors and composites each
translucent layer onto the first opaque one (`rgba(255,255,255,.5)` over `#000` → `rgb(128,128,128)`);
it returns null when no opaque ancestor exists. The contrast probe picks a `.tpl-text`/named node that
actually has text (`pickTextProbe`) and re-runs on story, template and palette changes. Touch targets
are measured from the rendered rect divided by the frame's zoom (`measureTouchTargets`, `toLogicalSize`),
so a 44 px target at 50 % zoom is not reported as 22 px. Overflow is `scrollWidth` against the logical
width at all three probes. The hidden probes are clipped to a 1 px inert box, so a 1180 px probe can no
longer widen the document at 390.

**FixtureFrame.** Native `load`/`error` listeners replace the JSX `onError` (React does not delegate
iframe error/load events, so the old handler never fired). The readiness timer lives in a ref and is
cleared on ready and on unmount; a mounted guard drops late transitions. A failure now offers "Try
again" via `onRetry`. `allow-same-origin` stays, with a comment naming the exact channel
(`preview-ready`, `rcl-preview-props-*`, `rcl-theme-apply`, `rcl-resolved-theme`) and why an opaque
origin would fail the harness's own module/token fetches.

**Accessibility.** New `InContextStage.a11y.test.tsx`: **axe 4.10 → 0 violations** for the staged view
and the no-story/phone state. The `.ic-side` panel became a labelled group inside the named
"In context" region (axe: complementary landmark must be top level). Device switch, instrument
toggles and slot options expose `aria-pressed`, are keyboard reachable in DOM order and use the
shared `:focus-visible` ring (plus explicit rings on reload/reset/link). Placement changes are
announced in a polite `role="status"` region ("Placed in KPI card in the stats grid, 76% affinity").

**Empty/error states.** Missing story, no slots, unsupported slot kind, harness failure (message +
Reload/Try again) and "in flow" each render a sentence and a next action.

**Placement fix.** Ties on total affinity are now broken by role specificity, then size, then
document order. At 390 the pure estimator cannot see the container-query grid collapse, so the KPI
slot tied the header action; the metric now lands in the KPI card at 390, 834 and 1180 (P3
acceptance), 76/89/96 %.

**Tests.** `pnpm exec vitest run src/workbench/component src/workbench/templates` → **11 files /
97 tests**. New: contextChecks "withholds every pass whose backing measurement is missing" (20
cases), overflow-from-scrollWidth, open-slot warn, squint; instruments translucent-stack and
logical-size/probe tests; FixtureFrame iframe-error, preview-ready, foreign-frame, unmount and
ready timer-clearing; InContextStage measurement/unmount/empty-state/no-slot/mismatch/slot-typing;
the axe suite.
`tsc` clean on every owned file (two pre-existing errors remain in `SidebarShell.test.tsx` and
`designs.test.ts`). `eslint` clean on every touched file.

**Served check.** Vite dev server on 23925 behind a temp proxy to the normal API (23906):
`/assets/3e1a19c3-5e96-451b-96a3-7b0c64fc9135?tab=preview` → In context. 1440×900 phone
`/tmp/opencode/ctxq-phone.png`, tablet `ctxq-tablet.png`; 390×844 `ctxq-390.png`. Zero console
errors. All eight shipped templates report `Measured at 390 ✓ · 834 ✓ · 1180 ✓`; at 390
`document.documentElement.scrollWidth === clientWidth === 390`. Touch reports `WARN measured`
(5/5 template controls under 44 px — dense desktop chrome), which is the honest reading.

**Still unmeasured.** Squint (human judgement by design); state coverage when no experience contract
exists; anything inside the specimen iframe — its touch targets, internal contrast and overflow are
cross-origin and the parent never reads its DOM, so those rows describe the page, not the specimen.
Contrast is unmeasured whenever the probe node has no opaque ancestor background.
