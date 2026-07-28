# Deployment — Content Desk

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness.

## Status — local now, Tier 2 desktop as the external target

Revised 2026-07-28 (D-017). This scenario runs locally through the Vrooli
lifecycle today. Its external delivery target is a **Tier 2 desktop
application** built through `scenario-to-desktop`, which is the fleet's existing
Electron ramp.

Hosted SaaS is explicitly **not** the target, and that is load-bearing rather
than a deferral. Local-first delivery removes three problems by construction:
no tenancy model, no hosted credential surface, and evidence checks that run as
ordinary local tool use instead of arbitrary remote code execution. A hosted
product would reopen all three at once.

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

## Supported Tiers

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Local Vrooli stack | active | Vrooli lifecycle, Go, Node/pnpm, SQLite path | Replace template reference domains before product deployment. |
| Desktop app (Tier 2) | **target** | `scenario-to-desktop` Electron packaging, bundled runtime, storage resolver, signing configuration | The P0 loop must work locally first. Run cross-platform readiness before packaging. |
| Managed cloud/SaaS | **rejected for now** | Hosted runtime, tenancy, sandboxed check execution, auth | Not a blocker list — a deliberate non-goal (D-017). Reopening it reopens D-016 and multi-user together. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Requires operational hardening. No demand signal. |

## Runtime Requirements

- API port: assigned by lifecycle as `API_PORT`.
- UI port: assigned by lifecycle as `UI_PORT`.
- Storage: `SQLITE_PATH` local file by default.
- Resources: none external by default.
- Network: local API/UI communication.

## Packaging

| Surface | Packaging Details |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/content-desk/`; generated clients are shared artifacts. |

## Release Checklist

- [ ] `make setup` passes.
- [ ] `make test` passes.
- [ ] PRD operational targets have linked requirements.
- [ ] Template reference domain has been replaced or explicitly retained
      with product justification.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.

## Rollback

Local development rollback is source-control based. For deployed
targets, document the deployment-specific rollback path before release.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
