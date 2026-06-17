# Deployment — Image Tools

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness for the local-first
image toolbox (Go API + React UI + Go CLI) in Vrooli's `*-tools` family.

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

## Supported Tiers

image-tools targets the Tier 1 local Vrooli stack today. Higher tiers
are documented in the [Deployment Hub](../../../../docs/deployment/README.md)
but are explicitly deferred — the scenario must reach P0 maturity and
clear cross-platform GPU hardening (deferred to P2 in the platform
`internal/hostinventory` collector, surfaced via the root CLI host-inventory
contract) before any packaged distribution.

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Local Vrooli stack | active | Full Vrooli install, lifecycle, Go, Node/pnpm, SQLite, api-core storage | None for deterministic ops; AI ops need opt-in model installs and host facts from the root `vrooli host inventory` CLI (via the `capabilities` seam). |
| Desktop app | deferred | Packaged UI/API, bundled standalone backends, storage resolver | Cross-platform GPU detection (AMD/Intel/Apple-Silicon) is P2; model-weight bundling/licensing unresolved. |
| Mobile | deferred | Remote API target, camera-capture upload path | No on-device inference; depends on a hosted API tier. |
| Managed cloud / SaaS | deferred | Hosted GPU runtime, auth, BYOK cost model, multi-tenant storage | Requires deployment + monetization review and GPU-pool capacity planning. |
| Enterprise / self-host | deferred | Install docs, backup/restore, support model | Requires operational hardening and license review of bundled models. |

## Runtime Requirements

Always-on, declared and started by the lifecycle:

- **Go API server** — Connect-RPC contracts plus a REST multipart edge
  for opaque binary uploads. Port assigned by lifecycle as `API_PORT`.
- **React + Vite UI** — production bundle served by `ui/server.js`.
  Port assigned by lifecycle as `UI_PORT`.
- **SQLite** — scenario metadata (jobs, recipes, model registry state,
  measures) at `SQLITE_PATH`.
- **api-core storage / blobstore** — image binaries stored outside the
  repo by default; per-request save-location override supported.
- **root `vrooli` CLI host inventory** — hardware capability/capacity facts
  read via `vrooli host inventory --json` (the shared `internal/hostinventory`
  collector) through `packages/vrooli-cli-go` behind the `capabilities` seam.
  AI ops depend on these host facts; deterministic ops do not. This is the
  local platform CLI, not a peer-scenario dependency (system-monitor is not
  used).

Declared `required:false`, NOT started by default — spin up only on demand:

- **Per-op model resources** — `/ready`-gated; downloaded on demand,
  checksummed, opt-in. Cheap deterministic ops need zero downloads.
- **ComfyUI** — optional second-tier backend; image-tools works fully
  via standalone backends (rembg, realesrgan, sd.cpp, LaMa, Tesseract,
  etc.) when ComfyUI is absent.
- **BYOK cloud providers** — optional third-tier fallback.

Hardware:

- **GPU is optional.** Every AI op ships a CPU-capable default model;
  CPU fallback always works (slower, with an explicit time warning).
- **Disk space** must accommodate opt-in model weights; the model
  manager enforces disk-space awareness and user confirmation before
  installs.
- GPU detection is strongest for NVIDIA today; other vendors are P2.

## Packaging

| Surface | Packaging Notes |
|---|---|
| API | Go binary built by the scenario lifecycle. Cheap deterministic ops require zero model downloads at build or first run. |
| UI | Vite production bundle served by `ui/server.js`. PWA manifest/icons under `ui/public/` remain valid. |
| CLI | Go CLI installed through scenario manifest install hooks; full headless parity with the UI. |
| Proto | Schemas under `packages/proto/schemas/image-tools/`; generated clients are shared artifacts. |
| AI models | Opt-in, on-demand installs through the model registry/manager — never bundled into the base scenario package. |

## Release Checklist

- [ ] `make setup` passes.
- [ ] `make test` passes (all required test-genie phases green).
- [ ] PRD operational targets have linked requirements that `validate`.
- [ ] cli/manifest.json measure blocks present (test-genie measures phase green).
- [ ] **Headless-completeness acceptance:** on a fresh GPU-less,
      ComfyUI-less machine, every operation runs end-to-end from the CLI
      (cheap ops instantly; AI ops via CPU-capable opt-in models).
- [ ] Backend-abstraction invariant holds: ≥1 standalone backend per AI op.
- [ ] Decompression-bomb / oversized-upload guards enforced at the ingestion boundary.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.

## Rollback

Local development rollback is the standard scenario lifecycle restart:
`make stop && make start` (or `make restart`) reverts to the last good
build. For a code-level revert, use the GCT baseline-restore path
(`git-control-tower baseline diff` / restore) to return the working
tree to a clean validated run, then restart the scenario. Opt-in model
weights are not rolled back by code reverts — manage them via the model
manager (`remove`/`install`) if a specific model entry is the culprit.
For any future deployed tier, document the deployment-specific rollback
path before release.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies (palette-gen, text-tools; host facts via the root `vrooli` CLI)
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
