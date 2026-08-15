# Family resemblance

Each style's nearest neighbour in the settled catalog, measured on the
arrangement of its source and the treatment chain over it. Colour is
deliberately excluded: a recolour is not a divergence.

**Reproduce:** `BACKDROP_STUDIO_WRITE_EVIDENCE=1 GOWORK=off go test ./internal/catalog/ -run TestResemblanceReportEvidence` from `scenarios/backdrop-studio/api`.

Cluster threshold: **0.90**. Styles at or above it read as one picture.

## Clusters

| Members | Strongest resemblance |
|---|---|
| `deep-field`, `ember-cloud` | 0.916 |

## Nearest neighbour, every style

| Style | Nearest | Kind | Source | Chain | Resemblance |
|---|---|---|---|---|---|
| `deep-field` | `ember-cloud` | scene | 0.916 | 1.000 | 0.916 |
| `ember-cloud` | `deep-field` | scene | 0.916 | 1.000 | 0.916 |
| `device-stage-mesh` | `ember-mesh` | scene | 0.785 | 1.000 | 0.785 |
| `ember-mesh` | `device-stage-mesh` | scene | 0.785 | 1.000 | 0.785 |
| `aurora-mesh` | `ember-mesh` | scene | 0.712 | 1.000 | 0.712 |
| `solar-mesh` | `ember-mesh` | scene | 0.639 | 1.000 | 0.639 |
| `engraved-colonnade` | `relief-plate` | scene | 0.543 | 0.915 | 0.497 |
| `relief-plate` | `engraved-colonnade` | scene | 0.543 | 0.915 | 0.497 |
| `silk-drift` | `type-mask-caustic` | scene | 0.456 | 1.000 | 0.456 |
| `type-mask-caustic` | `silk-drift` | scene | 0.456 | 1.000 | 0.456 |
| `glaze-mosaic` | `technical-field` | scene | 0.325 | 0.703 | 0.229 |
| `technical-field` | `glaze-mosaic` | scene | 0.325 | 0.703 | 0.229 |
| `riso-horizon` | `solar-bloom-horizon` | scene | 1.000 | 0.218 | 0.218 |
| `solar-bloom-horizon` | `riso-horizon` | scene | 1.000 | 0.218 | 0.218 |
| `city-pop-horizon` | `riso-horizon` | scene | 1.000 | 0.203 | 0.203 |
| `cyanotype-arcade` | `silk-drift` | scene | 0.374 | 0.500 | 0.187 |
| `swiss-contour` | `city-pop-horizon` | scene | 0.553 | 0.286 | 0.158 |
| `molten-terrain` | `riso-horizon` | scene | 0.484 | 0.288 | 0.139 |
| `long-exposure-flow` | `terrazzo-truchet` | scene | 0.332 | 0.406 | 0.135 |
| `terrazzo-truchet` | `long-exposure-flow` | scene | 0.332 | 0.406 | 0.135 |
| `tidal-caustic` | `ukiyo-tide` | scene | 0.385 | 0.300 | 0.115 |
| `ukiyo-tide` | `tidal-caustic` | scene | 0.385 | 0.300 | 0.115 |
| `night-contour` | `solar-bloom-horizon` | scene | 0.425 | 0.250 | 0.106 |
| `store-tile-truchet` | `glaze-mosaic` | scene | 0.294 | 0.291 | 0.085 |
| `vaporwave-drift` | `solar-bloom-horizon` | scene | 0.248 | 0.343 | 0.085 |
| `caption-wash-nebula` | `solar-bloom-horizon` | scene | 0.379 | 0.194 | 0.074 |
| `memphis-weave` | `swiss-contour` | scene | 0.158 | 0.333 | 0.053 |
| `blueprint-truchet` | `demoscene-terrain` | scene | 0.047 | 1.000 | 0.047 |
| `demoscene-terrain` | `blueprint-truchet` | scene | 0.047 | 1.000 | 0.047 |
| `guided-botanical` | `guided-industrial` | prompt | 0.068 | 0.653 | 0.045 |
| `guided-industrial` | `guided-botanical` | prompt | 0.068 | 0.653 | 0.045 |
| `synth-celestial` | `guided-industrial` | prompt | 0.118 | 0.242 | 0.029 |
| `guided-interior-riso` | `synth-celestial` | prompt | 0.094 | 0.283 | 0.027 |
| `ascii-field` | `technical-field` | scene | 0.857 | 0.000 | 0.000 |
| `constructivist-figure` | `guided-botanical` | prompt | 0.051 | 0.000 | 0.000 |
| `feature-band-mesh` | `device-stage-mesh` | scene | 0.835 | 0.000 | 0.000 |
| `filament-plot` | `caption-wash-nebula` | scene | 0.690 | 0.000 | 0.000 |
| `iron-attractor` | `ascii-field` | scene | 0.000 | 0.000 | 0.000 |
| `op-art-interior` | `guided-botanical` | prompt | 0.100 | 0.000 | 0.000 |
| `stipple-massif` | `demoscene-terrain` | scene | 1.000 | 0.000 | 0.000 |
