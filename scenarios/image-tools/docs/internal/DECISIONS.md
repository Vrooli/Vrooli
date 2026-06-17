# Decisions — Image Tools

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

The decisions below were locked in the 2026-06-16 design workshop. All
are **Accepted** as of that date unless a later supersession is recorded.

| Date | Decision | Rationale | Status |
|---|---|---|---|
| 2026-06-16 | **Headless-completeness is a HARD tenet.** Every operation runs from the CLI with no UI and no ComfyUI. | The CLI is the canonical surface; the UI is an enhancer, not a gate. Other scenarios and agents compose image-tools programmatically. Acceptance is verified by running the full op set on a fresh GPU-less / ComfyUI-less machine. | Accepted |
| 2026-06-16 | **ComfyUI is an OPTIONAL plug-in backend, never required.** Every AI op must have ≥1 standalone (non-ComfyUI) backend. | ComfyUI is heavy and not present on most hosts. Requiring it would break the headless tenet and tiny-laptop deployments. It is accepted as a second provider only when already installed. | Accepted |
| 2026-06-16 | **Text→image generation lives IN image-tools** as a primitive `generate` verb, not in the future rich-media-studio. | Generation is a foundational image capability, not campaign orchestration. Placing it here keeps it reusable by every consumer. | Accepted |
| 2026-06-16 | **image-tools and rich-media-studio are SEPARATE scenarios.** | image-tools is the capability primitive (single image ops); rich-media-studio is campaign orchestration that consumes it. Mixing the two would couple a reusable primitive to a higher-level workflow. | Accepted |
| 2026-06-16 | **No-GPU fallback chain = Local-GPU → Local-CPU → BYOK cloud.** CPU fallback is supported even with no BYOK key, with an explicit time warning. | Guarantees every AI op works on any host. CPU fallback prevents a hard dependency on a paid key; the time warning sets expectations. | Accepted |
| 2026-06-16 | **Persistent storage via api-core storage/blobstore** by default (outside repo); save location overridable per request; outputs are user-owned. | Reuses the platform storage substrate rather than reinventing it. Per-request override and user ownership keep outputs portable (copy/move/delete anywhere). | Accepted |
| 2026-06-16 | **image-tools OWNS image→text (OCR / caption / alt-text); text-tools owns document→text.** | Clear, non-overlapping boundary prevents duplicate capability and conflicting models across the `*-tools` family. | Accepted |
| 2026-06-16 | **Palette extraction delegates to palette-gen.** | palette-gen already owns this capability; image-tools calls it instead of reimplementing, avoiding duplication. | Accepted |
| 2026-06-16 | **Hardware detection reuses internal/hostinventory via the system-monitor capability/capacity probe.** | OS/hardware detection is owned by system-monitor/hostinventory. Reinventing it would fork detection logic and drift. The probe endpoint is a hard prerequisite that lands first. | Accepted |
| 2026-06-16 | **Model selection is driven by a declarative, schema/proto-validated model registry with hardware-fit.** Per-op defaults + user override; custom/fine-tuned local models supported. | A declarative registry makes selection auditable, testable, and extensible. Hardware-fit lets the same op pick the right model per host. Override and custom entries keep power users unblocked. | Accepted |
| 2026-06-16 | **Async jobs are durable + server-owned** (survive client disconnect) and follow the test-genie run-lifecycle philosophy: job-id + ETA up front, block-once wait, no polling. A GPU-serializing queue runs heavy jobs one at a time while cheap CPU ops run concurrently. | Client cancellation must not destroy in-flight work (mirrors the test-genie durable-run fix). Serializing GPU work prevents VRAM contention; concurrent CPU ops keep cheap work responsive. | Accepted |
| 2026-06-16 | **Face restoration is exposed as its own op.** | A face-restore model already ships for old-photo restoration; surfacing it directly gives users a first-class, composable operation. | Accepted |
| 2026-06-16 | **NSFW detection is both a standalone op and a configurable auto-scan** of generated output. Generation models carry capability labels (NSFW-capable, license/commercial-use, lineage, known-risks). No formal "copyright" label exists. | Safety must be usable directly and applied automatically to AI output. The industry has no standardized copyright label, so copyright/legal lineage risk is captured as free-text known-risks instead of a fabricated boolean. | Accepted |
| 2026-06-16 | **C2PA content credentials supported as opt-in, off by default.** | Provenance signing is valuable but adds a signing key and overhead; defaulting off keeps the common path lightweight and avoids surprising users. | Accepted |
| 2026-06-16 | **Webhook callbacks are MINIMAL: per-job signed URL + best-effort retry — NOT an event bus.** | Job-completion notification is a small, contained feature. A full pub/sub event bus would be disproportionate new infrastructure and attack surface. | Accepted |
| 2026-06-16 | **Image diff / visual-comparison is built as a general capability that test-genie can later ADOPT.** | Building it generically (successor, not overlap) lets test-genie consume one canonical visual-regression backend rather than maintaining a parallel one. | Accepted |
| 2026-06-16 | **NON-GOALS (explicitly rejected scope):** no face recognition / identity matching; no face-swap / deepfake / try-on; no video / motion content (→ rich-media-studio); no vector editing (SVG is raster import/export only). | These are ethically fraught, out of the single-image-primitive scope, or owned by another scenario. Recording them prevents scope creep and accidental relitigation. | Accepted |
| 2026-06-16 | **Model registry SEEDED with a CPU-capable, commercial-clean default + one quality tier per op** (49 entries / 26 ops). Seed: [`api/internal/models/registry.seed.json`](../../api/internal/models/registry.seed.json); catalog: [`reference/model-registry.md`](../reference/model-registry.md). | Gives the registry mechanism (OT-P0-006) a concrete, license-verified baseline. Every op works headless on CPU; quality tiers are opt-in. | Accepted |
| 2026-06-16 | **Stay GATE-FREE on generation licenses:** ship only no-cap licenses (SD 1.5 / SDXL OpenRAIL, Flux.1 schnell Apache). EXCLUDE Stability's $1M-revenue-gated SD 3.5 / SD-Turbo. | Avoids any revenue-tracking obligation or commercial-license exposure for downstream users of a monetizable scenario. | Accepted |
| 2026-06-16 | **Optional Python sidecar is ALLOWED for quality-tier models** (RestoreFormer++, SCUNet, Restormer, DDColor-large, RAM++, Marigold, Florence-2); never required. | Unlocks best-quality PyTorch-only models without breaking the headless / Go-native default tier. The CPU default for every op stays compiled-binary / ONNX / pure-Go. | Accepted |
| 2026-06-16 | **GFPGAN is EXCLUDED; RestoreFormer++ (Apache-2.0) is the sole face-restore backend.** | GFPGAN's "Apache" license embeds StyleGAN2 (NVIDIA non-commercial) + DFDNet (CC-BY-NC-SA) and its ncnn port is GPL — not commercial-safe. RestoreFormer++ is cleanly licensed. | Accepted |
| 2026-06-16 | **Registry carries an explicit BLOCKLIST of license-encumbered models** (CodeFormer, GFPGAN, bria-RMBG, FastSAM, Ultralytics-YOLO, YOLO-NAS, InsightFace, Surya, community-NC ESRGANs, MAT, Flux-dev/Fill, SD3.5/Turbo, Qwen-VL-3B, LLaVA, pyiqa, libpHash, tuotoo-qrcode, …). | These are the most-recommended models online and would be adopted by accident. Two rules: check *weights* license separately from *code*, and ONNX/GGUF export never strips AGPL or a non-commercial weight license. | Accepted |
| 2026-06-16 | **Model checksums are captured + pinned on first download — NEVER hand-written.** Seed ships empty `checksum.value` with `status: unverified-capture-on-download`. | A fabricated hash is false verification. Pinning on first real download gives genuine integrity checks thereafter. | Accepted |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | No durable decision has been replaced. Add an entry here when one is. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`SECURITY.md`](SECURITY.md) — security posture for the decisions above
- [`PERFORMANCE.md`](PERFORMANCE.md) — budgets driven by the async/queue decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
