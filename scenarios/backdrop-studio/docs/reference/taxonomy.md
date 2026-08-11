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

- **copy-safe zone** — the region reserved for type
- **contrast budget** — the legibility headroom the image must leave
- **focal weight** — where the eye settles; for `ambient` it should be *off* the copy
- **semantic distance** — how literally the image depicts the product

---

## Axis 2 · `treatment` — how the surface is rendered

A style may declare several; they apply in the order given. This is the axis with
the most vocabulary and the one the product exists to make usable.

### Reprographic — simulated print process

| Value | Definition |
|---|---|
| `halftone` | Amplitude-modulated dot screen: dot size varies on a fixed rotated grid. Parameters are line frequency, screen angle, dot shape. |
| `line_screen` | The same modulation expressed as lines rather than dots. Two overlaid at different angles produce deliberate moiré. |
| `risograph` | Spot inks with deliberate plate misregistration, paper tooth, and multiply overprint. The offset *is* the effect — perfect registration kills it. |

### Quantization — reduced tonal or colour depth

| Value | Definition |
|---|---|
| `dither_ordered` | Threshold against a fixed matrix (Bayer). Reads as machine and deliberate; the crosshatch is regular. |
| `dither_diffusion` | Quantization error propagated to neighbouring pixels (Floyd–Steinberg). Organic and grainy where ordered dithering is gridded. |
| `posterize` | Luminance collapsed to N levels. Hard tonal edges read as screenprint rather than photograph. |

### Ink mapping — palette substitution

| Value | Definition |
|---|---|
| `duotone` | Source colour discarded; luminance remapped across two inks. The cheapest route to brand cohesion available. |
| `tritone` | As duotone with a third ink, typically banded to a narrow luminance range as a spot highlight. |

### Photographic — simulated optics and film

| Value | Definition |
|---|---|
| `grain` | Stochastic noise plus a contrast lift. Keeps colour; the lightest-touch treatment. |
| `bloom` | Highlight bleed and halation. |
| `aberration` | Radial channel separation. Optical failure staged on purpose — it signals lens rather than render. |
| `long_exposure` | Motion smear, including intentional camera movement. |

### Synthetic — generated fields rather than filtered images

| Value | Definition |
|---|---|
| `mesh_gradient` | Soft overlapping colour fields. Maximum ambience, no subject. |
| `caustics` | Refracted-light patterning. |
| `flow_field` | Vector-field-driven line or particle structure. |

### Symbolic — the image rebuilt from discrete marks

| Value | Definition |
|---|---|
| `typographic_mosaic` | Luminance mapped to glyph density; the image reconstructed out of language. |
| `pixel` | Explicit low-resolution bitmap quantization. |

---

## Axis 3 · `subject` — what is depicted

| Value | Definition |
|---|---|
| `non_representational` | No subject. Pure field, geometry, or gradient. |
| `horizon` | Landscape with a dominant eye line. |
| `statuary_architecture` | Classical figures, columns, arcades, built structure. |
| `botanical` | Plant and organic forms. |
| `industrial` | Machinery, logistics, infrastructure. |
| `atmospheric` | Sky, cloud, water, weather. |
| `object_metaphor` | A physical object standing in for an abstract product idea. |

`non_representational` subjects are the ones most readily served by the
`procedural` strategies, which is why the free tier is not a token gesture — the
zero-cost lanes cover a genuinely useful part of the space.

---

## Axis 4 · `lineage` — the visual tradition being quoted

| Value | Definition |
|---|---|
| `cyanotype` | Blueprint and sun-print register: single blue ink on warm stock. |
| `metaphysical` | Impossible light, arcades, statuary, long shadow. |
| `city_pop` | Flat saturated colour planes, hard horizon, a single circular sun. |
| `swiss_international` | Grid discipline, geometric abstraction, restraint. |
| `op_art` | Optical interference and perceptual instability. |
| `demoscene` | 1-bit and low-palette computer graphics. |
| `riso_zine` | Small-press spot-colour printing. |
| `technical_minimalism` | Instrument and industrial-design restraint. |
| `solarpunk` | Optimistic ecological futurism. |
| `neo_brutalist` | Raw structure, high contrast, deliberate coarseness. |

**Authoring rule.** A lineage value names a *tradition*, never a living artist.
Prompt templates must describe formal properties — "flat saturated colour planes,
hard horizon, single circular sun, no visible brushwork" — rather than naming a
person. This avoids the obvious legal and ethical problem, and it also generates
more reliably, because formal description constrains the model where a name
merely gestures.

---

## Axis 5 · `placement` — how the image meets the page

A style declares every placement it is fit for. The legibility gate measures each
one independently (`LEG-004`).

| Value | Definition | Legibility note |
|---|---|---|
| `full_bleed` | Fills the section; copy overlaid. | Highest impact; the only placement that strictly requires a scrim. |
| `split_panel` | Copy one side, image the other, hard seam. | Safest — no contrast risk, so the image can be as loud as it likes. |
| `framed_inset` | Image contained with margin, copy outside it. | Reads editorial rather than promotional. |
| `corner_bleed` | Image occupies one corner and fades out. | Lets a very loud treatment coexist with a very quiet page. |
| `type_mask` | Image clipped to the headline itself. | One per page at most; loses legibility quickly at small sizes. |

**Supporting devices**, applied rather than declared: `scrim` (gradient overlay),
`vignette`, and blur-under-text. These are amendments the gate can propose
(`LEG-005`), not placements in their own right.
