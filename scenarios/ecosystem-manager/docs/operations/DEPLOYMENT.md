# Deployment — Ecosystem Manager

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness for Ecosystem Manager —
the internal Vrooli control plane for autonomous resource/scenario
generation and improvement.

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions and dependencies must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

Ecosystem Manager is an **internal platform service**. It is not sold
standalone; it ships and runs as part of the Vrooli monorepo on the
Tier-1 local stack.

## Supported Tiers

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Tier-1 local Vrooli stack | active | Vrooli lifecycle, Go, Node/pnpm, a writable storage data root (embedded SQLite — no database resource), agent-manager running; scenario-completeness-scoring optional for cached status views | None — this is the only supported tier today. |
| Desktop/mobile app | out of scope | Cross-platform runtime, packaged UI/API | Internal control-plane service; not packaged for end-user devices. |
| Managed cloud/SaaS | out of scope | Hosted runtime, multi-tenant auth, hosted data store | Not an externally sold product; see the [Deployment Hub](../../../../docs/deployment/README.md) for future-tier direction. |
| Enterprise/self-host | out of scope | Install docs, support model | Ships only inside a full Vrooli installation. |

## Runtime Requirements

- **Dashboard / UI**: `http://localhost:30500`.
- **API**: served under `http://localhost:30500/api`; port assigned by
  lifecycle via `API_PORT` (UI fixed at `21110`).
- **Health**: `GET /health` reports API + DB reachability.
- **Database**: embedded SQLite file at
  `<data-root>/vrooli/<namespace>/ecosystem-manager.db` (resolved via
  `api/pkg/storagepaths`). Domain-owned schemas
  (`api/pkg/{autosteer,steering}/schema.sql`) are applied
  at API boot by `database.EnsureSchemas`.
- **Hard dependencies (must be running)**:
  - `agent-manager` — executes the agent runs that perform generation and
    improvement work.
- **Optional readers**:
  - `scenario-completeness-scoring` — fast cached maturity/freshness/
    completeness status for operators and reports.
- **Filesystem state**: git-tracked `profiles/` (auto-steer profile JSON +
  `metadata.json`) in the scenario tree, and the runtime task queue YAML
  under `<data-root>/vrooli/<namespace>/queue/<status>/`.

## Packaging

Ecosystem Manager is part of the Vrooli monorepo and is built and managed
entirely through the scenario lifecycle. There is no standalone artifact.

| Surface | Packaging Notes |
|---|---|
| API | Go binary built by the scenario lifecycle (`make build` / `vrooli scenario`). Never direct-exec the binary. |
| UI | Vite production bundle served by the lifecycle-managed UI server. |
| CLI | Go CLI installed through the scenario manifest install hooks. |
| Schema | Domain-owned `api/pkg/{autosteer,steering}/schema.sql`, applied at API boot via `database.EnsureSchemas`; idempotent (`CREATE TABLE ... IF NOT EXISTS`). |

## Release Checklist

- [ ] `make setup` passes.
- [ ] `make test` passes (`vrooli scenario test ecosystem-manager`).
- [ ] The storage data root is writable; the embedded SQLite schema is
      applied at boot (no external database to provision).
- [ ] `agent-manager` is running and healthy.
- [ ] Optional: `scenario-completeness-scoring` is healthy when cached
      status views are part of the validation.
- [ ] `GET /health` reports API + DB healthy.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md` and `OBSERVABILITY.md` are current.

## Rollback

Rollback is source-control based:

1. Revert the scenario code to the previous known-good commit.
2. Re-run `make setup` to rebuild the API/UI/CLI; schemas re-apply at the
   next API boot.

The domain-owned SQLite schemas are idempotent (`CREATE TABLE/INDEX ... IF
NOT EXISTS`), so re-applying them against the existing database file is
safe and does not drop or recreate durable state. Restore durable data
from backup (data-backup-manager coverage of the storage root) only if a
destructive change is being rolled back (see
[`RUNBOOK.md`](RUNBOOK.md) → Backup / Restore).

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures and incident response
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health, logs, and product signals
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system structure and dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
- [`../../../../docs/deployment/README.md`](../../../../docs/deployment/README.md) — Vrooli deployment-tier hub (future direction)
