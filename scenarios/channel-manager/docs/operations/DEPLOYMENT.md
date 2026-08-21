# Deployment — Channel Manager

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
| Desktop/mobile app (**Tier 1**) | **target** | Cross-platform runtime, packaged UI/API, storage resolver | The intended shape. Runs on the operator's own device and network, which is what makes warming function at all. See `../business/MONETIZATION.md` § Deployment tier. |
| Self-hosted full runtime (Tier 2) | viable | Standard Vrooli runtime | Same reasoning as Tier 1 — the operator's own hardware and network. |
| Managed cloud / hosted Vrooli (**Tier 3**) | **ruled out** | — | **Not deferred — excluded on technical grounds.** Precondition `residential-proxy-locked` requires a residential IP pinned per identity for the account's life. Hosted execution originates from datacenter IPs, which is precisely the fingerprint warming exists to avoid. Revisit only if the warming capability is dropped entirely. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model, **auth** | Blocked on the auth gap in `../internal/SECURITY.md`. The console exposes the whole portfolio and has no access control. |

### Why deployment target is a correctness concern here

For most scenarios the deployment tier decides where data lives. For this one it
decides **whether the product works**. An identity's distribution depends on its
traffic originating from a consistent residential IP in one region; a hosted tier
breaks that invariant no matter how well the software behaves. That is why Tier 3
is recorded as excluded rather than as future work.

The corollary is a genuine constraint on Tier 1 packaging: the app must run
unattended for scheduled actions to fire in their windows. "Runs locally" and "runs
only when someone is looking at it" are different things, and the second one is not
sufficient.

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
| Proto | Schemas live under `packages/proto/schemas/channel-manager/`; generated clients are shared artifacts. |

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
