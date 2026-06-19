# Integrations — Tunnel Manager

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

> Status: documentation-first (Phase 1). The integrations below are the
> **planned** dependency contracts; none are wired yet. Required/optional
> status reflects design intent. `service.json` declarations are made in
> Phase 2 (do not edit them from this doc).

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, all persistence-backed domains | `SQLITE_PATH` lifecycle env var; no external DB | API reports unhealthy if unreachable. |
| Vrooli lifecycle (`internal/lifecycle`) | local platform + seam | yes | API, UI, CLI; `exposure` ensure-running | `.vrooli/service.json`, Makefile targets; ensure-running seam | Start through lifecycle commands; ensure-running failure surfaces as an Expose error (TM does not reimplement lifecycle). |
| `cloudflared` daemon | host tool (required) | yes | `tunnel` (systemd + `/ready` + Prometheus), `config` (local mode), `recovery` (restart) | systemd unit + `/ready` + Prometheus endpoint (default `127.0.0.1:20241`); `systemctl` for restart | If down: `tunnel` reports unhealthy, `recovery` restarts it (single owner). TM does NOT install it (setup handles that). |
| Cloudflare API v4 | third-party service | remote mode only | `config` ingress push/sync | HTTPS API + account/tunnel id + API token (credential reference, not inlined) | On outage: classified as `cloudflare-outage`; `recovery` waits + alerts rather than restarting. Fallback to local config mode (P1). |
| `api-core/coreset` | local package seam | yes | `exposure` core reconcile | Queryable SSOT of core scenarios | If unavailable: skip-and-alert; never tear down existing CORE routes. |
| scenario `service.json` files | local files | yes | `audit` port compliance | Read each exposed scenario's fixed UI port | Missing/ranged/mismatched port → audit finding (does not crash TM). |
| `redis` | Vrooli resource | optional | UI real-time updates (pub/sub) | Declared **disabled** in `.vrooli/service.json` | Absent/disabled → UI falls back to HTTP polling. |

## Vrooli Resources

The generated template does not declare external Vrooli resources. Add
resources to `.vrooli/service.json` only when a real scenario domain
requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| `redis` | optional, declared disabled | UI pub/sub for real-time updates; UI falls back to HTTP polling when absent. Foundational infra must not hard-depend on it. | Enable if/when real-time UI updates outweigh the polling fallback. |
| SQLite (embedded) | required (not a Vrooli resource) | Self-contained store; must work when other resources are down (see [`../internal/DECISIONS.md`](../internal/DECISIONS.md)). | Revisit only if a domain truly needs a shared relational store. |
| Postgres | rejected | Old scenario wrongly required Postgres with an empty schema; foundational infra must avoid that dependency. | Revisit only if a domain truly needs a shared relational store. |

## Scenario Dependencies

These are runtime seams / contracts, not hard build-time dependencies.

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `vrooli-autoheal` | planned (contract) | Avoid dueling cloudflared restarters: TM is the single authoritative owner of cloudflared restart; autoheal downgrades its cloudflared check to **alert-only** and remains as defense-in-depth. | Autoheal alerts but does not restart cloudflared; flip happens in launch sequencing. |
| `app-monitor` | planned (consumer) | app-monitor's "open in new tab" calls TM's exposure-query (`IsExposed` / `ExposeAndGetURL`). The app-monitor-side change is a **separate task**, not this scenario. | TM exposes the query API; reverse proxy stays in `packages/api-base`, unchanged. |
| `internal/lifecycle` (seam) | planned | ensure-running delegation when exposing a scenario; TM does not own process management. | `exposure` calls the ensure-running seam. |
| `api-core/coreset` (seam) | planned | Core-tier membership SSOT. | `exposure` reads the core set during reconcile. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| Cloudflare API v4 | planned (remote mode) | Programmatic ingress management replaces the manual dashboard step (OT-P0-002). | HTTPS API + account/tunnel id + API token; token referenced via credential, never inlined in `tunnel_config`. |
| `cloudflared` daemon | planned (host tool) | The tunnel itself; health via systemd + `/ready` + Prometheus, restart via `systemctl`. | Required host tool; installed by setup, not by TM. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| `cloudflared` | `/ready` fail or HA-connections=0 | `tunnel` reports unhealthy; `recovery` restarts (backoff + circuit breaker). | Planned `tunnel` + `recovery` tests (Phase 2). |
| Cloudflare API v4 | API error / timeout | Classified `cloudflare-outage`; wait + alert, do not restart; local-mode fallback (P1). | Planned `config` + `probes` tests (Phase 2). |
| `api-core/coreset` | seam error / empty result | Skip-and-alert; never tear down existing CORE routes. | Planned `exposure` reconcile tests (Phase 2). |
| `internal/lifecycle` | ensure-running error | Expose returns a typed error; no route/ingress orphaned. | Planned `exposure` tests (Phase 2). |
| `service.json` (audited scenario) | missing/ranged/mismatched UI port | `audit` emits a compliance finding; TM keeps running. | Planned `audit` tests (Phase 2). |
| `redis` (optional) | unavailable/disabled | UI falls back to HTTP polling; no functional loss. | Planned UI tests (Phase 2). |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
