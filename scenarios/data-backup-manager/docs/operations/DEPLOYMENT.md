# Deployment — Data Backup Manager

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
| Local Vrooli stack | active (target) | Vrooli lifecycle, Go, Node/pnpm, SQLite path, `kopia` + `vault` resources | Not yet implemented; remove example `notes` domain. Companion `kopia` resource must be ready (`docs/plans/kopia-resource-plan.md`). |
| Desktop/mobile app | deferred | Cross-platform runtime, packaged UI/API, storage resolver | Run cross-platform readiness before adoption. |
| Managed cloud/SaaS | deferred | Hosted runtime, auth, observability, cost model, offsite destinations | Requires deployment and monetization review. |
| Enterprise/self-host | deferred (primary monetization path) | Install docs, encrypted + offsite destinations, restore/verify procedures, support model | Requires operational hardening; DR is a paid expectation at this tier (see `../business/MONETIZATION.md`). |

## Runtime Requirements

- API port: assigned by lifecycle as `API_PORT`.
- UI port: assigned by lifecycle as `UI_PORT`.
- Storage: `SQLITE_PATH` local file for the manager's own catalog and run
  history. Backup artifacts do **not** live here — they live in kopia
  repositories (destinations).
- Resources (required): `kopia` (backup engine — all repository,
  snapshot, restore, dedup, encryption, retention work) and `vault`
  (destination passphrases and backend access keys, read at runtime).
- Resources (on demand, per source kind): `postgres`, `redis`, `qdrant`,
  `minio` — needed only when a registered target uses that source kind.
- Network: local API/UI communication; outbound to remote destination
  backends (e.g., S3/MinIO) when configured.
- Storage roots: every destination must satisfy the separate-root rule —
  it must not live under the storage root it protects; an offsite
  destination is preferred for at least one tier.

## Packaging

| Surface | Packaging Notes |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/data-backup-manager/`; generated clients are shared artifacts. |

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
