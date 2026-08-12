# Subject collapse: sixteen art directions, four pictures

Recorded 2026-08-12, before and after the lane-coherence work.

## The defect

The procedural lane answered every subject by picking the nearest scene it
happened to have. The mapping was a `switch` in `render.go`:

| Subject | Rendered | Depicts it? |
|---|---|---|
| `horizon` | `horizon` | yes |
| `atmospheric` | `horizon` | no |
| `aquatic` | `horizon` | **no** |
| `statuary_architecture` | `arcade` | yes |
| `interior` | `arcade` | **no** |
| `geological` | `terrain` | yes |
| `cartographic` | `terrain` | **no** |
| `non_representational` | `field` | yes |
| `textile_material` | `field` | **no** |
| `object_metaphor` | `field` | **no** |

So four generators served ten subjects, and the substitution was silent. Ukiyo
Tide, Riso Horizon, City Pop Horizon and Solar Bloom Horizon were the same
picture with different filters over it. A catalog that names sixteen art
directions drew four.

Grouped by the source they actually rendered, before:

| Source | Styles |
|---|---|
| `horizon` | city-pop-horizon, riso-horizon, solar-bloom-horizon, **ukiyo-tide** |
| `arcade` | cyanotype-arcade, engraved-colonnade, **op-art-interior** |
| `terrain` | demoscene-terrain, stipple-massif, **swiss-contour** |
| `field` | ascii-field, technical-field, **memphis-weave**, **vaporwave-drift** |

The styles in bold were rendering a subject they did not declare.

## What changed

**A generator declares what it depicts, and a style may only use one that
depicts its subject.** A subject with no generator is refused — at catalog write
time for an operator's style, and over the settled catalog after seeding — with
a message naming the subject and what the lane can draw. Refusing is the honest
answer: it says "this needs the model lane" rather than shipping a different
picture under the requested name.

**Four new non-representational generators**, so the abstract half of the
catalog has real breadth instead of one generator wearing four names:

| Preset | What it is | Why it belongs here |
|---|---|---|
| `flow` | Curl-noise particle advection | Depicts nothing; it is a record of motion. Density carries tone, so the histogram is continuous end to end. |
| `voronoi` | F2−F1 cellular partitioning | Cracked-glaze surface. Shading by the *gap* between nearest and second-nearest site keeps it continuous instead of a partition of flat plates. |
| `reaction` | Gray-Scott simulation | Coral and maze structure no noise function produces. The one generator whose pattern has a length scale of its own. |
| `caustics` | Refracted-light accumulation | Rays are actually refracted and accumulated, so caustic lines appear where they converge — thin, very bright, with real cusps. |

`caustics` claims `aquatic`, not `non_representational`: it depicts water, and
pretending otherwise would repeat the mistake this work exists to correct.

## After

| Source | Styles |
|---|---|
| `horizon` | city-pop-horizon, riso-horizon, solar-bloom-horizon |
| `arcade` | cyanotype-arcade, engraved-colonnade |
| `terrain` | demoscene-terrain, stipple-massif, swiss-contour |
| `field` | ascii-field, technical-field |
| `flow` | silk-drift, vaporwave-drift |
| `voronoi` | glaze-mosaic |
| `reaction` | memphis-weave |
| `caustics` | tidal-caustic, ukiyo-tide |
| model lane | op-art-interior, guided-botanical, constructivist-figure |

Three styles were mislabelled and had their declared subject corrected;
`op-art-interior` genuinely claims to depict an interior, which no procedural
generator draws, so it moved to the model lane rather than being relabelled to
fit the generators available. Reasons for each are recorded in
`api/internal/catalog/seed/v4.json`.

Every generator is enforced against three standards, which is why the new ones
took several rounds to land: `TestScenesSpanFullTonalRange` (a screening source
must occupy the ramp), `TestSceneNoiseIsCoherent` (neighbours agree far more
often than chance), and `TestScenesScaleWithSize` (features are a fraction of
the frame, not a pixel count). The first version of `flow` failed coherence and
the resolution test; the first `caustics` rendered as speckle; the first
`reaction` rendered blank. Each was a real fault the standard caught.

## Reproduce

```
backdrop-studio render submit --style tidal-caustic --placement full_bleed --seed 7 --surface web.hero
backdrop-studio render submit --style silk-drift    --placement full_bleed --seed 7 --surface web.hero
backdrop-studio render submit --style glaze-mosaic  --placement full_bleed --seed 7 --surface web.hero
```

The PNGs beside this file are those renders.
