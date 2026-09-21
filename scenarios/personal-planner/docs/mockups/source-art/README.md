# Source art — raw plates (NOT drop-in assets yet)

Generated via ChatGPT (2026-09-20) for the **Plan** page "star chart" concept, plus two landscape
panoramas. These are **source plates**, not finished scenery — they need processing (below) before
they can be wired into the UI. Final processed scenery lives in `ui/public/public/scenes/`.

## Files

| File | Dims | What it is |
|---|---|---|
| `plan-starchart-desktop-day.greenscreen.png` | 1672×941 | Plan star-chart on a desk, day-lit, **green-screen aperture** (desktop) |
| `plan-starchart-desktop-night.greenscreen.png` | 1672×941 | Same, night-lit (desktop) |
| `plan-starchart-mobile-night.greenscreen.png` | 941×1672 | Same, night-lit (portrait / mobile) |
| `plan-starchart-mobile-day.greenscreen.png` | 941×1672 | Same, day-lit (portrait / mobile) |
| `landscape-lake-day.panorama.png` | 1672×940 | Daytime lake/mountains panorama — **use as-is** (baked sky is opaque, nothing composites behind it). Add procedural **hot-air balloons** drifting across. |
| `landscape-lake-night.greenscreen-sky.png` | 1672×941 | Moonlit lake panorama with **only the upper sky green-screened** → procedural **stars + comets** show there. **This is the one to use for night.** |
| `landscape-lake-night.panorama.png` | 1672×940 | Earlier night version with a fully baked sky — superseded by the greenscreen-sky version above; kept for reference. |
| `focus-scope-desktop.greenscreen.png` | 1672×941 | **Focus** — brass telescope in a dark dome, large **circular green-screen aperture** (desktop) |
| `focus-scope-mobile.greenscreen.png` | 941×1672 | Same, portrait / mobile |
| `settings-instrument-desktop.greenscreen.png` | 1672×941 | **Settings** — brass telescope mechanism/dials, dark dome, **arched green-screen window**, warm lantern (desktop) |
| `settings-instrument-mobile.greenscreen.png` | 941×1672 | Same, portrait / mobile |

> Double-check the day/night labels by eye — assigned from generation order, not verified pixel-by-pixel.

## Verdict (see chat 2026-09-20)

- **Star-chart desk (all 4): GOOD for Plan.** Correct concept ("Plan = the command chart"), full
  day/night × desktop/mobile matrix. Shape B (interior with a sky aperture), per
  `../image-generation-brief.md` §3. Green-screen window → procedural sky behind.
- **Landscape panoramas: the VIEW THROUGH THE PLAN WINDOW** (confirmed with operator — prompted as
  "the view from the observatory window overlooking the lake"). They sit *behind* the star-chart desk
  plate's green-screen window. So Plan is a layered diorama (see the "Plan page composite" in
  `../../visual-compositing.md`). Handling per "Partial apertures & the baked-sky exception":
  - **Day** → baked panorama as-is (opaque sky). Add procedural **hot-air balloons** drifting across
    the window (day-only ambient FX).
  - **Night** → `...greenscreen-sky.png`; key the upper-sky band so procedural **stars + comets** show
    above the painted clouds/moon. Keep clouds/moon/lake baked.
  - **Sizes:** the landscape view needs **one size only** — it's seen through the window aperture, so
    any viewport change is handled by cropping/panning within the window. It's the **interior desk
    plate** that needs two sizes (desktop landscape + mobile portrait, both already generated) because
    it fills the whole background.
- **Focus scope (both): GOOD for Focus.** Shape B interior "through the scope," desktop + mobile,
  clean circular green aperture. The focus timer/ring + a single procedural focus-star (and comets)
  composite *inside* the keyed circle; procedural sky behind. Dark interior leaves room for controls.
  One mood only (dim) — day/night is expressed by what shows through the aperture, not the interior.
- **Settings instrument (both): GOOD for Settings.** Shape B interior "the instrument," desktop +
  mobile. Brass mechanism/dials are the hero (calibration = tuning your instrument); the arched green
  window is the secondary aperture → procedural sky. Ample dark negative space for the settings list.
  Note desktop biases the mechanism right (dark space left) — inverse of the brief's suggested left
  bias, but fine; the settings list just takes the dark side.

## Processing needed before these are usable

The green rectangle is a **chroma-key aperture**, not a finished sky. To honor the compositing
contract (procedural sky through the window so comets/day-night keep working):

1. **Key out the green → transparent.** Produce a transparent PNG/WebP where the green becomes alpha.
   (ChatGPT can't emit reliable transparency; do this in an image tool or a small script.)
2. **Export to `ui/public/public/scenes/plan/`** as `webp` (match the existing `day/night-panorama.webp`
   convention), e.g. `plan-desk-day.webp`, `plan-desk-night.webp`.
3. **Wire via the generalized compositing hook** — Plan is Shape B, so it does NOT use Today's
   bottom-band sampling. It needs the aperture approach: the interior plate on top, the procedural sky
   (L0) composited *behind* the transparent window. This is the "generalize compositing to all pages"
   task in `../../FEATURE_BACKLOG.md` §G — do that first, then Plan is the first consumer.
4. **Do NOT composite the baked landscape panoramas into the window** — use the procedural sky so
   comets/parallax/day-night work through the glass.

Until steps 1–3 are done, these stay here as raw source and are not referenced by the app.
