# Integrations — Image Tools

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API (jobs, recipes, models, automation, measures, storage refs) | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| api-core storage/blobstore | platform storage | yes | storage seam, all operation domains | `storage` domain seam | Image bytes cannot be persisted; ops fail with a storage error. |
| root `vrooli` CLI host inventory | local platform | yes (for AI tier) | models selector, backends fallback | `vrooli host inventory --json` via `packages/vrooli-cli-go`, behind the `capabilities` seam (`internal/hostinventory` collector) | No host facts → AI ops cannot select a model; deterministic ops unaffected. |
| ComfyUI | Vrooli resource | no (`required:false`, opt-in) | generation, enhancement | optional plug-in provider via `backends`, `/ready` gated | If absent/not ready, `backends` uses standalone or CPU provider. |
| Per-op model resources | Vrooli resource | no (`required:false`, on-demand) | generation, enhancement, analysis | on-demand spin-up, `/ready` gated, idle-shutdown by orchestrator | If not ready, fallback ladder or "install model" prompt. |
| BYOK cloud providers | third-party | no (optional fallback tier) | backends fallback, models cost tracking | per-provider API + key (secret) | Used only as last tier; cost estimate required before use. |
| palette-gen | scenario | no (delegated) | analysis (palette extraction) | scenario call | Palette extraction unavailable; other analysis ops unaffected. |
| claude-code / configured agent | Vrooli resource | no | AI labeling / authoring paths only | agent invocation | Capability-labeling/authoring features degrade; core ops unaffected. |

## Vrooli Resources

image-tools keeps deterministic ops zero-dependency and makes every heavy
AI resource opt-in. Heavy resources are declared `required:false` with
`/ready` gating; idle-shutdown is delegated to the platform orchestrator
(mirroring the audio-tools optional-resource pattern). No AI resource is
ever a hard dependency — the CPU fallback model guarantees every AI op can
still run.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| ComfyUI | optional (`required:false`) | Opt-in plug-in provider for generation/enhancement, used only when already installed and `/ready`; never required. | Promote only if a profile mandates ComfyUI as a baseline backend. |
| Per-op model resources | optional (`required:false`, on-demand) | Each AI op's weights spin up on demand, `/ready`-gated, idle-shutdown by orchestrator; CPU-capable default install per op. | Add entries as new AI ops/models land (OT-P0/P1). |
| claude-code (or configured agent) | optional | Needed only where AI authoring or capability-labeling is required, not for image ops themselves. | Add hard requirement only if an op becomes agent-dependent. |

## Scenario Dependencies

Integration strategy is shared workflows > resource CLI > direct API.

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| palette-gen | optional (delegated) | image-tools delegates palette extraction to palette-gen rather than reimplementing it. | Scenario call for palette extraction. |
| text-tools | boundary peer (no call) | Clear, non-overlapping boundary: image-tools owns image→text/OCR; text-tools owns document→text. No runtime dependency. | Documented ownership boundary. |
| rich-media-studio | future consumer (P2) | Will compose image-tools via an embeddable UI component + stable contract (OT-P2-001). | Embeddable component + internal API contract. |
| test-genie | future consumer (P2) | Will adopt the image-diff/visual-comparison capability (OT-P1-009) as its visual-regression backend (OT-P2-002). | image-diff capability contract. |

Host hardware facts (OS/arch, CPU, memory, GPU/VRAM) are **not** a peer-scenario
dependency: they are read from the root `vrooli` CLI (`vrooli host inventory --json`,
the shared `internal/hostinventory` collector) through `packages/vrooli-cli-go`
behind the image-tools `capabilities` seam. image-tools does not call or depend on
system-monitor.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| BYOK cloud image providers | optional (third-tier fallback) | Last tier in the Local-GPU → Local-CPU → BYOK ladder; used only when local tiers cannot run an op. A cost estimate is presented and accepted before any cloud op executes (cost-tracking reuses the audio-tools usage-tracking pattern). | Per-provider API + user-supplied key (secret, never stored with images). |
| Webhook callback targets | optional (per-job) | A job may carry a signed callback URL; image-tools delivers a best-effort POST with retry on completion/failure. This is not an event bus — no pub/sub infrastructure. | Signed POST to a user-supplied URL. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| GPU contention | concurrent heavy jobs / VRAM pressure | Heavy jobs serialize in the dedicated queue; cheap CPU ops keep running concurrently. | queue serialization tests |
| Model download / disk pressure | download failure, checksum mismatch, low disk | Disk-space check + user confirmation before install; checksummed opt-in download; clear error on mismatch. | models install tests |
| GPU absent / insufficient VRAM | host probe reports no fit | Fallback ladder: Local-CPU (guaranteed model), then BYOK; surface "needs ≥X GB VRAM" + time warning. | selector + fallback tests |
| BYOK cost / availability | provider down or over budget | Pre-op cost estimate gates execution; user must accept; degrade gracefully if declined/unavailable. | cost-estimate + fallback tests |
| NVIDIA-only GPU detection | AMD/Intel/Apple-Silicon host | GPU not detected today → CPU/BYOK path; AMD/Intel/Apple hardening lands in the platform `internal/hostinventory` collector (surfaced through the root CLI host-inventory contract) and is then consumed via the `capabilities` seam (OT-P2-003). | detection tests (NVIDIA path) |
| Decompression bomb / oversized image | abnormal dimensions/ratio at ingestion | Ingestion-boundary guard rejects before processing. | ingestion-safety tests |
| ComfyUI / model resource not ready | `/ready` not green | `backends` falls through to standalone/CPU provider; never blocks deterministic ops. | backends fallback tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
