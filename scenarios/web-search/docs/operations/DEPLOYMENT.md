# Deployment — Web Search

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness.

> **Status (2026-06-09):** Not implemented yet. This describes the
> *intended* deployment shape from `PRD.md` and
> [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md).

## Purpose Of This Document

Use this document to answer:

- Where can this scenario run?
- What runtime assumptions must hold?
- What blocks desktop, mobile, cloud, SaaS, or enterprise packaging?
- What must pass before deployment?

## Supported Tiers

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Tier 1 local Vrooli stack | active (intended) | Vrooli lifecycle, Go, Node/pnpm, SQLite path, healthy SearXNG/Qdrant/Ollama/reranker resources, search-hub reachable for registration | Not yet implemented; SearXNG must be verified healthy first (see precondition below). |
| Desktop/mobile app | deferred | Cross-platform runtime, packaged UI/API, storage resolver, bundled/remote SearXNG | Resource (SearXNG/Qdrant/Ollama) availability off the local stack is unsolved. |
| Managed cloud/SaaS | deferred | Hosted runtime, auth, observability, cost model, hosted search backend | Requires monetization review and external-engine rate-limit strategy at scale. |
| Enterprise/self-host | deferred | Install docs, backup/restore, support model | Requires operational hardening (findings backup/restore). |

## SearXNG-Healthy Precondition (read first)

**Before any P0 work or deployment, verify the SearXNG resource is
healthy and standards-current on this host.** SearXNG is the live-web
engine behind L0/L1 (and the L2 source) — the entire external-search
value depends on it. It exists and appears maintained, but it must be
confirmed reachable before web-search can serve `web-search.live`.

If SearXNG is down at deploy time, web-search still starts: the
`web-search.live` provider degrades to unavailable (graceful
"rate-limited / try later"), while the `web-search.learnings` corpus is
unaffected and keeps serving. See
[`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) for the full
degraded-behavior matrix.

## Runtime Requirements

- **API port:** assigned by lifecycle as `API_PORT` (proto-first
  Connect-RPC + REST `/health`).
- **UI port:** assigned by lifecycle as `UI_PORT` (react-vite + Tailwind).
- **Storage:** `SQLITE_PATH = ${SCENARIO_DATA_DIR}/web-search.db` for
  findings/briefs/audit, applied on startup via `api-core/database`.
- **Resources (P0, `try_start`):** SearXNG (`SEARXNG_URL`) for live web;
  Qdrant for the `web-search-findings` semantic index; Ollama
  (`nomic-embed-text` + a small chat model) for embeddings, L1/L2/L3
  synthesis, and distillation; reranker (TEI cross-encoder) for findings
  ranking. All `try_start` — web-search degrades rather than fails if any
  are missing.
- **Scenario dependency (P0):** search-hub reachable at boot so the two
  providers can self-register (idempotent upsert from `.vrooli/search.json`).
- **Resources/scenarios (P1, currently disabled):** browserless (L2 page
  fetch + readable-text extraction) and agent-manager (L3 iterative
  research runs). Enabled when their levels ship (OT-P1-001 / OT-P1-002).
- **Network:** local API/UI communication; outbound web access only via
  the local SearXNG resource (and, in P1, browserless) — never direct
  external-engine calls.

## Packaging

| Surface | Packaging Notes |
|---|---|
| API | Go binary built by scenario lifecycle. |
| UI | Vite production bundle served by `ui/server.js`. |
| CLI | Go CLI (`web-search`) installed through scenario manifest install hooks. |
| Proto | Schemas live under `packages/proto/schemas/web-search/`; generated clients are shared artifacts. |
| Search providers | `web-search.live` (SCOPE_EXTERNAL) and `web-search.learnings` (SCOPE_PROJECT) declared in `.vrooli/search.json`, self-registered with search-hub at boot. |

## Deploy / Lifecycle

Start, stop, and inspect only through the Vrooli lifecycle — never run
the API/UI binaries directly:

```bash
vrooli scenario start web-search   # or: make start
make status
make logs
make stop
```

On startup the scenario applies its SQLite schema, wires the resource
clients (SearXNG/Qdrant/Ollama/reranker), and self-registers its two
providers with search-hub. If search-hub is unreachable at boot it
retries briefly, serves locally, and re-registers on the next boot.

## Release Checklist

- [ ] SearXNG resource verified healthy and standards-current on this host.
- [ ] `make setup` passes.
- [ ] `make test` passes (test-genie required phases green).
- [ ] PRD operational targets (OT-P0-001..008) have linked requirements.
- [ ] Both providers register with search-hub; default federated query
      fires no live-web request (scope-aware blending verified).
- [ ] Cache + budget governor return graceful "rate-limited" rather than
      hammering SearXNG.
- [ ] Template `notes`/`health` reference domains replaced or explicitly
      retained with justification.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, `SECURITY.md`, and
      `MONETIZATION.md` are active or explicitly not-applicable.

## Rollback

Local development rollback is source-control based. The findings store is
**never hard-deleted** (supersede/archive only) — treat
`web-search.db` and the Qdrant `web-search-findings` collection as
durable, non-disposable data and migrate rather than recreate on schema
change (Vrooli SQLite policy). For deployed targets, document the
deployment-specific rollback path (including findings backup/restore)
before release — currently deferred (see [`RUNBOOK.md`](RUNBOOK.md)).

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependencies
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
