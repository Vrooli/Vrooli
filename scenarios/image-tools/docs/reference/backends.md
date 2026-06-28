# AI Backends & Provisioning

image-tools runs AI operations through a provider ladder: **local GPU → local
CPU → BYOK cloud → refuse**. This page documents what each backend needs on the
host and how weights are provisioned. Deterministic ops (resize, crop, format,
metadata, …) and `analyze probe` are pure-Go and need **none** of this.

## How an AI op becomes runnable

A backend program and model weights must both be present, and the selector
reports them distinctly:

1. **A backend program** — the engine that runs the model. The selector probes
   "is the program on PATH?" (`Provider.Available`).
2. **The model weights** — installed on disk under
   `<data-dir>/models/<model-id>/`. `models install <id>` fetches + validates
   them; until then the op refuses with an actionable hint (HTTP 409).

A backend present but weights missing → `models install` hint. Weights present
but backend program missing → "no available provider" / enable BYOK. Both must
line up. The tier label (`local-gpu` / `local-cpu` / `byok-cloud`) reflects where
the op *actually* ran — a CPU-only backend reports `local-cpu` even on a GPU host.
GPU fit is based on live **free** VRAM from `vrooli host inventory --json`, not
adapter total VRAM, so a busy shared GPU falls back to CPU instead of being
optimistically scheduled into an OOM.

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
| `onnxruntime` (sidecar) | `python3` + onnxruntime | `background_removal`, `denoise`, `deblur`, `colorize`, `depth_map`, `object_detection`, `segment`, `tagging`, `nsfw_classify`, `embedding` | no (CPU) | Embedded sidecar code; Python runtime deps managed through Scenario Dependency Analyzer (SDA). The provider probes ONNX Runtime execution providers and requires `CPUExecutionProvider`; `CUDAExecutionProvider` presence is reported in doctor output, but this row stays CPU-labeled until a separate GPU-capable ONNX provider is promoted. The provider prefers the warm JSONL worker and falls back to one-shot `python3 -m image_tools_sidecar.<op>` if the worker dies. |
| `python-sidecar` | `python3` modules | `colorize`, `face_restore`, `old_photo_restore` | no (CPU for registered rows) | Registered embedded sidecar providers. `colorize` uses the ONNX/Pillow/numpy sidecar path; restoration probes the heavier RestoreFormer++ stack (`torch`, `basicsr`, `facexlib`) and stays red until provisioned through SDA. |
| `diffusers` (sidecar) | `python3` + diffusers | `edit_instruct`, `inpaint`, `outpaint`, `background_replace` | yes | Embedded sidecar pattern; `diffusers`/`torch` runtime deps managed through SDA. |
| `stable-diffusion.cpp` | `sd` | `text_to_image`, `image_to_image` | yes | Standalone binary managed through SDA / host-tool provisioning. |
| `iopaint` | `iopaint` | `object_removal` | yes (`--device cuda`) | Standalone CLI managed through SDA. |
| `llama.cpp` | `llama-mtmd-cli` / compatible `llama-cli` | `caption` | yes | Registered runtime provider; standalone multimodal llama.cpp binary managed through SDA before caption E2E is promoted. |
| `rembg` | `rembg` | `background_removal` (alt), `background_replace` (solid-color replace) | no (CPU) | Optional standalone CLI managed through SDA; ONNX sidecar remains the CPU floor for removal. |
| `realesrgan-ncnn-vulkan` | `realesrgan-ncnn-vulkan` | `upscale`, `denoise` (`-dn` mode) | yes (Vulkan) | Standalone ncnn-vulkan release binary managed through SDA / host-tool provisioning. |
| `realcugan-ncnn-vulkan` | `realcugan-ncnn-vulkan` | anime upscale variants | yes (Vulkan) | Standalone ncnn-vulkan release binary managed through SDA before enabling. |
| `builtin` | in-process Go | `naturalize` | no (CPU) | Shipped in the API binary; no provisioning. |
| `computed` | in-process math | `normal_map`, `quality_assessment` | no (CPU) | Registered runtime provider shipped in the API binary; no provisioning. |
| `library-go` | linked Go library | `duplicate_detect`, `qr_barcode_read` | no (CPU) | Registered runtime provider shipped in the API binary; no model weights. |
| `library-cgo` | host C/C++ library / binary | `ocr`, `face_detection` | no (CPU) | Registered Tesseract OCR and OpenCV YuNet providers; host binaries/libraries/data are managed through SDA. |

### Host-tool provisioning (generated)

The table below is generated from `providerSpecs()` (`api/internal/ai`) — the
single source of truth for which platform host tool each backend needs and the
exact remediation command. Do not edit it by hand; run `make backends-doc` to
regenerate. A test (`TestBackendsDocHostToolMatrixUpToDate`) fails the build if
it drifts. Host tools are declared as data in `internal/tools/<name>/tool.json`
and surfaced (never auto-fetched) via `image-tools` `service.json` `hostTools`.

<!-- BEGIN GENERATED: host-tool-matrix (regenerate with `make backends-doc`) -->
| Backend (provider) | Host tool | Operations | Install / remediation |
|---|---|---|---|
| `diffusers` | `uv` | `text_to_image`, `image_to_image`, `inpaint`, `outpaint`, `background_replace`, `edit_instruct` | `vrooli host install uv` |
| `iopaint` | `iopaint` | `object_removal` | `vrooli host install iopaint` |
| `llama.cpp` | `llama-cpp` | `caption` | `vrooli host install llama-cpp` |
| `onnxruntime` | `uv` | `denoise`, `deblur`, `background_removal`, `colorize`, `depth_map`, `object_detection`, `segment`, `tagging`, `nsfw_classify`, `embedding` | `vrooli host install uv` |
| `python-sidecar` | `uv` | `colorize` | `vrooli host install uv` |
| `python-sidecar` | `uv` | `face_restore`, `old_photo_restore` | `vrooli host install uv` |
| `realesrgan-ncnn-vulkan` | `realesrgan-ncnn-vulkan` | `upscale`, `denoise` | `vrooli host install realesrgan-ncnn-vulkan` |
| `rembg` | `rembg` | `background_removal`, `background_replace` | `vrooli host install rembg` |
| `stable-diffusion.cpp` | `sd` | `text_to_image`, `image_to_image` | `vrooli host install sd` |
<!-- END GENERATED: host-tool-matrix -->

## Operation Support Matrix

This matrix is the operator lookup for the enabled default path of each
model-backed or AI-adjacent operation. It mirrors the seed catalog's
`default_for` rows and should be updated in the same change as any default model
or backend reassignment.

| Operation | Default model | Backend | Tier | Host requirement |
|---|---|---|---|---|
| `background_removal` | `isnet-general-use` | `onnxruntime` | `default` | Isolated uv venv ONNX sidecar (onnxruntime/Pillow/numpy from the lock); CPU-capable. |
| `background_replace` | `birefnet-general` | `rembg` | `quality` | `rembg` CLI; CPU-capable but GPU preferred for throughput. |
| `caption` | `moondream2` | `llama.cpp` | `default` | `llama-mtmd-cli` or compatible multimodal `llama-cli`; CPU-capable GGUF path. |
| `colorize` | `ddcolor-tiny` | `python-sidecar` | `default` | Isolated uv venv ONNX sidecar (onnxruntime/Pillow/numpy from the lock); CPU-capable. |
| `deblur` | `nafnet` | `onnxruntime` | `default` | Embedded ONNX sidecar; CPU-capable, GPU preferred for large images. |
| `denoise` | `dncnn` | `onnxruntime` | `default` | Embedded ONNX sidecar; CPU-capable. |
| `depth_map` | `depth-anything-v2-small` | `onnxruntime` | `default` | Embedded ONNX sidecar; CPU-capable. |
| `duplicate_detect` | `goimagehash` | `library-go` | `default` | In-process pure Go; no host provisioning. |
| `edit_instruct` | `instruct-pix2pix` | `diffusers` | `default` | Isolated uv venv (diffusers/torch from `internal/pydeps/requirements.lock`); GPU recommended, CPU-capable but slow. |
| `embedding` | `nomic-embed-vision-v1.5` | `onnxruntime` | `default` | Embedded ONNX sidecar; CPU-capable. |
| `face_detection` | `yunet` | `library-cgo` | `default` | `python3` with OpenCV (`cv2`) and numpy; CPU-capable. |
| `face_restore` | `restoreformer-plus-plus` | `python-sidecar` | `default` | **Gated (not yet promoted)** — its stack (`torch`, `basicsr`, `facexlib`) is deliberately excluded from the lock; the op fails loud until the Phase 2 restore path lands (DECISIONS.md). |
| `image_to_image` | `sd-1.5` | `stable-diffusion.cpp` | `default` | `sd` binary; CPU-capable, GPU recommended. |
| `inpaint` | `sd-1.5-inpainting` | `diffusers` | `default` | Isolated uv venv (diffusers/torch from `internal/pydeps/requirements.lock`); GPU recommended, CPU-capable but slow. |
| `naturalize` | `naturalize-detail-v1` | `builtin` | `default` | In-process Go; no weights or host provisioning. |
| `normal_map` | `normals-from-depth` | `computed` | `default` | In-process Go math; no weights or host provisioning. |
| `nsfw_classify` | `adamcodd-vit-nsfw` | `onnxruntime` | `default` | Embedded ONNX sidecar; CPU-capable. |
| `object_detection` | `yolox-tiny` | `onnxruntime` | `default` | Embedded ONNX sidecar; CPU-capable. |
| `object_removal` | `mi-gan` | `iopaint` | `default` | `iopaint` CLI; CPU-capable, GPU recommended. |
| `ocr` | `tesseract` | `library-cgo` | `default` | Tesseract binary and language data; CPU-capable. |
| `old_photo_restore` | `restoreformer-plus-plus` | `python-sidecar` | `default` | **Gated (not yet promoted)** — same excluded-from-lock stack as `face_restore`; fails loud until the Phase 2 restore path lands (DECISIONS.md). |
| `outpaint` | `sd-1.5-inpainting` | `diffusers` | `default` | Isolated uv venv (diffusers/torch from `internal/pydeps/requirements.lock`); GPU recommended, CPU-capable but slow. |
| `qr_barcode_read` | `gozxing` | `library-go` | `default` | In-process pure Go; no host provisioning. |
| `quality_assessment` | `laplacian-blur` | `computed` | `default` | In-process Go analysis; no weights or host provisioning. |
| `segment` | `mobilesam` | `onnxruntime` | `default` | Embedded ONNX sidecar; CPU-capable. |
| `tagging` | `wd14-vit-v3` | `onnxruntime` | `default` | Embedded ONNX sidecar; CPU-capable. |
| `text_to_image` | `sd-1.5` | `stable-diffusion.cpp` | `default` | `sd` binary; CPU-capable, GPU recommended. |
| `upscale` | `real-esrgan` | `realesrgan-ncnn-vulkan` | `default` | `realesrgan-ncnn-vulkan` binary on a Vulkan-capable host; provision through SDA. |

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
> `library-cgo`, `python-sidecar`) and
> also emits red rows for enabled catalog-declared backend operations with no
> runtime provider yet.
> Selection errors include the same provisioning details for registered
> providers; the declared-but-unregistered rows become green as their operation
> verticals are promoted. The latest promotions add embedded ONNX sidecars for
> `deblur`, `object_detection`, `segment`, `tagging`, `nsfw_classify`, and
> `embedding`, registered operation coverage for `realesrgan-ncnn-vulkan`
> `denoise` and `rembg` `background_replace`, and a registered
> `python-sidecar` row for `colorize`, `face_restore`, and
> `old_photo_restore`. RestoreFormer++ restoration still requires SDA-managed
> Python packages before its E2E vertical can turn green.

> Status (2026-06-19): The first in-process Phase 2 promotion is complete.
> `computed` now has a CPU provider for `normal_map` (depth/luma to normal-map
> PNG) and `quality_assessment` (structured quality metrics). `library-go` now
> has a CPU provider for `duplicate_detect` and `qr_barcode_read`; duplicate
> detection reuses the production pure-Go perceptual-hash implementation, while
> QR/barcode read is a registered lightweight seam pending the full decoder
> vertical. These rows require no host provisioning and should report available
> in `image-tools backends doctor`.

> Status (2026-06-19): Phase 3 golden-contract coverage has started with the
> no-provisioning CPU rows. Tests now pin structural contracts for `normal_map`
> (PNG, dimensions, opaque alpha, Z-dominant normal channels, nonblank X
> gradients), `quality_assessment` (stable flat-fixture metrics and JSON field
> shape), and `duplicate_detect` (stable checker-fixture hashes and JSON field
> shape). The embedded ONNX sidecar also has structural helper contracts for
> background-removal normalization families, ImageNet preprocessing for
> depth/tagging/embedding/NSFW models, generic [0,1] tensor preprocessing for
> detection/segmentation, depth scaling, segmentation masks, detector box
> normalization, color/deblur output shaping, tag sigmoid scoring, and NSFW
> payload labeling. Remaining model-backed sidecars still need full
> fixture-backed E2E golden gates as their weights/provisioning become runnable.

> Status (2026-06-20): Phase 4 warm-sidecar hardening has started for the
> `onnxruntime` backend. Production ONNX sidecar calls now prefer a persistent
> `image_tools_sidecar.worker` JSONL process, and `_common.make_session()` caches
> ONNX Runtime sessions by model path inside that process so repeated requests do
> not pay model-load cost every time. The existing one-shot sidecar path remains
> the fallback when the worker exits or cannot start; this is an internal
> reliability fallback, not a compatibility surface. Runtime performance levers
> are now explicit: `IMAGE_TOOLS_CPU_WORKERS` bounds CPU-lane concurrency and
> `IMAGE_TOOLS_INSTALL_MB_PER_SECOND` tunes model-install ETA math without
> changing download behavior. A synthetic warm-worker benchmark now records the
> control result in `docs/internal/PERFORMANCE.md`: one-shot Python sidecar calls
> measured 46.2 ms/op versus 4.3 ms/op through the warm JSONL worker on
> 2026-06-20, using `go test ./internal/ai -run '^$' -bench
> 'BenchmarkWarmSidecarRunner_AmortizesModuleLoad' -benchtime=10x -count=1`.

> Status (2026-06-20): Phase 5 observability hardening has started. Every
> finalized durable job now records a structured `job_trace` row with operation,
> model id, backend, tier, lane, terminal state, queue wait, run duration, result
> ref, and error detail. AI submit payloads carry the selected backend/tier so
> traces and aggregate measures no longer need to infer those facts from logs.

> Status (2026-06-20): Phase 6 GPU-selection hardening has started. Model
> hardware fit now compares `min_vram_gb` to the best known **free** VRAM from
> the shared host-inventory probe (`vram_bytes - vram_used_bytes`), not total
> adapter VRAM. If another process is occupying the GPU, image-tools chooses a
> CPU-capable tier and emits a free-VRAM shortfall warning instead of claiming
> `local-gpu`. The ONNX sidecar decision is also now explicit: `backends doctor`
> reports the actual ONNX Runtime execution providers from host Python. On this
> host the probe reports `AzureExecutionProvider,CPUExecutionProvider`, not
> `CUDAExecutionProvider`, so ONNX jobs are honestly labeled `local-cpu`. GPU E2E
> should use the GPU-capable `diffusers`, `stable-diffusion.cpp`, `iopaint`, or
> ncnn-vulkan rows after SDA-managed runtime provisioning is present; a future
> ONNX GPU row should only be promoted once `CUDAExecutionProvider` is available
> and tested. `make gpu-e2e` is the attended Phase 6 proof gate for the
> `stable-diffusion.cpp` text-to-image path: it verifies free-VRAM fit and
> backend readiness, installs the selected model if needed, then runs a small
> `ai generate --wait` job. On unprovisioned hosts it exits as a documented skip
> with the missing backend detail rather than claiming a green GPU run.

> Status (2026-06-20): A no-download headless AI E2E gate now complements the
> attended GPU gate. `make headless-ai-e2e` generates a local PNG fixture and
> exercises `analyze probe`, `analyze quality`, `analyze duplicate`, `ai
> naturalize --wait`, and `ai normal-map --wait` through the public CLI. This
> proves the built-in/computed/library-Go model-lifecycle paths stay runnable on
> a clean local host without model downloads or external AI packages.

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

## Python isolation (isolated uv venv)

Every Python backend (`diffusers`, `onnxruntime`, `python-sidecar`) runs from a
**private, lock-pinned virtualenv** that is fully isolated from the host's
`python3` and site-packages. This is deliberate: a single shared interpreter is a
shared mutable namespace, and an unrelated `pip` upgrade on the box once pulled
`transformers` 5.x and broke `edit_instruct`. The venv closes that class of bug.

How it works (`api/internal/pyenv`, wired in `main.go`):

1. `uv` (a host tool — `vrooli host install uv`) builds the venv at
   `<data-dir>/pyenv/` and syncs it from the embedded, fully-pinned + hashed
   `api/internal/pydeps/requirements.lock` (the single version source of truth).
   The first build downloads torch (multi-GB) in the background.
2. The Go providers invoke the venv interpreter by **absolute path**
   (`<data-dir>/pyenv/bin/python`) — never a bare `python3` off `PATH`. There is
   **no host-interpreter fallback**: until the venv is built, the Python backends
   report *unavailable* (surfaced before use via `models doctor`, `/health`, and
   `ListOperationModels` ready_state), rather than silently running unisolated.
3. The venv is rebuilt automatically only when the lock's content hash changes
   (a sentinel file records the synced hash), so steady-state starts are instant.

Consequence: poisoning the host's `transformers` (or any other package) cannot
affect image-tools, and the proven dependency set is reproducible from the lock.
To change a version, edit `internal/pydeps/requirements.in` and regenerate the
lock (`internal/pydeps/README.md`) — never edit the lock or `pip install` by hand.

## The in-repo Python sidecar (`image_tools_sidecar`)

The CPU-tractable backend is a small Python package shipped **inside the Go
binary** (`api/internal/sidecar/py/image_tools_sidecar`). At boot the server
materializes it under `<data-dir>/sidecar/` and prepends that directory to
`PYTHONPATH`, so the venv interpreter's `python -m image_tools_sidecar.<op>`
resolves regardless of the working directory. You do **not** install the sidecar
code separately.

For ONNX Runtime operations, the Go provider starts the **venv interpreter**
`<data>/pyenv/bin/python -m image_tools_sidecar.worker` and sends one JSON
request per line. The worker invokes the same module entrypoints as the one-shot
path but keeps the process alive, allowing ONNX sessions to be cached per model
path. If the worker crashes or a request context is cancelled, the provider
restarts on the next request and falls back to the one-shot command for the
failed request.

You do **not** install the Python stack by hand. Every Python backend runs from a
private, lock-pinned uv venv that image-tools builds on start (see
[Python isolation](#python-isolation-isolated-uv-venv) below). The single thing
to provision is `uv` itself:

```bash
vrooli host install uv   # then (re)start image-tools; it builds the venv from the lock
```

The exact, fully-pinned + hashed dependency set lives in
`api/internal/pydeps/requirements.lock` (governed in SDA, regenerated with the
`uv pip compile` command in `internal/pydeps/README.md`) — it is the single
version source of truth. There is no separate `pip install` / per-package SDA
`deps install` step, and the host's own `python3`/site-packages are never used.

`bg_removal.py` runs a U^2-Net / IS-Net family ONNX model on
`CPUExecutionProvider` and writes an RGBA PNG (subject over transparency).
`deblur.py` runs image-to-image restoration exports such as NAFNet and writes an
RGB PNG. `detect.py` writes normalized bounding-box JSON for YOLOX-like detector
exports. `segment.py` writes a single-channel PNG mask for generic mask-output
ONNX exports and MobileSAM-style encoder/decoder directories. `tagging.py`
writes tag-score JSON; when no model-specific label asset is installed it emits
stable anonymous tag ids rather than pretending to know a vocabulary.
`denoise.py` is a classical Pillow denoise (no model needed). If a
dependency is missing the sidecar exits non-zero with an actionable message (the
job error surfaces it) — it never silently succeeds.

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
