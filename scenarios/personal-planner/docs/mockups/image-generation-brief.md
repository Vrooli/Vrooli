# Personal Planner — Image Generation Brief (for ChatGPT / DALL·E)

> **How to use this file:** Paste this whole document into ChatGPT, and **attach the observatory
> images you already generated for the Today/home page** (the ones on your phone). Then say:
> *"Here's the full context and style for my app. We already have the Today image (attached).
> Please generate the images described under each Image Class below, matching the style **and the
> compositing contract in §3**."* Generate one class at a time so you can steer each result.

---

## 1. What this is

**Personal Planner** is a calm, premium personal planning app — a beautifully crafted "daily driver"
for goals, schedule, focus sessions, and weekly review. It is **not** a corporate productivity
dashboard; it should feel like a quiet, intelligent companion.

The whole app is built around **one metaphor: celestial navigation from an observatory.** You plan
your time the way a navigator reads the sky — set a course by fixed stars (goals), read your position
(review), steer daily (today). Every screen lives in this single world. The imagery reinforces that
world; it does not just decorate.

**We already generated the home/Today image** (attached): an observatory under a star field. That
sets the standard. Everything new must feel like the *same world and same hand* — just a **different
vantage point** per screen.

---

## 2. The house style (keep constant)

**Mood:** serene, spacious, cinematic. Deep-night calm. Awe without busyness. Lots of negative space.

**Palette (this is the brand — use it exactly):**
- Night sky gradient: deep navy `#15243c` (top) → near-black blue `#0b1728` (bottom/horizon).
- Signal / accent: luminous cyan `#22d3ee` (sparingly — one highlighted star, a glowing dome slit).
- Warm secondary: soft amber `#f5b971` (dawn light, lamp glow).
- Ink / stars: cool off-white `#e8eefc`.

**Rendering:** painterly-but-clean digital illustration. Smooth gradients, subtle grain, gentle
atmospheric depth. **Not** flat vector, **not** clip-art, **not** photoreal, **not** 3D-render look.
Match the softness and detail of the attached Today image.

---

## 3. ⭐ THE COMPOSITING CONTRACT (read this — it's why earlier images looked wrong)

The app renders every screen in **three layers**. Your generated image is **only the middle layer.**

```
 L2  Procedural FX + glow + UI      ← the app draws this (stars, COMETS, halos, text)
 L0  Procedural SKY (gradient)      ← the app draws this, full-screen, animated, day/night
 L1  YOUR GENERATED IMAGE           ← foreground scenery ONLY, anchored to the bottom
```

**The #1 rule: DO NOT fill the frame with your own detailed sky.** The app owns the sky so it can
animate stars, drift comets, and cross-fade day↔night. If your image contains a rich baked-in sky, it
collides with the app's sky and creates a visible seam. That was the bug.

So every image must obey ONE of two shapes:

### Shape A — EXTERIOR band (use for Today, Plan, Goals, Review)
- **Bottom ~55–60%** = the scenery / landmark on its horizon. All the detail lives here.
- **Top ~40–45%** = **plain, flat sky gradient in our palette** (`#15243c` fading down toward the
  horizon). **No stars, no clouds, no detail** up here — just smooth color. The app fades this top
  region into its own sky and paints stars/comets over it. Think of the top as a "handoff zone."
- **Aspect ratio: wide panorama, about 3:1** (e.g. 2172×724, matching the existing Today assets).
  Landmark biased to the **left third**; keep the right side open for UI text.

### Shape B — INTERIOR with an aperture (use for Focus, Settings)
- A **full-frame interior** (telescope mechanism, the view down the scope) is fine here —
- BUT include a **clear window / aperture / opening filled with flat chroma-key green (`#00e000`)** —
  the app keys the green to transparent and composites the live, animated sky (and comets) through it.
  Paint **nothing** inside the green.
- Aspect: 16:9 for desktop; keep the aperture toward the upper/left area.

**Keying rule (bands vs apertures):** for a **Shape-A bottom band**, leave the top sky *flat navy*
(the app blends it, no keying). For a **Shape-B aperture** (or any sub-region you want live sky/FX
inside an otherwise-painted scene, like the night lake sky), fill exactly that region with **flat
chroma-key green** so the app can key it out. Green = "cut a hole here"; flat navy = "blend into sky."

**In one line for ChatGPT:** *"Never paint your own star field. For outdoor scenes leave the top ~40%
flat navy; for indoor scenes / windows fill the opening with flat chroma-key green (#00e000). The app
paints the real animated sky there."*

---

## 4. Variants — how many versions?

Avoid a 24-file explosion. Recommended minimum:

**Theme (Night vs Day):** Night is primary — **generate Night first for every screen.** Once happy,
ask for a **Day variant of the same composition** (same landmark/framing, warm pre-dawn light, amber
low sun, flat top sky in a lighter navy). Same scene, different hour.

**Device (Desktop vs Mobile):**
- Exterior **Shape-A bands are naturally responsive** — the app scales the 3:1 band to screen width
  and the flat top blends into the sky, so **one band serves both** devices. (This is how Today works.)
- Interior **Shape-B scenes DO need two sizes** — a **16:9 desktop** and a **9:16 mobile portrait** —
  because the interior fills the whole background and a landscape crop won't cover a tall phone.
- A **view seen *through* an aperture** (e.g. the Plan window's landscape) needs only **one size** —
  the app crops/pans it within the opening, so viewport changes are absorbed there.

**Output:** high resolution, no text, no logos, no UI, no watermarks. Flat sky where §3 requires it.

---

## 5. Image Classes (one per screen)

Each block already bakes in the compositing contract. Paste §2 + §3 once, then feed these one at a time.

### 5.1 — TODAY (already have it ✅)
Reference/style anchor — the attached image. Observatory exterior, bottom-anchored band, flat navy
sky up top. All new images match this one.

### 5.2 — PLAN — "The star chart" *(Shape A — exterior band)*
> A wide panorama, ~3:1. In the **bottom 55%**, on a dark observatory desk/parapet biased to the left,
> a large circular **celestial star-chart / astrolabe** — faint gridlines, ecliptic curves, delicate
> cyan `#22d3ee` linework on aged dark material. The **top 45% is plain flat night sky**, smooth deep
> navy `#15243c` fading toward the horizon — **no stars, no detail** (the app adds them). Painterly-
> clean, palette-accurate, no text, no UI. Landmark left, open right.

### 5.3 — GOALS — "The constellations" *(Shape A — exterior band)*
> A wide panorama, ~3:1. In the **bottom 50%**, the low silhouette of the observatory dome on a dark
> ridge, biased left. Above the ridge, near the horizon only, a few **faint constellation shapes**
> just beginning — but keep the **upper 45% of the frame as plain flat navy sky** (`#15243c`), smooth
> and empty, so the app can paint the full star field and light the constellations itself. One warm
> amber `#f5b971` accent near the horizon. Painterly-clean, no text, no UI.

### 5.4 — FOCUS — "Through the scope" *(Shape B — interior with aperture)* — HAVE IT ✅ (`source-art/focus-scope-{desktop,mobile}.greenscreen.png`)
> An intimate interior view **looking through the observatory telescope**: the dark curved interior of
> the tube/dome framing a **circular aperture**. Fill that circle with **flat chroma-key green
> (`#00e000`)** — paint nothing inside it; the app keys it out and places the single focus-star and any
> comets there. Everything else is quiet dark interior, a whisper of cyan `#22d3ee` on the metal rim.
> Serene, minimal, high-contrast, lots of dark negative space. Painterly-clean, no text, no UI.
> **Generate both a 16:9 desktop and a 9:16 mobile portrait** (interior = full background).

### 5.5 — REVIEW — "Star trails" *(Shape A — exterior band, special case)*
> A wide panorama, ~3:1. **Bottom 45%**: the observatory dome silhouette anchored lower-left on a dark
> horizon, hint of warm amber `#f5b971` dawn at the horizon line. The **upper portion stays mostly
> flat navy** (`#15243c`) — you *may* suggest very faint concentric **star-trail arcs** as subtle
> texture, but keep them low-contrast and leave room; the app overlays the real animated trails/stars.
> When in doubt, flatter is better. Painterly-clean, contemplative, no text, no UI.

### 5.6 — SETTINGS — "The instrument" *(Shape B — interior with aperture)* — HAVE IT ✅ (`source-art/settings-instrument-{desktop,mobile}.greenscreen.png`)
> A close, warm study of the **observatory telescope mechanism** — brass/dark-metal tube, gears, focus
> dials, calibration rings — biased left, shallow depth, mostly dark with the metal catching soft amber
> `#f5b971` lamp light and one cyan `#22d3ee` dial highlight. Include a small **window/opening** (upper
> area) **filled with flat chroma-key green (`#00e000`)** — the app keys it out and paints the live sky
> through it. Painterly-clean, no text, no UI.
> **Generate both a 16:9 desktop and a 9:16 mobile portrait** (interior = full background).

---

## 6. Checklist before accepting an image
- [ ] Sky is **flat, empty, palette-colored** where §3 requires (top 40% for exteriors, inside the
      aperture for interiors) — **no baked star field**?
- [ ] Correct shape: exterior = ~3:1 bottom-anchored band; interior = 16:9 with a sky aperture?
- [ ] Palette exact (`#15243c`→`#0b1728`, cyan `#22d3ee`, amber `#f5b971`, ink `#e8eefc`)?
- [ ] Landmark biased left, right side open for text?
- [ ] No baked-in text, UI, logos, or heavy glow halos (glow is added in-app)?
- [ ] Feels like the same world and hand as the Today image?

Once the 6 Night versions are approved, come back for Day variants and any portrait interior crops.

---

## 7. Note for the app side (not for ChatGPT)
The app composites per §3 and **derives the sky from the image at runtime** — see
`docs/visual-compositing.md` for the full mechanism. In short: the app samples the panorama's top
strip for its colour and reads its aspect ratio, then builds the procedural sky gradient and the
scenery band from those, and feathers the band's top edge into the sky with a mask. This means a
regenerated or slightly-different panorama **cannot** leave a mismatched seam — you don't have to hit
an exact sky colour, only follow the Shape-A / Shape-B contract in §3. Comets/FX live on the
procedural sky layer, which spans the full viewport on every page.
