# Deployment — Audio Tools

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
| Local Vrooli stack | existing development surface, not broad voice certification | Lifecycle-managed API/UI, writable domain storage, a serviceable permitted inference route | Qualify real streaming, quality, recovery and the nominated host profile. |
| Desktop/mobile browser consumer | staged target | Shared capture plus thin host adapter and reachable permitted inference | Browser compatibility does not prove a local model runs on that device; native D1–D6 receipts remain required for claims. |
| Packaged standalone runtime | deferred | Cross-platform artifacts, storage/secret resolver, update and rollback story | Qualify the actual packaged API, engine and browser host independently. |
| Managed cloud/SaaS | target, not launch-qualified | Authenticated hosted delivery, shared entitlement/ledger, observability and approved cost policy | Gateway implementation and live delivery/settlement evidence remain open. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Requires operational hardening. |

## Runtime Requirements

- API port: assigned by lifecycle as `API_PORT`.
- UI port: assigned by lifecycle as `UI_PORT`.
- Storage: embedded SQLite, resolved from the scenario id by `api-core/storage`.
- Resources: capability-selected local models and audio-format tools when the
  chosen route requires them. Optional dependencies permit degraded startup,
  not successful voice with no provider. See INTEGRATIONS.md.
- Network: same-origin consumer API transport; BYOK and owned service also require
  authorized remote provider access. An offline/local-only mode cannot silently
  fall back to those routes.

## Packaging

| Surface | Packaging Notes |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/audio-tools/`; generated clients are shared artifacts. |

## Release Checklist

- [ ] `make setup` passes.
- [ ] `make test` passes.
- [ ] PRD operational targets have linked requirements.
- [ ] Required targets and exact route/device/language cohorts have current
      accepted evidence, including partial rendering, final tails, privacy and recovery.
- [ ] Served build and lifecycle metadata identify the qualified source; stale
      healthy processes are not accepted as proof of a new build.
- [ ] Owned-route claims have approved catalog/policy, real gateway delivery,
      durable settlement/reconciliation and isolated fixture controls.
- [ ] Native support claims have matching native receipts; simulated device or
      provider results remain labeled as simulations.
- [ ] Retention limits, backup/restore, credential handling, resource bounds and
      deployment-specific rollback are qualified.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.

## Rollback

Retain the prior deployable artifact and compatible data/secret backup before
release. Specify recovery for schema, retained sessions and pending settlements;
do not delete a reservation or captured turn to make rollback appear complete.
Use the lifecycle/deployment owner to restore the approved artifact. Never reset
a shared dirty worktree to simulate rollback. The deployment-specific procedure
and evidence are still required before release.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
