# Design studio — debt inventory

> **Re-baseline 2026-11-19 (workbench program).** This file is a dated audit,
> not a current line-by-line truth. Items 1–11 below were resolved by the
> Nocturne and workbench waves. The 2026-11-19 program added a workspace
> aggregate, one search service, server-owned Review/Activity/Health, five
> destinations and a shared search surface; it removed the browser findings and
> activity reconstructions, the duplicate Home page and 126 unused catalog
> strings. Remaining studio debt is tracked in
> [PROBLEMS.md](../internal/PROBLEMS.md#2026-11-19--workbench-program-implemented-slice-and-remaining-gaps):
> legacy design/template owners, workspace capture/assist, Review pagination and
> the post-change capture sweep and workspace asset/assist/capture. The
> workspace studio now reuses the mature studio surfaces (layers, system,
> canvas, inspector, status bar, present) rather than a reduced layout.
> Re-audit before acting on any line below.

Snapshot: **2026-11-19 13:44 EST** (audited 13:19–13:44), HEAD `e1a86f6e83d`, working tree dirty (all
items below reproduced in this tree). Read-only audit; no product file was changed.

This tree is under **active multi-stream construction**. Between 13:22 and 13:39 the studio shell
(`ui/src/workbench/studio/**`), the in-context stage (`InContextStage.tsx`, `incontext.css`,
`placement.ts`, `contextChecks.ts`, `InContextChecks.tsx`), the designs API (`api/handlers/designs`,
`api/internal/designs`), the brand API and `ComponentPage.tsx` were all rewritten or added. Items
already being corrected by an active stream are listed under "Do not fix now" instead of the list
below.

---

## Status after wave 2 (2026-11-19)

The list below is the audit snapshot that drove the rebuild. Items 1–11 are resolved; the file is
kept as the record and as the place where new debt is filed.

| # | Item | State | Evidence |
|---|---|---|---|
| 1 | String catalog disconnected | **fixed** | `scripts/sync-catalog.mjs`, 980 keys, `strings:no-unused-keys` clean, incremental ja/ar |
| 2 | Checks pass without evidence | **fixed** | `studio/checks.ts` verdicts; contrast/evidence report `unmeasured` |
| 3 | Page selector decorative | **fixed** | `scenario`/`page` on the document + API filter; deep link opens/creates the bound design |
| 4 | PreviewPage hand-built chrome | **fixed** | `PreviewPage` uses `DeviceFrame`; one metrics table in `component/viewports.ts` |
| 5 | FixtureFrame spins forever / sandbox | **fixed** | HTTP probe, 8 s timeout, native `load`/`error`, retry; sandbox rationale documented |
| 6 | 294 lines dead studio CSS | **fixed** | deleted; shared 980 px collapse restored separately |
| 7 | Deep links ignored | **fixed** | `useParams` → same state; `designPath()` has call sites |
| 8 | DeviceFrame as one image / clickable div | **fixed** | `role="group"` + real `.df-select` button |
| 9 | `pnpm lint` red | **fixed** | **0 problems** (was 649 errors); `HANDOFF-lint.md` |
| 10 | Present overflows 390 | **fixed** | `max-width:100%` + fit zoom; verified at 390 |
| 11 | API errors mislabelled | **fixed** | default `connect.CodeInternal`, table test |
| 12 | Four check renderers / three vocabularies | **mostly fixed** | `workbench/verdicts.ts` owns the type, icon and catalog key; density-specific row markup remains (intentional) |

New debt introduced by wave 2 is recorded in the sections below under their categories; the
"do not fix now" list still applies to the three pre-existing Go failures.

### Found and fixed during wave-2 verification

- **Component Test report timestamps crashed the Tests tab.** `api/componentTests.ts` typed
  `createdAt` as a proto `Timestamp` and the drawer called `.toDate()`; Connect JSON delivers an
  ISO **string**, so any asset with a test report threw `r.createdAt.toDate is not a function`.
  `lib/timestamp.ts` is now the one reader (string, number, `Date`, `seconds`/`nanos`, `toDate()`),
  `createdAt` is `unknown`, and both call sites use it. Live `/assets/{id}?tab=tests` renders 24
  rows with zero console errors.
- **The page selector silently picked the first page.** Two legacy effects filled `scenario`/`page`
  into the URL, which — once pages bind — created a design for an arbitrary page on every `/design`
  load. Removed; a bare `/design` opens the existing design, and only an explicit page selection (or
  a `/design/:scenario/:page` deep link) creates a blank bound design.
- **Slot placeholders made the sample design look unfinished.** Slots now accept authored fallback
  children (`slotNode(..., { children })`); the studio shows a finished KPI card and in-context
  replaces that region with the live component.

---

## Fix before/while building the studio

Twelve items, highest value first. Effort: S < 1h, M 1–4h, L > 4h.

1. **The string catalog is disconnected from every workbench string.**
   `ui/src/i18n/locales/en.json` (556 leaves, last changed Nov 13) contains **0** `workbench.*` keys,
   while the workbench makes **≈940** `t("...")` calls (`ui/src/workbench/**`). Every studio and
   component string renders from its inline `defaultValue`; `ja.json`/`ar.json` are structurally
   incapable of translating the workbench, and `pnpm lint` fails with 519 `strings/no-unused-keys`
   errors on the generated registry.
   *Why:* locale files will drift on every new string, and the handoff gate in PLAN §5.6 cannot pass.
   *Owner:* platform. *Effort:* L (generate keys from usage, then have each stream adopt `strings.x`).

2. **Studio checks can report "pass" without evidence.**
   `ui/src/workbench/pages/DesignStudioPage.tsx:261` — "A primary action is declared" passes when
   `primary + slots > 0`, so a component slot alone satisfies it.
   `:286` — "Component evidence is attached" passes on the presence of a `componentId` query param.
   `:232` — "Page route is declared" passes from the selected API page, not from the design.
   *Why:* PLAN P3 acceptance is "no check is pass without evidence".
   *Owner:* studio. *Effort:* M (reuse `studio/checks.ts` verdicts; make these `unmeasured` or
   measured).

3. **The page selector is decorative; it does not bind a design.**
   `useStudio()` accepts no scenario/page, `DesignStudioPage.tsx:89` reads `page` from the URL and
   `:162` only uses it for the route claim and a `<Select>`. Switching page changes the label and
   "route" verdict but not the frames being edited or saved. This is the surviving form of
   "pages come from the API while the design is local".
   *Why:* PLAN P2/P4 assume a design belongs to a page; export/review will attach to the wrong one.
   *Owner:* studio (with API if frames gain a page binding). *Effort:* M.

4. **`PreviewPage` is the last hand-built browser chrome, and its device metrics disagree.**
   `ui/src/workbench/pages/PreviewPage.tsx:147-155` still renders `.wb-device*` dots + URL bar; the
   new `DeviceFrame` exists and is used only by the studio canvas. Heights disagree across three
   tables: `PreviewPage.tsx:11,155` phone 720/tablet 760; `InContextStage.tsx:88-98` phone 844/tablet
   1112; `studio/model.ts:8-12` phone 844/tablet 900.
   *Why:* three device truths means a screenshot of "phone" describes three different viewports.
   *Owner:* context (frames owns `DeviceFrame`). *Effort:* S–M (delete `wb-device` markup, use the
   primitive, one metrics table).

5. **`FixtureFrame` has no failure path: a bad harness URL loads forever.**
   `ui/src/workbench/component/FixtureFrame.tsx:149-172` only leaves `"loading"` when the iframe
   posts `preview-ready`; there is no `onError`, no timeout, and no HTTP-status check. A 404/500 from
   `/preview/{id}/harness.html` leaves a permanent spinner. The same iframe grants
   `sandbox="allow-scripts allow-same-origin allow-forms"` (`:172`), which negates origin isolation.
   *Why:* every fixture and in-context slot depends on this one component.
   *Owner:* context. *Effort:* S.

6. **294 lines of dead studio CSS after the rewrite.**
   `ui/src/workbench/pages.css:1139-1432` (`.wb-studio*`, `.wb-artboard*`, `.wb-layer*`,
   `.wb-page-item*`, `.wb-inspector-section*`, `.wb-contract-row*`, `.wb-compare`, `.wb-swatches`,
   `.wb-color`, `.wb-brief*`) has **zero** references in any `.tsx`; the `.wb-studio` rules inside the
   ≤1180/≤980/≤720 media queries (`:2111-2231`) are dead too. The live studio uses
   `studio/studio.css`.
   *Why:* PLAN §5.5 "delete what you replace"; dead rules still ship in the 90 KB CSS bundle.
   *Owner:* studio / debt. *Effort:* S (pure deletion).

7. **Deep links to a design are silently ignored.**
   `ui/src/routes.ts:21-22` and `App.tsx:128` register `/design/:scenario` and `/design/:scenario/:page`,
   but `DesignStudioPage.tsx:85` reads only `useSearchParams`; path params are never read. The URL
   helper `designPath()` (`routes.ts:90`) has **zero** call sites, so nothing builds those links today —
   but any external link or bookmark does the wrong thing.
   *Why:* the public URL contract is advertised but unimplemented.
   *Owner:* studio. *Effort:* S (read `useParams` into the same state, or delete the two routes and
   `designPath`).

8. **`DeviceFrame` exposes the whole composition as an image.**
   `ui/src/workbench/component/DeviceFrame.tsx:99` gives the viewport `role="img"` + `aria-label`, with
   the rendered page as its children. Screen readers announce a single image and none of the page
   text or controls inside. The wrapper also takes `role="button"`/`tabIndex` when `onSelect` is set
   (`:78-79`), which lint flags (`jsx-a11y/no-static-element-interactions`,
   `no-noninteractive-tabindex`).
   *Why:* the canvas is the primary surface; "the layer tree is the keyboard path" only holds if the
   canvas is otherwise exposed honestly.
   *Owner:* frames. *Effort:* S (aria-hidden + text summary, or a labelled `group`).

9. **`pnpm lint` is red, so the PLAN gate is not satisfiable.**
   Measured 13:26: exit 1, **664 problems / 649 errors** in 71 s. 519 are unused catalog keys
   (item 1), 87 are `no-restricted-syntax` (mostly literal Testing Library queries in existing
   tests), plus 15 `react-refresh`, 13 `require-await`, accessibility and `no-floating-promises`
   errors. The new `DeviceFrame.tsx` alone contributes 3.
   *Why:* PLAN §5.6 requires `pnpm lint` before handoff; today every stream hands off red.
   *Owner:* platform (mostly resolved by item 1). *Effort:* M.

10. **Present mode overflows a 390 px viewport.**
    `.studio-present-frame` (`studio/studio.css:1429-1449`) is `overflow:auto` but has no
    `max-width`, while the inline width is the frame's authored width — up to 1180 px
    (`DesignStudioPage.tsx:935-937`). At 390 the dialog's grid child is wider than the viewport; the
    internal scroller never engages.
    *Why:* PLAN §7 requires a 390×844 pass.
    *Owner:* studio. *Effort:* S (`max-width:100%` + scale-to-fit, as the canvas already does).

11. **API errors are mislabelled as client errors.**
    `api/handlers/sketch/connect_handler.go:456-457` logs and returns
    `connect.CodeInvalidArgument` for *every* unmapped error, so a disk/permission/parse failure
    surfaces to the studio as HTTP 400 "invalid argument".
    *Why:* the studio cannot distinguish "my request was wrong" from "the server is broken", and
    `retry` policies fire on the wrong class.
    *Owner:* api. *Effort:* S (default to `Internal`, keep explicit mappings).

12. **Four check renderers, three verdict vocabularies.**
    `ComponentPage.tsx:631` (`ChecksList`, `pass|warn|fail|unmeasured`), `InContextChecks.tsx:27`
    (`pass|fail|warn|unmeasured|info`), `StudioInspector.tsx:346` (studio claims), and
    `AssetDataDrawer` experience chips (`pass|fail|unmeasured`). Each re-implements the icon/label/
    colour mapping.
    *Why:* a new verdict (e.g. "stale") has to be added in four places; the studio and in-context
    disagree about whether `info` counts toward the score.
    *Owner:* context (owns the measured rows) with studio as consumer. *Effort:* M.

---

## Correctness and contradictions

- **Studio route claim is about the API page, not the design** — `DesignStudioPage.tsx:230-236`
  (see item 2). Selecting a page also does not load that page's design (item 3).
- **"Evidence attached" is a query-param check** — `DesignStudioPage.tsx:280-288`: `componentId`
  present ⇒ pass, regardless of whether the fixture resolved (`StudioSlot` can render
  "Component unavailable" and the claim still passes).
- **Contrast is permanently unmeasured in the studio** — `DesignStudioPage.tsx:289-296` hardcodes
  `unmeasured` even though `instruments.ts` now computes real contrast ratios and the in-context view
  reports them.
- **`save`/`accept` used to flash success on a failed write** — the rewritten page delegates to
  `useStudio` and now surfaces `saveFailed` (`StudioStatusBar`), which is the right shape; keep it.
  (Old inline `localStorage.setItem` + success toast is gone; do not reintroduce.)
- **Template shelf is still browser-local while designs are server-owned** —
  `templates/local.ts:12,40-56` writes `rcl.workbench.templates.v1`; `designs.ts` now writes
  `/api/designs` with a localStorage fallback (`designs.ts:45-96`). Saving a frame as a template
  (`DesignStudioPage.tsx:420-442`) produces an object that only this browser can see.
- **Fixture arrangement is a third local store** — `ComponentPage.tsx:52-65,140-150` keeps
  `rcl.workbench.fixtures.<asset>.v1`; it is not tied to the design or the component version.
- **Local fallback can silently win** — `designs.ts:69-71` treats 404/5xx/network as "API
  unavailable" and falls back; a misconfigured API base therefore looks like a working local studio
  until the browser is cleared. Surface the fallback in the status bar.
- **Unknown template node types disappear silently** — `TemplateRenderer.tsx:706-708`
  (`default: return null`); a typo in `spec.ts` renders an incomplete page with no error.

## Duplication

- **Browser chrome:** `PreviewPage.tsx:147-155` (live), `DeviceFrame.tsx:89-96` (live),
  plus the shared `.wb-device*` rules in `pages.css:1047-1100` that PreviewPage still uses. (The
  in-context stage adopted `DeviceFrame` at 13:31 — good.)
- **Device metrics:** `PreviewPage.tsx:11,155`, `InContextStage.tsx:88-98`, `studio/model.ts:8-12`
  (see item 4).
- **Device segmented control:** the same three-icon `Segmented` block is written out in
  `StudioTopBar.tsx` (device switch), `PreviewPage.tsx:98-124`, and previously in
  `InContextStage` — one `DeviceSwitch` primitive would own the labels.
- **Verdict → icon/colour mapping:** `ComponentPage.tsx:645-668`, `InContextChecks.tsx:45-56`,
  `StudioInspector.tsx:346-360`, `pages.css:1373-1381` (dead) — see item 12.
- **`checks` computation:** `DesignStudioPage.tsx:214-298` walks the spec inline while
  `studio/checks.ts` already exposes `stateAudit`, `layoutVerdicts`, `underTargetButtons` and is not
  used for these six claims (`layoutVerdicts` is imported but unused).
- **localStorage read/write + try/catch:** `templates/local.ts:28-47`, `designs.ts:83-99`,
  `ComponentPage.tsx:56-65`, `preferences.ts:26-38` — four copies of the same guarded JSON store.
- **"Take into studio" URL is built twice** — `ComponentPage.tsx:293-299` and `:595-601`.
- **Dead duplicate DDL for `reconcile_evidence`** — `internal/gates/calibration_surface.go:29-35`,
  `calibration_databases.go:60-66` and `:102-108` (see API section).

## Hardcoded strings

The catalog has no `workbench.*` keys (item 1), so every workbench string is inline. Beyond that,
strings that bypass `t()` entirely and are user-visible:

- `ui/src/workbench/studio/model.ts:176-193` — 14 `ADDABLE_NODES` labels ("Text", "Heading",
  "Card", "Action", …) render directly in the "Add a layer" `<option>` list
  (`StudioLayers.tsx:195-206`).
- `ui/src/workbench/templates/spec.ts:310-327` — `LAYER_NAMES` (17 labels) is the visible layer-tree
  label via `layerLabel()`; also composes `${name} · ${suffix}`.
- `ui/src/workbench/data.ts:490-530` — `formatRelative`/`formatDay` return English "just now",
  "Today", "Yesterday" outside `t()`; `formatRelative` is used by the studio revision history and
  the asset drawer.
- `ui/src/workbench/plural.ts` — `pluralDefault` only has English `_one`/other rules.
- `ui/src/workbench/templates/local.ts:113-154` — the blank template seeds five English copy strings
  into the composition (`blankTemplate`), which then become design content.
- `ui/src/workbench/templates/spec.ts:347-382` — `templateStyle` hardcodes `--color-danger`,
  `--color-success`, `--color-warning` as hex (see below), so those strings of "theme" cannot be
  localized by palette.

Counts: ≈940 literal-key `t()` calls in `ui/src/workbench` (≈1,350 across all `.tsx`); 0 of those
keys exist in `en.json` (measured 13:44).

## Hardcoded design values

The new `studio/studio.css` (1,696 lines) and `component/incontext.css` (526 lines) contain **zero**
hex/rgba literals — the token rule is being honoured in new code. The offences are concentrated in
legacy files:

- `ui/src/workbench/pages.css` — 55 hex + 42 `rgb()`/`rgba()` literals. Live ones include the preview
  chrome: `#fff` (`:1053`, `:1090`), `#141219` (`:1064`), `#3a3340` (`:1071`), `#0d0b11` (`:1080`),
  `rgba(0,0,0,.45)` (`:1054`). The 294 dead lines (item 6) contain a further 9 literals.
- `ui/src/workbench/nocturne.css:880` — `.wb-input`/`.wb-select`/`.wb-textarea` background
  `#0d1018`, a cool blue-black in a warm palette and not a `--wb-*` token.
- `ui/src/workbench/templates/TemplateRenderer.tsx:539` — card shadow `0 18px 50px rgba(0,0,0,.14)`;
  `:649` — swatch radius `6`.
- `ui/src/workbench/component/device-frame.css` — 6 hex + 9 rgba for bezel/island/home/safe-area
  (`#0a0a0c`, `#14151a`, `#fff`, `rgba(255,179,92,…)`). Physical hardware colours are defensible, but
  they should be `--df-*` locals so the frame has one palette owner.
- **Type scale:** 26 declarations of 7–9 px mono text in `pages.css` (e.g. `:1082`, `:1249`, `:1285`)
  and 2 in `nocturne.css`. Several are essential labels (layer type, frame bar, revision hash), not
  decorative.
- `ui/src/workbench/pages/PreviewPage.tsx:63-77` — inline `fontSize:13`, truncation and layout in
  style props rather than a class.

Contrast check (computed, WCAG 2.1): `--wb-muted-2` on `--wb-panel` is 5.16:1 (passes AA small
text); on `--wb-panel-4` it is 4.32:1 (fails AA for the 9 px labels used there). `--wb-line-2`
control borders are 1.53:1 against the panel, below the 3:1 non-text requirement.

## Dead code

- **`ui/src/workbench/pages.css:1139-1432`** — 294 lines of orphaned studio CSS, plus
  `.wb-studio*` rules in the ≤1180/≤980/≤720 media queries (`:2111-2231`). Zero TSX references.
- **`ui/src/workbench/designs.ts:197`** — `deleteDesign` exported, never called. The API implements
  DELETE; the studio has no delete affordance (PLAN P2 lists delete).
- **`ui/src/workbench/data.ts:72`** — `useHealthOverview` never called.
- **`ui/src/workbench/data.ts:556,560`** — `assetEffectiveAdoptions`, `isRenderable` never called.
- **`ui/src/routes.ts:90`** — `designPath()` never called (see item 7).
- **`ui/src/workbench/component/TemplateRenderer.tsx:706-708`** — `default: return null` silently
  drops unknown node types.
- **Stale one-off scripts** — `ui/scripts/codemod-stylesheet-keys.mjs` (+ test) and
  `ui/scripts/strip-token-fallbacks.mjs` are not referenced by any package script.
- **Kept on purpose** — `appRoutes.legacySketch` + `SketchRedirect` (`App.tsx:78-89`) preserve the
  published `/sketch/...` URLs; `/catalog` redirect (`App.tsx:72-76`) likewise. Do not delete these.

## Performance and reliability

- **Every route pays for the shell's queries** — `WorkbenchShell.tsx:76-78` mounts `useCatalog()`
  (600 assets), `useCoverage()` (gate corpus, 60 s deadline, 8 s polling while warming) and
  `useDesignScenarios()` on *every* page, including `/design` and `/assets/:id`.
- **`ListDesignPages({})` walks the whole fleet** — `api/internal/sketch/inventory.go:35-80` reads
  `scenarios/` and opens each `experience/pages` directory. It is cached in react-query for 120 s but
  has no server-side cache and is called by the shell as well as the studio.
- **`useComponentBundle` is 5 requests** (`data.ts:228-279`) per component; the studio calls it per
  active component and per `StudioSlot`, deduped only by the query key. Fine today, but a design with
  N distinct components is 5N requests with no batching.
- **Un-cleaned timers** — `ComponentPage.tsx:140-150` debounced position write has no unmount
  cleanup; `ComponentPage.tsx:224` and `PreviewPage.tsx:39` clear the "Copied" state with
  `setTimeout` and no cleanup (setState after unmount).
- **Drag listeners can outlive the component** — `TemplateRenderer.tsx:196-198` and `:225-227` add
  `pointermove`/`pointerup` to `window` inside the handler with no effect cleanup; unmounting
  mid-drag leaves them attached until the next pointerup.
- **No error boundary inside the studio** — `App.tsx` wraps routes in `ErrorBoundary`, but a render
  error in one frame's `TemplateRenderer` replaces the whole studio. A boundary per frame pane would
  keep the shell usable.
- **Bundle** — the last measured build (`ui/dist`, 11:40) is already split: `CodeEditor` 2.5 MB +
  `ts.worker` 6 MB are lazy (`SourceWorkbench.tsx:32`), `index` 387 KB, `App` 221 KB,
  `page-stories` 230 KB. No action needed now.
- **Fallback honesty** — see "Local fallback can silently win" above.

## Accessibility

- **Canvas as image** — `DeviceFrame.tsx:99` `role="img"` around the composition (item 8).
- **Non-native interactive wrapper** — `DeviceFrame.tsx:72-87` div with click/keydown + `role=button`;
  lint already reports it. Use a real button for the frame selector, or keep selection on the frame
  header.
- **Present mode at 390** — fixed 1180 px frame, no `max-width` (item 10).
- **Small-text contrast** — `--wb-muted-2` on `--wb-panel-4` is 4.32:1 and is used for 9 px labels
  (revision, layer type, frame bar); `--wb-line-2` borders 1.53:1.
- **No focus target on the artboard** — `TemplateRenderer` is intentionally not focusable (comment
  at `:1-4`), but the studio offers no shortcut to move focus into the layer tree or canvas; a
  keyboard user landing on `/design` must tab through rail → top bar → pane tabs.
- **Global Escape handling is duplicated** — `WorkbenchShell.tsx:147-157` closes the command palette
  on every Escape; `useFocusTrap` also listens for Escape. Escape inside a studio sheet closes both
  surfaces.
- **Missing page-level a11y coverage** — `axe-evidence.test.ts` tests the axe producer, not the
  studio or in-context DOM; there is no axe run for `/design` or `/assets/:id?mode=context`.

## API

- **Error classification** — `handlers/sketch/connect_handler.go:456-457` (item 11).
- **Mixed error shapes in one module** — `handlers/sketch/module.go:136-185` (reference-asset upload
  / serve) returns plain-text `http.Error`, while the Connect service in the same module returns
  JSON `ErrorEnvelope`s; `handlers/designcapture/target.go` returns plain text too. A client cannot
  branch on one shape.
- **Schema ownership** — `internal/gates/calibration_surface.go:29-35`,
  `calibration_databases.go:60-66,102-108` create `reconcile_evidence` /
  `reconcile_evidence_viewports` with inline `CREATE TABLE` in **production** files, three copies.
  Other domains use `Schema()` + `EnsureSchemas` (e.g. `internal/designs/schema.sql`). This is the
  one domain that bypasses the seam.
- **Missing handler test** — `ListDesignPages` (the studio's only sketch RPC) has no
  `handlers/sketch` test; only `internal/sketch/inventory_test.go` covers the reader. A proto/route
  mismatch would not be caught.
- **Fleet scan without cache** — `internal/sketch/inventory.go:35-80` (see Performance).
- **Validation is thin on the REST surface** — `handlers/sketch/module.go:136-150` trusts
  `scenario`/`page` form values; the internal `SaveReferenceAsset` performs the real checks, so the
  handler cannot return a field-specific 400.
- **Designs API is new and coherent** — `api/handlers/designs/module.go` (5 routes),
  `api/internal/designs/schema.sql`, JSON envelope `{"error":{code,message}}`, 5 endpoint tests.
  It is registered in `api/main.go:332` and the UI calls it with a bounded fetch. No debt found
  beyond "saves are last-write-wins (no 409)" being an explicit product decision recorded in
  `docs/studio/HANDOFF-api.md`.

## Test gaps

- No page-level test for `/design` or the in-context view: `App.test.tsx` mocks the API but only
  asserts routes resolve; the studio ships `DesignStudioPage.story.json` with no `.test.tsx`.
- `pnpm test` covers pure studio maths well (`studio/model.test.ts`, `history.test.ts`,
  `checks.test.ts`, `rcl.test.ts`, `component/DeviceFrame.test.tsx`), but not the studio shell,
  in-context instruments, or the `designs.ts` fallback (only `designs` handler tests on the API side).
- No test asserts that a "pass" claim has evidence — the exact P3 acceptance criterion.

---

## Do not fix now

| Item | Reason |
|---|---|
| `TestCatalogSelectorResolvesThroughRealImportedSource` (`api/internal/reconcile`) | **Fixture drift, not a product defect.** The page contract binds `[data-experience-surface="catalog-results"]` (`experience/pages/catalog-workspace.json`). That literal now lives in `ui/src/workbench/pages/ComponentsPage.tsx:194`; `ui/src/App.tsx` contains none. The test pins a fake scanner at `ui/src/App.tsx` (`resolver_test.go:231`), so it resolves zero candidates. `resolve-imports.mjs` confirms the literal is detectable in ComponentsPage and absent in App.tsx, so this is neither an environment nor an AST failure. Commit `d70387a700d` (workbench redesign) moved the surface. Fix belongs to the test owner: point the fixture at `ComponentsPage.tsx` (or scan the real inventory). One line; do not change product code. |
| `TestLiveUtilityClassAllowlistIsExactAndShrinkOnly` (`api/internal/gates`) | Committed library debt: `library/components/AppNotificationSettings/versions/1.0.6/AppNotificationSettings.tsx:133-154` emits `grid`, `gap-space-md`, `text-sm`, `divide-y`, `rounded-lg`… not in `library/utility-class-allowlist.json`. Requires the library/asset workstream, not the studio. Reproduced with the designs changes stashed (see `HANDOFF-api.md`). |
| `TestLiveReleaseProvenanceCoversEveryMaterializedRelease` (`api/internal/gates`) | Committed corpus debt: `BottomSheet@1.3.7`, `Dialog@1.3.10`, `DrawerShell@1.1.9`, `FullPageDrawer@1.5.2`, `OverlayActionBar@1.0.0` have no publish/backfill record in `library/release-provenance.json`. Needs a governed publish or an explicit backfill decision; out of scope for the studio. |
| New-stream type/lint errors in `workbench/studio/**` and `InContextStage.tsx` | At snapshot `tsc` reports 2 errors in `InContextStage.tsx` (unused `SlotBox`, one `t()` overload). They are being actively edited (files changed at 13:32-13:35); re-run before filing. |
| `ui/src/workbench/component/FixtureFrame.tsx` sandbox tightening | Changing `allow-same-origin` can break the harness message protocol; do it with the context owner and a fixture that proves `postMessage` still works. |
| Splitting `page-stories` / RCL vendor chunks | Already lazy where it matters (Monaco); no measured user-facing cost. |
| Deleting `/sketch` redirects and `/catalog` redirect | Published URL contract; keep. |
| Converting the four check renderers in one pass | Wait until the studio/in-context verdict vocabularies settle (item 12 is the seam; the rest is mechanical). |

---

## Validation record

All commands run from `/home/matthalloran8/Vrooli/scenarios/react-component-library` unless noted.
`make check` was **not** run: it invokes `pnpm format` / `lint:fix` (write paths), which this audit is
not allowed to execute.

### UI tests

| Time | Command | Result |
|---|---|---|
| 13:22 | `cd ui && pnpm test` | **95 files / 462 tests passed**; duration 10.15 s; wall 10.83 s. Matches the stated baseline. |
| 13:30 | `cd ui && pnpm test` | 100 files / 490 tests, **2 failed** — `studio/checks.test.ts` ("expected 5 to be 4"), `studio/rcl.test.ts` ("expected meter… to deeply equal card…"). Both files were created at 13:23-13:24 and are uncommitted. |
| 13:38 | `cd ui && pnpm test` | **100 files / 490 tests passed**; duration 9.86 s; wall 10.80 s. The two WIP tests were fixed by their owner in the interim. |

### API tests

| Time | Command | Result |
|---|---|---|
| 13:20 | `cd api && go test ./... -count=1` | **3 failing tests in 2 packages**, wall 38.86 s: `internal/reconcile` (0.65 s) and `internal/gates` (12.68 s, two tests). Everything else `ok`. |
| 13:19 | `cd api && go test ./internal/reconcile -run TestCatalogSelectorResolvesThroughRealImportedSource -v` | `resolver_test.go:239: catalog binding not proved: … ReasonCode:"binding_unresolved" … Reason:"no source evidence resolved the region"`; 0.30 s. |

The stated baseline ("one pre-existing failure") under-counts: the two `internal/gates` failures were
reproduced here and are independently confirmed as pre-existing in `docs/studio/HANDOFF-api.md`
("reproduced with this stream's changes stashed"). They are library-content gates, not studio code.

### Static gates

| Time | Command | Result |
|---|---|---|
| 13:26 | `cd ui && pnpm lint` | **exit 1**, 664 problems (649 errors, 15 warnings), 71.03 s. Top: `strings.generated.ts` 519 (`strings/no-unused-keys`), `RegionCorrectionProvider.tsx` 26, `async-boundary.test.tsx` 16, `designs.ts` 6 (now fixed by the API stream), `DeviceFrame.tsx` 3. |
| 13:31 | `cd ui && pnpm type-check` | **exit 1**, 69.06 s. `catalog-conformance` tsc: 3× TS6133 in `workbench/studio/*`; token/pins/exports/usage-contract checks all passed (219 library defaults, 25 app-local tokens; 274 assets / 587 versioned exports). |
| 13:34 | `cd ui && ./node_modules/.bin/tsc --noEmit -p tsconfig.json` | 3 errors: `InContextStage.tsx` unused `SlotBox`, `InContextStage.tsx` `t()` overload, `ComponentPage.tsx:476` passing removed props. |
| 13:36 | same | 2 errors, both `InContextStage.tsx`; the `ComponentPage` seam was fixed by the context stream at 13:35. |
| 13:39 | `pnpm test` (see above) | green at 100/490. |

### Evidence for the reconcile diagnosis

```
node packages/react-component-library/tooling/resolve-imports.mjs \
  --binding-facts-root scenarios/react-component-library/ui/src/workbench/pages/ComponentsPage.tsx
→ [{"file":".../ComponentsPage.tsx","elements":[{"tag":"div","attributes":{"data-experience-surface":["\"catalog-results\""], ...}}]}]

node .../resolve-imports.mjs --binding-facts-root .../ui/src/App.tsx
→ no data-experience-surface attribute in the output
```

### Screenshots (PLAN §7)

**Both absent.** `docs/studio/` contained only `PLAN.md` and `HANDOFF-api.md` before this file was
added. A repo-wide search (`find . -path ./library -prune -o -name '*.png'`) returns no `/design` or
in-context capture at 1440×900 or 390×844; the only images in the scenario are published
`library/**/preview.png` assets. No served-run artifact, interaction trace, or handoff note records a
capture, and no served run is evidenced in the tree at snapshot.

### What changed during the audit (so the record is reproducible)

`git status --short` at 13:43 shows modified `api/{main.go,internal/module/module.go,
internal/modules/registry.go}`, `WorkbenchShell.tsx`, `FixtureFrame.tsx`, `InContextStage.tsx`,
`ComponentPage.tsx`, `DesignStudioPage.tsx`, `templates/library.ts`, `templates/spec.ts`; new
`workbench/studio/**` (20 files), `component/{DeviceFrame.tsx,DeviceFrame.test.tsx,device-frame.css,
InContextChecks.tsx,contextChecks.ts,contextChecks.test.ts,incontext.css,instruments.ts,
instruments.test.ts,placement.ts,placement.test.ts}`, `workbench/{brand.css,useBrand.ts,designs.ts}`,
`api/{handlers,internal}/{designs,brand}/**`, and `docs/studio/**`. Re-run the commands above before
acting on any single line number.

---

## Wave 3 (2026-11-19): found and fixed

| Finding | Why it mattered | Fix | Evidence |
|---|---|---|---|
| **Template create posted to `/api/templates/{id}`** | The server has `PUT` there, not `POST`, so every new template was silently browser-local despite the green unit suite (the mock accepted any URL). | Create posts to the collection; update addresses the id. Added two route-asserting tests. | Live: save → `/api/templates` lists it; reload → the shelf shows it; delete → 204. |
| **A network-failed save read as "saved"** | `saveDesign` fell back to localStorage and returned a doc; `persist` adopted its revision, cleared dirty and **broadcast a server revision that never existed**. | `putDesign` returns `{design, local}`; `SavedState.local`; no broadcast for local writes; status shows **Local draft**; reconciliation treats it as dirty; retry on `online`. Local writes no longer invent a revision, so the retry can succeed. | Live: blocking PUT showed Local draft, then the conflict banner instead of silently overwriting the only copy. |
| **`Field` labels were not associated with controls** | No `htmlFor`/`id` on any of 39 inspector fields; screen readers announced an unlabelled input and clicking a label did not focus it. | `Field` generates a stable `useId`, keeps a child's own id, and clones the single control with it. | `primitives.test.tsx`: `getByLabelText` resolves both cases. |
| **Selecting a template destroyed the composition** | A `<select onChange>` applied the template immediately: browsing the shelf replaced all frames. Undoable, but surprising and easy to lose work. | Selection only changes the choice; an explicit **Apply template** button commits (with a title explaining Undo). | `DesignStudioPage` shelf actions; live shelf test. |
| **Gateway rejected the patch schema** | The gateway enforces a JSON Schema **subset**. `type` arrays and `maxProperties` produced `INFERENCE_ERROR_CODE_UNSUPPORTED_SCHEMA`, which surfaced as a generic 502. | Schema uses only supported keywords; scalar values are checked semantically. The subset is documented in `HANDOFF-ai.md`. | Three iterations against the live gateway; final proposal in 2.6 s. |
| **Ctrl+Z did nothing on the canvas** | A design tool without the standard shortcut is a daily papercut; the command stack existed but only the toolbar reached it. | `shortcuts.ts` (+ tests) and a page listener that ignores text fields. | `shortcuts.test.ts`. |

### Wave-3 debt that remains (not hidden)

- `DesignStudioPage.tsx` is ~1,150 lines and `useStudio.ts` ~780. They are cohesive but large; the
  next structural move is extracting page-context (binding + templates) and the proposal wiring
  into their own hooks.
- AI is one-shot by design. There is no streaming, model picker, conversation, or image input; the
  gateway owns model routing.
- The AI schema validator is duplicated in Go and TypeScript. The two serve different jobs
  (server: applicability against stored JSON; client: applicability against the live tree) but a
  shared fixture set would catch drift earlier. `patch.test.ts` and `designassist` tests currently
  cover the same cases independently.
- HTML import and extractability remain open (PLAN next wave 1–2).
- The three pre-existing Go failures are still library/fixture debt; do not weaken the tests.

---

## Wave 4 (2026-11-19): found and fixed

| Finding | Why it mattered | Fix | Evidence |
|---|---|---|---|
| **The chrome controls disagreed with the canvas** | `StudioChrome` fell back to 14/sans while `frameTemplate` falls back to `DEFAULT_TEMPLATE` (16/serif). An old design with `options:{}` showed a value the frame was not rendering. | Both use the template defaults; the fallback test asserts 16 + Serif. | `StudioChrome.test.tsx`; live Aperture (options `{}`) matches. |
| **An AI proposal could follow a tab switch** | `StudioAssist` captured only a signature. A proposal made for design A could be offered (and, with two identical copies, applied) after switching to design B. | Proposals carry `designId`; the design-change effect aborts and clears; apply refuses a mismatched design; only the signature decides staleness so autosave does not invalidate a valid patch. | Live: start a proposal, switch tabs — zero proposals and no stuck busy state; `useStudio.test.tsx` covers the mismatch. |
| **Extraction overflowed the 245 px pane** | The candidate list was a flex item with `min-width:auto`, so the nowrap label made the list 300 px and pushed "Select example" over the canvas. | `min-width:0` on the list/item, ellipsis on the label, a two-row block layout. | Live measurement: list 245 px, button ends inside the pane; screenshot `studio-extract.png`. |
| **Imported layers read as `Stack · html 9 stack`** | The layer label fell back to the import's synthetic id; a tree of ids is unreadable. | Containers carry a source-role label with their own heading (`Section · Why now`, `Header`, `List`); the extraction label uses the authored role, not twice. | `htmlImport.test.ts`; screenshot `studio-imported.png`. |
| **AI idempotency reuse and trailing JSON** | A repeated key with different inputs surfaced as a 503 "gateway unavailable"; the trailing check used `Decoder.More` (array semantics). | `ErrConflict` maps to 409 `conflict`; trailing JSON is checked with `io.EOF`. | `handlers/designs` tests pass. |

### Wave-4 debt that remains

- HTML import does not interpret CSS or fetch assets (by design). A remote `<img>` becomes a warning,
  not a broken layer.
- The region list is every importable element, so a large document produces a long list. It is
  scrollable and keyboard reachable, but a block-level filter would be a better default.
- Extraction is single-frame and shape-only: it cannot see that two different shapes mean the same
  thing, and it does not compare across designs or ignore the three device copies.
- A design created while the API is unreachable keeps a browser id and cannot be promoted later;
  it is retained as a local draft. This is the next offline feature, not a hidden success.
- `DesignStudioPage.tsx` is now ~1,190 lines and `useStudio.ts` ~800. Both are cohesive, but the
  page-context (binding + templates + import + chrome) and the asset slice are the next extraction
  targets.
