# Integrations — Image Tools

## Purpose Of This Document

This document records integrations present in the current service declaration
and source. `.vrooli/service.json` and the shared capacity/host-tool contracts
are the executable dependency sources; this table explains the behavior users
see when an optional route is absent.

## Scenario Dependencies

Required scenario integrations are limited to the platform packages and
services declared by the scenario. Optional scenario consumers are listed
below with their fallback behavior.

## Vrooli Resources

Image Tools uses the platform lifecycle, database, blobstore, host-inventory,
capacity and host-tool owners described below. It does not own host repair.

## Dependency Inventory

The authoritative scenario declaration is [`.vrooli/service.json`](../../.vrooli/service.json).
Configuration options and their behavior are documented in
[`../reference/configuration.md`](../reference/configuration.md).

## Third-Party Services

Image Tools' optional external routes are limited to the service and host
integrations enumerated below.

## Required platform boundaries

| Dependency | Owner | Purpose | Degradation |
|---|---|---|---|
| Scenario lifecycle and control plane | Vrooli | Starts/builds/stops the API, UI and CLI; assigns ports and process identity. | Use managed lifecycle commands; direct process launches are unsupported. |
| Embedded SQLite and `api-core/database` | Platform package | Applies domain schemas and stores jobs, catalog state, measures and Look/preparation metadata. | API health is degraded/unhealthy when the database cannot be opened. |
| `api-core` blobstore | Platform package | Stores opaque image inputs and outputs behind Image Tools' `storage` seam. | Operations that need to retain/read blobs fail with an actionable storage error. |
| Host inventory | Control plane `internal/hostinventory`, exposed by `vrooli host inventory --json` | Supplies OS/architecture, CPU, RAM and GPU facts to model selection and GPU binding. | Model fit and execution requiring unavailable facts refuse; deterministic operations remain independent. |
| Shared capacity clients | Control plane / `packages/capacity` | Cross-process CPU, RAM and GPU admission, renewal and release for local jobs; supports warm-residency ownership. | Local work is queued, deferred or rejected before inference. A denied claim never authorizes unadmitted execution. |
| Scenario Dependency Analyzer host-tool owner | SDA plus platform host-tool declarations | Installs and verifies declared system/runtime tools such as Tesseract and supported backend binaries. | A missing tool is reported as unavailable; Image Tools does not run private host-repair commands. |

## Optional host tools

Image Tools declares these tools in `.vrooli/service.json`; their exact
acquisition and platform applicability come from the platform host-tool owner.

| Tool | Used for | Absent behavior |
|---|---|---|
| `google-chrome-stable` | High-fidelity SVG rendering when the pure-Go renderer cannot preserve filters, masks, patterns or CSS. | Named fidelity error for those inputs; ordinary supported SVG stays on the pure-Go path. |
| `ffmpeg` | Format conversions that the supported Go codecs do not provide. | Unsupported conversions return a clear error; other deterministic conversions remain available. |
| `realesrgan-ncnn-vulkan` | Real-ESRGAN upscale/denoise provider. | Related operations show provider/model unavailability; the tool is not downloaded by default setup. |
| `sd` | Stable Diffusion C++ text-to-image and image-to-image provider. | The provider is unavailable; GPU-capable routes are not silently claimed. |
| `iopaint` | Optional object-removal provider. | The backend reports unavailable; another declared compatible route may be selected. |
| `rembg` | Background-removal/background-replace provider. | The backend reports unavailable; another declared compatible route may be selected. |
| `llama-cpp` | Optional local caption provider. | Caption route is unavailable until the multimodal binary and model are present. |
| `uv` | Builds the private lock-pinned Python environment used by diffusers, ONNX, SAM2 and related sidecars. | Python providers report unavailable; the host `python3` is not used as a fallback. |
| `tesseract` | Local OCR plus English language data. | OCR reports its managed host-tool/model requirement; provisioning delegates installation to SDA. |

Weights are installed through the model/adaptor lifecycle and are not host-tool
packages. The one-command preparation surface is an Image Tools job over the
runtime/backend/model owners; the fresh-host all-capabilities showcase remains
open in the mission.

## Optional scenario integrations

| Scenario | Status | Contract |
|---|---|---|
| `ai-gateway` | Optional | Routes allowed cloud-provider operations and owns credentials, policy, transport and execution receipts. Local model operations do not require it. |
| `device-studio` | Optional consumer | Accepts a reviewed prepared-photograph candidate through its declared scenario API. Preparation review remains available if Device Studio is unavailable. |

There is no current dependency on `palette-gen`, Claude Code, a watch-folder
automation service or per-operation model resources. These were present in older
planning prose and must not be treated as runtime integrations.

## Optional cloud route and real-effects boundary

OpenRouter-backed model execution is an optional BYOK route through
AI Gateway. Image Tools identifies the route and forwards caller intent; the
credential owner, model policy and remote execution receipts remain outside
Image Tools. Cloud use stays opt-in and is not required for the local-first
mission. No paid request or credential is implied by an installed tool or
catalog row.

## Failure Modes

| Signal | Expected behavior |
|---|---|
| Host facts unavailable or stale | Refuse hardware-dependent planning; do not infer GPU fit from a model name. |
| Capacity broker defers/denies a local claim | Keep the job queued/retryable or fail an explicit `require_gpu` request without starting inference. Release every partial claim before retry/terminal completion. |
| Model artifact, backend tool or Python environment unavailable | Doctor/picker report the concrete lifecycle reason; the installer/provisioner uses its managed owner route. No silent switch to an unqualified interpreter or CPU path when the request requires GPU. |
| Blobstore unavailable | Fail the affected read/write path; deterministic in-memory transforms that need no retained blob may still be usable. |
| AI Gateway unavailable or cloud route not permitted | Local providers continue; cloud-only or BYOK-required operations report the unavailable route. |
| Optional Device Studio unavailable | Keep Image Tools preparation/review artifacts; defer only scene import. |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — service composition
- [`DATA.md`](DATA.md) — persistence and storage boundaries
- [`DOMAINS.md`](DOMAINS.md) — owners and implementation surfaces
- [`FLOWS.md`](FLOWS.md) — lifecycle and recovery behavior
