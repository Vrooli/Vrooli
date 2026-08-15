# Render matrix

Every seeded style rendered through a really running `image-tools`, at seed 7,
with no brand bound — the path a CLI caller always takes.

**Reproduce:** `make integration-evidence` from `scenarios/backdrop-studio`.

| Field | Value |
|---|---|
| API build | `95954b987998` |
| Catalog seed version | 8 applied of 8 shipped |
| Image models installed | `sd-1.5`, `instruct-pix2pix`, `sd-1.5-inpainting`, `mi-gan`, `big-lama`, `real-esrgan`, `dncnn`, `nafnet`, `ddcolor-tiny`, `restoreformer-plus-plus`, `isnet-general-use`, `u2netp`, `birefnet-general`, `mobilesam`, `depth-anything-v2-small`, `tesseract`, `adamcodd-vit-nsfw`, `moondream2`, `smolvlm-256m`, `yolox-tiny`, `wd14-vit-v3`, `yunet`, `nomic-embed-vision-v1.5`, `imported-sd15` |
| Conditioning adapters ready | `lcm-lora-sdv1-5`, `ip-adapter-sd15`, `controlnet-canny-sd15` |
| Result | **44 rendered, 0 failed, 0 skipped** of 44 |

| Style | Strategy | Result | Geometry | Detail |
|---|---|---|---|---|
| `ascii-field` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `aurora-mesh` | procedural | pass | 1440x900 | procedural → treatment |
| `blueprint-truchet` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `caption-wash-nebula` | procedural-treated | pass | 2048x2732 | procedural → treatment |
| `city-pop-horizon` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `constructivist-figure` | synthesized | pass | 1440x720 | image-tools inference → treatment |
| `cyanotype-arcade` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `deep-field` | procedural | pass | 1440x900 | procedural → treatment |
| `demoscene-terrain` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `device-stage-mesh` | procedural | pass | 2048x2732 | procedural → treatment |
| `ember-cloud` | procedural | pass | 1440x900 | procedural → treatment |
| `ember-mesh` | procedural | pass | 1440x900 | procedural → treatment |
| `engraved-colonnade` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `engraved-colonnade-vector` | vector | pass | 1440x900 | vector → image-tools rasterize → treatment |
| `feature-band-mesh` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `filament-plot` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `glaze-mosaic` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `guided-botanical` | guided | pass | 1440x900 | scaffold → image-tools inference → treatment |
| `guided-industrial` | guided | pass | 1440x900 | scaffold → image-tools inference → treatment |
| `guided-interior-riso` | guided | pass | 1440x900 | scaffold → image-tools inference → treatment |
| `iron-attractor` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `long-exposure-flow` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `memphis-weave` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `molten-terrain` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `night-contour` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `op-art-interior` | guided | pass | 1440x900 | scaffold → image-tools inference → treatment |
| `pale-moon` | vector | pass | 1440x900 | vector → image-tools rasterize → treatment |
| `relief-plate` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `riso-horizon` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `silk-drift` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `solar-bloom-horizon` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `solar-mesh` | procedural | pass | 1440x900 | procedural → treatment |
| `stipple-massif` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `store-tile-truchet` | procedural-treated | pass | 2048x2732 | procedural → treatment |
| `survey-relief` | vector | pass | 1440x900 | vector → image-tools rasterize → treatment |
| `swiss-contour` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `synth-celestial` | synthesized | pass | 1440x900 | image-tools inference → treatment |
| `technical-field` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `terrazzo-truchet` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `tidal-caustic` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `tidal-halftone` | vector | pass | 1440x900 | vector → image-tools rasterize → treatment |
| `type-mask-caustic` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `ukiyo-tide` | procedural-treated | pass | 1440x900 | procedural → treatment |
| `vaporwave-drift` | procedural-treated | pass | 1440x900 | procedural → treatment |
