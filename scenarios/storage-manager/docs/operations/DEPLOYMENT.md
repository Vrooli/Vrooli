# Deployment — Storage Manager

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness.

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
| Desktop/mobile app | deferred | Cross-platform runtime, packaged UI/API, storage resolver | Run cross-platform readiness before adoption. |
| Managed cloud/SaaS | deferred | Hosted runtime, auth, observability, cost model | Requires deployment and monetization review. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Requires operational hardening. |

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
| Proto | Schemas live under `packages/proto/schemas/storage-manager/`; generated clients are shared artifacts. |

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
