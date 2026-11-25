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
the op *actually* ran — a CPU-only backend reports `local-cpu` even on a GPU host. A finished local job also carries `execution_tier` and `execution_device` (`gpu` or `cpu`) in its meta: the submit-time tier is the plan, and admission can still move a job to the CPU when the GPU lease is denied.
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

## Local inference capacity and lifetime

Every production local CPU or GPU inference reserves the selected catalog
model's positive `min_ram_gb` through the shared host capacity broker, using
no-reclaim admission. The host RAM reservation precedes any GPU reservation or
provider execution. A denial does not start a worker, reclaim another service,
or silently choose a paid provider. GPU jobs also retain their separately
validated, UUID-bound VRAM claim.

The engine renews both leases during execution and cancels the operation if
renewal fails. Before releasing RAM, an admitted JSONL sidecar is stopped and
reaped, including successful inference. This prevents a resident model from
outliving its operation's reservation. The result records
`ram_capacity_claim_bytes`, `ram_capacity_policy`, and
`ram_capacity_release`; an uncertain release is reported with native TTL
recovery rather than asserted successful. The declared RAM reservation is an
admission budget, not a claim of measured peak RSS or an operating-system memory
ceiling. Model export builds have a separate owned-process/cgroup lifecycle.

The native MoGe CPU proof on 2026-10-22 completed with a 4 GiB claim and no
remaining JSONL worker after completion. The exact claim was independently
observed released. Cancellation, denial-before-provider, worker teardown and
release-error behavior are covered by focused engine tests and the race-enabled
AI package suite. This proof qualifies that selected execution path, not every
catalog model's memory estimate.

The broker never infers idleness: a claim reads idle until its work-owner reports
otherwise, and an idle claim that is unprotected or yields can be reclaimed by
higher-priority work. Image-tools therefore reports every claim it holds through
`vrooli capacity activity`: active when admission completes and the job starts
(RAM, CPU, GPU and support claims alike), and for a warm worker's resident claims
active while a job serves on the worker and idle during its 30-second idle window.
Each report advances the claim generation, which the shared lease adopts so
heartbeats stay valid. A failed report is recorded in the job meta
(`capacity_activity`, `capacity_resident_activity`) and never stops the work.
The ledger cannot attribute observed GPU bytes to the sidecar processes, so these
claims still read `observed=0`; only the reported activity distinguishes a running
job from an idle one.

## Backends

| Backend | Program | Ops | GPU-capable | Management decision |
|---|---|---|---|---|
| `onnxruntime` (sidecar) | `python3` + onnxruntime | `background_removal`, `denoise`, `deblur`, `colorize`, `depth_map`, `face_restore`, `object_detection`, `segment`, `tagging`, `nsfw_classify`, `embedding` | CUDA only for qualified ops (`providerSpec.gpuQualifiedOps`): `background_removal`, `segment`, `depth_map`, `face_restore`, `face_swap`, `colorize`, `denoise`; CPU for the rest. Measured 2026-11-23 (warm session, CPU 4 threads vs CUDA): depth-anything-v2-large 2.9 s vs 0.11 s, ddcolor-large 2.4-2.9 s vs 0.95-1.2 s, bg removal 0.67 s vs 0.45 s; SCUNet denoise on 768x1024 18-20 s vs 1.0 s (needs about 6 GB); the small classifiers (nsfw, age, embedding, detection) gain 0.02-0.2 s, less than a lease costs; OCR stays on the CPU (its detector pyramid needs 1.8 GB buffers and failed on CUDA with 3-7 GB free), and deblur stays too (the NAFNet export is int8, which the CUDA provider cannot run). A job that cannot get its GPU lease runs on the CPU (`capacity_fallback=cpu`), and one whose GPU run dies of GPU memory exhaustion repeats once on the CPU in the same job (`capacity_fallback=cpu_after_oom`, `execution_device=cpu`). The ledger counts every claim, including the VRAM an idle warm worker keeps for cached weights, so when a GPU claim is denied the engine first unloads this service's own idle warm workers on that GPU (least recently used first, retrying the claim after each) and records `evicted_residents=<models>`; a worker serving a job and any other owner's claim are never touched | Embedded sidecar code; Python runtime deps managed through Scenario Dependency Analyzer (SDA). The provider probes ONNX Runtime execution providers and requires `CPUExecutionProvider`; an op claims the GPU only when `CUDAExecutionProvider` is present and the op's CUDA path has been qualified. The provider uses the JSONL sidecar protocol, reports failures once, and starts fresh on the next request. Production local calls reap the worker before releasing operation-scoped RAM; the unleased control benchmark retains bounded warm reuse. |
| `python-sidecar` | `python3` modules | `colorize`, `old_photo_restore` | no (CPU for registered rows) | Registered embedded sidecar providers. `colorize` uses the ONNX/Pillow/numpy sidecar path; `old_photo_restore` (scratch repair + colour + faces) has no promoted pipeline and stays red. |
| `diffusers` (sidecar) | `python3` + diffusers | `edit_instruct`, `inpaint`, `outpaint`, `background_replace` | yes | Embedded sidecar pattern; `diffusers`/`torch` runtime deps managed through SDA. |
| `stable-diffusion.cpp` | `sd` | `text_to_image`, `image_to_image` | yes | Standalone binary managed through SDA / host-tool provisioning. |
| `iopaint` | `iopaint` | `object_removal` | yes (`--device cuda`) | Standalone CLI managed through SDA. |
| `llama.cpp` | `llama-mtmd-cli` / compatible `llama-cli` | `caption` | yes | Registered runtime provider; standalone multimodal llama.cpp binary managed through SDA before caption E2E is promoted. |
| `gateway-vision` | AI Gateway (`ExecuteRoute`) over Ollama | `caption`, `object_detection`, `tagging` | no (the gateway places the model) | General vision-language models next to the Florence-2 specialist, one registry row each (`runtime.gateway_role`, `box_order`, `box_scale`); see "Add a gateway-served vision model" in `model-registry.md`. Local, no consent; reports the `gateway-local` tier (the gateway owns the model's CPU, GPU and memory, so no CPU or GPU claim is made). Needs `ai-gateway` and its Ollama resource running. |
| `openrouter` (vision rows) | AI Gateway (`ExecuteRoute`), remote-only | `caption`, `object_detection`, `tagging` (and the image operations) | no | The hosted rung of the same executor: BYOK, spend cap, and a consent affirmation when the image shows people (the people check runs at submit; a picture with nobody in it needs none, see `SECURITY.md`). |
| `rembg` | `rembg` | `background_removal` (alt), `background_replace` (solid-color replace) | no (CPU) | Optional standalone CLI managed through SDA; ONNX sidecar remains the CPU floor for removal. |
| `realesrgan-ncnn-vulkan` | `realesrgan-ncnn-vulkan` | `upscale`, `denoise` (`-dn` mode) | yes (Vulkan) | Standalone ncnn-vulkan release binary managed through SDA / host-tool provisioning. |
| `realcugan-ncnn-vulkan` | `realcugan-ncnn-vulkan` | anime upscale variants | yes (Vulkan) | Standalone ncnn-vulkan release binary managed through SDA before enabling. |
| `builtin` | in-process Go | `naturalize`, `color_transfer`, `mask_composite` | no (CPU) | Shipped in the API binary; no provisioning. |
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
| `diffusers` | `uv` | `text_to_image`, `image_to_image`, `inpaint`, `outpaint`, `background_replace`, `edit_instruct`, `face_swap`, `garment_replace` | `vrooli host install uv` |
| `florence-transformers` | `uv` | `caption`, `object_detection` | `vrooli host install uv` |
| `iopaint` | `iopaint` | `object_removal` | `vrooli host install iopaint` |
| `llama.cpp` | `llama-cpp` | `caption` | `vrooli host install llama-cpp` |
| `onnxruntime` | `uv` | `denoise`, `deblur`, `background_removal`, `object_removal`, `colorize`, `depth_map`, `face_restore`, `face_swap`, `pose_extract`, `scene_geometry`, `object_detection`, `segment`, `tagging`, `nsfw_classify`, `ocr`, `age_estimate`, `embedding` | `vrooli host install uv` |
| `python-sidecar` | `uv` | `colorize`, `old_photo_restore` | `vrooli host install uv` |
| `realesrgan-ncnn-vulkan` | `realesrgan-ncnn-vulkan` | `upscale`, `denoise` | `vrooli host install realesrgan-ncnn-vulkan` |
| `rembg` | `rembg` | `background_removal`, `background_replace` | `vrooli host install rembg` |
| `sam2-transformers` | `uv` | `segment` | `vrooli host install uv` |
| `segformer-transformers` | `uv` | `segment` | `vrooli host install uv` |
| `stable-diffusion.cpp` | `sd` | `text_to_image`, `image_to_image` | `vrooli host install sd` |
| `tesseract` | `tesseract` | `ocr` | `vrooli host install tesseract` |
| `triposr` | `uv` | `image_to_object` | `vrooli host install uv` |
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
| `caption` | `florence-2` | `florence-transformers` | `quality` | Isolated uv venv (torch + transformers from `internal/pydeps/requirements.lock`) and the pinned Florence-2 large checkpoint (~1.55GB); about 1 s warm on GPU, CPU-capable. `smolvlm-256m` / `moondream2` (`llama.cpp`, `llama-mtmd-cli`) remain selectable CPU options. |
| `colorize` | `ddcolor-large` | `onnxruntime` | `quality` | Embedded ONNX sidecar; CUDA when the capacity lease grants about 4 GB (0.95-1.2 s warm), otherwise the CPU (2.4-2.9 s warm, 5-11 s cold). `ddcolor-tiny` is the fast, oversaturated tier. |
| `deblur` | `nafnet` | `onnxruntime` | `default` | Embedded ONNX sidecar, CPU only (the NAFNet export is int8 and has no CUDA kernel). Runs at the image's own resolution in 1024 px tiles; works well on motion blur (PSNR +1.8 dB on a 15 px blur, whiskers and text recovered), only mildly on defocus/Gaussian blur. About 15 s for 768x1024. |
| `denoise` | `dncnn` | `onnxruntime` | `default` | Embedded ONNX sidecar; CPU-capable. |
| `depth_map` | `depth-anything-v2-large` | `onnxruntime` | `quality` | Embedded ONNX sidecar; CPU-capable (~1.9s CPU, ~60ms CUDA, 2.3GB VRAM). CC-BY-NC-4.0: normal use only; `depth-anything-v2-small` (Apache) stays selectable for monetized use. Chosen on the 2026-11-21 evaluation (finer branch/sky and fur edges). |
| `duplicate_detect` | `goimagehash` | `library-go` | `default` | In-process pure Go; no host provisioning. |
| `edit_instruct` | `instruct-pix2pix` | `diffusers` | `default` | Isolated uv venv (diffusers/torch from `internal/pydeps/requirements.lock`); GPU recommended, CPU-capable but slow. |
| `embedding` | `siglip2-base` | `onnxruntime` | `quality` | Embedded ONNX sidecar (reads the installed `preprocessor_config.json`); CPU-capable. |
| `face_detection` | `yunet` | `library-cgo` | `default` | `python3` with OpenCV (`cv2`) and numpy; CPU-capable. |
| `face_restore` | `restoreformer-plus-plus` | `onnxruntime` | `quality` | Embedded ONNX sidecar (`face_restore.py`): bundled YuNet detection, FFHQ-template alignment, 512px restore, feathered paste-back; no `basicsr`/`facexlib`. CPU-capable. `codeformer` (S-Lab non-commercial) is the selectable alternative; RestoreFormer++ won the 2026-11-21 identity comparison. |
| `face_swap` | `hyperswap-1a-256` | `onnxruntime` | `quality` | Embedded ONNX sidecar (`face_swap.py`), no `insightface`/`cv2` and no `~/.insightface` download: bundled YuNet detection, ArcFace identity embedding, 256 px swapper, XSeg occlusion mask, feathered paste-back with LAB colour matching. One managed model directory holds all four graphs. Swapper alternatives (same pipeline, picked by filename): `hyperswap-1b-256`, `hyperswap-1c-256`, `inswapper-128`, `alphaface-256`. Non-commercial research weights: normal local use only. A cloud route (`openrouter-image`, role `image.edit.identity`, both images sent) exists and needs a consent affirmation, but it is not recommended: on the 2026-11-23 comparison Gemini returned the original face unchanged. Chosen on that comparison (five swappers, two SFW tests, ArcFace similarity plus by-eye blending): `hyperswap-1a-256` is the sharpest and cleanest; `inswapper-128` (128 px) shows hairline and lash artefacts; `alphaface-256` is smoother. Prefer the raw swap: RestoreFormer++ afterwards smooths skin and lowers identity similarity. Swap extent: the manifest's `extent` block declares independent stages (`face` with a 0..1 `face_strength` that mixes the two identity embeddings, `skin`, `hair_style`, `hair_colour`, and `head`, a whole-head swap through `flux-2-klein-4b` that replaces `face`) and presets over them (`face`, `face_skin`, `full_look`, `whole_head`). The UI panel, the CLI (`--extent`, `--match skin,hair-style,hair-colour`, `--face-strength`) and the engine all read that block. A restyle stage (`hair_style`) has `flux-2-klein-4b` redo the working image with the reference's hairstyle (a two-image edit; the reference should be a tight portrait), segments the hair of the working image and of the edit (`segformer-b2-clothes`), and lays only that union region of the edit into the working image with `mask_composite`, so the swapped face is not re-rendered. The `skin` stage takes a declared option, `skin_target` (CLI `--skin-target`): `source` (default) moves the parsed skin (face and neck, arms and hands) toward the reference person's skin tone with a wide feather, so the change fades into skin the parser has no class for (a V-neck chest); `body` instead matches the face to the working image's own hands and arms. `--face-strength` is met by measurement: the sidecar swaps at the two ends, measures the identity of each result with the same ArcFace embedder and bisects the embedding-mix weight until the result sits that fraction of the way between the target's identity and the source's (the swapper's response saturates differently for every pair of faces, so the raw mix weight is not linear); the job summary reports `identity_mix` and `identity_progress`. A recolour stage is `segment` (the `segformer-b2-clothes` parser, on both the working image and the reference) then `color_transfer`, each through the normal admission, safety review and persistence; the job meta lists the stages that ran (`extent_stages`) and each stage's mask model and seconds, and the intermediate images are deleted. Every stage after the swap is local. Evidence: `research:runs/capabilities-20261122/outputs/face-swap/README.md`. |
| `garment_replace` | `sd-1.5-inpainting` | `diffusers` | `default` | Isolated uv venv (diffusers/torch from `internal/pydeps/requirements.lock`); masked inpaint driven by a prompt. GPU recommended, CPU-capable but slow. |
| `image_to_image` | `sd-1.5` | `stable-diffusion.cpp` | `default` | `sd` binary; CPU-capable, GPU recommended. |
| `inpaint` | `sd-1.5-inpainting` | `diffusers` | `default` | Isolated uv venv (diffusers/torch from `internal/pydeps/requirements.lock`); GPU recommended, CPU-capable but slow. |
| `naturalize` | `naturalize-detail-v1` | `builtin` | `default` | In-process Go; no weights or host provisioning. |
| `color_transfer` | `color-transfer-lab-v1` | `builtin` | `default` | In-process Go; CIELAB mean/spread transfer inside a mask onto the colour of a reference region (the reference's alpha selects its region), keeping the region's own shading. The skin-tone and hair-colour stages of a face swap run on it. |
| `mask_composite` | `mask-composite-lab-v1` | `builtin` | `default` | In-process Go; lays a second image (resampled to the first's size) into the first through a mask, with a grown, feathered seam. The hair-style stage of a face swap runs on it. |
| `normal_map` | `normals-from-depth` | `computed` | `default` | In-process Go math; no weights or host provisioning. |
| `nsfw_classify` | `falconsai-nsfw` + `adamcodd-vit-nsfw` | `onnxruntime` | `default` | Embedded ONNX sidecar; the shared review profile supplies both classifiers and their bars. CPU-capable. |
| `object_detection` | `florence-2` | `florence-transformers` | `quality` | Same worker and checkpoint as `caption`. `yolox-tiny` is a fast fixed-vocabulary alternative (verdict `works`, 80 COCO labels, 0.2-0.3 s on the CPU; fixed 2026-11-23 by decoding the raw head in the sidecar). |
| `object_removal` | `big-lama` | `onnxruntime` | `quality` | Embedded ONNX sidecar; CPU-capable. `mi-gan` (`iopaint` CLI) is disabled: the IOPaint backend needs a manual pip/pipx install. |
| `ocr` | `paddleocr` | `onnxruntime` | `quality` | Embedded ONNX sidecar (`ocr.py`, PP-OCRv5 detector + recogniser, no OpenCV); CPU-capable (about 15-40 s on a large photo). Tesseract (`library-cgo`) stays selectable. |
| `old_photo_restore` | `ms-bringing-old-photos` (disabled) | `python-sidecar` | `default` | **Not promoted** — needs a scratch-repair + colour + face pipeline; `face_restore` covers the face stage alone. |
| `outpaint` | `sd-1.5-inpainting` | `diffusers` | `default` | Isolated uv venv (diffusers/torch from `internal/pydeps/requirements.lock`); GPU recommended, CPU-capable but slow. |
| `qr_barcode_read` | `gozxing` | `library-go` | `default` | In-process pure Go; no host provisioning. |
| `quality_assessment` | `laplacian-blur` | `computed` | `default` | In-process Go analysis; no weights or host provisioning. |
| `segment` | `mobilesam` | `onnxruntime` | `default` | Embedded ONNX sidecar; CPU-capable. |
| `tagging` | `qwen3-vl-4b` | `gateway-vision` | `quality` | A general vision model served by AI Gateway's Ollama (role `vision.default`); 33-54 s a call on a shared GPU, one local call at a time. Default since 2026-11-23: `wd14-vit-v3` (`onnxruntime`) is not installed and its sidecar names tags `tag_<index>`. `gemini-flash-lite-vision` (hosted, BYOK, consent) answers in about 1 s. |
| `text_to_image` | `sd-1.5` | `stable-diffusion.cpp` | `default` | `sd` binary; CPU-capable, GPU recommended. |
| `upscale` | `real-esrgan` | `realesrgan-ncnn-vulkan` | `default` | `realesrgan-ncnn-vulkan` binary on a Vulkan-capable host; provision through SDA. Runs on the leased GPU (the Vulkan device whose UUID matches the capacity lease, bound with `-g`) or, on the CPU tier, on the software Vulkan device (llvmpipe/lavapipe). 4x upscale uses the bundled `realesrgan-x4plus`; 2x/3x and denoise use `realesr-animevideov3`. |

`object_removal` has a second, always-available provider: the builtin
`mask-fill` backend (model `mask-fill`) is a pure-Go membrane fill — dilate the
mask, grow it over a colour halo, solve Laplace inside the region, composite. It
needs no host tool, so object removal succeeds even when `iopaint` is absent.
`iopaint` remains the quality path and is ranked ahead of `mask-fill`; the
builtin is deterministic and exact on flat or smooth backgrounds but smears on
textured ones. `vrooli host install iopaint` reports `manual_action_required`
until a governed installer is provisioned.

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
> `python-sidecar` row for `colorize` and `old_photo_restore`. `face_restore`
> moved to the ONNX sidecar on 2026-11-21 (RestoreFormer++ / CodeFormer ONNX
> exports), so it needs no extra Python packages.

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
> `onnxruntime` backend. The `image_tools_sidecar.worker` JSONL process caches
> ONNX Runtime sessions by model path, bounded to one active model (at most two
> encoder/decoder sessions). Since 2026-10-22, production local inference holds
> an operation-scoped RAM lease and reaps the worker before that lease is
> released. Unleased control/benchmark callers retain model-switch cleanup and
> the 30-second idle bound; their warm timing is not a production latency promise.
> Failed requests are reported once; the next request starts a fresh worker.
> Runtime performance levers
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

How it works (`packages/pyenv-go`, wired in `main.go`):

1. `uv` (a host tool — `vrooli host install uv`) builds the venv at
   `<cache-dir>/pyenv/` and syncs it from the embedded, fully-pinned + hashed
   `api/internal/pydeps/requirements.lock` (the single version source of truth).
   The first build downloads torch (multi-GB) in the background.
2. The Go providers invoke the venv interpreter by **absolute path**
   (`<cache-dir>/pyenv/bin/python`) — never a bare `python3` off `PATH`. There is
   **no host-interpreter fallback**: until the venv is built, the Python backends
   report *unavailable* (surfaced before use via `models doctor`, `/health`, and
   `ListOperationModels` ready_state), rather than silently running unisolated.
3. The venv is rebuilt automatically only when the lock's content hash changes
   (a sentinel file records the synced hash), so steady-state starts are instant.

The synchronous `analysis/nsfw` route and generated-output auto-scan use this
same absolute interpreter. Both now run the same classifier ensemble and
thresholds as the bundled reviewer (the resolved safety profile), so the
standalone op and the generation gate cannot answer the same image differently.
If provisioning is absent or incomplete, analysis reports the backend
unavailable and auto-scan keeps its documented best-effort behavior; neither
path probes or falls back to host `python3`.

Consequence: poisoning the host's `transformers` (or any other package) cannot
affect image-tools, and the proven dependency set is reproducible from the lock.
To change a version, edit `internal/pydeps/requirements.in` and regenerate the
lock (`internal/pydeps/README.md`) — never edit the lock or `pip install` by hand.

## The in-repo Python sidecar (`image_tools_sidecar`)

The CPU-tractable backend is a small Python package shipped **inside the Go
binary** (`api/internal/sidecar/py/image_tools_sidecar`). At boot the server
materializes it under `<cache-dir>/sidecar/` and prepends that directory to
`PYTHONPATH`, so the venv interpreter's `python -m image_tools_sidecar.<op>`
resolves regardless of the working directory. You do **not** install the sidecar
code separately.

For ONNX Runtime operations, the Go provider starts the **venv interpreter**
`<cache>/pyenv/bin/python -m image_tools_sidecar.worker` and sends one JSON
request per line. The worker invokes the same module entrypoints as the one-shot
path but keeps the CPU process alive for at most 30 seconds idle, with two
sessions maximum and cache eviction on model changes. A crash or cancellation
is reported without replaying inference. Teardown joins the child and protocol
reader before returning; the next request starts fresh. API shutdown closes
providers after stopping jobs.

Diffusion processes remain one-shot, so framework/model memory ends with each
process. The scheduler passes an explicit `--device cpu|cuda`; CUDA availability
cannot override a CPU decision. GPU work requires a valid host capacity grant,
renews it every one-third of its ten-minute TTL with the generation guard, and
stops when renewal fails. Cleanup releases with a separate bounded context even
when the job was canceled. A GPU-only model cannot be relabeled as CPU fallback.
The stable-diffusion.cpp CPU route pins both `--backend cpu` and
`--params-backend cpu`: [upstream backend selection](https://github.com/leejet/stable-diffusion.cpp/blob/master/docs/guides/backend.md)
distinguishes runtime compute from parameter residency. CPU offloading alone
does not prove CPU execution.

Prompted segmentation is available as `image-tools ai segment` / the durable
`POST /api/v1/ai/segment` edge with schema-validated `AIParams.op_params.segmentation` points/box.
The qualified existing runtime is CPU MobileSAM (SAM1 export contract), not
SAM2. The synchronous `select segment` region-grow remains heuristic and does
not invoke this model path. See [layered photography](layered-photography.md)
for usage, source-image variants and the unqualified hybrid stages.

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
RGB PNG. `detect.py` writes normalized bounding-box JSON (class id and COCO label) for YOLOX
exports: it letterboxes, decodes the raw head, applies class-wise NMS and undoes the letterbox. `segment.py` writes a single-channel PNG mask for generic mask-output
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
when no local backend is available (opt-in with `--byok`, metered). It is the
**last** tier: a registered cloud provider is chosen only when `allow_byok` is set
AND no local standalone backend can run the op, or when explicitly forced via
`--model`. Local-first behaviour is never affected by the cloud provider's
presence.

The registered cloud provider is **`openrouter`** (backend `openrouter`, seed
model `openrouter-image`). It serves `text_to_image`, `image_to_image`, and
`edit_instruct` through OpenRouter's dedicated image API
(`POST /api/v1/images`): the prompt (and, for edits, the source image as an
`input_references` data URL) goes up, and the response returns the image as
base64 in `data[].b64_json` (vector models add a `media_type`). It holds no local
weights, so `RequiresWeights()` is false and it is always "installed";
runnability is gated instead by `Available()` — a usable `OPENROUTER_API_KEY`.

- **Key resolution:** `OPENROUTER_API_KEY` from the environment, injected by
  the scenario lifecycle from the canonical credential authority. The provider
  never invokes a resource CLI or reads a user credential file. A usable key
  must look like `sk-or-…` (length floor). No key → the provider reports
  unavailable and only-BYOK ops refuse cleanly with guidance.
- **Model selection (greenfield):** no concrete slug is hard-coded. Each request
  resolves an OpenRouter **policy role** through `resource-openrouter policy
  resolve` — `text_to_image` → `image.generate.default`, edits →
  `image.edit.default`. A caller carries intent through the request param
  `openrouter_role` (e.g. brand-manager requesting `image.generate.logo` or
  `image.edit.identity`). The concrete model, endpoint, and request defaults all
  come from `resources/openrouter/model-policy.json` — the single update point.
- **Costs + privacy:** every call is billed to your key and sends the prompt (and
  any input image) to a third-party cloud — not for private/local-only data.
