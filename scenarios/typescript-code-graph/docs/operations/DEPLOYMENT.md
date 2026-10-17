# Deployment — TypeScript Code Graph

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
| Local Vrooli stack | target for v1 | Vrooli lifecycle, Go toolchain, Node runtime (≥20.x), pnpm, SQLite path (optional, P1 only) | Implementation pending — scenario is freshly initialized. The Node sidecar adds Node runtime to the dependency list. |
| Desktop/mobile app | deferred | Cross-platform runtime, packaged UI/API + sidecar, storage resolver | typescript-code-graph is infrastructure used by other scenarios; standalone desktop packaging is unlikely to be valuable. Bundling a Node runtime would also bloat the package. |
| Managed cloud/SaaS | deferred | Hosted runtime, auth, observability, multi-tenant story, sidecar per-tenant isolation | Add auth model + path-traversal hardening + sidecar isolation per tenant before any remote-exposure scenario. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model, sidecar bootstrap automation | Same as managed cloud. |

## Runtime Requirements

- **API port**: assigned by lifecycle as `API_PORT`.
- **UI port**: assigned by lifecycle as `UI_PORT`.
- **Storage**: embedded SQLite resolved by `api-core/storage` (only required if/when P1 Operation Log lands).
- **Go toolchain**: required for the API + CLI binaries to build.
- **Node runtime ≥20.x**: required to run the sidecar. Provided by the react-vite template's lifecycle.
- **pnpm**: required to install the sidecar's `ts-morph` dependency.
- **Resources**: none external by default. No Ollama, Qdrant, Postgres.
- **Network**: local API/UI communication only. The sidecar communicates with the API over inherited stdio (or a local Unix socket); no network port.

## Packaging

| Surface | Packaging Notes |
|---|---|
| API | Go binary built by scenario lifecycle. Spawns the sidecar as a child process at startup. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Sidecar | Node code under `sidecar/` (to be created). Built artifact at `sidecar/dist/index.js`. Lifecycle wires `pnpm install` and `pnpm build` for the sidecar in addition to the UI. |
| Proto | Schemas live under `packages/proto/schemas/typescript-code-graph/` plus the co-owned `packages/proto/schemas/common/v1/code_graph.proto`. Generated clients are shared artifacts. |

## Release Checklist

- [ ] `make setup` passes from a clean clone, including sidecar dependency install.
- [ ] `make test` passes including the determinism gate (`bas/fixtures/ts-*` including the load-bearing `ts-jsdoc-tags/` fixture).
- [ ] `make test` includes the performance regression suite (small + medium project budgets).
- [ ] `make test` includes the sidecar chaos tests (kill child mid-call, verify restart-with-backoff).
- [ ] PRD operational targets have linked requirements (`vrooli scenario requirements validate typescript-code-graph` reports `healthy`).
- [ ] Template reference `notes` domain has been removed (Gate 7 in [`../START-HERE.md`](../START-HERE.md)).
- [ ] `docs/manifest.json` maturity values reflect current docs state.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and `../business/MONETIZATION.md` are active or explicitly not-applicable.
- [ ] Path-traversal validation is wired on `Rewrite` ops (`ErrPathTraversal` returned for any out-of-project destination).
- [ ] At minimum two fixture TS projects ship in `bas/fixtures/` with hand-curated `expected-graph.json`, including the `ts-jsdoc-tags/` leading-comment contract fixture.
- [ ] Sidecar startup handshake rejects incompatible versions.
- [ ] Sidecar status is surfaced via `/health` and the diagnostics UI panel.

## Rollback

Local development rollback is source-control based. For deployed targets, document the deployment-specific rollback path before release.

For consumer scenarios that have applied a `Rewrite` and want to undo it: there is no scenario-side undo. Use `git restore .` in the operator's working tree. This is by design — see the "Never invoke git" decision in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

If the sidecar enters `permanently_unhealthy`, the scenario refuses to service `graph` / `rewrite` calls until manual recovery (typically `make restart`).

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
- [`../../PRD.md`](../../PRD.md) — operational targets
