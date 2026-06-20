# Model Registry Reference

The canonical, machine-readable registry seed is
[`api/internal/models/registry.seed.json`](../../api/internal/models/registry.seed.json)
(schema + load/checksum policy: [`api/internal/models/README.md`](../../api/internal/models/README.md)).
This page is the human catalog: what each operation ships, where to download it,
where to watch for updates, and which models are deliberately **blocked**.

Authored 2026-06-16 from license-verified research. Policy choices are recorded in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Design rules baked into the seed

- **Headless + ComfyUI-free.** Every op has ≥1 standalone backend; `requires_comfyui` is always `false`. ComfyUI is an optional plug-in only.
- **CPU-capable default per op**, plus one higher-quality (often GPU) tier.
- **Commercial-clean licenses only** — gate-free (no Stability $1M-gated SD3.5/Turbo), no AGPL, no non-commercial weights. Traps are listed in the blocklist below.
- **Optional Python sidecar** unlocks PyTorch-only quality tiers (RestoreFormer++, SCUNet, Restormer, DDColor-large, RAM++, Marigold, Florence-2) without ever being required.
- **Sizes/VRAM are approximate**; **checksums are captured on first download** (never hand-written).
- **Enabled weight-backed models must be installable.** `image-tools models doctor`
  checks that enabled seed entries have direct `source.assets[]` records with
  artifact `kind` and positive `min_bytes`. Phase 1 has migrated the enabled
  seed catalog off documentation-page stubs; new enabled weight-backed entries
  must add direct assets before they ship.
- **Weightless catalog entries are explicit.** `builtin`, `computed`, and
  `library-go` entries have no model weights and are treated as always
  installed. `library-cgo` remains a provisioned backend until its host/library
  assets are made explicit by the backend doctor work.

## Adding or changing a model

Use this checklist for every seed-catalog change. The catalog is a runtime
contract, not just documentation: if a model is enabled, operators must be able
to install it and understand which backend makes it runnable.

1. Edit [`api/internal/models/registry.seed.json`](../../api/internal/models/registry.seed.json)
   by hand. Do not mass-edit the seed.
2. Choose the operation and tier deliberately:
   - exactly one enabled default should serve each operation;
   - quality tiers may be enabled only when their license, backend, and host
     requirements are truthful;
   - disabled quality tiers still need a clear reason in `io.notes` or
     `capability_labels.known_risks`.
3. For weight-backed models, add direct `source.assets[]` entries:
   - use a direct artifact URL, not a model-card or HTML page;
   - set `filename`, `kind`, and positive `min_bytes`;
   - include a published `sha256` only when the upstream publishes one. Otherwise
     leave `checksum.status` as capture-on-download so install records pin the
     verified bytes.
4. For weightless models, use only the backends the registry treats as explicit
   no-weight providers (`builtin`, `computed`, `library-go`) unless backend
   doctor has a registered runtime provider for the family.
5. Confirm the backend row exists in [`backends.md`](backends.md). If the backend
   needs host software, route provisioning through Scenario Dependency Analyzer;
   never install with raw `pip`, `go get`, `npm`, or an OS package manager.
6. Run the fast gates:

```bash
cd scenarios/image-tools/api && go test ./internal/models ./internal/ai ./internal/backends -count=1
cd scenarios/image-tools && make coverage-model-lifecycle
image-tools models doctor --json
image-tools backends doctor --json
```

`models doctor` must be green before the model ships enabled. `backends doctor`
may stay red for missing host provisioning, but it must identify a registered
provider and concrete provisioning path rather than a catalog-only gap.

## Backend engines (what the Go API shells out to)

| Engine | Used for | Notes |
|---|---|---|
| `stable-diffusion.cpp` | t2i / i2i / inpaint / outpaint | CPU-capable diffusion; supports SD1.x/2.x, SDXL, SD3.x, Flux. Not PixArt/Cascade/Kandinsky. |
| `IOPaint` | object removal (LaMa / MI-GAN) | Batteries-included standalone server. |
| `rembg` + `onnxruntime` | background removal | MIT lib; ONNX is the cross-platform path. |
| `realesrgan-ncnn-vulkan` (+ONNX) | upscale / denoise | Vulkan-first binary; ship ONNX for CPU. |
| `onnxruntime-go` (CGo) | colorize, NSFW, detection, tagging, embeddings, SAM, depth | Shared ONNX backbone. |
| `llama.cpp` (CGo) | captioning (moondream2 / SmolVLM) | GGUF vision-language. |
| `gocv` / OpenCV (CGo) | face detection (YuNet), blur/quality (Laplacian, BRISQUE) | Shared CV backbone. |
| pure Go | duplicate detect (goimagehash), QR/barcode (gozxing), resolution math | Zero CGo. |
| `python-sidecar` (optional) | RestoreFormer++, SCUNet, Restormer, DDColor-large, RAM++, Marigold, Florence-2 | Opt-in; never required. |

## Registry by operation

Legend: **D**=default · **Q**=quality tier · **V**=size/CPU variant · **N**=nice-to-have · 🟢 CPU-capable · 🔴 GPU-tier

### Generation & editing
| Op | Tier | Model | License (commercial) | Approx size | Source / updates |
|---|---|---|---|---|---|
| text_to_image / image_to_image | 🟢 D | SD 1.5 | OpenRAIL-M ✅ | ~4.0GB safetensors (~1GB q4) | [direct safetensors](https://huggingface.co/stable-diffusion-v1-5/stable-diffusion-v1-5/resolve/main/v1-5-pruned-emaonly.safetensors) · [sd.cpp](https://github.com/leejet/stable-diffusion.cpp) |
| text_to_image / image_to_image | 🔴 Q | SDXL 1.0 | OpenRAIL++-M ✅ | ~6.9GB (~2GB q4) | [HF](https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0) |
| text_to_image / image_to_image | 🔴 Q+ | FLUX.1 schnell | Apache-2.0 ✅ | ~6.8GB q4 | [HF](https://huggingface.co/black-forest-labs/FLUX.1-schnell) |
| edit_instruct | 🟢 D | InstructPix2Pix | OpenRAIL-M ✅ | ~7.7GB safetensors | [direct safetensors](https://huggingface.co/timbrooks/instruct-pix2pix/resolve/main/instruct-pix2pix-00-22000.safetensors) |
| inpaint / outpaint | 🟢 D | SD 1.5 inpainting | OpenRAIL-M ✅ | ~4.2GB ckpt | [direct checkpoint](https://huggingface.co/stable-diffusion-v1-5/stable-diffusion-inpainting/resolve/main/sd-v1-5-inpainting.ckpt) |
| inpaint / outpaint | 🔴 Q | SDXL inpainting 0.1 | OpenRAIL++-M ✅ | ~6.9GB | [HF](https://huggingface.co/diffusers/stable-diffusion-xl-1.0-inpainting-0.1) |
| object_removal | 🟢 D | MI-GAN | MIT ✅ | ~27MB | [IOPaint traced weight](https://github.com/Sanster/models/releases/download/migan/migan_traced.pt) · [repo](https://github.com/Picsart-AI-Research/MI-GAN) |
| object_removal | 🟢 Q | Big-LaMa | Apache-2.0 ✅ | ~198MB | [direct ONNX](https://huggingface.co/Carve/LaMa-ONNX/resolve/main/lama_fp32.onnx) · [repo](https://github.com/advimman/lama) |
| background_replace | 🔴 Q | IC-Light V1 (relight) | Apache-2.0 ✅ | ~2GB | [repo](https://github.com/lllyasviel/IC-Light) — *V2 is non-commercial* |

> background_replace **default** is non-AI: BiRefNet/u2net cutout → composite (see below).

### Enhancement & restoration
| Op | Tier | Model | License (commercial) | Approx size | Source / updates |
|---|---|---|---|---|---|
| upscale | 🟢 D | Real-ESRGAN | BSD-3 / MIT ✅ | ~10MB default pair | [x4](https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesr-general-x4v3.pth) · [wdn](https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesr-general-wdn-x4v3.pth) |
| upscale (anime) | 🟢 V | Real-CUGAN | MIT ✅ | ~45MB | [ncnn](https://github.com/nihui/realcugan-ncnn-vulkan/releases) |
| upscale | 🔴 Q | SwinIR | Apache-2.0 ✅ | ~142MB | [repo](https://github.com/JingyunLiang/SwinIR) |
| denoise | 🟢 D | DnCNN | BSD-3 / MIT ✅ | <5MB | [KAIR direct weight](https://github.com/cszn/KAIR/releases/download/v1.0/dncnn_color_blind.pth) |
| denoise | 🟡 Q | SCUNet | Apache-2.0 ✅ | ~70MB | [repo](https://github.com/cszn/SCUNet) |
| deblur | 🟢 D | NAFNet | MIT ✅ | ~92MB ONNX | [OpenCV ONNX export](https://huggingface.co/opencv/deblurring_nafnet/resolve/main/deblurring_nafnet_2025may.onnx) · [repo](https://github.com/megvii-research/NAFNet) |
| deblur | 🔴 Q | Restormer | MIT ✅ | ~135MB | [repo](https://github.com/swz30/Restormer) |
| colorize | 🟢 D | DDColor tiny | Apache-2.0 ✅ | ~220MB | [direct weight](https://huggingface.co/piddnad/DDColor-models/resolve/main/ddcolor_paper_tiny.pth) · [repo](https://github.com/piddnad/DDColor) |
| colorize | 🔴 Q | DDColor large | Apache-2.0 ✅ | ~912MB | [repo](https://github.com/piddnad/DDColor) |
| face_restore | 🟡 D | RestoreFormer++ | Apache-2.0 ✅ | ~294MB | [direct checkpoint](https://github.com/wzhouxiff/RestoreFormerPlusPlus/releases/download/v1.0.0/RestoreFormer%2B%2B.ckpt) |
| old_photo_restore | 🟡 D | pipeline: Real-ESRGAN + DDColor + RestoreFormer++ | all ✅ | ~0.5–1GB | (composed) |
| old_photo_restore | 🔴 Q | MS Bringing-Old-Photos | MIT ✅ | ~1–2GB | [repo](https://github.com/microsoft/Bringing-Old-Photos-Back-to-Life) |

### Segmentation & depth
| Op | Tier | Model | License (commercial) | Approx size | Source / updates |
|---|---|---|---|---|---|
| background_removal | 🟢 D | IS-Net general-use | Apache-2.0 ✅ | ~44MB | [rembg](https://github.com/danielgatis/rembg) |
| background_removal | 🟢 V | U²-Net (u2netp/silueta) | Apache-2.0 ✅ | ~4.7–43MB | [rembg](https://github.com/danielgatis/rembg) |
| background_removal / replace | 🔴 Q | BiRefNet (tiny ONNX) | MIT ✅ | ~224MB | [direct ONNX](https://github.com/ZhengPeng7/BiRefNet/releases/download/v1/BiRefNet-general-bb_swin_v1_tiny-epoch_232.onnx) |
| segment | 🟢 D | MobileSAM | Apache-2.0 ✅ | ~45MB ONNX pair | [encoder](https://huggingface.co/PulpCut/mobilesam-onnx/resolve/main/mobilesam.encoder.onnx) · [decoder](https://huggingface.co/PulpCut/mobilesam-onnx/resolve/main/mobilesam.decoder.onnx) · [repo](https://github.com/ChaoningZhang/MobileSAM) |
| segment | 🔴 Q | SAM 2.1 large (tiny ~128MB) | Apache-2.0 ✅ | ~900MB | [repo](https://github.com/facebookresearch/sam2) |
| segment | 🟢 N | HQ-SAM (sharpest edges) | Apache-2.0 ✅ | ~40MB (Light) | [repo](https://github.com/SysCV/sam-hq) |
| depth_map | 🟢 D | Depth-Anything-V2 Small | Apache-2.0 ✅ | ~99MB | [direct ONNX](https://github.com/fabio-sim/Depth-Anything-ONNX/releases/download/v2.0.0/depth_anything_v2_vits.onnx) · [HF](https://huggingface.co/depth-anything/Depth-Anything-V2-Small) |
| depth_map | 🔴 Q | Depth-Anything **V1** Large | Apache-2.0 ✅ | ~1.3GB | [repo](https://github.com/LiheYoung/Depth-Anything) — *V2 large is NC* |
| normal_map | 🟢 D | normals-from-depth (computed) | n/a ✅ | 0 | (math) |
| normal_map | 🔴 Q | Marigold-Normals v1-1 | Apache-2.0 ⚠️ | ~2GB | [HF](https://huggingface.co/prs-eth/marigold-normals-v1-1) — *verify weight card at bundle* |

### Analysis & extraction
| Op | Tier | Model/lib | License (commercial) | Approx size | Source / updates |
|---|---|---|---|---|---|
| ocr | 🟢 D | Tesseract (gosseract) | Apache/MIT ✅ | ~4MB English data | [eng.traineddata](https://raw.githubusercontent.com/tesseract-ocr/tessdata_fast/main/eng.traineddata) · [gosseract](https://github.com/otiai10/gosseract) |
| ocr | 🟢 Q | PaddleOCR | Apache-2.0 ✅ | ~140MB | [repo](https://github.com/PaddlePaddle/PaddleOCR) |
| nsfw_classify | 🟢 D | AdamCodd ViT-base NSFW | Apache-2.0 ✅ | ~85MB int8 | [direct ONNX](https://huggingface.co/AdamCodd/vit-base-nsfw-detector/resolve/main/onnx/model_int8.onnx) |
| nsfw_classify | 🟢 Q | Falconsai NSFW | Apache-2.0 ✅ | ~85MB | [HF](https://huggingface.co/Falconsai/nsfw_image_detection) |
| caption | 🟢 D | moondream2 | Apache-2.0 ✅ | ~3.7GB GGUF pair | [text](https://huggingface.co/ggml-org/moondream2-20250414-GGUF/resolve/main/moondream2-text-model-f16_ct-vicuna.gguf) · [mmproj](https://huggingface.co/ggml-org/moondream2-20250414-GGUF/resolve/main/moondream2-mmproj-f16-20250414.gguf) |
| caption | 🟢 V | SmolVLM-256M | Apache-2.0 ✅ | ~279MB GGUF pair | [model](https://huggingface.co/ggml-org/SmolVLM-256M-Instruct-GGUF/resolve/main/SmolVLM-256M-Instruct-Q8_0.gguf) · [mmproj](https://huggingface.co/ggml-org/SmolVLM-256M-Instruct-GGUF/resolve/main/mmproj-SmolVLM-256M-Instruct-Q8_0.gguf) |
| caption / detect / tag | 🟢 Q | Florence-2 | MIT ✅ | ~460MB | [HF](https://huggingface.co/microsoft/Florence-2-large) |
| object_detection | 🟢 D | YOLOX-Tiny | Apache-2.0 ✅ | ~20MB | [direct ONNX](https://github.com/Megvii-BaseDetection/YOLOX/releases/download/0.1.1rc0/yolox_tiny.onnx) |
| object_detection | 🟢 V | NanoDet | Apache-2.0 ✅ | ~1–5MB | [repo](https://github.com/RangiLyu/nanodet) |
| object_detection | 🟢 Q | RT-DETRv2 (official) | Apache-2.0 ✅ | ~76MB | [repo](https://github.com/lyuwenyu/RT-DETR) |
| tagging | 🟢 D | WD14 wd-vit-v3 | Apache-2.0 ✅ | ~379MB | [direct ONNX](https://huggingface.co/SmilingWolf/wd-vit-tagger-v3/resolve/main/model.onnx) |
| tagging | 🟢 Q | RAM++ | Apache-2.0 ✅ | ~300MB | [repo](https://github.com/xinyu1205/recognize-anything) |
| face_detection | 🟢 D | YuNet | MIT ✅ | ~230KB | [direct ONNX](https://media.githubusercontent.com/media/opencv/opencv_zoo/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx) |
| quality_assessment | 🟢 D | Laplacian + resolution (computed) | own ✅ | 0 | in-process pure Go |
| quality_assessment | 🟢 Q | BRISQUE | OpenCV Apache + attribution ✅ | ~25KB | [opencv_contrib](https://github.com/opencv/opencv_contrib) |
| quality_assessment | 🟢 N | LAION aesthetic (+CLIP) | MIT ✅ | ~350MB | [repo](https://github.com/christophschuhmann/improved-aesthetic-predictor) |
| duplicate_detect | 🟢 D | goimagehash (pure Go) | BSD-2 ✅ | ~0 | [repo](https://github.com/corona10/goimagehash) |
| embedding | 🟢 D | nomic-embed-vision-v1.5 | Apache-2.0 ✅ | ~97MB int8 | [direct ONNX](https://huggingface.co/nomic-ai/nomic-embed-vision-v1.5/resolve/main/onnx/model_int8.onnx) |
| embedding | 🟢 Q | SigLIP2 base | Apache-2.0 ✅ | ~375MB | [HF](https://huggingface.co/google/siglip2-base-patch16-224) |
| qr_barcode_read | 🟢 D | gozxing (pure Go) | MIT ✅ | ~0 | [repo](https://github.com/makiuchi-d/gozxing) |

## ⛔ Blocklist — do NOT ship (verified license traps)

The registry carries these as `blocklist` entries so they can't be added by accident.

| Model | Op | License | Why blocked |
|---|---|---|---|
| **CodeFormer** | face_restore | S-Lab 1.0 | Non-commercial; the #1 face restorer online → use RestoreFormer++ |
| **GFPGAN** | face_restore | "Apache" but embeds StyleGAN2 (NVIDIA-NC) + DFDNet (CC-BY-NC-SA); ncnn port GPL | Excluded by policy |
| **bria RMBG-1.4/2.0** | bg_removal | CC-BY-NC-4.0 | Bundled in rembg zoo — block explicitly |
| **FastSAM** | segment | AGPL-3.0 (via Ultralytics) | Copyleft incl. SaaS |
| **Ultralytics YOLOv8/v11** | detection | AGPL-3.0 | ONNX export does not remove AGPL |
| **YOLO-NAS** | detection | weights Deci-NC | Weights non-commercial |
| **InsightFace SCRFD/RetinaFace** | face_detect | weights NC | Paid license required → use YuNet |
| **Surya OCR** | ocr | weights OpenRAIL revenue-capped (<$5M) | Cap hidden in README |
| **4x-UltraSharp & community ESRGAN** | upscale | CC-BY-NC-SA | Often mislabeled "mit" |
| **SUPIR** | upscale | custom NC | + GPU-only (12–30GB) |
| **APISR** | upscale | GPL + academic weights | Copyleft + academic |
| **MAT** | object_removal | CC-BY-NC-4.0 | IOPaint "recommends" it but it's NC |
| **FLUX.1 dev / Fill** | t2i / inpaint | non-commercial | Use schnell (Apache) |
| **Stable Cascade** | t2i | non-commercial | Also unsupported by sd.cpp |
| **IC-Light V2** | bg_replace | non-commercial | Use V1 (Apache) |
| **Depth-Anything-V2 Base/Large/Giant** | depth | CC-BY-NC-4.0 | Use V2-Small or V1-Large |
| **DSINE** | normal_map | custom NC | Use depth-derived / Marigold |
| **SD 3.5 / Turbo** | t2i | Stability $1M gate | Excluded by gate-free policy |
| **Qwen2.5-VL-3B** | caption | Qwen Research NC | 7B is Apache; never bundle 3B |
| **LLaVA** | caption | Llama/Vicuna NC weights | Use moondream2 |
| **pyiqa MUSIQ / CLIP-IQA** | quality | PolyForm/NTU NC | Use BRISQUE |
| **libpHash** | dup_detect | GPL-3.0 | Use goimagehash |
| **tuotoo/qrcode** | qr_barcode | GPL-3.0 | Use gozxing |

## Keeping this current

- **Per-model updates:** follow each entry's `update_source` (HF model card or GitHub releases).
- **Re-verify licenses periodically** — several here changed over time (NAFNet/Restormer dropped "non-commercial"; bria/Depth-Anything-V2 added NC tiers; prs-eth migrated Marigold cards). Re-fetch the canonical LICENSE before bundling a new weight.
- **Always check the *weights* license separately from the *code* license**, and remember ONNX/GGUF export never strips AGPL or a non-commercial weight license.
