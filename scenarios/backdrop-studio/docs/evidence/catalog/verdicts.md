# Per-style ship verdicts

## The bar

> **A style meets the bar when a designer would put it on a paying customer's
> landing page without editing it, and when the thing the style says it depicts
> is visible in the delivered picture.**

Two clauses, and the second one is the one that has been failing. A style may be
a handsome texture and still be below the bar if it names a subject the
delivered picture does not contain, because the catalog sells the subject: a row
called Cyanotype Arcade promises an arcade, and a reader who gets a field of
dots has been told something untrue by the product.

**Judged at:** the style's own delivery geometry, read as its own file at 100%.
Every style below was rendered through a really running `image-tools` at seed 7
with no brand bound, to the landing-page hero where the style permits it and to
its largest permitted surface otherwise — the same geometry the contact sheets
use, but read full size rather than as a 640×320 cell.

**Why the resolution is part of the bar.** Every quality failure this scenario
has actually shipped survives downscaling as something that looks fine. The
previous pass recorded `cyanotype-arcade` as "the reference case. Arches,
columns, statue and canopy all read through the dot density." Read at 1440×720,
that file is a cream field carrying a dot canopy and four rectangular dot
blocks. There is no arch in it, no column, and no statue. The sheet cell and the
delivered file are different pictures, and only one of them is the product.

**Reproduce:**

```bash
cd scenarios/backdrop-studio && make integration-evidence          # sheets, corpus, matrix
# and, for the full-resolution files this pass was written against:
cd api && BACKDROP_STUDIO_STYLE_RENDER_DIR=<dir> BACKDROP_STUDIO_WRITE_EVIDENCE=1 \
  GOWORK=off go test -tags integration -count=1 -timeout 40m \
  -run TestCatalogContactSheetEvidence ./integration/...
```

The renders go outside the repository on purpose: delivery-resolution PNGs of
the whole catalog are about 34 MB and D-014 keeps them out of git.

## What this pass found

Sixteen styles were re-rendered and read at full resolution on 2026-08-13.
**Eight of the sixteen are below the bar.** The previous pass recorded every one
of those eight as shipping.

The disagreement is not a matter of taste, and it is recorded rather than
reconciled away. In each case the previous verdict describes something that is
not in the delivered file — an arcade, a contour map, an engraved survey sheet —
and the delivered file is a screen with no subject under it. The previous text
is preserved in full at the bottom of this document, because it is evidence of
what a reviewer saw at sheet size and therefore evidence about the sheet.

### One cause dominates

Seven of the eight are the same defect: **a screen applied to a source that has
no picture in it.** The procedural generators draw three to five flat tonal
zones, so a halftone, stipple, engraving or line screen over them has nothing to
modulate and returns its own texture. This is `flat-source`, and it is the
plan's Ceiling 1 measured rather than argued.

The evidence that it is the source and not the screen: `demoscene-terrain`,
`stipple-massif` and `iron-attractor` run the same class of screen over sources
that *do* have structure — layered ridges, an attractor's filaments — and all
three read correctly at full size. The screens work. The pictures under seven of
them do not exist.

### The one image that reaches the reference standard is model-backed

`synth-celestial` is a gilded art-nouveau celestial chart: gold ornament on
lapis, real linework, real depth. It is the only file in this catalog that would
survive next to the reference material this scenario is calibrated against, and
it was drawn by a model from a prompt with no conditioning at all. That is the
argument for the frontier tier stated as a measurement rather than a prediction.

## The ledger

`read` marks a style re-rendered and read at full resolution in this pass.
`carried` marks a style whose 2026-08-12 verdict stands unchanged and which was
not re-read at full resolution here; those rows are re-judged in the maturation
phase, and none of them is claimed as proven by this pass.

| Style | Verdict | Cause | Basis | Note |
|---|---|---|---|---|
| `cyanotype-arcade` | **below-bar** | `flat-source` | read | A cream field with a dot canopy and four rectangular dot blocks. No arch, no column, no statue. The arcade generator draws flat cut-outs; the screen has nothing to carry. |
| `engraved-colonnade` | **below-bar** | `flat-source` | read | The arches are readable as white voids and the statue as a silhouette, so this is the best of the arcade styles — and it is still cut-out geometry under a burin, with no modelling in the wall, no gradation in the water, and no perspective. |
| `ascii-field` | **below-bar** | `flat-source` | read | Large areas of identical `@` glyphs on both flanks with a lighter centre. The v7 note says a tonal grain gives every large area a reason to break up; at full size the constant-tone blocks are still the picture. |
| `relief-plate` | **below-bar** | `parameter-defect` | read | Diagonal engraving moire across the whole frame with one light blob. No relief, no survey sheet, no readable terrain. |
| `swiss-contour` | **below-bar** | `parameter-defect` | read | A curtain of vertical lines with a light blob. The contour generator's lines are finer than the screen that is drawn over them, so the screen replaces them. |
| `ukiyo-tide` | **below-bar** | `flat-source` | read | A near-uniform halftone field. No wave, no tide, no subject at any scale. |
| `tidal-caustic` | **below-bar** | `flat-source` | read | The same near-uniform halftone field as `ukiyo-tide`. The caustic source has fine structure everywhere and large-scale composition nowhere. |
| `op-art-interior` | **below-bar** | `flat-source` | read | Recorded as `not judged` before because it would not allocate on this host; it rendered here. A coarse line screen over a dark blob: no interior, no op-art, nothing legible. |
| `synth-celestial` | meets-bar | — | read | Gold on lapis, real ornament, real linework. The reference case for what this catalog should look like, and the only file that currently reaches it. |
| `demoscene-terrain` | meets-bar | — | read | Dithered green ranges with genuine atmospheric layering, a moon, and a textured foreground. The closest thing in the set to the dithered-mountain reference. |
| `stipple-massif` | meets-bar | — | read | Real tonal gradation; the mountain silhouettes read through the stipple. Proof the screen is not the problem elsewhere. |
| `blueprint-truchet` | meets-bar | — | read | Bold navy-and-white tiling with a dither that gives the ground a paper texture. Finished. |
| `city-pop-horizon` | meets-bar | — | read | Posterised sun over a banded sea. Still the most finished-looking style in the catalog. |
| `iron-attractor` | meets-bar | — | read | Delicate stippled filaments with real density gradation; reads as a scientific plate. The v6 repair holds at full size. |
| `filament-plot` | meets-bar | — | read | A defocused attractor on near-black. The empty left third reads as deliberate negative space for copy rather than as a gap. |
| `aurora-mesh` | meets-bar | — | read | Clean blades with a real light source. Professional, though undistinguished: it is the look every SaaS hero already has. |
| `deep-field` | meets-bar | `family-resemblance` | carried | Ships on its own, but see the cluster below: it and `ember-cloud` measure 0.916 against each other. |
| `ember-cloud` | meets-bar | `family-resemblance` | carried | Same cluster as `deep-field`. |
| `riso-horizon` | meets-bar | `family-resemblance` | carried | Draws the identical horizon scene as `city-pop-horizon` and `solar-bloom-horizon` — source similarity 1.000. Only the chain separates them. |
| `solar-bloom-horizon` | meets-bar | `family-resemblance` | carried | Same source as the two rows above. |
| `ember-mesh` | meets-bar | `family-resemblance` | carried | Mesh family: 0.785 against `device-stage-mesh`, 0.712 against `aurora-mesh`. |
| `device-stage-mesh` | meets-bar | `family-resemblance` | carried | Mesh family. |
| `solar-mesh` | meets-bar | `family-resemblance` | carried | Mesh family, the most separated of the four at 0.639. |
| `technical-field` | meets-bar | — | carried | Not re-read at full resolution. Shares the metaball source with `ascii-field`, which failed this pass, so treat this row as unproven until the maturation phase re-judges it. |
| `night-contour` | meets-bar | — | carried | Not re-read. Shares the contour generator with `swiss-contour`, which failed this pass. |
| `molten-terrain` | meets-bar | — | carried | Not re-read. |
| `long-exposure-flow` | meets-bar | — | carried | Not re-read. |
| `vaporwave-drift` | meets-bar | — | carried | Not re-read. |
| `memphis-weave` | meets-bar | — | carried | Not re-read. |
| `glaze-mosaic` | meets-bar | — | carried | Not re-read. |
| `silk-drift` | meets-bar | — | carried | Not re-read. |
| `terrazzo-truchet` | meets-bar | — | carried | Not re-read. |
| `store-tile-truchet` | meets-bar | — | carried | Not re-read. |
| `type-mask-caustic` | meets-bar | — | carried | Not re-read. Shares the caustics generator with `tidal-caustic` and `ukiyo-tide`, both of which failed this pass. |
| `feature-band-mesh` | meets-bar | — | carried | Not re-read. |
| `caption-wash-nebula` | meets-bar | — | carried | Not re-read. Deliberately quiet; a caption band that competes has failed. |
| `guided-industrial` | meets-bar | — | carried | Not re-read; skipped on this host for device memory in the run before this one. |
| `guided-interior-riso` | meets-bar | — | carried | Not re-read; same skip. |
| `guided-botanical` | meets-bar | — | carried | Not re-read; same skip. |
| `constructivist-figure` | not judged | — | — | Generation has not completed on this host at its permitted geometry. Recorded, never passed. |

## Family resemblance, measured

`TestNoTwoSettledStylesRenderTheSamePicture` compares declared configuration and
reports two styles as one only when the generator, the chain and every parameter
agree exactly. It passes today while four mesh styles read as one family,
because changing a single parameter is enough to make two configurations
distinct and nowhere near enough to make two pictures distinct.

The measure that does answer the question is in
[`resemblance.md`](resemblance.md). It scores structure and chain and
deliberately ignores colour, because the operator's repair direction for a
cluster is "give each member a different source, not a different colour ramp" —
a measure that counted the ramp would report a cluster as diverged the moment
someone recoloured it.

| Cluster | Resemblance | What it means |
|---|---|---|
| `deep-field`, `ember-cloud` | 0.916 | Above the 0.90 threshold. Two nebulae that read as one picture. |
| `aurora-mesh`, `device-stage-mesh`, `ember-mesh`, `solar-mesh` | 0.64 – 0.79 | Below the threshold but visibly one family. The threshold is set where a reader stopped calling them duplicates. |
| `city-pop-horizon`, `riso-horizon`, `solar-bloom-horizon` | source 1.000 | The strongest finding in the report: all three draw the *identical* horizon scene. Only the treatment chain separates them, which is why their combined score is low and their sources are indistinguishable. |

## Repair lanes

Every `below-bar` cause maps to one lane, and this table is the input to the
maturation phase.

| Cause | Styles | Repair |
|---|---|---|
| `flat-source`, line and ink subject | `cyanotype-arcade`, `engraved-colonnade`, `ascii-field` | Re-source through a vector generator that draws the line work natively instead of screening it out of a flat raster. |
| `flat-source`, photographic subject | `op-art-interior`, `ukiyo-tide`, `tidal-caustic` | Re-source through the tier router at the frontier tier, which is what `synth-celestial` already demonstrates. |
| `parameter-defect` | `relief-plate`, `swiss-contour` | Repair at cause: the screen's ruling is coarser than the structure it is drawn over, so it replaces that structure instead of carrying it. Do not loosen a threshold to make either pass. |
| `family-resemblance` | the three clusters above | Diverge the sources. A different colour ramp is explicitly not a divergence. |

---

# The previous review, preserved

The text below is the 2026-08-12 pass, kept verbatim. It is not deleted, because
a verdict that later turns out to be wrong is evidence about how it was reached
— in this case, that eight styles read as shipping at 640×320 and do not at
delivery size.

Written 2026-08-12 against the contact sheets in this directory, at seed 7, each
style at the largest surface it permits.

**Why this file exists.** The perceptual gate passed all forty styles. Read the
sheets and seven of them are images no designer would put on a landing page.
That gap is not a defect in the gate — its job is to make *unusable* impossible
to ship, not to enforce taste — but it is the reason a written human verdict is
a step rather than a nicety. Metrics prove a treatment did not destroy its
subject. They do not prove the result is worth showing anyone.

Every `would not ship` below was repaired at cause in seed v6 or v7 rather than
dropped, because in each case the fault was traceable to a generator or a
parameter rather than to the art direction.

## Untreated procedural

| Style | Verdict | Reasoning |
|---|---|---|
| `aurora-mesh` | ship | Dark teal-and-violet blades with a real light source. The reference look for a dark hero. |
| `ember-mesh` | ship | Warm blades over cream; reads as raking light rather than as a gradient. |
| `solar-mesh` | ship | The widest smear in the set. Bold, and the only style that reads as a poster on its own. |
| `deep-field` | ship | Emission, dust and stars all present; the core follows the cloud instead of sitting on it. |
| `ember-cloud` | ship | The strongest of the two nebulae — filamentary and warm without becoming orange soup. |
| `device-stage-mesh` | ship | Quiet centre where the device goes, bright band above it. Repaired in v6: it shared the aurora ramp with `aurora-mesh` and read as the same picture at a glance. |

## Screens

| Style | Verdict | Reasoning |
|---|---|---|
| `cyanotype-arcade` | ship | The reference case. Arches, columns, statue and canopy all read through the dot density. |
| `engraved-colonnade` | ship | Legible colonnade with real burin weight. This is the style the whole plan started from. |
| `blueprint-truchet` | ship | Bold, clean, and the dither gives the ground a paper texture the flat version lacked. |
| `demoscene-terrain` | ship | Green dithered ranges on a dark sky; the closest thing in the set to the dithered-mountain reference. |
| `riso-horizon` | ship | Sun, sea, and a mis-registered ink edge. Reads as printed rather than rendered. |
| `stipple-massif` | ship | Still the strongest texture in the catalog. |
| `relief-plate` | ship | An engraved survey sheet. Handsome, and legitimately different from the terrain styles. |
| `swiss-contour` | ship | Corrected in v5 to draw real contours; it had been a mountain wearing stripes. |
| `ascii-field` | **repaired twice** | *Would not ship*, and the gate agreed — it failed outright at 600×240 with subject survival 0.560. The source was the fault: twenty-two alpha-blended gaussians have no surfaces, so the glyph mosaic had no composition to preserve. The metaball rewrite cleared the gate but left the second half of the original audit's complaint standing — a wall of identical `@` characters with a hard vertical edge. That is what a glyph mosaic does to an area of constant tone, and the metaball interiors were constant. A fine tonal grain across the field gives every large area a reason to break up. Ships. |
| `iron-attractor` | **repaired** | *Would not ship*: stipple spacing wider than the attractor's filaments, so the dots landed on nothing and the plate read as dirt on paper. v6 halves the spacing and thickens the source. |
| `tidal-caustic` | **repaired** | *Would not ship*: a fine screen over an equally fine caustic field returned noise. Repaired twice — see the note below, the second attempt was wrong in an instructive way. |
| `ukiyo-tide` | **repaired** | Same fault and same repair as `tidal-caustic`, with a posterize ahead of the screen. |

## Optical

| Style | Verdict | Reasoning |
|---|---|---|
| `filament-plot` | ship | A defocused attractor. Elegant, and the softest thing in the catalog. |
| `night-contour` | ship | The infrastructure-branding topographic look, done properly. One of the best in the set. |
| `memphis-weave` | ship | Bold pink-and-purple zigzag; unmistakably its lineage. |
| `molten-terrain` | ship | Pixel-sorted ridges. Reads more as spiked terrain than as molten rock, but it is striking and it is the only `pixel_sort` in the catalog. |
| `solar-bloom-horizon` | ship | A sunlit coast with real halation. |
| `long-exposure-flow` | **repaired** | *Would not ship*: the smear was long enough to average the flow field's warm and cool filaments into grey marble. v6 roughly halves it and flattens the curve. |
| `vaporwave-drift` | **repaired** | *Would not ship as its lineage*: silver-lilac in a tradition that promises magenta and cyan. v6 strengthens the aberration and lightens the scrim. |

## Tonal

| Style | Verdict | Reasoning |
|---|---|---|
| `city-pop-horizon` | ship | Posterised sun over a banded sea. The most finished-looking style in the catalog. |
| `glaze-mosaic` | ship | Cracked-glaze voronoi with genuine depth between cells. |
| `silk-drift` | ship | Marbled blue and white; the duotone lets the flow field's structure carry it. |
| `terrazzo-truchet` | ship | Warm stone with a polished inlay. The tonal curve is doing quiet, real work. |
| `store-tile-truchet` | ship | The same tiling composed for a store tile, with a quiet centre. |
| `type-mask-caustic` | ship | Bright cyan web on near-black — exactly what a type mask wants behind it. |
| `feature-band-mesh` | ship | Posterised mesh folds. Serviceable rather than exciting, and correct for a feature graphic. |
| `caption-wash-nebula` | ship | Deliberately quiet. It is a caption band, and a caption band that competes has failed. |
| `technical-field` | **repaired** | *Would not ship*: a white smudge on grey mist, for the same reason as `ascii-field`. Ships after the metaball rewrite. |

## Model-backed

| Style | Verdict | Reasoning |
|---|---|---|
| `guided-industrial` | ship | A real industrial hall in red duotone, with depth and structure. The lane works. |
| `guided-interior-riso` | ship | Sunlit modernist interior with a single chair. The best evidence that edge conditioning over a structured scaffold does what it claims. |
| `synth-celestial` | ship | Gilded art-nouveau celestial chart. The most beautiful image in the catalog, and it needed no conditioning at all. |
| `op-art-interior` | not judged | Generation did not complete on this host at its permitted geometry. Recorded, not passed. |
| `constructivist-figure` | not judged | Same — the VAE allocation fails at 1440×720. |
| `guided-botanical` | **repaired** | *Would not ship*: a flat green rectangle. Its scaffold was the abstract field under an `edge` conditioner, and a Canny edge map of a soft blob field is nearly empty, so ControlNet was handed no structure and produced none. v6 conditions on `depth` over the terrain scaffold. |

## The instructive failure

The first repair of `tidal-caustic` and `ukiyo-tide` coarsened their screens, on
the theory that a fine screen over a busy source produces mush. The gate then
refused both — subject survival 0.598 and 0.345 against a 0.600 floor — and
measuring the relationship explained why.

The gate reduces a composition to a 64-column field before correlating it, so at
1440px wide its cell is 22.5px. A 42-line screen has a 34px cell and a 34-line
screen an 85px one. Once the screen cell exceeds the gate's, the gate is
measuring the screen rather than the picture — and so is a viewer. Measured
across rulings on one caustic source:

| lpi | subject survival |
|---|---|
| 34 | 0.386 |
| 48 | 0.638 |
| 64 | 0.924 |
| 96 | 0.891 |

The knee sits exactly where the two cells cross. The correct repair was the
other half of the same idea: keep the screen fine and give the *source* a
stronger large-scale composition, which is what v6's caustics retune and the
generator's deepened light gradient do. `TestSeededScreensResolveFinerThanTheGateSamples`
now holds the rule so it cannot be re-broken by the same reasoning.

**Read this section against the 2026-08-13 pass above.** The measurement is
sound and the conclusion it reached was half of the repair: the screen was made
to resolve finer than the gate samples, and both styles still deliver a uniform
dot field with no subject. Giving the source a stronger large-scale composition
was the other half, and it was never done — the caustics generator still draws
fine structure everywhere and composition nowhere.
