# Deployment — Go Code Graph

This document records supported delivery tiers, packaging assumptions, runtime dependencies, and deployment readiness.

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

## Supported Tiers

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Local Vrooli stack | target for v1 | Vrooli lifecycle, Go toolchain, Node/pnpm (UI), SQLite path (optional, P1 only) | Implementation pending — scenario is freshly initialized. |
| Desktop/mobile app | deferred | Cross-platform runtime, packaged UI/API, storage resolver | Run cross-platform readiness audit before adoption. Go Code Graph is infrastructure used by other scenarios; standalone desktop packaging is unlikely to be valuable. |
| Managed cloud/SaaS | deferred | Hosted runtime, auth, observability, multi-tenant story | Add auth model + path-traversal hardening (see [`../internal/SECURITY.md`](../internal/SECURITY.md)) before any remote-exposure scenario. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Same as managed cloud. |

## Runtime Requirements

- **API port**: assigned by lifecycle as `API_PORT`.
- **UI port**: assigned by lifecycle as `UI_PORT`.
- **Storage**: `SQLITE_PATH` local file (only required if/when P1 Operation Log lands).
- **Go toolchain**: required for the API + CLI binaries to build. Runtime does not require the toolchain on the host — `golang.org/x/tools/go/packages` is statically linked in, but it does require the target Go module to be loadable, which transitively means a recent Go SDK on the host is recommended.
- **Resources**: none external by default. No Ollama, Qdrant, Postgres.
- **Network**: local API/UI communication only.

## Packaging

| Surface | Packaging Notes |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/go-code-graph/` plus the co-owned `packages/proto/schemas/common/v1/code_graph.proto`. Generated clients are shared artifacts. |

## Release Checklist

- [ ] `make setup` passes from a clean clone.
- [ ] `make test` passes including the determinism gate (`bas/fixtures/go-*`).
- [ ] `make test` includes the performance regression suite (small + medium module budgets).
- [ ] PRD operational targets have linked requirements (`prd-control-tower requirements validate go-code-graph` reports `healthy`).
- [ ] Template reference `notes` domain has been removed (Gate 7 in [`../START-HERE.md`](../START-HERE.md)).
- [ ] `docs/manifest.json` maturity values reflect current docs state.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and `../business/MONETIZATION.md` are active or explicitly not-applicable.
- [ ] Path-traversal validation is wired on `Rewrite` ops (`ErrPathTraversal` returned for any out-of-module destination).
- [ ] At minimum two fixture Go modules ship in `bas/fixtures/` with hand-curated `expected-graph.json`.

## Rollback

Local development rollback is source-control based. For deployed targets, document the deployment-specific rollback path before release.

For consumer scenarios that have applied a `Rewrite` and want to undo it: there is no scenario-side undo. Use `git restore .` in the operator's working tree. This is by design — see the "Never invoke git" decision in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
- [`../../PRD.md`](../../PRD.md) — operational targets
