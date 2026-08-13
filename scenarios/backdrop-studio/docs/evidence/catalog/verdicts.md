# Per-style ship verdicts

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

Regenerate the sheets with `make integration-evidence`.

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
