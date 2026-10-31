# Visual Compositing — how the observatory sky & scenery are handled

How Personal Planner renders its celestial-navigation world, and the rule every page follows so the
generated art, the procedural sky, and effects (stars, comets, day/night) always fit together.

## The three-layer model

Every observatory surface is composed of three layers. The **generated image is only the middle one.**

| Layer | Owner | Contents |
|---|---|---|
| **L2 · FX + UI** | code | glow/halos, animated stars, **comets**, page content & text |
| **L0 · Sky** | code (procedural) | full-viewport gradient + star field; animates; cross-fades day↔night |
| **L1 · Scenery** | generated art | the observatory / telescope / chart — *foreground only*, edge-anchored |

**Rule: the sky is always procedural; generated art is always foreground.** Art never contains its own
usable sky. This is what lets comets, parallax, and day/night work identically on every page,
independent of the artwork in front — and it's why art must leave the sky flat/empty (see the image
brief `mockups/image-generation-brief.md` §3: Shape A exterior band, Shape B interior aperture).

## Deriving the sky from the image (no hard-coded colours)

The seam where L1 meets L0 must be invisible. Rather than hard-code a sky colour to match a specific
asset (which breaks the moment the image changes), the app **derives the sky from the image at
runtime**:

1. **Sample.** On load, `sampleSceneAsset()` (in `ui/src/theme/observatoryAppearance.ts`) draws the
   panorama to an offscreen canvas and averages its **top ~6% strip** — the flat "handoff" sky the
   image hands to us — yielding a seam colour. It also reads the image's intrinsic **aspect ratio**.
   `useSampledSceneColors()` does this for the day and night assets together.
2. **Publish.** `DashboardPage` writes the results as CSS custom properties on the observatory root:
   `--scene-sky-seam-day`, `--scene-sky-seam-night`, `--scene-band-aspect`.
3. **Build the sky from them** (`ui/src/styles.css`):
   - The scenery band height is derived: `--scene-band-block: calc(100vw / var(--scene-band-aspect, 3))`
     — so the band matches the image's real aspect (falls back to a 3:1 panorama).
   - The procedural gradient reaches the **sampled seam colour exactly at the band's top edge**:
     `linear-gradient(180deg, <palette-top> 0%, var(--scene-sky-seam-*, <fallback>) calc(100vh - var(--scene-band-block)), …)`.
   - The band feathers into that sky with a mask:
     `mask-image: linear-gradient(to top, #000 58%, transparent 100%)`.

Because the seam colour is sampled *from the image itself* and the gradient is pinned to reach it at
the exact join line, the two sides match by construction. The mask feather then softens any residual
texture. **A regenerated or slightly-off panorama cannot produce a visible seam** — the app adapts to
the image instead of the image having to hit an exact target.

### Fallbacks & edge cases
- Sampling failure (canvas taint, decode error): the `var(--scene-sky-seam-*, <palette>)` fallback
  and `var(--scene-band-aspect, 3)` keep a sensible palette sky. Assets are same-origin, so tainting
  shouldn't occur.
- **Art-free** preference: `.observatory.art-free .observatory-sky` restores the plain palette
  gradient (no derived band), since there is no image to hand off to.
- **Reduced scenery / subdued night**: unaffected — they only change layer opacity.

## Partial apertures & the baked-sky exception

The "leave the sky flat/empty" rule is really about **only the region where live content must show
through.** You do not have to flatten a whole sky:

- **Baked sky is fine when nothing moves behind it.** The daytime lake panorama keeps its painted
  blue sky and clouds as-is — it is fully opaque and static, so there is no procedural layer behind it
  to reveal. Don't waste effort keying a sky you'll never composite through.
- **Key only the FX region.** The night lake panorama keeps its painted clouds, moon, mountains and
  lake, and green-screens **only the upper sky above the clouds** — that keyed band becomes the window
  onto the procedural sky (stars) and moving FX (comets). Keep the pretty baked detail; open a hole
  only where live content belongs.

So an asset can be: fully baked (day landscape), a bottom band with procedural sky above (Today
observatory), or a mostly-baked scene with a **partial keyed aperture** (night landscape, Plan window).
Pick the least amount of keying that still lets the needed FX show.

Processing a keyed plate: chroma-key the green to alpha, export `webp`, and composite the procedural
sky/FX behind the transparent region. Source plates live in `mockups/source-art/`.

## Where comets & ambient FX go

Moving elements belong on the **procedural layers** (`.observatory-sky-stars` and siblings), confined
to the keyed sky region, never baked into art:

- **Comets / shooting stars / twinkle** — night sky FX, drift across the keyed band occasionally.
- **Hot-air balloons** — a *daytime* ambient touch: an occasional balloon drifts slowly across the
  sky region. Same pattern as comets — a sparse, slow, procedural sprite so the scene feels alive
  without being busy. Gate by appearance (balloons = day, comets = night).
- **Parallax** — subtle layer offset on scroll/tilt, applied to the procedural sky.

Add these as procedural layers so they animate and respond to day/night for free. Keep them **sparse
and slow** — the mood is calm; one balloon every so often, not a fleet.

## Worked example — the Plan page diorama

Plan is the richest composite: a **nested, parallax-capable diorama** where you look across the desk,
through a window, at the lake, up into the sky. Front-to-back:

```
 (front) L1a  Star-chart desk interior plate  — green-screen WINDOW keyed → transparent
         L1b  Lake landscape (the view through the window)
                • day:   baked opaque sky, used as-is
                • night: upper-sky band keyed → transparent
         L2   Ambient FX in the sky region:  day → hot-air balloons · night → comets
 (back)  L0   Procedural sky (gradient + stars), shows through the night keyed band
```

Keying happens twice, at two depths: the desk's window opens onto the landscape; the landscape's
night sky opens onto the procedural stars. Day short-circuits the back layers (the landscape sky is
opaque), so day just needs the desk window keyed with balloons drifting in the opening. Assets:
`mockups/source-art/plan-starchart-*.greenscreen.png` (desk) + `landscape-lake-*.png` (view).

**Asset sizing:** only the **interior desk plate** needs two sizes (desktop landscape + mobile
portrait), because it fills the whole background. The **landscape view needs just one size** — it's
seen through the window, so viewport changes are absorbed by cropping/panning it within the aperture.

## Extending this to the other pages

Today is the reference implementation. The other surfaces (Plan, Goals, Focus, Review, Settings) reuse
the same L0 procedural sky + sampled-seam mechanism; only their L1 art differs (Shape A band for
exteriors; Shape B interior-with-aperture for Focus/Settings — the aperture reveals the same procedural
sky). Factor the sampling + sky layers into a shared component/hook when generalizing, rather than
copying the Today CSS. Tracked in `FEATURE_BACKLOG.md` §G.
