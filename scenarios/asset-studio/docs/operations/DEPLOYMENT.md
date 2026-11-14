# Deployment — Asset Studio

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
| Local Vrooli stack | active | Vrooli lifecycle, Go, Node/pnpm, SQLite path, writable blob directory, reachable ai-gateway | No product domain exists yet; only the scaffold runs. |
| Desktop app (Tier 2) | candidate (`ASSET-P2-004`) | `scenario-to-desktop` packaging, packaged UI/API, storage resolver, bundled or reachable ai-gateway | Should ship the same way as `content-desk` or deliberately not; decide the two together. |
| Managed cloud/SaaS | **rejected for now** | Hosted runtime, auth, tenancy, cost attribution per tenant | Deliberately not the plan. Local-first keeps unreleased marketing material and generated persona assets on the operator's machine, and avoids attributing generation spend across tenants — which is a billing product, not a feature. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Requires the conformance gate to be validated first; the differentiator has to work before anyone is asked to run it. |

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
| Proto | Schemas live under `packages/proto/schemas/asset-studio/`; generated clients are shared artifacts. |

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
