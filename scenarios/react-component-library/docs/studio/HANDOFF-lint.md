# Handoff — UI lint gate (`pnpm lint` → 0 problems)

Baseline was **136 problems (121 errors, 15 warnings)** in
`scenarios/react-component-library/ui`. This stream owned everything outside
`src/workbench/**`; the workbench streams took the rest.

## Result

| Gate | Command | Result |
|---|---|---|
| Lint | `pnpm lint` (ui) | **0 problems**, exit 0 |
| Scoped lint | `pnpm exec eslint src/components src/lib src/providers src/test-utils src/hooks src/i18n src/axe-evidence.test.ts src/catalog-draft-story-behavior.test.tsx` | **0 errors**, exit 0 |
| Types | `pnpm exec tsc --noEmit` | **0 errors**, exit 0 |
| Tests | `pnpm test` | **108 files / 577 tests passed** |
| Prettier | `pnpm exec prettier --check` on touched files | clean, except files that were already unformatted at `HEAD` (see *Left as-is*) |

`pnpm lint` is now usable as a gate. Nothing was committed, built, or
restarted.

## Rule → fix pattern

| Rule | Count | Pattern used |
|---|---|---|
| `no-restricted-syntax` (tests) | 66 | Name the string once in the test file, then query by reference: `const copy = { loading: "Loading content" } as const; … getByText(copy.loading)`. Catalog-owned copy uses `strings.x.y` (`getByLabelText(strings.catalog.sourcePath)`, `strings.regionCorrection.*`). Fixture children (`<p>Ready results</p>`) get the same treatment so the render and the query share one symbol. No assertion was deleted, no `getByTestId` invented, no test became less specific. |
| `no-restricted-syntax` (prod) | 19 | `t("key", { defaultValue: "…" })` with interpolation for `{{ownerScenario}}`, `{{component}}`, `{{evidenceId}}`, `{{key}}`, `{{label}}` (3 in `ActionLauncher`, 16 in `RegionCorrectionProvider`). |
| `react-refresh/only-export-components` | 15 | One file-level comment per `export *` barrel: `/* eslint-disable react-refresh/only-export-components -- published re-export barrel; the rule cannot see through `export *`. */` |
| `@typescript-eslint/require-await` | 8 | Dropped `async` where the body was synchronous (`async-boundary`, `catalog-draft-story-behavior`, `live-announcer`). Where the async `act()` was load-bearing, the callback now resolves the deferred **and** awaits a microtask: `await act(async () => { resolve(); await Promise.resolve(); })`. |
| `@typescript-eslint/no-floating-promises` | 5 | `act(() => vi.advanceTimersByTime(n))` returned the `Vitest` object (a thenable), so the arrow now uses a block body: `act(() => { vi.advanceTimersByTime(n); })`. |
| `@typescript-eslint/no-deprecated` | 5 | `SidebarShell.test.tsx` migrated off deprecated `width` / `resizeHandleProps` to `resizable={{ containerRef, min, max, defaultSize }}` and asserts `--rcl-panel-size` instead of `style.width`. `useDeviceEmulation.test.ts` hoisted the `querySelector` spy into a local const instead of reading the DOM method off the object (which also cleared `unbound-method`). |
| `no-implied-eval` | 2 | Kept the `new Function(...)` boundary probes, added a one-line `eslint-disable-next-line … -- test constructs a function to prove the boundary rejects it` (and the axe-helper equivalent). |
| `no-unnecessary-condition` (`i18n/index.ts:64`) | 1 | `window.navigator.language` is always a string → removed `?? ""`. |
| `design-system/no-raw-dimensions` | 1 | `max-h-48` (12rem) → the published token class `max-h-content-short` (`--layout-content-short: 12rem`) — same pixels, no raw dimension. |
| `no-unnecessary-type-assertion` | 1 | `value={value as string}` → `value={value}`; `SelectHTMLAttributes.value` already accepts `string \| readonly string[]`. |
| `jsx-a11y/no-static-element-interactions` | 1 | Scoped disable on the provider wrapper with the rationale that capture-phase pointer handlers only *observe* gestures. |
| `jsx-a11y/no-noninteractive-element-interactions` | 1 | Scoped disable on the dialog backdrop with the rationale that it is overlay dismissal, not element interaction (`role="button"` deliberately not added). |
| parse error | 1 | `src/hooks/useMediaQuery.test.tsx` was never in the TS program: `useMediaQuery.test.ts` and `.tsx` share a basename and the project service (and `tsc --listFilesOnly`) only picks the `.ts`. Renamed to `useMediaQueryTablet.test.tsx` (it covers the `useIsTablet` listener contract). Both files and all their coverage survive. |

## RegionCorrectionProvider: the type was the lie

`validEvidence` validates **untrusted host input**, so its parameter is now
`unknown`:

```ts
function validEvidence(value: unknown): value is ResolvedRegionCorrection
```

Every field is re-read through small narrowing helpers — `isRecord`,
`asString`, `asNumber` — so `schema`, `kind`, `storage`, `private`, `origin`,
`fidelity`, ids, paths, viewport and bounds are all checks that can actually
fail. `no-unnecessary-condition` / `no-unnecessary-boolean-literal-compare`
went away because the comparisons are now meaningful, not because a check was
dropped: all previous checks survive, plus structural checks that the old code
got "for free" from the type (missing `origin`/`capture`/`region`/`crop`/
`bounds` objects, a `hub-packet` origin without `attachment`, an unknown origin
kind).

New tests in `RegionCorrectionProvider.test.tsx` (each asserts a status line
and no dialog, i.e. rejected without throwing):

- wrong `schema` (`ux-region-feedback/v2`)
- wrong `kind` (`feedback`)
- `private: false`
- missing `origin`
- out-of-range `bounds` (`x 0.9 + width 0.5`)

## New catalog keys

The coordinator's `strings:catalog` run has already picked these up (they are
in `en.json` and `strings.generated.ts` as of this handoff), which is why the
provider's test can now assert `strings.regionCorrection.*` instead of literal
key paths.

- `launcher.filterChoices`, `launcher.searchPlaceholder` (`Search {{label}}`),
  `launcher.noMatches`, `launcher.selectOption`
- `regionCorrection.cancelSelection`, `.close`, `.closeLabel`, `.component`,
  `.cropAlt` (`Evidence crop for {{component}}`), `.cropCaption`
  (`Saved crop · {{evidenceId}}`), `.description` (`… queued locally for
  {{ownerScenario}} …`), `.messageLabel`, `.messagePlaceholder`, `.region`,
  `.report`, `.route`, `.shortcut` (`Shortcut: Shift + Alt + {{key}} …`),
  `.submit`, `.tapInstruction`, `.title`, `.viewport`

## Files touched

**Production (3):** `src/components/ActionLauncher.tsx`,
`src/providers/RegionCorrectionProvider.tsx`, `src/i18n/index.ts`.

**Re-export barrels (15):** `src/components/{BottomNav,Button,Card,DataTable,Dialog,EmptyState,Icon,IconButton,Input,Pressable,Select,SidebarShell,StatusBadge,Text,Textarea}.tsx`.

**Tests / hooks / utils (28):** `src/axe-evidence.test.ts`,
`src/catalog-draft-story-behavior.test.tsx`,
`src/components/ActionLauncher.test.tsx`, `OverlayActionBar.test.tsx`,
`SidebarShell.test.tsx`, `adopted-foundations.test.tsx`,
`async-boundary.test.tsx`, `data-table.test.tsx`, `navigation-assets.test.tsx`,
`table.test.tsx`, `markdown-harvest/{CodeBlock,MarkdownRenderer,MermaidDiagram,useCodeCopy}.test.tsx`,
`src/hooks/{useDeviceEmulation.test.ts,useMediaQueryTablet.test.tsx (renamed),
useMediaQuery.test.ts}`, `src/lib/{ApprovalPrompt,ComposerAttachmentTrayDraft,IntegrationCardDraft,PromptComposer,useHoverIntentDraft}.test.tsx`,
`src/lib/PreviewCaptureBoundary.test.ts`,
`src/providers/RegionCorrectionProvider.test.tsx`,
`src/services/live-announcer.test.tsx`,
`src/test-utils/{archetypeLayouts,asyncPanel,commandCenterShell,experienceSurface}.test.tsx`.

**Not mine, changed concurrently:** `src/consts/strings.generated.ts`,
`src/i18n/locales/{en,ja,ar}.json`, `src/i18n/locales/locales.test.ts`
(catalog sync), everything under `src/workbench/**`.

## One fix that reached outside the lint list

Removing the `?? ""` in `detectInitialLocale` exposed a leak: three
`markdown-harvest` suites did `vi.stubGlobal("navigator", { clipboard })`,
replacing the whole navigator with a clipboard-only object and never restoring
it. In the single-threaded run that stub survived into the next file's
`test-setup.ts` import, so `i18n` threw `Cannot read properties of undefined
(reading 'split')` for six unrelated suites. Their `afterEach` now calls
`vi.unstubAllGlobals()` before `vi.restoreAllMocks()` (which never restored
globals). Same intent, no test meaning changed.

## Left as-is (with reason)

- **`src/workbench/**`** — owned by two other streams; untouched.
- **Non-JSX copy inside `RegionCorrectionProvider`** — the status/notice
  strings (`"Finding the page evidence…"`, `"Sending correction…"`, …) are set
  through state, not rendered as JSX text or attributes, so
  `no-restricted-syntax` does not cover them. Moving them would need keys plus
  new assertions in a file that already changes here; flagged as a follow-up
  rather than smuggled in.
- **Test-only fixture strings are not in the catalog.** Library-owned copy
  (`AsyncBoundary` headings, `ApprovalPrompt` decisions, `TYPESCRIPT` label,
  `More`, `Access is limited`) belongs to the published package, not this app's
  catalog, so each test names it once at the top of the file. Putting it in
  `en.json` would claim ownership of copy this scenario cannot translate.
- **Prettier** — 10 touched files were already unformatted at `HEAD`
  (`prettier --check` on the `HEAD` blobs reproduces it), and most of the churn
  is pre-existing one-line statements (e.g. `RegionCorrectionProvider.tsx`
  would take ~236 changed lines). Only `src/components/table.test.tsx` — clean
  at `HEAD` and broken by my edit — was reformatted. Running `pnpm format` over
  the whole UI is a separate, mechanical change.
- **`eslint.config.js`** — no config change was needed; every disable is a
  single-line scoped comment with a rationale.

## Follow-ups worth carrying

1. `RegionCorrectionProvider`'s state-held status/notice copy should move to
   `regionCorrection.*` keys in a follow-up (needs the provider's status
   assertions to switch to key paths).
2. `useMediaQuery.test.ts` + `useMediaQueryTablet.test.tsx` are still two files
   for one hook; merging them is safe now that the basename collision is gone.
3. `pnpm format` over the UI would delete the remaining pre-existing
   formatting drift.
