# Deployment — Lighting Control

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
| Local Vrooli stack | implemented; partially operated | Vrooli lifecycle, Go, Node/pnpm, owned SQLite namespace | Actual authenticated setup, selected-device access, headless persistence and physical output remain qualification work. |
| Desktop/mobile app | deferred | Cross-platform runtime, packaged UI/API, storage resolver | Run cross-platform readiness before adoption. |
| Managed cloud/SaaS | deferred | Hosted runtime, auth, observability, cost model | Requires deployment and monetization review. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Requires operational hardening. |

## Runtime Requirements

- API port: assigned by lifecycle as `API_PORT`.
- UI port: assigned by lifecycle as `UI_PORT`.
- Storage: embedded SQLite, resolved from the scenario id by `api-core/storage`.
- Resources: none external by default; host-native OpenRGB is an explicit machine opt-in. Its current native runtime supports Linux amd64. Other platforms remain unqualified/unsupported as stated in the [resource operations guide](../../../../resources/openrgb/docs/OPERATIONS.md).
- Network: local API/UI communication.

## Packaging

| Surface | Packaging Details |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/lighting-control/`; generated clients are shared artifacts. |

## Release Checklist

- [ ] `make setup` passes.
- [ ] `make test` passes.
- [ ] PRD operational targets have linked requirements.
- [ ] Applicable platform and device qualification is recorded; cross-builds and SDK acknowledgements are not physical-output proof.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.

## Rollback

Use the normal scenario lifecycle to adopt or return to a reviewed application
build. Source rollback does not restore machine-local SQLite policy or
controller enrollment; follow the [runbook state recovery procedure](RUNBOOK.md#backup--restore)
for a separately requested data restore. Keep lighting disabled until schema,
stable identity and desired-state checks pass. OpenRGB artifact/update rollback
and selected-device permission rollback stay with the [resource owner](../../../../resources/openrgb/docs/OPERATIONS.md);
reverting this scenario does not revert host grants, root intent or firmware.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
