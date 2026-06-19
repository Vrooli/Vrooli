# AI Backends & Provisioning

image-tools runs AI operations through a provider ladder: **local GPU → local
CPU → BYOK cloud → refuse**. This page documents what each backend needs on the
host and how weights are provisioned. Deterministic ops (resize, crop, format,
metadata, …) and `analyze probe` are pure-Go and need **none** of this.

## How an AI op becomes runnable

Two independent things must be present, and the selector reports them
distinctly:

1. **A backend program** — the engine that runs the model. The selector probes
   "is the program on PATH?" (`Provider.Available`).
2. **The model weights** — installed on disk under
   `<data-dir>/models/<model-id>/`. `models install <id>` fetches + validates
   them; until then the op refuses with an actionable hint (HTTP 409).

A backend present but weights missing → `models install` hint. Weights present
but backend program missing → "no available provider" / enable BYOK. Both must
line up. The tier label (`local-gpu` / `local-cpu` / `byok-cloud`) reflects where
the op *actually* ran — a CPU-only backend reports `local-cpu` even on a GPU host.

## Backends

| Backend | Program | Ops | GPU-capable | Provisioning |
|---|---|---|---|---|
| `onnxruntime` (sidecar) | `python3` + onnxruntime | `background_removal`, `deblur`, `segment`, detection/tagging/embeddings | no (CPU) | Python + `onnxruntime`, `pillow`, `numpy` (see below) |
| `rembg` | `rembg` | `background_removal` (alt) | no (CPU) | `pip install rembg` (optional alt to the sidecar) |
| `stable-diffusion.cpp` | `sd` | `text_to_image`, `image_to_image` | yes | build/install the `sd` binary |
| `diffusers` (sidecar) | `python3` + diffusers | `edit_instruct`, `inpaint`, `outpaint` | yes | Python + `diffusers`/`torch` (heavy) |
| `iopaint` | `iopaint` | `object_removal` | yes (`--device cuda`) | `pip install iopaint` |
| `realesrgan-ncnn-vulkan` | `realesrgan-ncnn-vulkan` | `upscale`, `denoise` | yes (Vulkan) | install the ncnn-vulkan release binary + models |
| `builtin` | in-process Go | `naturalize` | no (CPU) | no provisioning; always installed |
| `computed` | in-process math | `normal_map` | no (CPU) | no model weights; depends on the depth-map input |
| `library-go` | linked Go library | `duplicate_detect`, `qr_barcode_read` | no (CPU) | no model weights; shipped in the API binary |
| `library-cgo` | host C/C++ library | `ocr`, `face_detection`, `quality_assessment` | no (CPU) | Phase 2 backend doctor will probe libraries/binaries and data files |

> Status (2026-06-18): only the **onnxruntime sidecar `background_removal`** path
> is wired + proven end-to-end on CPU. The other backends are declared; their
> verticals (weights resolution + provisioning + proof) are built in later phases
> of the advanced-editing plan.

> Status (2026-06-19): Phase 1 catalog hardening has direct install assets for
> every enabled weight-backed seed model. The final migrated slice added
> `instruct-pix2pix`, `sd-1.5-inpainting`, `mi-gan`, `real-esrgan`, `dncnn`,
> `nafnet`, `ddcolor-tiny`, `restoreformer-plus-plus`, `mobilesam`,
> `moondream2`, and `smolvlm-256m`. `models doctor` should now be green; any
> future enabled weight-backed model without direct `source.assets[]` is a
> regression.

## The in-repo Python sidecar (`image_tools_sidecar`)

The CPU-tractable backend is a small Python package shipped **inside the Go
binary** (`api/internal/sidecar/py/image_tools_sidecar`). At boot the server
materializes it under `<data-dir>/sidecar/` and prepends that directory to
`PYTHONPATH`, so `python3 -m image_tools_sidecar.<op>` resolves regardless of the
working directory. You do **not** install the sidecar code separately.

What you *do* provision is the Python runtime it imports:

```bash
python3 -m pip install onnxruntime pillow numpy
```

`bg_removal.py` runs a U^2-Net / IS-Net family ONNX model on
`CPUExecutionProvider` and writes an RGBA PNG (subject over transparency).
`denoise.py` is a classical Pillow denoise (no model needed). If a dependency is
missing the sidecar exits non-zero with an actionable message (the job error
surfaces it) — it never silently succeeds.

## Model weights & artifact validation

`models install <id>` resolves a model's `Source.Assets` (direct, resolvable
weight URLs — HuggingFace `resolve/main/...`, GitHub release assets), downloads
each, and **validates the artifact before recording the install**:

- rejects HTML pages (content-type sniff) — the exact "downloaded a landing
  page" failure mode;
- enforces a size floor and the asset's declared `min_bytes`;
- checks format magic per `Kind` (ONNX, GGUF, safetensors);
- verifies an upstream `sha256` when published, else pins the computed hash
  **after** validation passes.

A seed model whose source is only a documentation page (not yet migrated to a
real asset) fails loud rather than recording a fake install.

### Example: background removal on CPU

```bash
image-tools models install u2netp --wait          # fetches + validates u2netp.onnx (~4.5 MB)
image-tools ai bg-removal photo.jpg --model u2netp --out cutout.png --wait
# → background_removal succeeded on u2netp/local-cpu
```

## BYOK cloud

The selection ladder supports a bring-your-own-key cloud tier as the last resort
when no local backend is available (opt-in with `--byok`, metered, cost-estimated
up front). The ladder + refusal semantics exist; a concrete key-gated cloud
provider is not yet registered (designed, deferred to a later phase). Until then,
on a host with no local backend an only-BYOK op refuses cleanly with guidance.
