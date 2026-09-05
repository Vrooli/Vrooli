# Taxonomy — the five axes

The catalog validates every style against these enums (`CAT-001`). This file is
the canonical definition; the proto enum is generated from it.

The axes are **independent**. A style is a point in a five-dimensional space, not
a member of a single named genre. That independence is what makes the catalog
searchable, promptable, diffable, and — most usefully — combinable into
compositions nobody has drawn yet.

---

## Axis 1 · `role` — what job the image does

| Value | Definition |
|---|---|
| `ambient` | The image is the stage. It carries mood, brand and craft signal while attention passes *through* it to the copy. Non-literal, low-competition, designed to be overlaid. |
| `focal` | The image is the message — a product shot, a screenshot, a portrait. Attention lands on it. Must be literal. |
| `evidential` | The image is proof — a chart, a real capture, a metric. Truth claims attach to it. |

Backdrop Studio produces `ambient` imagery. The other two values exist so the
axis is honest about what it excludes, and so a consumer can state what it
needs. Producing `focal` or `evidential` imagery is out of scope (see PRD
non-goals).

**Supporting properties**, held on the style rather than as axis values:

- **reserved region** — an area foreground content claims, declaring whether it overlays (text, gated on contrast) or occludes (a device frame, gated on focal placement)
- **contrast budget** — the legibility headroom the image must leave
- **focal weight** — where the eye settles; for `ambient` it should be *off* the copy
- **semantic distance** — how literally the image depicts the product

---

## Axis 2 · `treatment` — how the surface is rendered

A style may declare several; they apply in the order given. This is the axis with
the most vocabulary and the one the product exists to make usable.

> **This list is open, not closed.** It is the space worth covering first, not a
> boundary. Adding a value is a row here plus either an `image-tools` operation
> or a `scaffold` generator — see the mapping at the end of this file. An
> implementing agent should build broadly across these families rather than
> deeply into one; breadth is what makes the catalog read as a system.

Two kinds of treatment, and the difference decides where the code lives (D-003):

- **Filters** transform an input image. They belong in `image-tools` as generic,
  reusable operations.
- **Generators** synthesise a field from parameters and a seed. They are content,
  so they live in this scenario's `scaffold` domain.

### Reprographic — simulated print process · *filters*

| Value | Definition |
|---|---|
| `halftone` | Amplitude-modulated dot screen: dot size varies on a fixed rotated grid. Parameters are line frequency, screen angle, dot shape. |
| `line_screen` | The same modulation expressed as lines rather than dots. Two overlaid at different angles produce deliberate moiré. |
| `risograph` | Spot inks with deliberate plate misregistration, paper tooth, and multiply overprint. The offset *is* the effect — perfect registration kills it. |
| `stipple` | Density of discrete dots at irregular positions rather than on a grid. Reads as hand-drawn or as scientific illustration where halftone reads as mechanical. |
| `engraving` | Tonal value carried by line weight and spacing — hatching, cross-hatching, contour-following burin lines. The banknote and old-encyclopaedia register. |
| `letterpress` | Impression, ink squeeze at edges, slight plate offset. Subtle; works as a finishing pass over another treatment. |

### Quantization — reduced tonal or colour depth · *filters*

| Value | Definition |
|---|---|
| `dither_ordered` | Threshold against a fixed matrix (Bayer). Reads as machine and deliberate; the crosshatch is regular. |
| `dither_diffusion` | Quantization error propagated to neighbouring pixels (Floyd–Steinberg). Organic and grainy where ordered dithering is gridded. |
| `posterize` | Luminance collapsed to N levels. Hard tonal edges read as screenprint rather than photograph. |

### Ink mapping — palette substitution · *filters*

| Value | Definition |
|---|---|
| `duotone` | Source colour discarded; luminance remapped across two inks. The cheapest route to brand cohesion available. |
| `tritone` | As duotone with a third ink, typically banded to a narrow luminance range as a spot highlight. |
| `thermal` | Luminance mapped onto a false-colour ramp — infrared, elevation, or instrument palettes. Reads as data even when the source is not. |

### Optical and device — simulated lenses, film and displays · *filters*

| Value | Definition |
|---|---|
| `grain` | Stochastic noise plus a contrast lift. Keeps colour; the lightest-touch treatment. |
| `bloom` | Highlight bleed and halation. |
| `aberration` | Radial channel separation. Optical failure staged on purpose — it signals lens rather than render. |
| `long_exposure` | Motion smear, including intentional camera movement. |
| `bokeh` | Defocused highlights rendered as aperture-shaped discs. |
| `godray` | Volumetric shafts from a occluded light source. |
| `solarization` | Partial tonal inversion above a threshold — the Sabattier darkroom effect. |
| `cross_process` | Deliberately mismatched film chemistry: crushed blacks, shifted casts per channel. |
| `crt_scanline` | Horizontal raster lines, phosphor bleed, slight barrel curvature. The display artifact, not the print one. |
| `anaglyph` | Offset channel pairs read as stereoscopic depth without glasses. |

### Synthetic fields — continuous generated colour · *generators*

| Value | Definition |
|---|---|
| `mesh_gradient` | Soft overlapping colour fields. Maximum ambience, no subject. |
| `caustics` | Refracted-light patterning, as through water or glass. |
| `noise_field` | Layered coherent noise — Perlin, simplex, fBm — as a continuous tonal field. |
| `metaball` | Merging implicit surfaces; organic blobs that fuse as they approach. |

### Procedural structure — algorithmic pattern · *generators*

| Value | Definition |
|---|---|
| `flow_field` | Vector-field-driven line or particle structure. Coherent noise steering direction. |
| `voronoi` | Cellular partitioning around seed points. Reads as crystalline, geological, or organic depending on point distribution. |
| `reaction_diffusion` | Turing patterns — spots, stripes, and labyrinths emerging from two competing rates. |
| `cellular_automata` | Rule-driven grid evolution. Rule 30, Life, and their relatives. |
| `wave_function_collapse` | Constraint-solved tiling from a small example. Produces non-repeating structure that still reads as authored. |
| `truchet` | Tile sets whose rotations join into continuous curves or mazes across a grid. |
| `l_system` | Recursive rewriting into branching structure — plants, lightning, coral. |
| `strange_attractor` | Trajectory plots of chaotic systems. Dense, filamentary, deterministic from a seed. |
| `contour` | Topographic isolines over a height field. The technical-cartographic register, and a mainstay of infrastructure branding. |

### Displacement and distortion — the source rearranged · *filters*

| Value | Definition |
|---|---|
| `pixel_sort` | Runs of pixels reordered by luminance, hue or saturation within a threshold band. Streaked, molten, and unmistakably digital. |
| `glitch` | Compression and transmission failure staged deliberately — channel misalignment, block displacement, datamosh smearing. |
| `displacement` | One image's luminance used to offset another's pixels. |
| `fluted_glass` | Periodic vertical refraction, as through ribbed glass. Currently ubiquitous in interface design. |
| `kaleidoscope` | Mirrored radial symmetry from a source wedge. |
| `slit_scan` | One column or row sampled over time and assembled into a frame. Temporal smear as spatial structure. |

### Symbolic — the image rebuilt from discrete marks · *filters*

| Value | Definition |
|---|---|
| `typographic_mosaic` | Luminance mapped to glyph density; the image reconstructed out of language. |
| `pixel` | Explicit low-resolution bitmap quantization. |
| `photomosaic` | The image assembled from many smaller images. |

---

## Axis 3 · `subject` — what is depicted

Open list. The `Drawn by` column is the load-bearing part: a procedural style
may only name a subject some generator actually depicts, and the catalog refuses
one that does not at write time. Everything marked `model lane` needs a
model-backed strategy — the procedural lane will not substitute a nearby scene
for it.

That rule is new, and it is the correction to a real defect. The lane used to
answer every subject with the nearest scene it happened to have: `aquatic`,
`atmospheric` and `horizon` all rendered the horizon, `interior` rendered the
arcade, `cartographic` rendered the terrain, and `textile_material` and
`object_metaphor` rendered the abstract field. Sixteen named art directions drew
four pictures and nothing anywhere said so.

`non_representational` is where the procedural lane is genuinely strong — seven
distinct generators, selectable by name through a style's `scaffold.preset` —
which is why the free tier is not a token gesture.

| Value | Definition | Drawn by |
|---|---|---|
| `non_representational` | No subject. Pure field, geometry, or gradient. | procedural — `field`, `flow`, `voronoi`, `reaction`, `mesh`, `truchet`, `attractor` |
| `horizon` | Landscape with a dominant eye line. | procedural — `horizon` |
| `statuary_architecture` | Classical figures, columns, arcades, built structure. | procedural — `arcade` |
| `interior` | Enclosed architectural space — halls, stairwells, vaulting. | model lane |
| `botanical` | Plant and organic forms. | model lane |
| `industrial` | Machinery, logistics, infrastructure. | model lane |
| `atmospheric` | Sky, cloud, water, weather. | procedural — `nebula` |
| `celestial` | Astronomical bodies, orbits, deep field. | model lane |
| `aquatic` | Submerged and surface water, marine forms. | procedural — `caustics` |
| `geological` | Strata, minerals, erosion, crystal. | procedural — `terrain` |
| `textile_material` | Weave, fibre, paper, metal, and surface close-up. | model lane |
| `cartographic` | Maps, charts, survey and navigational imagery. | procedural — `contour` |
| `figure` | The human form, abstracted or partial. Use with care — a recognisable person is identity-bearing and belongs to `asset-studio`. | model lane |
| `object_metaphor` | A physical object standing in for an abstract product idea. | model lane |

`atmospheric` and `cartographic` moved off the model lane when the `nebula` and
`contour` generators landed. They are not consolation substitutes: a nebula's
emission field, dust occlusion and star magnitudes are each modelled, and a
contour map's lines are genuine level sets that crowd where the ground is steep.
The distinction that matters is whether the generator depicts the subject or
merely resembles it, and these do.

---

## Axis 4 · `lineage` — the visual tradition being quoted

Open list. A lineage narrows a prompt and a palette at once, which is why it is
worth naming rather than leaving to the subject and treatment alone.

| Value | Definition |
|---|---|
| `cyanotype` | Blueprint and sun-print register: single blue ink on warm stock. |
| `metaphysical` | Impossible light, arcades, statuary, long shadow. |
| `city_pop` | Flat saturated colour planes, hard horizon, a single circular sun. |
| `swiss_international` | Grid discipline, geometric abstraction, restraint. |
| `bauhaus` | Primary colour, elemental geometry, workshop rationalism. |
| `constructivist` | Diagonal dynamism, red and black, photomontage and heavy type. |
| `art_deco` | Symmetry, stepped forms, metallic gradation, streamline. |
| `art_nouveau` | Organic whiplash line, botanical ornament, flat decorative panels. |
| `ukiyo_e` | Flat colour areas, keyblock outline, stylised wave and weather. |
| `mid_century_modern` | Muted saturation, textured flats, cheerful geometry. |
| `wpa_poster` | Bold flat shapes, limited palette, civic monumentality. |
| `scientific_plate` | Engraved or lithographed specimen illustration with taxonomic restraint. |
| `op_art` | Optical interference and perceptual instability. |
| `psychedelic` | Vibrating complements, liquid lettering, radial symmetry. |
| `memphis` | Arbitrary geometry, clashing pastels, squiggle and terrazzo. |
| `demoscene` | 1-bit and low-palette computer graphics. |
| `vaporwave` | Recuperated corporate imagery, gradient chrome, statuary and grid. |
| `cyberpunk` | Dense signage, wet neon, atmospheric depth. |
| `frutiger_aero` | Glossy skeuomorphic optimism — water, glass, bubbles, sky. |
| `riso_zine` | Small-press spot-colour printing. |
| `technical_minimalism` | Instrument and industrial-design restraint. |
| `solarpunk` | Optimistic ecological futurism. |
| `neo_brutalist` | Raw structure, high contrast, deliberate coarseness. |
| `wabi_sabi` | Asymmetry, negative space, weathered material, deliberate incompleteness. |

**Authoring rule.** A lineage value names a *tradition*, never a living artist.
Prompt templates must describe formal properties — "flat saturated colour planes,
hard horizon, single circular sun, no visible brushwork" — rather than naming a
person. This avoids the obvious legal and ethical problem, and it also generates
more reliably, because formal description constrains the model where a name
merely gestures.

---

## Axis 5 · `placement` — how the image meets the foreground

A style declares every placement it is fit for. A surface declares which
placements it permits (`SUR-002`), so the usable set is the intersection. The
legibility gate measures each one independently (`LEG-004`).

### Page placements

| Value | Definition | Legibility note |
|---|---|---|
| `full_bleed` | Fills the section; copy overlaid. | Highest impact; the only placement that strictly requires a scrim. |
| `split_panel` | Copy one side, image the other, hard seam. | Safest — no contrast risk, so the image can be as loud as it likes. |
| `framed_inset` | Image contained with margin, copy outside it. | Reads editorial rather than promotional. |
| `corner_bleed` | Image occupies one corner and fades out. | Lets a very loud treatment coexist with a very quiet page. |
| `type_mask` | Image clipped to the headline itself. | One per page at most; loses legibility quickly at small sizes. |

### Store-listing placements

These arrange a supplied application screenshot over a backdrop (`CMP-008`). The
device frame is registered as an **occlusion** region, not an overlay one — no
contrast is measured beneath it, but focal detail placed there is wasted.

| Value | Definition | Legibility note |
|---|---|---|
| `device_center` | Device frame alone, centred over the backdrop. | No overlay region; only the occlusion footprint matters. |
| `caption_above_device` | Caption band at the top, device below. | The caption band is an overlay region and is gated. |
| `caption_below_device` | Device at the top, caption band beneath. | Same, inverted. Common on Play listings. |
| `caption_only` | Type over the backdrop, no device. | Behaves exactly like `full_bleed`; gated identically. |
| `feature_graphic` | Wide banner, no device, title-safe centre. | Stores overlay their own furniture and may crop; keep the centre quiet. |

**Supporting devices**, applied rather than declared: `scrim` (gradient overlay),
`vignette`, and blur-under-text. These are amendments the gate can propose
(`LEG-005`), not placements in their own right.

---

## Axis values are not operation names

The `treatment` axis and the execution layer are **two namespaces**, and they do
not have to agree. An axis value is vocabulary an operator filters by; an
operation name is a verb `image-tools` executes, or a generator this scenario's
`scaffold` domain runs. One axis value may compile to several operations, and the
names differ where each namespace has its own convention.

**Where each treatment is implemented** decides who owns it (D-003):

| Implementation | Owner | Which treatments |
|---|---|---|
| `image-tools` operation | upstream, generic | every *filter* — anything that transforms an input image |
| `scaffold` generator | this scenario | every *generator* — anything that synthesises a field from a seed |

### Filters → `image-tools` operations

| `treatment` axis value | Operation(s) |
|---|---|
| `halftone` · `line_screen` · `posterize` · `duotone` · `grain` · `aberration` · `stipple` · `engraving` · `displacement` | same name |
| `dither_ordered` · `dither_diffusion` | same name |
| `tritone` | `duotone` (three-stop ramp) |
| `thermal` | `duotone` (multi-stop false-colour ramp) |
| `risograph` | `posterize` + `duotone` ×2 + `grain`, composed with offset |
| `letterpress` | `displacement` (edge) + `grain`, low amplitude |
| `bloom` · `godray` | `bloom` |
| `solarization` · `cross_process` | `curve` (tone-curve op) |
| `bokeh` | `defocus` |
| `crt_scanline` | `line_screen` + `aberration` + `bloom` |
| `anaglyph` | `aberration` (channel offset, no radial falloff) |
| `long_exposure` | `motion_blur` |
| `pixel_sort` · `glitch` · `kaleidoscope` · `slit_scan` · `fluted_glass` | same name |
| `typographic_mosaic` | `ascii_mosaic` |
| `pixel` | `posterize` + `resample` (nearest) |
| `photomosaic` | `photomosaic` |

### Treatment parameters are relative, not pixels

A spatial parameter expressed in pixels ties a style to exactly one output size.
The catalog renders the same style at short edges from 390px (`web.hero-mobile`)
to 2732px (`app-store-12.9-screenshot`) — a 7× span — so a pixel value that
looks right on one surface is wrong on every other. Proof, with images:
[`docs/evidence/treatments/resolution-proof/`](../evidence/treatments/resolution-proof/).

**The rule.** Every spatial parameter a seeded style sends uses the `_rel` form,
which is *a fraction of the delivered image's short edge*:

```
px = round(rel × min(width, height))
```

clamped up to the operation's minimum. `image-tools` still accepts the pixel
form and always will — a caller that means pixels keeps meaning pixels — but no
seeded style may use it, and `TestSeededStylesSendNoAbsoluteSpatialParameter`
fails the build if one does. The resolved pixel value comes back on
`OpResult.resolved_params`, so what actually ran is always visible.

| Operation | Relative parameter | Seed default | Resolves to at `web.hero` |
|---|---|---|---|
| `line_screen` | `spacing_rel` | 0.0125 | ~9px pitch |
| `stipple` | `spacing_rel` | 0.0097 | ~7px cell |
| `engraving` | `spacing_rel` | 0.0125 | ~9px period |
| `ascii_mosaic` | `block_size_rel` | 0.02 | 14px = 2 glyph advances |
| `displacement` | `spacing_rel` · `amplitude_rel` | 0.072 · 0.007 | ~52px wave, ~5px throw |
| `aberration` | `distance_rel` | 0.0056 | ~4px separation |
| `bloom` | `radius_rel` | 0.008 | ~6px lift |
| `defocus` | `radius_rel` | 0.0056 | ~4px blur |
| `motion_blur` | `distance_rel` | 0.014 | ~10px smear |

**Two parameters are exceptions, for stated reasons.**

`halftone`'s `lpi` has no relative form because it never needed one: it is a
count of screen lines across the image *width*, so the same value gives the same
coarseness at any size. Measured drift between a 768px and a 2304px frame is
below 0.5% at every ruling — the plan's finding B3 listed it as absolute and was
wrong. What it *does* have is a floor: a screen cell drawn from fewer than 8
pixels cannot carry tone, and at `lpi=130` on a 768px frame (5.9px cell) the
rendered density falls 29.6% short of the identical ruling on a larger frame.
`image-tools` clamps any ruling finer than the grid supports and reports the
clamp in `resolved_params`, so a style tuned for a hero degrades legibly on a
phone instead of rendering mud.

`ascii_mosaic`'s cell snaps to a whole multiple of the 7px bitmap glyph advance
(nearest, floor of one advance), because a cell that is not a whole multiple
resamples the glyph and smears the characters the operation exists to draw. The
advance is a coarse quantum at small cells, so this treatment holds density
across resolutions less tightly than the screens do — hence its wider test
tolerance.

**Tuning rule.** Retunes are a seed version, not an edit. Each new
`api/internal/catalog/seed/vN.json` carries a `retune_reasons` entry naming the
value chosen and why; an operator-authored style is never overwritten by one.

### Generators → `scenes` presets

These synthesise rather than filter, so they are content and live in this
scenario. Each is deterministic from its seed (`SCAF-001`).

**Built.** The `scenes` package draws these today, one file per generator:

| Preset | Depicts | What it is |
|---|---|---|
| `horizon` | `horizon` | Layered landscape with a sky model and a sun |
| `arcade` | `statuary_architecture` | Colonnade with sea, statue and canopy |
| `terrain` | `geological` | Ridged-noise elevation with atmospheric depth |
| `field` | `non_representational` | Soft overlapping colour masses (`noise_field`, `metaball`) |
| `flow` | `non_representational` | Curl-noise particle advection (`flow_field`) |
| `voronoi` | `non_representational` | F2−F1 cellular partitioning (`voronoi`) |
| `reaction` | `non_representational` | Gray-Scott simulation (`reaction_diffusion`) |
| `caustics` | `aquatic` | Refracted-light accumulation (`caustics`) |
| `mesh` | `non_representational` | Inverse-distance colour stops, smeared along a turning axis (`mesh_gradient`) |
| `truchet` | `non_representational` | Two-rotation arc tiling at two scales, lit from a fixed direction (`truchet`) |
| `attractor` | `non_representational` | De Jong trajectory density plot (`strange_attractor`) |
| `contour` | `cartographic` | Screen-space-width isolines over a stretched height field (`contour`) |
| `nebula` | `atmospheric` | Domain-warped emission, multiplied dust lanes, magnitude-distributed stars |

A procedural style selects one through `scaffold.preset`, and the generator must
depict the style's declared subject — the catalog refuses the pair otherwise.

Three of these carry a `palette` parameter selecting one of two or three
art-directed ramps. It is a float because scene parameters are a
`map[string]float64` on the wire; the alternative — one preset per palette —
would make three generators out of one and overstate how much of the space
ships.

**Not built.** `cellular_automata`, `wave_function_collapse` and `l_system` are
vocabulary with no generator behind them. They are listed in Axis 2 because the
axis is an open description of the space worth covering, not a claim about what
ships. A style cannot name one: there is no preset to select.

**Every generator must carry tonal depth.** A dither or halftone over a flat
colour region does nothing — the treatment has no gradient to modulate — so a
generator that fills areas with flat colour cannot carry the treatments this
catalog is built on. `TestScenesSpanFullTonalRange` enforces it: p2 luma below
0.18, p98 above 0.80, and at least half the histogram buckets occupied.
`TestScenesScaleWithSize` and `TestSceneNoiseIsCoherent` enforce the other two
standards — features are a fraction of the frame, not a pixel count, and
neighbouring pixels agree far more often than chance.

### Why this table exists

Three rows carry it. `typographic_mosaic` and `ascii_mosaic` are the same effect
under two conventions. `risograph` is one axis value that compiles to four
operations. And `crt_scanline` is a *display* artifact built from three
operations that individually simulate *print* and *lens*.

A style's `treatment` axis is therefore **classification, not a call list** — the
call list is the treatment chain, and `compose` is what maps between them.

Surface geometry is a separate registry again; see [`surfaces.md`](surfaces.md).
The seed set and its coverage rule are in
[`starter-catalog.md`](starter-catalog.md).
