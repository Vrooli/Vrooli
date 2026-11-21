# Handoff — design-owned radius and display (`options`)

Status: server, client and studio model are implemented and tested. The studio
**controls** and the `updateOptions` command are the coordinator's remaining
work (this file gives the exact code); `make endpoints` and a scenario restart
are also still pending on the coordinator's side.

Before this change every frame rendered with `DEFAULT_TEMPLATE.radius` and
`DEFAULT_TEMPLATE.display` (`studio/model.ts:frameTemplate`), so a design could
change its palette but not its chrome. `options` is now a durable, validated,
design-owned object; the library template remains the fallback when a design
never sets it.

## Contract

```ts
interface DesignOptions {
  radius?: number;              // 0–48 inclusive (decimals allowed)
  display?: "sans" | "serif";
}
```

Only those two keys are accepted. `options` is a JSON **object**; it is stored
verbatim once validated and always returned as an object (never `null`).

| Method | Path | Body | Result |
|---|---|---|---|
| POST | `/api/designs` | `{name,frames,theme,options?,scenario?,page?}` | `201 {design}` |
| PUT | `/api/designs/{id}` | `{name,frames,theme,options?,scenario?,page?,revision?}` | `200 {design}` |
| GET | `/api/designs/{id}` | — | `200 {design}` with `options` always present |

`DesignDocument = {id,name,updatedAt,revision,frames,theme,options,scenario,page}`.
`DesignSummary` is unchanged (it does not carry options; the studio loads full
documents).

### Presence semantics

- **Absent or explicit `null` → `{}`.** A client that predates this change keeps
  working unchanged.
- **PUT replaces the whole options object**, exactly like `theme` and `frames`.
  An old PUT body without `options` therefore saves successfully and resets the
  design's chrome to `{}` (library defaults). There is no server-side merge —
  the studio always saves the whole document, so that is already how the other
  fields behave. Call the caveat out if a second writer with an old build can
  target the same design.
- `options` keys are optional and independent:
  `{}` → both defaults; `{radius:8}` → design radius, default display.

### Validation (all `400 invalid_argument`)

The message names the offending field so the studio can surface it verbatim:

| Input | Message |
|---|---|
| not a JSON object (array, string, number, `true`) | `options must be a JSON object` |
| unknown key | `options.<key> is not a supported option` |
| `radius` not a number / `null` / `<0` / `>48` | `options.radius must be a number between 0 and 48` |
| `display` not a string / `null` / not `sans`\|`serif` | `options.display must be "sans" or "serif"` |
| options value over 4 KiB | `options must be at most 4 KiB` |

The 4 KiB ceiling applies to the `options` value, not the whole document (the
document keeps its existing 8 MiB bound). Bad keys are checked in sorted order,
so a document with several mistakes always reports the same one.

Error envelope, unchanged:

```json
{"error":{"code":"invalid_argument","message":"options.radius must be a number between 0 and 48"}}
```

## Schema and migration

`api/internal/designs/schema.sql`:

```sql
options_json TEXT NOT NULL DEFAULT '{}'
```

`Migrate` (idempotent, same pattern as `scenario`/`page`) runs
`ensureColumn(ctx, db, "studio_designs", "options_json", "TEXT NOT NULL DEFAULT '{}'")`
after re-applying the declarative schema. On an old database:

- the column is added in place; existing rows read `{}` through the column
  default (no data rewrite);
- a second `Migrate` is a no-op;
- the migrated row still saves and reads options back.

`Migrate` runs at boot from `api/main.go`, so **restart the scenario** after
this lands (`vrooli scenario restart react-component-library`). Until the
restart, the running server ignores `options` on the wire and the UI would save
without the column — restart before exercising the controls.

## Server internals (for reviewers)

- `internal/designs/types.go` — `Document.Options`, `CreateInput.Options`,
  `UpdateInput.Options` are `json.RawMessage`; `normalizeOptions` validates and
  returns the trimmed input (verbatim storage, no re-serialisation).
- `internal/designs/sqlite.go` — create/save write `options_json`; `get` maps
  the column through `optionsFromSQL`, which degrades a damaged/empty value to
  `{}` so chrome can never fail a read.
- `api/handlers/designs/module.go` — `documentRequest.Options` is optional and
  passed straight to the domain; endpoint descriptors mention `options` on
  create/save and the `invalid_argument` conditions.

## Client API (`ui/src/workbench/designs.ts`)

```ts
type DesignDisplay = "sans" | "serif";
interface DesignOptions { radius?: number; display?: DesignDisplay }

interface DesignDocument {
  …;
  /** Chrome owned by the design; `{}` when unset. Older fixtures may omit it. */
  options?: DesignOptions;
}

interface CreateDesignInput {
  …;
  options?: DesignOptions;
}
```

- `DesignDocument.options` is optional **in the type** so older fixtures and
  local drafts keep compiling; read it with `doc.options ?? {}` (the model
  already does).
- `createDesign` always sends `options: input.options ?? {}`.
- `putDesign` / `saveDesign` always send `options: doc.options ?? {}`.
- The localStorage fallback stores `options` too; a browser-only create
  normalises absent options to `{}`.

## Studio model (`ui/src/workbench/studio/model.ts`)

- `seedInput(template)` → `options: {radius: template.radius, display: template.display}`.
  A new design starts from the template's chrome, so later edits are design
  changes.
- `duplicateInput(doc, name)` → `options: {...(doc.options ?? {})}`.
- `docSignature(doc)` includes `options: {radius: doc.options?.radius ?? null,
  display: doc.options?.display ?? null}`. Consequences:
  - an options-only edit is a real change and is pushed onto the undo stack;
  - absent options and `{}` fingerprint identically;
  - revision/updatedAt remain excluded.
- `frameTemplate(doc, frame)` →
  `radius: doc.options?.radius ?? DEFAULT_TEMPLATE.radius`,
  `display: doc.options?.display ?? DEFAULT_TEMPLATE.display`.
  One fixture with no `options` still renders exactly as before.

## Coordinator: `useStudio.updateOptions`

One command, undoable, coalesced by `mergeKey` — the same shape as
`updateTheme`. Add the type import first:

```ts
import type { DesignOptions, … } from "../designs";
```

`StudioApi` interface (next to `updateTheme`):

```ts
updateOptions: (patch: DesignOptions, label: string, mergeKey?: string) => void;
```

Implementation (next to `updateTheme`):

```ts
const updateOptions = useCallback(
  (patch: DesignOptions, label: string, mergeKey?: string) => {
    const doc = docsRef.current.find((entry) => entry.id === activeIdRef.current);
    if (!doc) return;
    commit(
      doc.id,
      label,
      (current) => ({ ...current, options: { ...current.options, ...patch } }),
      mergeKey,
    );
  },
  [commit],
);
```

Return it from the hook (add `updateOptions` to the object returned at the
bottom, next to `updateTheme`).

Notes:

- `commit` already skips no-op signatures, so re-selecting the current value is
  a no-op rather than a revision bump.
- Pass `mergeKey` while dragging: use `"options:radius"` for the radius slider
  and `"options:display"` for a segmented control. Consecutive edits with the
  same key collapse into one undo step; releasing the slider then changing the
  typeface produces two.
- **Reset to the library default:** pass `{radius: undefined}` /
  `{display: undefined}`. `frameTemplate` falls back on `undefined`, the
  signature matches an absent key, and `JSON.stringify` drops the key on the
  wire, so the server stores the remaining object (or `{}`). Do not pass
  `null` — the server rejects it.
- **One existing call site needs a one-line follow-up:** `applyTemplate` builds
  `const input = seedInput(template)` but copies only `name`, `theme` and
  `frames` into the commit. Add `options: input.options` there too, so applying
  a template adopts its chrome instead of leaving the design's previous
  radius/display behind. That is the same bug this change fixes, on the
  template path. Do the same in `provisionalDesign` (`options: input.options`)
  for the pre-load draft, so it renders the template's chrome rather than the
  library defaults. `createFromTemplate` already passes `seedInput` straight to
  `createDesign`, so new designs carry options.

## System controls to build

`StudioSystem` currently receives `theme`, `frames`, `activeFrame`, `token`,
`coverage` and the callbacks wired in `DesignStudioPage.tsx`. The coordinator
owns that data flow; add `options={design.options ?? {}}` and one callback,
e.g. `onOptionsChange={(patch, label, mergeKey) =>
studio.updateOptions(patch, label, mergeKey)}`.

Suggested controls (translations are the coordinator's to add; locale JSON is
outside this stream):

1. **Corner radius** — slider or number input, `min=0`, `max=48`, `step=1`
   (`step="any"` if you want the decimals the server already accepts). Value is
   `options.radius ?? DEFAULT_TEMPLATE.radius`. On change:
   `updateOptions({radius: next}, "Adjust corner radius", "options:radius")`.
   Keep the bound at 48 in the control; an out-of-range value is a 400 and the
   save fails.
2. **Typeface** — segmented `Sans` / `Serif` from `options.display ??
   DEFAULT_TEMPLATE.display`:
   `updateOptions({display: next}, "Change typeface", "options:display")`.
3. Optional **Reset chrome** action: `updateOptions({radius: undefined,
   display: undefined}, "Reset chrome")` (no mergeKey — one discrete step).
4. Show the effective value, not a blank control, when the design has never set
   the option: `options.radius ?? DEFAULT_TEMPLATE.radius`. A blank slider
   would read as "0", which is a real radius.

Verify on save that the response document's `options` matches what the control
sent; the studio adopts the server document, so a rejected value surfaces as a
save error rather than silently snapping back.

## What stays library-owned

- **Template defaults** — `DEFAULT_TEMPLATE.radius` / `.display` (and every
  template's own values) remain the fallback. A design that never sets options
  renders identically to today.
- **The bridge** — `frameTemplate` is still the only place a design's chrome
  becomes a `Template`; `ui/src/workbench/templates/spec.ts` keeps emitting
  `--t-radius` / `--t-display` from that Template. Do not add a second CSS
  override path.
- **Palette** — colours stay in `DesignDocument.theme`; `options` is not a
  colour surface. Brand colours keep flowing through the palette.
- **Layer-level geometry** — `radius` props on nodes inside a frame are frame
  data, not design chrome; design radius is the frame-level default.
- **Templates themselves** — the shelf still owns template chrome; a design
  copies it at seed time and may diverge.

## Verification

```bash
cd api && go test ./internal/designs/... ./handlers/designs/...
cd api && go vet ./internal/designs/... ./handlers/designs/...
cd api && go test ./...
cd ui && pnpm exec vitest run src/workbench/designs.test.ts src/workbench/studio/model.test.ts
cd ui && pnpm exec tsc --noEmit
cd ui && pnpm exec eslint src/workbench/designs.ts src/workbench/designs.test.ts \
  src/workbench/studio/model.ts src/workbench/studio/model.test.ts
```

Covered here: create/update/get options round-trip; absent and explicit `null`
→ `{}`; unknown key, bad radius (both bounds, wrong type, `null`), bad display,
non-object and >4 KiB rejected with the field named; old PUT bodies still save;
migration adds `options_json` to an old database, old rows read `{}`, and a
second `Migrate` is a no-op. Client/model tests cover seed chrome, duplicate
copy (including a legacy fixture → `{}`), signature change on a radius-only
edit, `frameTemplate` design values plus per-key fallback, create/save payloads
including `options`, and legacy fixtures rendering with defaults.

`go test ./...` retains exactly the three known failures
(`internal/reconcile TestCatalogSelectorResolvesThroughRealImportedSource`,
`internal/gates TestLiveUtilityClassAllowlistIsExactAndShrinkOnly`,
`internal/gates TestLiveReleaseProvenanceCoversEveryMaterializedRelease`).
`tsc --noEmit` and `eslint` are clean on the four UI files.

Still coordinator-owned: `make endpoints` (descriptors changed in
`api/handlers/designs/module.go`), the scenario restart to run `Migrate`, the
`updateOptions` command, and the System controls.

## Coordinator integration (landed)

- `useStudio.updateOptions(patch, label)` commits one command; the merge key is derived from the
  keys in the patch, so two radius keystrokes coalesce but radius and display remain separate undo
  steps. `applyTemplate` adopts the template's radius/display; `provisionalDesign` carries
  `seedInput` options.
- `StudioChrome` (System pane) edits both values, clamps radius to 0–48, offers a reset to the
  library defaults, and is axe-clean. Its fallback is `DEFAULT_TEMPLATE` (16/serif), matching
  `frameTemplate` — the earlier 14/sans fallback showed a value the canvas was not rendering.
- Live: radius 24 + serif on the imported design changed `--t-radius`/`--t-display`, saved at
  revision 3, and survived reload; `GET /api/designs/{id}` returns
  `{"options":{"radius":24,"display":"serif"}}`. Evidence: `studio-chrome.png`.
