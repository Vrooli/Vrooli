# Decisions — Tunnel Manager

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-18 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-06-18 | **Regenerate from react-vite + port logic** rather than migrate the old scenario in place. | The prior tunnel-manager predated template 1.0.0 (no `generation` block, REST/JSON, no proto, horizontal layers). | Clean Connect-RPC + screaming-architecture foundation; old logic ported from `/tmp/tunnel-manager-OLD-reference`. | n/a (foundational). Mirrors `scenario-authenticator` precedent. |
| 2026-06-18 | **SQLite only** (no Postgres). | Foundational infra must keep working when other resources are down; old scenario wrongly required Postgres with an empty schema. | Self-contained; manifest/leases/metrics/probes/recovery in SQLite. | Revisit only if a domain truly needs a shared relational store. |
| 2026-06-18 | Reframe as a **tiered exposure broker** (CORE always-on + LEASED on-demand TTL), per-route subdomains. | Operator currently adds Cloudflare public hostnames manually; hostname budget is finite; core scenarios must always be reachable. | New `exposure` domain; CF ingress automation promoted P1→P0. | Revisit tiers if Cloudflare cap proves very low or multi-tunnel arrives. |
| 2026-06-18 | **CORE tier = `packages/api-core/coreset`.** | A queryable SSOT of essential scenarios already exists and drives the self-improvement loop. | `exposure` reconciles coreset members as always-exposed. | Revisit if coreset semantics change or a tunnel-specific pin-list is needed. |
| 2026-06-18 | **LEASED tier**: TTL ≈1 week, request/extend/revoke, auto-reaped; requestable by the operator AND other scenarios via API. | "Expose me, I'll be used soon" use case. | `leases` table + exposure-request API (OT-P0-005). | Revisit default TTL after real usage. |
| 2026-06-18 | **Ensure-running by delegation** to `internal/lifecycle`; do NOT reimplement scenario lifecycle. | PRD non-goal: TM monitors/exposes, it does not own process management. | `exposure` calls the lifecycle seam; keeps blast radius small. | Revisit if lifecycle seam is insufficient. |
| 2026-06-18 | **Auto-recovery LIVE from day one**; TM is the single authoritative owner of cloudflared restart. | Operator chose immediate self-healing over a monitor-only soak. | `recovery` acts live (backoff + circuit breaker); `vrooli-autoheal`'s cloudflared check must downgrade to alert-only to avoid dueling restarts. | Revisit if false-positive restarts occur; tighten circuit breaker. |
| 2026-06-18 | **Reverse proxy stays in `packages/api-base`, unchanged.** Only app-monitor's "open in new tab" integrates with TM. | app-monitor's iframe proxy is the main viewing path and works; TM should not absorb proxying. | TM exposes `IsExposed`/`ExposeAndGetURL`; the app-monitor consumer change is a separate later task (not this scenario). | Revisit if the proxy is deprecated in favor of always-tunnel. |
| 2026-06-18 | **`domain` is a manifest field** (default `itsagitime.com`), not a hardcoded constant. | Old code hardcoded `.vrooli.com`; real domain is `.itsagitime.com`. | Fixes a latent bug; enables multi-domain (P2). | n/a. |
| 2026-06-18 | **UI reimagined as 5 surfaces** (Overview / Exposure / Recovery & Events / Metrics / Audit). | Old UI sprawled to 7 score-chasing pages contradicting the PRD's "minimal UI". | Same capabilities, organized around the exposure lifecycle. | Revisit per operator feedback. |
| 2026-06-18 | **Fixed UI port 21240** for tunnel-manager itself. | It enforces the fixed-UI-port contract on others, so it must obey it. | `service.json` ports.ui pinned. | Revisit only on port-band changes. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
