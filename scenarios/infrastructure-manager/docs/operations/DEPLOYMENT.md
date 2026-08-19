# Deployment — Infrastructure Manager

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
| Local Vrooli stack | **the only supported tier** | Vrooli lifecycle, Go, Node/pnpm, SQLite path | The scenario reads local infrastructure through local scenario surfaces. It has no meaning on a host it is not observing. |
| Desktop/mobile app | not-applicable | — | Nothing to package. This is operator-and-agent tooling for the host it runs on. |
| Managed cloud/SaaS | not-applicable | — | The plant is a local platform. A hosted board would be observing nothing. Also no buyer — see [`../business/MONETIZATION.md`](../business/MONETIZATION.md). |
| Enterprise/self-host | not-applicable | — | Every Vrooli install is already self-hosted; this ships with the platform rather than as a distributed product. |

**Deployment is deliberately single-tier, and that is a property of the plant
rather than an unfinished roadmap.** This scenario supervises the layer stack
of the host it runs on: autoheal's check registry, the capacity broker's
claims, local storage growth. None of that is portable, so there is no tier-2+
story to defer — the honest answer is not-applicable rather than later.

One consequence worth stating: **the board must not be a critical scenario in
its own plant.** It observes the platform; it is not part of what the platform
needs to function. If it is down, the team falls back to reading sensors
directly — every member declares that fallback per the degradation contract in
`docs/agent-system/TARGET_MODEL.md`. The board makes the good path cheap; it
never makes the manual path illegal.

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
| Proto | Schemas live under `packages/proto/schemas/infrastructure-manager/`; generated clients are shared artifacts. |

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
