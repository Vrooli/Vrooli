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

Run the backend doctor before an attended AI run:

```bash
image-tools backends doctor
image-tools backends doctor --json
```

It calls `ModelsService/DoctorBackends` and reports, per backend, whether the
software is present on this host, which ops it serves, whether it can claim GPU,
and the provisioning path. It also reports enabled catalog backend families that
do not yet have a registered runtime provider, so declared-but-unpromoted
verticals are visible before an operator tries a job. Hardware fit is still
reported by `models select` / AI planning; backend doctor answers "is this
backend actually probeable/executable on this host?".

## Backends

| Backend | Program | Ops | GPU-capable | Management decision |
|---|---|---|---|---|
| `onnxruntime` (sidecar) | `python3` + onnxruntime | `background_removal`, `denoise`, `colorize`, `depth_map` | no (CPU) | Embedded sidecar code; Python runtime deps managed through Scenario Dependency Analyzer (SDA). |
| `python-sidecar` | `python3` modules | restoration/color/depth quality tiers | mixed | Embedded sidecar pattern; Python runtime deps managed through SDA before enabling a vertical. |
| `diffusers` (sidecar) | `python3` + diffusers | `edit_instruct`, `inpaint`, `outpaint`, `background_replace` | yes | Embedded sidecar pattern; `diffusers`/`torch` runtime deps managed through SDA. |
| `stable-diffusion.cpp` | `sd` | `text_to_image`, `image_to_image` | yes | Standalone binary managed through SDA / host-tool provisioning. |
| `iopaint` | `iopaint` | `object_removal` | yes (`--device cuda`) | Standalone CLI managed through SDA. |
| `llama.cpp` | `llama-mtmd-cli` / compatible `llama-cli` | `caption` | yes | Registered runtime provider; standalone multimodal llama.cpp binary managed through SDA before caption E2E is promoted. |
| `rembg` | `rembg` | `background_removal` (alt) | no (CPU) | Optional standalone CLI managed through SDA; ONNX sidecar remains the CPU floor. |
| `realesrgan-ncnn-vulkan` | `realesrgan-ncnn-vulkan` | `upscale` | yes (Vulkan) | Standalone ncnn-vulkan release binary managed through SDA / host-tool provisioning. |
| `realcugan-ncnn-vulkan` | `realcugan-ncnn-vulkan` | anime upscale variants | yes (Vulkan) | Standalone ncnn-vulkan release binary managed through SDA before enabling. |
| `builtin` | in-process Go | `naturalize` | no (CPU) | Shipped in the API binary; no provisioning. |
| `computed` | in-process math | `normal_map`, `quality_assessment` | no (CPU) | Registered runtime provider shipped in the API binary; no provisioning. |
| `library-go` | linked Go library | `duplicate_detect`, `qr_barcode_read` | no (CPU) | Registered runtime provider shipped in the API binary; no model weights. |
| `library-cgo` | host C/C++ library / binary | `ocr`, `face_detection` | no (CPU) | Registered Tesseract OCR and OpenCV YuNet providers; host binaries/libraries/data are managed through SDA. |

> Status (2026-06-19): Phase 1 catalog hardening has direct install assets for
> every enabled weight-backed seed model. The final migrated slice added
> `instruct-pix2pix`, `sd-1.5-inpainting`, `mi-gan`, `real-esrgan`, `dncnn`,
> `nafnet`, `ddcolor-tiny`, `restoreformer-plus-plus`, `mobilesam`,
> `moondream2`, and `smolvlm-256m`. `models doctor` should now be green; any
> future enabled weight-backed model without direct `source.assets[]` is a
> regression.

> Status (2026-06-19): Phase 2 backend doctor is wired and catalog-aware.
> `backends doctor` reports software readiness for registered runtime providers
> (`stable-diffusion.cpp`, `diffusers`, `iopaint`, `realesrgan-ncnn-vulkan`,
> `rembg`, `onnxruntime`, `llama.cpp`, `builtin`, `computed`, `library-go`,
> `library-cgo`) and
> also emits red rows for enabled catalog-declared backend operations with no
> runtime provider yet (currently `python-sidecar`, plus operation-level gaps
> under otherwise registered families such as `onnxruntime`
> classifiers/detectors, `realesrgan-ncnn-vulkan` `denoise`, and `rembg`
> `background_replace`).
> Selection errors include the same provisioning details for registered
> providers; the declared-but-unregistered rows become green as their operation
> verticals are promoted.

> Status (2026-06-19): The first in-process Phase 2 promotion is complete.
> `computed` now has a CPU provider for `normal_map` (depth/luma to normal-map
> PNG) and `quality_assessment` (structured quality metrics). `library-go` now
> has a CPU provider for `duplicate_detect` and `qr_barcode_read`; duplicate
> detection reuses the production pure-Go perceptual-hash implementation, while
> QR/barcode read is a registered lightweight seam pending the full decoder
> vertical. These rows require no host provisioning and should report available
> in `image-tools backends doctor`.

> Status (2026-06-19): `llama.cpp` is promoted to a registered caption backend.
> The provider probes `llama-mtmd-cli` first, then compatible `llama-cli`
> installations, builds argv from the installed text-model GGUF plus `mmproj`
> GGUF assets, captures the runner's stdout, and writes a structured caption JSON
> result. On hosts without a llama.cpp multimodal binary, `backends doctor`
> reports the row red with SDA provisioning guidance instead of the older
> "no runtime provider" catalog gap.

> Status (2026-06-19): `library-cgo` is partially promoted. The OCR vertical is
> now a registered provider that probes the Tesseract binary, executes the same
> `tesseract <image> stdout -l eng` command used by the synchronous analysis API,
> and writes structured OCR JSON when selected through the shared backend seam.
> Backend doctor now compares catalog backend+operation coverage, so this OCR
> provider did not hide the former `library-cgo` `face_detection` gap.

> Status (2026-06-19): `library-cgo` `face_detection` is promoted to a
> registered OpenCV YuNet provider. It probes `python3` plus `cv2`/`numpy`, runs
> the embedded `image_tools_sidecar.face_detection` module against the installed
> YuNet ONNX asset, and writes anonymous face-count/bounding-box JSON. On hosts
> without OpenCV Python/native libraries, `backends doctor` reports the row red
> with SDA provisioning guidance instead of "no runtime provider."

## The in-repo Python sidecar (`image_tools_sidecar`)

The CPU-tractable backend is a small Python package shipped **inside the Go
binary** (`api/internal/sidecar/py/image_tools_sidecar`). At boot the server
materializes it under `<data-dir>/sidecar/` and prepends that directory to
`PYTHONPATH`, so `python3 -m image_tools_sidecar.<op>` resolves regardless of the
working directory. You do **not** install the sidecar code separately.

What you *do* provision is the Python runtime it imports. Use Scenario
Dependency Analyzer rather than a raw package manager; the package names below
are the runtime requirements the SDA action should install/approve for this
scenario surface:

```bash
scenario-dependency-analyzer deps install pip/onnxruntime --scenario image-tools --surface api --apply
scenario-dependency-analyzer deps install pip/pillow --scenario image-tools --surface api --apply
scenario-dependency-analyzer deps install pip/numpy --scenario image-tools --surface api --apply
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
