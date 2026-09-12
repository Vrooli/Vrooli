# Deployment — Device Control

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness.

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

## Supported Tiers

This scenario runs as a Tier 1 local service and is **not itself a deployment
target**. It is the thing that *validates* deployment targets, so the deferred
rows below are deferred by design rather than by backlog.

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Local Vrooli stack | active | Vrooli lifecycle, Go, Node/pnpm, SQLite path | Replace template reference domains before product deployment. |
| Desktop/mobile app | not-applicable | — | Deliberate. Packaging the device control plane as a device app inverts the layering. |
| Managed cloud/SaaS | deferred | Hosted runtime, auth, observability, cost model | Owner-scoped bridge trust makes multi-tenancy an architecture project; see `../business/MONETIZATION.md` obstacle 1. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Requires the three open security questions answered first. |

### Placement constraint

Device control is only useful on a node that can actually reach devices. An
instance on a machine with no attached devices and no bridge reach is correct
but useless — it reports an empty inventory rather than failing, which is the
intended behavior. Placement is therefore a capability question, not a
capacity one.

## Runtime Requirements

- API port: assigned by lifecycle as `API_PORT`.
- UI port: assigned by lifecycle as `UI_PORT`.
- Storage: embedded SQLite, resolved from the scenario id by `api-core/storage`.
- Resources: none external by default.
- Network: local API/UI communication.

## Packaging

| Surface | Packaging Details |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/device-control/`; generated clients are shared artifacts. |

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
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — why self-host precedes SaaS
