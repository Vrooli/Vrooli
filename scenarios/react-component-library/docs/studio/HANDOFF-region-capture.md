# Handoff — region capture (Assets pane)

Status: **UI complete, coordinator wiring pending.** Region capture is implemented and
tested inside the Assets pane; the page and `useStudio` still upload every asset as
`reference` until the coordinator lands the signature described below.

## Files

| File | Change |
|---|---|
| `ui/src/workbench/studio/StudioAssets.tsx` | `onImport` gains a fidelity argument; fidelity badge; **Crop region** action; dialog mount |
| `ui/src/workbench/studio/RegionCapture.tsx` | New accessible crop dialog (pointer marquee, keyboard, numeric fields, aspect presets, canvas encode) |
| `ui/src/workbench/studio/region.ts` | New pure geometry module |
| `ui/src/workbench/studio/region.test.ts` | 24 geometry tests |
| `ui/src/workbench/studio/StudioAssets.test.tsx` | 11 component tests (fidelity, upload, dialog, drag, keyboard, aspect, capture, failures) |
| `ui/src/workbench/studio/region-capture.css` | New dialog styles, all `--wb-*` tokens |

No other file was hand-edited. The Vite strings plugin refreshed `en.json` /
`strings.generated.ts` during the test run (see *Left for the coordinator*).

## UI flow

1. Every image card in the Assets pane has **Crop region**. The card still has the
   existing **Place** and **Remove** actions.
2. **Crop region** opens a modal dialog over the pane with the full image scaled to
   fit (natural aspect preserved).
3. On open the marquee is the whole image, so **Capture region** can save the full
   image without an edit. A drag on the full-image marquee starts a new rectangle
   (the full rectangle cannot be moved); after that, dragging inside the marquee
   moves it and dragging a corner handle resizes it.
4. The precise path is the numeric `X` / `Y` / `W` / `H` fields. Typing is not
   clamped mid-word; values commit as you type and normalize on blur/Enter.
5. Aspect presets — **Free**, **1:1**, **4:3**, **16:9** — constrain subsequent
   drags, resizes, keyboard resizes and width/height edits. `Free` removes the lock.
6. **Capture region** draws the rectangle through `<canvas>` with
   `drawImage(img, sx, sy, sw, sh, 0, 0, sw, sh)`, encodes `image/png`, wraps it in a
   `File` named `capture-<asset base name>.png`, and calls
   `onImport(file, "captured")`. The dialog closes only after that call resolves.
7. A null canvas context, null `toBlob` result, thrown `toBlob`, or rejected
   `onImport` keeps the dialog open with a role-alert message; the primary button
   returns to idle so the person can retry. While encoding the button reads
   **Capturing…**; while the upload ran it reads **Saving capture…**; both are
   disabled. There is no indefinite spinner.
8. An image that fails to load shows its own error and disables capture.

### `onImport` change (coordinator action)

`StudioAssets` now declares:

```ts
onImport: (file: File, fidelity: "reference" | "captured") => void;
```

The upload input passes `"reference"`; the dialog passes `"captured"`. **The
coordinator must update `DesignStudioPage` and `useStudio.importAsset` to accept the
fidelity and forward it to `uploadDesignAsset(designId, file, fidelity)`.** Until
that lands, cropping still produces the right PNG but the page records it as a
reference.

`tsc --noEmit` is currently **clean** — there is no page error to report. A one-arg
callback is assignable to the new two-arg prop, so type-checks cannot catch the
remaining wiring gap; only the behaviour does. The page handler also currently
catches upload errors internally and returns `void`; make it return the promise (or
rethrow) so the dialog can keep itself open on upload failure.

## Crop math (`region.ts`)

The marquee is stored in **natural image pixels** for its whole lifetime. Pointer
coordinates are converted once from the rendered `<img>` box; numeric fields and
keyboard steps are already natural pixels, so the canvas call uses the region
verbatim.

- `clampRegion` rounds to whole pixels, clamps `w/h` to `[min, bounds]`, then nudges
  `x/y` so the rectangle cannot overflow an edge. `MIN_REGION = 16`.
- `moveRegion` translates and re-clamps; size is preserved.
- `resizeRegion(handle = nw|ne|sw|se)` moves one corner, pins the opposite corner,
  and stops at the image edge or the minimum — a drag past the opposite edge never
  flips the rectangle.
- `applyAspect` fits the current width to `w/h = ratio`, shrinks it when the fitted
  height would overflow, enforces the minimum, and centres (or keeps top-left with
  `anchor: "nw"`).
- `regionFromDrag` normalises inverted drags and applies the minimum size without
  leaving the image.
- `regionToPixels` converts displayed-preview coordinates to natural pixels with
  independent X/Y scale factors; the studio uses the same conversion concept, and
  the tests cover a 2× and a non-uniform 4× preview plus an edge clamp.

Filename note: `hero.png` becomes `capture-hero.png` (one extension is stripped) so
the result is never `capture-hero.png.png`.

## Accessibility

- `role="dialog"`, `aria-modal="true"`, `aria-labelledby` a real `<h2>`.
- Focus enters the dialog on open (the close button is the first focusable), is
  trapped by the shared `useFocusTrap`, and returns to the **Crop region** trigger
  when the dialog unmounts. Escape closes; the backdrop is a pointer-only close
  surface, so there is no duplicate hidden focus stop.
- The image has alt text naming the asset.
- The marquee is a focusable `role="slider"` with an accessible name,
  `aria-valuemin`, `aria-valuemax`, `aria-valuenow` and `aria-valuetext` describing
  x/y/w/h. Arrow keys move 2 px; Shift+arrows resize from the south-east corner by
  2 px.
- All controls are real `<button>`/`<input>` elements with `:focus-visible` rings;
  numeric fields have visible labels; errors are `role="alert"`; loading is a
  `role="status"`.

## Validation

```bash
cd ui
pnpm exec vitest run src/workbench/studio/region.test.ts src/workbench/studio/StudioAssets.test.tsx
pnpm exec eslint src/workbench/studio/StudioAssets.tsx src/workbench/studio/RegionCapture.tsx \
  src/workbench/studio/region.ts src/workbench/studio/region.test.ts \
  src/workbench/studio/StudioAssets.test.tsx
pnpm exec tsc --noEmit
```

Results: **35/35 tests pass** (24 region + 11 component), **ESLint 0 errors**, **`tsc`
clean**. `pnpm build` and the scenario were intentionally not run, per the task.

## Left for the coordinator

1. Wire `onImport(file, fidelity)` through `DesignStudioPage` and
   `useStudio.importAsset` → `uploadDesignAsset`.
2. The string catalog is already in sync: the Vite strings plugin refreshed
   `en.json` / `strings.generated.ts` during the test run, and
   `pnpm strings:catalog:check` reports **1068 keys, up to date**. No locale JSON
   was hand-edited; re-run the sync if a later edit changes a default.
3. Optional visual pass at 1440 and 390; the dialog is capped at 58 vh of image
   and the four numeric fields collapse to two columns under 520 px.
4. Captured assets intentionally keep the **Crop region** action so they can be
   re-cropped; only the badge changes to `captured` (green via
   `[data-fidelity="captured"]`).
