# Scene Integration Contract — Observatory

This document is the concrete integration recipe for the Observatory day/night
landscape. `DESIGN.md` owns the *design language* (tokens, feel, appearance
model, accessibility floors); this reference owns the *how*: the layer stack,
the sky-to-app blending technique, focal points, responsive scenery behavior,
and the asset inventory. It exists because "make it beautiful like the mockups"
is not a spec — this is.

The approved visual target is the pair of mockups in
[`mockups/`](./mockups/). They are the **acceptance oracle**: a surface is
"done" when it matches them (Day and Night) at the reference width, not when it
merely renders.

> Binding vs. tuned. The *structure* here (layer order, blending mechanism,
> responsive rules, focal-point anchoring, accessibility behavior) is binding.
> Specific color hex values are **starting points to tune against real
> screenshots**, never eyedropped from the PNGs as ground truth — see
> `DESIGN.md` §Color and the source plan §6.3.

---

## 1. Asset inventory

All source masters are preserved, durable, in-repo. Optimized, resized, and
modern-format (AVIF/WebP + fallback) exports are **derived during
implementation** from these masters; never ship the raw multi-MB PNGs.

| Asset | Path | Dimensions | Aspect | Role |
|---|---|---|---|---|
| Day — wide | `scene-masters/scene-day-wide-2x1.png` | 1774×887 | 2:1 | Desktop/tablet Day |
| Day — panorama | `scene-masters/scene-day-panorama-3x1.png` | 2172×724 | 3:1 | Wide/ultrawide Day |
| Night — wide | `scene-masters/scene-night-wide-2x1.png` | 1774×887 | 2:1 | Desktop/tablet Night |
| Night — panorama | `scene-masters/scene-night-panorama-3x1.png` | 2172×724 | 3:1 | Wide/ultrawide Night |
| Mockup — Today Day | `mockups/mockup-today-day.png` | 1585×992 | ~1.6:1 | Acceptance oracle (Day) |
| Mockup — Today Night | `mockups/mockup-today-night.png` | 1585×992 | ~1.6:1 | Acceptance oracle (Night) |

**Known gap — no portrait composition.** The provided masters are all
landscape (2:1 and 3:1). There is **no compact/portrait scene** for phones. The
mobile Today surface therefore either (a) uses a deliberately composed compact
crop of the wide master with the observatory kept in frame, or (b) commissions a
true portrait composition derived from the same master (source plan step 3;
`DESIGN.md` §Responsiveness). Simply `cover`-cropping a landscape into a portrait
viewport will amputate either the observatory or the horizon and is prohibited.
Record which path is taken in `docs/internal/DECISIONS.md`.

**Landmark constant.** In every master the observatory sits on the **left**
(~10–18% from the left edge, straddling the horizon). The observatory is the
identity of the app; no crop at any width may push it out of frame.

---

## 2. The layer stack

Compose Today as five layers, back to front. Only the last layer contains
interactive or textual content; the illustrations are scenery only.

| # | Layer | Implementation | Notes |
|---|---|---|---|
| 1 | Base atmosphere | Full-viewport CSS color/gradient (`bg-app-background` token) | Extends to any size; equals the scene's top-sky color so the seam vanishes (§3) |
| 2 | Landscape + observatory | The composite master illustration, pinned bottom | One composite per appearance for R1; independent parallax layers are a later refinement |
| 3 | Ambient details (optional) | A few restrained CSS stars/lights; respects reduced motion | Off when `prefers-reduced-motion` or reduced-scenery setting is on |
| 4 | Readability scrim | `linear-gradient` overlay, app-bg-opaque at top → transparent near the horizon | Melts the sky into the atmosphere **and** gives headline/nav a clean surface |
| 5 | Application interface | Real React: nav rail, headline, next-action card, capacity, timeline | All text, buttons, cards, calendar blocks live here — never in the image |

Rationale for keeping content in code: text must wrap, cards must grow, a11y
settings must work, and timeline blocks must keep accurate dimensions. Baked-in
text defeats all four.

---

## 3. Sky-to-app blending (the seam)

The masters include their own sky. The seam between the illustration's sky and
the app background disappears with two coordinated moves:

1. **Match the atmosphere to the top-edge sky.** Set the base-atmosphere token
   (layer 1) to the illustration's top-most sky color for that appearance, so
   where the image ends and the background continues there is no color step.
   *Starting points, tune against screenshots:*
   - Night: deep plum-navy (~`#171634`–`#1c1b3a` family; cf. the LPBS
     constellation-midnight palette).
   - Day: pale warm blue (~`#bcd2e4`–`#cfe0ec` family).
2. **Dissolve the residual with the readability scrim (layer 4).** Overlay a
   `linear-gradient` from the atmosphere color (fully opaque at the top of the
   viewport) to transparent at roughly the scene's horizon line. Any small
   mismatch between the token and the painted sky is absorbed by the fade, and
   the upper region gains a legible surface for the "Today / Monday…" headline
   and nav.

Geometry: pin the scene with `background-position: <focal-x> bottom` (or an
`<img>`/`object-position` equivalent) and size it to the **lower band** of the
viewport. The scene occupies the bottom; layers 1+4 own the top. See the
mockups: content sits over the sky; scenery fills the bottom ~40–55%.

Never regenerate artwork on load. Never let a remote image host gate the first
Today view (`DESIGN.md` §Scene Assets).

---

## 4. Focal points and cropping

- **Anchor left, keep the horizon.** Cover-crops must preserve the observatory
  (left ~10–18%) and the mountain horizon. Recommended focal point:
  `object-position: 15% 70%` (favor left, favor lower-third), tuned per width.
- **Preserve aspect ratio.** Use `aspect-ratio` / intrinsic sizing; do not
  stretch. Choose a *composition* (wide vs. panorama vs. portrait) that fits the
  viewport shape, then a *resolution* that fits its displayed pixel size —
  these are two separate decisions (§5).
- **Shared coordinate system for any layered art.** If the scene is ever split
  into independent layers (mountains/trees/buildings/clouds), all layers share
  one coordinate system and transform together. A separate `cover` crop per
  layer makes the observatory drift off its hillside — prohibited.

---

## 5. Responsive behavior

The interface and the scenery follow **separate** responsive rules. Breakpoints
below are a starting point; adjust when real content feels cramped
(`DESIGN.md` §Responsiveness owns the interface side in full).

| Width | Interface | Scenery |
|---|---|---|
| Wide desktop ≥1280px | Nav rail; next-action + capacity side by side; horizontal timeline | Broad landscape (panorama 3:1) across the lower canvas |
| Tablet / small desktop 768–1279px | Smaller nav; cards stack as needed; fewer timeline columns | Controlled crop of the wide 2:1, observatory kept visible |
| Phone <768px | Single column; prominent Start/Resume; **vertical agenda**; bottom nav | Compact composition in a shallow decorative region (see §1 portrait gap) |
| Short window / phone landscape | Prioritize current action + essential controls | Reduce or suppress scenery height |

**Composition vs. resolution:**
- Use `<picture>` with `media` to switch **composition** (panorama ↔ wide ↔
  compact) by viewport shape.
- Use `srcset` + `sizes` to select **resolution** (DPR/size) within a chosen
  composition.
- MDN: <https://developer.mozilla.org/en-US/docs/Web/HTML/Guides/Responsive_images>

**Container queries for embedded cards.** Reusable planner cards (next-action,
capacity, timeline block) may be embedded in narrow panels on a large monitor
(e.g. inside Daily / Cadence / Iris). They must adopt their compact layout based
on **container width**, not viewport width, via CSS container queries.
- MDN: <https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_containment/Container_queries>

---

## 6. Cross-cutting rules

- Keep important content in normal document flow; long titles must never collide
  with artwork.
- Give text sufficiently opaque surfaces where the scene is busy (the cream
  next-action card, translucent timeline blocks).
- Load the interface independently of decorative artwork; scenery must not block
  the first task view. Provide an **art-free / reduced-scenery** setting and a
  subdued night-surface option (`DESIGN.md` §Customization).
- **Appearance switching never moves controls or changes scroll position**; it
  is a restrained opacity/color transition only.
- Disable ambient movement and appearance transitions under
  `prefers-reduced-motion`.
  MDN: <https://developer.mozilla.org/en-US/docs/Web/CSS/@media/prefers-reduced-motion>

---

## 7. Visual acceptance (the self-check loop)

Fidelity to the mockups is the primary acceptance signal for the Today surface.
To keep image reads bounded (they are expensive):

- **Do not screenshot during iteration.** Build from tokens + this contract +
  the mockups already reviewed; reason before you render.
- **Check at slice checkpoints only.** When a surface is claimed done, capture
  **one Day + one Night** at the reference desktop width, read them once, diff
  against `mockups/mockup-today-day.png` / `mockup-today-night.png`, fix, and
  re-capture **at most once or twice** before it either passes or the residual
  gap is logged as bounded follow-up.
- **Responsive sweep is a separate, later checkpoint** — only after the desktop
  Today matches — capturing only the named breakpoints (§5), not a continuous
  range.

A surface that renders but does not match the mockup is **not done**; it is a
green that lies. The visual gate is a hard R1 release gate (`DESIGN.md`
§Scene Assets).

---

## References

- `DESIGN.md` — design language, tokens, appearance model, accessibility floors.
- `docs/concepts/EXPERIENCE.md` + `experience/` — which surfaces exist.
- `docs/concepts/UI-ARCHITECTURE.md` — component decomposition.
- `docs/reference/source-implementation-plan.md` — the original source plan
  (external input; §6.x referenced throughout DESIGN.md).
- `mockups/` — the approved visual acceptance oracle.

## 8. Delivery reference handoff and fidelity receipts (2026-10-03)

The active I18 delivery team must read the **actual full-resolution files**, not
only this inventory, a dossier thumbnail, a PDF, or a remote link. The durable
[visual reference manifest](./mockups/visual-reference-manifest.json) records
17 original PNGs (37,875,533 bytes), full SHA-256 hashes, decoded dimensions,
repository source revisions, and page/state/theme/viewport/requirement mappings.
All were read completely and decoded with `Image.load()`; originals were neither
resized nor replaced. These repository paths are available to the native team's
in-place workers. Each receiving worker must acknowledge which applicable image
files it actually opened; file presence alone does not prove a model viewed them.

### Authority and reference matrix

| Surface | Reference to open | Authority and limits |
|---|---|---|
| Today desktop, Day/Night | `mockups/mockup-today-{day,night}.png`, both 1585×992 | Original accepted D08 pair. Day hash matches the retained dossier pin `dcd6d9687b2bfb0fd84729de6933ef79a18fb9a6d72411d795463c72f3952063`. |
| Today scenery | `scene-masters/scene-{day,night}-{wide-2x1,panorama-3x1}.png` | Four original decorative masters; no UI text or controls belong in scenery. |
| Today mobile | `../mockups/redesign-desktop-and-mobile.html#today`, `../mockups/today-mobile-nextstep.html`, this contract §5 and `../../DESIGN.md` | Historical composition guidance; no approved portrait full-UI PNG or portrait scene exists in these sources. Compose faithfully for phones; do not claim an exact mobile image approval. |
| Plan / Week | `../mockups/redesign-desktop-and-mobile.html#plan`; four `../mockups/source-art/plan-starchart-*.png` plus lake plates | Historical desktop Week/mobile Day composition and source-art GOOD verdict. Desk plates are 1672×941 desktop / 941×1672 portrait; not full-UI screenshots. Week has no separately referenced original PNG. |
| Focus | HTML `#focus`; `../mockups/source-art/focus-scope-{desktop,mobile}.greenscreen.png` | Historical full-page composition guidance and GOOD plate verdict; brass scope aperture, live timer and controls. Raw green circle is an aperture, never a working UI color. |
| Settings | HTML `#settings`; `../mockups/source-art/settings-instrument-{desktop,mobile}.greenscreen.png` | Historical composition and GOOD plate verdict; brass instrument, dark negative space, live settings. |
| Goals / Review | HTML `#goals` / `#review`, `../../DESIGN.md` | Historical composition guidance; no corresponding standalone original full-UI PNG established. Review figures must come from actual eligible evidence. |
| New I18 event/Month/editor, empty/error/unknown/stale/conflict states | Accepted specification §6, `../../DESIGN.md`, nearest established surface above | Routine engineering within the accepted language; no exact pre-approved specimen for every new state. No new owner checkpoint for ordinary composition/accessibility/data fixes. |

The historical HTML is self-contained source, not a thumbnail: desktop specimen
760×520, outer phone frame 320×660 (inner screen 298×638). Those are specimen
frames, not mandatory runtime viewport sizes. Use the accepted Today 1585×992
reference and the named responsive bands for runtime evidence.

D08 accepts the Observatory direction; the repository source-art README records
historical GOOD verdicts and the operator-confirmed Plan lake interpretation.
The precise remote IDs and chat-version approval receipts for those plates are
not available in the inspected sources. Preserve this limit. Do not convert
source-plate approval into approval of every UI pixel. The expanded proposal's
**D1–D7 remain unreviewed**; they are distinct from accepted D01–D09 and from
these older repository composition references. A newer candidate, generated
replacement or changed dossier image does not supersede the original pair.
The baked-sky night lake panorama is historical/superseded; use the keyed-sky
plate identified by its README for the live Plan composition.

### Close-fidelity acceptance for affected surfaces

At an epoch surface checkpoint, retain current-runtime screenshots and compare
side by side against the applicable full-size reference: Day and Night desktop,
plus the supported narrow composition. Record exact route/state, viewport/DPR,
resolved appearance, source/build/runtime identity, synthetic data fixture and
capture receipt/path. Inspect hierarchy, spacing, typography, scene framing and
blend, landmark/aperture placement, action placement, calendar readability and
responsive composition. A successful render or a screenshot without comparison
is insufficient. Use the bounded checkpoint loop in §7; do not continuously
capture during implementation or expand this finite I18 into unrelated redesign.

For each material difference, retain a short disposition: fix required; justified
responsive/accessibility adaptation; truthful-data correction; or explicitly
unverified/open. Record why and the supporting requirement. Correct time geometry,
real dates/values, unknown evidence, readable labels, keyboard and contrast rules
take precedence over illustrative arithmetic or decorative copy. No pixel-identical
claim is required or appropriate. Justified deviations must still preserve close
visual fidelity and the established Observatory language; generic replacement
layouts do not pass. If actual captures are unavailable, leave visual acceptance
open and state the concrete limitation. Independent epoch review checks the
comparison evidence under the existing native gate, without adding a parent-chat
release gate or changing permissions, budgets, scope or provider authority.
