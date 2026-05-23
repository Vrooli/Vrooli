# Deployment — Architecture Cartographer

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
| Local Vrooli stack | target for v1 | Vrooli lifecycle; Go toolchain; Node/pnpm; SQLite path; `go-code-graph` and `typescript-code-graph` scenarios running on same host | Cartographer is pre-implementation; language-graph dependency scenarios exist (initialized 2026-05-23) but are not yet implemented (see [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md)). |
| Desktop/mobile app | deferred | Cross-platform runtime; packaged UI/API; bundled language-graph scenarios | Run `cross-platform-readiness` audit; resolve how language toolchains (Go, TS) are packaged for offline use. |
| Managed cloud/SaaS | deferred | Hosted runtime; auth; per-tenant isolation; observability; cost model | Cartographer reads source code — multi-tenant exposure requires authorization (see [`../internal/SECURITY.md`](../internal/SECURITY.md)). Substantial work. |
| Enterprise/self-host | deferred | Install docs; backup/restore; support model; license model | Requires operational hardening; not in v1 scope. |

## Runtime Requirements

- **API port**: assigned by lifecycle as `API_PORT` (range 15000-19999 per `.vrooli/service.json`).
- **UI port**: assigned by lifecycle as `UI_PORT` (range 20000-24999).
- **Storage**: `SQLITE_PATH` local file by default.
- **Required scenarios on same host**:
  - `go-code-graph` — must be running for any Go target scenario.
  - `typescript-code-graph` — must be running for any TS target scenario.
  - Cartographer's CLI surfaces actionable errors when either is unreachable (see [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — Failure Modes).
- **Local toolchains**:
  - `git` (required) — for `git-co-edit` signal and `apply` commits.
  - Go toolchain (`go build`) — required if target scenarios include Go code.
  - TypeScript toolchain (`tsc`, `pnpm`) — required if target scenarios include TS code.
- **Resources**: none external in v1. No Ollama, no Qdrant, no shared database. SQLite is bundled.
- **Network**: local API/UI/CLI communication only. No outbound calls beyond localhost.

## Packaging

| Surface | Packaging Notes |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/architecture-cartographer/`; generated clients are shared artifacts. |

## Release Checklist

Pre-flight gates before declaring a cartographer release ready:

- [ ] `make setup` passes.
- [ ] `make test` passes.
- [ ] PRD operational targets have linked requirements (`prd-control-tower prd validate architecture-cartographer --json` reports `healthy`).
- [ ] All P0 requirements report status `complete` or `in_progress` with passing automated validations.
- [ ] Template `notes` reference domain has been removed (Gate 7 in [`../START-HERE.md`](../START-HERE.md)).
- [ ] `go-code-graph` and `typescript-code-graph` are both at release-ready maturity and listed as runtime dependencies in `.vrooli/service.json`.
- [ ] `docs/manifest.json` maturity values reflect current docs (no doc shipped as `stub` if it covers active capability).
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md` are active.
- [ ] `MONETIZATION.md` and `GO-TO-MARKET.md` are either active or explicitly `not-applicable` with documented rationale.
- [ ] Cartographer dogfoods itself: `arch-cart conflicts list architecture-cartographer` against the cartographer's own manifest reports no unresolved conflicts above `warn` severity (see [`../internal/DECISIONS.md`](../internal/DECISIONS.md) entry on dogfooding).
- [ ] Fixture scenarios under `bas/fixtures/` produce stable expected-graph and expected-conflicts outputs across three independent runs.
- [ ] No `--force --note` usage in the analytics log for the release candidate's own migrations (or, if present, each is reviewed and accepted).

## Rollback

Local development rollback is source-control based. For deployed
targets, document the deployment-specific rollback path before release.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
