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

The integrations below are the current dependency contracts. Required
dependencies are wired through domain seams and lifecycle configuration;
optional dependencies are explicitly fallback-safe.

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, all persistence-backed domains | resolved by `api-core/storage` from the scenario id; no external DB | API reports unhealthy if unreachable. |
| Vrooli lifecycle (`internal/lifecycle`) | local platform + seam | yes | API, UI, CLI; `exposure` ensure-running | `.vrooli/service.json`, Makefile targets; ensure-running seam | Start through lifecycle commands; ensure-running failure surfaces as an Expose error (TM does not reimplement lifecycle). |
| `cloudflared` daemon | managed resource (required) | yes | `tunnel` (resource status + `/ready` + Prometheus), `config` (local mode), `recovery` (resource restart) | Vrooli-managed service exports its `/ready` and Prometheus endpoint; standalone fallback is `127.0.0.1:20241` | If down: `tunnel` reports unhealthy, `recovery` restarts it (single owner). TM does NOT install it (setup handles that). |
| Cloudflare API v4 | third-party service | remote mode only | `config` ingress push/sync | HTTPS API + account/tunnel id + API token (credential reference, not inlined) | Remote sync returns a typed setup/upstream error; local config mode remains available. Cloudflare-outage classification needs richer signals and is deferred. |
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
| `vrooli-autoheal` | contract | Avoid dueling cloudflared restarters: TM is the single authoritative owner of cloudflared restart; autoheal downgrades its cloudflared check to **alert-only** and remains as defense-in-depth. | Autoheal alerts but does not restart cloudflared; flip happens in launch sequencing. |
| `app-monitor` | consumer contract exposed by TM; app-monitor-side work separate | app-monitor's "open in new tab" can call TM's exposure-query (`IsExposed` / `ExposeAndGetURL`). The app-monitor-side change is a **separate task**, not this scenario. | TM exposes the query API; reverse proxy stays in `packages/api-base`, unchanged. |
| `internal/lifecycle` (seam) | implemented seam | ensure-running delegation when exposing a scenario; TM does not own process management. | `exposure` calls the ensure-running seam. |
| `api-core/coreset` (seam) | implemented seam | Core-tier membership SSOT. | `exposure` reads the core set during reconcile. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| Cloudflare API v4 | implemented for remote mode | Programmatic ingress management replaces the manual dashboard step (OT-P0-002). | HTTPS API + account/tunnel id + API token; credentials resolve through the Vrooli credential authority and are never inlined in `tunnel_config`. |
| `cloudflared` daemon | managed resource | The tunnel itself; health via resource status + `/ready` + Prometheus, restart through the control plane. | Required managed resource; installed by setup, not by TM. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| `cloudflared` | `/ready` fail or HA-connections=0 | `tunnel` reports unhealthy; `recovery` can restart with backoff + circuit breaker. Background recovery evaluation is opt-in. | `tunnel`, `recovery`, and scheduler tests. |
| Cloudflare API v4 | API error / timeout | Remote sync/reconcile reports a typed error and local-mode fallback remains available. | `config` production wiring tests and exposure module tests with fake Cloudflare ingress. |
| `api-core/coreset` | seam error / empty result | Reconcile reports failure without tearing down existing CORE routes. | `exposure` reconcile tests. |
| `internal/lifecycle` | ensure-running error | Expose returns a typed error; no route/ingress orphaned. | `exposure` service tests. |
| `service.json` (audited scenario) | missing/ranged/mismatched UI port | `audit` emits a compliance finding; TM keeps running. | `audit` service/handler tests. |
| `redis` (optional) | unavailable/disabled | UI falls back to HTTP polling; no functional loss. | UI tests cover polling client flows. |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
