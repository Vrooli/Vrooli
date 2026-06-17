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
| text_to_image / image_to_image | 🟢 D | SD 1.5 | OpenRAIL-M ✅ | ~2.1GB (~1GB q4) | [HF](https://huggingface.co/stable-diffusion-v1-5/stable-diffusion-v1-5) · [sd.cpp](https://github.com/leejet/stable-diffusion.cpp) |
| text_to_image / image_to_image | 🔴 Q | SDXL 1.0 | OpenRAIL++-M ✅ | ~6.9GB (~2GB q4) | [HF](https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0) |
| text_to_image / image_to_image | 🔴 Q+ | FLUX.1 schnell | Apache-2.0 ✅ | ~6.8GB q4 | [HF](https://huggingface.co/black-forest-labs/FLUX.1-schnell) |
| inpaint / outpaint | 🟢 D | SD 1.5 inpainting | OpenRAIL-M ✅ | ~2GB | [HF](https://huggingface.co/stable-diffusion-v1-5/stable-diffusion-inpainting) |
| inpaint / outpaint | 🔴 Q | SDXL inpainting 0.1 | OpenRAIL++-M ✅ | ~6.9GB | [HF](https://huggingface.co/diffusers/stable-diffusion-xl-1.0-inpainting-0.1) |
| object_removal | 🟢 D | MI-GAN | MIT ✅ | ~30MB | [repo](https://github.com/Picsart-AI-Research/MI-GAN) · [IOPaint](https://www.iopaint.com/models) |
| object_removal | 🟢 Q | Big-LaMa | Apache-2.0 ✅ | ~196MB | [repo](https://github.com/advimman/lama) |
| background_replace | 🔴 Q | IC-Light V1 (relight) | Apache-2.0 ✅ | ~2GB | [repo](https://github.com/lllyasviel/IC-Light) — *V2 is non-commercial* |

> background_replace **default** is non-AI: BiRefNet/u2net cutout → composite (see below).

### Enhancement & restoration
| Op | Tier | Model | License (commercial) | Approx size | Source / updates |
|---|---|---|---|---|---|
| upscale | 🟢 D | Real-ESRGAN | BSD-3 / MIT ✅ | ~64MB | [releases](https://github.com/xinntao/Real-ESRGAN/releases) |
| upscale (anime) | 🟢 V | Real-CUGAN | MIT ✅ | ~45MB | [ncnn](https://github.com/nihui/realcugan-ncnn-vulkan/releases) |
| upscale | 🔴 Q | SwinIR | Apache-2.0 ✅ | ~142MB | [repo](https://github.com/JingyunLiang/SwinIR) |
| denoise | 🟢 D | DnCNN | BSD-3 / MIT ✅ | <5MB | [KAIR](https://github.com/cszn/KAIR) |
| denoise | 🟡 Q | SCUNet | Apache-2.0 ✅ | ~70MB | [repo](https://github.com/cszn/SCUNet) |
| deblur | 🟡 D | NAFNet | MIT ✅ | ~257MB (w64) | [repo](https://github.com/megvii-research/NAFNet) |
| deblur | 🔴 Q | Restormer | MIT ✅ | ~135MB | [repo](https://github.com/swz30/Restormer) |
| colorize | 🟢 D | DDColor tiny/quant | Apache-2.0 ✅ | ~55–215MB | [repo](https://github.com/piddnad/DDColor) |
| colorize | 🔴 Q | DDColor large | Apache-2.0 ✅ | ~912MB | [repo](https://github.com/piddnad/DDColor) |
| face_restore | 🟡 D | RestoreFormer++ | Apache-2.0 ✅ | ~290MB | [repo](https://github.com/wzhouxiff/RestoreFormerPlusPlus) |
| old_photo_restore | 🟡 D | pipeline: Real-ESRGAN + DDColor + RestoreFormer++ | all ✅ | ~0.5–1GB | (composed) |
| old_photo_restore | 🔴 Q | MS Bringing-Old-Photos | MIT ✅ | ~1–2GB | [repo](https://github.com/microsoft/Bringing-Old-Photos-Back-to-Life) |

### Segmentation & depth
| Op | Tier | Model | License (commercial) | Approx size | Source / updates |
|---|---|---|---|---|---|
| background_removal | 🟢 D | IS-Net general-use | Apache-2.0 ✅ | ~44MB | [rembg](https://github.com/danielgatis/rembg) |
| background_removal | 🟢 V | U²-Net (u2netp/silueta) | Apache-2.0 ✅ | ~4.7–43MB | [rembg](https://github.com/danielgatis/rembg) |
| background_removal / replace | 🔴 Q | BiRefNet (lite ~115MB) | MIT ✅ | ~490MB fp16 | [repo](https://github.com/ZhengPeng7/BiRefNet) |
| segment | 🟢 D | MobileSAM | Apache-2.0 ✅ | ~40MB | [repo](https://github.com/ChaoningZhang/MobileSAM) · [export](https://github.com/vietanhdev/samexporter) |
| segment | 🔴 Q | SAM 2.1 large (tiny ~128MB) | Apache-2.0 ✅ | ~900MB | [repo](https://github.com/facebookresearch/sam2) |
| segment | 🟢 N | HQ-SAM (sharpest edges) | Apache-2.0 ✅ | ~40MB (Light) | [repo](https://github.com/SysCV/sam-hq) |
| depth_map | 🟢 D | Depth-Anything-V2 Small | Apache-2.0 ✅ | ~99MB | [HF](https://huggingface.co/depth-anything/Depth-Anything-V2-Small) · [ONNX](https://github.com/fabio-sim/Depth-Anything-ONNX) |
| depth_map | 🔴 Q | Depth-Anything **V1** Large | Apache-2.0 ✅ | ~1.3GB | [repo](https://github.com/LiheYoung/Depth-Anything) — *V2 large is NC* |
| normal_map | 🟢 D | normals-from-depth (computed) | n/a ✅ | 0 | (math) |
| normal_map | 🔴 Q | Marigold-Normals v1-1 | Apache-2.0 ⚠️ | ~2GB | [HF](https://huggingface.co/prs-eth/marigold-normals-v1-1) — *verify weight card at bundle* |

### Analysis & extraction
| Op | Tier | Model/lib | License (commercial) | Approx size | Source / updates |
|---|---|---|---|---|---|
| ocr | 🟢 D | Tesseract (gosseract) | Apache/MIT ✅ | ~10MB+lang | [tessdata](https://github.com/tesseract-ocr/tessdata_best) · [gosseract](https://github.com/otiai10/gosseract) |
| ocr | 🟢 Q | PaddleOCR | Apache-2.0 ✅ | ~140MB | [repo](https://github.com/PaddlePaddle/PaddleOCR) |
| nsfw_classify | 🟢 D | AdamCodd ViT-base NSFW | Apache-2.0 ✅ | ~85MB int8 | [HF](https://huggingface.co/AdamCodd/vit-base-nsfw-detector) |
| nsfw_classify | 🟢 Q | Falconsai NSFW | Apache-2.0 ✅ | ~85MB | [HF](https://huggingface.co/Falconsai/nsfw_image_detection) |
| caption | 🟢 D | moondream2 | Apache-2.0 ✅ | ~1.8GB | [HF](https://huggingface.co/vikhyatk/moondream2) |
| caption | 🟢 V | SmolVLM-256M | Apache-2.0 ✅ | ~400MB | [HF](https://huggingface.co/HuggingFaceTB/SmolVLM-256M-Instruct) |
| caption / detect / tag | 🟢 Q | Florence-2 | MIT ✅ | ~460MB | [HF](https://huggingface.co/microsoft/Florence-2-large) |
| object_detection | 🟢 D | YOLOX-Tiny | Apache-2.0 ✅ | ~20MB | [releases](https://github.com/Megvii-BaseDetection/YOLOX/releases) |
| object_detection | 🟢 V | NanoDet | Apache-2.0 ✅ | ~1–5MB | [repo](https://github.com/RangiLyu/nanodet) |
| object_detection | 🟢 Q | RT-DETRv2 (official) | Apache-2.0 ✅ | ~76MB | [repo](https://github.com/lyuwenyu/RT-DETR) |
| tagging | 🟢 D | WD14 wd-vit-v3 | Apache-2.0 ✅ | ~350MB | [HF](https://huggingface.co/SmilingWolf/wd-vit-tagger-v3) |
| tagging | 🟢 Q | RAM++ | Apache-2.0 ✅ | ~300MB | [repo](https://github.com/xinyu1205/recognize-anything) |
| face_detection | 🟢 D | YuNet | MIT ✅ | ~300KB | [opencv_zoo](https://github.com/opencv/opencv_zoo) |
| quality_assessment | 🟢 D | Laplacian + resolution (computed) | Apache/own ✅ | 0 | [gocv](https://github.com/hybridgroup/gocv) |
| quality_assessment | 🟢 Q | BRISQUE | OpenCV Apache + attribution ✅ | ~25KB | [opencv_contrib](https://github.com/opencv/opencv_contrib) |
| quality_assessment | 🟢 N | LAION aesthetic (+CLIP) | MIT ✅ | ~350MB | [repo](https://github.com/christophschuhmann/improved-aesthetic-predictor) |
| duplicate_detect | 🟢 D | goimagehash (pure Go) | BSD-2 ✅ | ~0 | [repo](https://github.com/corona10/goimagehash) |
| embedding | 🟢 D | nomic-embed-vision-v1.5 | Apache-2.0 ✅ | ~180MB | [HF](https://huggingface.co/nomic-ai/nomic-embed-vision-v1.5) |
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
