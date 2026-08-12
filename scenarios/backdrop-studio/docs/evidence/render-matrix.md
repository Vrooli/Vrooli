# Render matrix

Every seeded style rendered through a really running `image-tools`, at seed 7,
with no brand bound — the path a CLI caller always takes.

**Reproduce:** `make integration-evidence` from `scenarios/backdrop-studio`.

| Field | Value |
|---|---|
| API build | `1b2924d585bd` |
| Catalog seed version | 4 applied of 4 shipped |
| Image models installed | `sd-1.5`, `instruct-pix2pix`, `sd-1.5-inpainting`, `mi-gan`, `big-lama`, `real-esrgan`, `dncnn`, `nafnet`, `ddcolor-tiny`, `restoreformer-plus-plus`, `isnet-general-use`, `u2netp`, `birefnet-general`, `mobilesam`, `depth-anything-v2-small`, `tesseract`, `adamcodd-vit-nsfw`, `moondream2`, `smolvlm-256m`, `yolox-tiny`, `wd14-vit-v3`, `yunet`, `nomic-embed-vision-v1.5`, `imported-sd15` |
| Conditioning adapters ready | `lcm-lora-sdv1-5`, `ip-adapter-sd15`, `controlnet-canny-sd15` |
| Result | **16 rendered, 0 failed, 3 skipped** of 19 |

| Style | Strategy | Result | Geometry | Detail |
|---|---|---|---|---|
| `ascii-field` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `city-pop-horizon` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `constructivist-figure` | synthesized | skip | — | SKIP(gpu-capacity): reached the model, but the host could not allocate device memory |
| `cyanotype-arcade` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `demoscene-terrain` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `engraved-colonnade` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `glaze-mosaic` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `guided-botanical` | guided | skip | — | SKIP(gpu-capacity): reached the model, but the host could not allocate device memory |
| `memphis-weave` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `op-art-interior` | guided | skip | — | SKIP(gpu-capacity): reached the model, but the host could not allocate device memory |
| `riso-horizon` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `silk-drift` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `solar-bloom-horizon` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `stipple-massif` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `swiss-contour` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `technical-field` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `tidal-caustic` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `ukiyo-tide` | procedural-treated | pass | 1440x720 | procedural → treatment |
| `vaporwave-drift` | procedural-treated | pass | 1440x720 | procedural → treatment |

Every skip names the capability it is waiting on. A skip is not a pass.
