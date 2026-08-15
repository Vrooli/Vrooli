# Reserved-copy legibility

Worst-pixel contrast inside each style's own declared overlay region, against
its own declared text colour and threshold, measured on the picture a really
running `image-tools` produced.

**Reproduce:** `make integration-evidence` from `scenarios/backdrop-studio`.

| Style | Worst ratio | Threshold | Verdict |
|---|---|---|---|
| `aurora-mesh` | 19.82 | 4.50 | passes |
| `caption-wash-nebula` | 19.30 | 4.50 | passes |
| `deep-field` | 19.17 | 4.50 | passes |
| `demoscene-terrain` | 16.28 | 4.50 | passes |
| `ember-mesh` | 17.10 | 4.50 | passes |
| `engraved-colonnade` | 15.30 | 4.50 | passes |
| `engraved-colonnade-vector` | 9.20 | 4.50 | passes |
| `feature-band-mesh` | 13.55 | 4.50 | passes |
| `filament-plot` | 18.61 | 4.50 | passes |
| `guided-botanical` | 1.00 | 4.50 | **fails** |
| `guided-industrial` | 1.01 | 4.50 | **fails** |
| `long-exposure-flow` | 18.59 | 4.50 | passes |
| `memphis-weave` | 8.90 | 4.50 | passes |
| `night-contour` | 17.50 | 4.50 | passes |
| `pale-moon` | 11.11 | 4.50 | passes |
| `stipple-massif` | 15.87 | 4.50 | passes |
| `store-tile-truchet` | 14.96 | 4.50 | passes |
| `survey-relief` | 9.19 | 4.50 | passes |
| `swiss-contour` | 14.70 | 4.50 | passes |
| `synth-celestial` | 1.09 | 4.50 | **fails** |
| `technical-field` | 17.00 | 4.50 | passes |
| `terrazzo-truchet` | 16.32 | 4.50 | passes |
| `tidal-halftone` | 8.57 | 4.50 | passes |

**20 of 23 pass.**

Not measured — did not render on this host: `city-pop-horizon`, `constructivist-figure`, `cyanotype-arcade`, `op-art-interior`, `riso-horizon`, `solar-bloom-horizon`.
