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
| 2026-06-21 | **`SwitchMode` is PURE** — it persists the mode and performs ZERO ingress writes. | Apply-before-persist meant "switch to remote" silently overwrote the live tunnel with TM's manifest (the central footgun, old `service.go:203`). | Mode switches are safe; ingress is applied only by an explicit `Sync`. Behavioral break, acceptable (greenfield, no external consumers). Switching to remote still does a read-only credential resolve so the operator isn't stranded. | Revisit only if an operator needs switch-implies-apply (prefer an explicit follow-up sync). |
| 2026-06-21 | **`Sync` is additive by default; removal is explicit.** | A full-document PUT dropped any live ingress TM didn't author; there was no additive option and no per-entry control. | `applyAdditive` publishes desired merged onto current live (union), preserving unmanaged/ignored entries. `--prune` removes only orphaned (ledger-managed, route-gone) entries; per-hostname removal is `PruneIngress`. Unmanaged drift is NEVER auto-removed. | Revisit only if additive causes ingress bloat near the Cloudflare cap (tiering + prune already mitigate). |
| 2026-06-21 | **Ownership ledger (`ingress_ownership`) is authoritative** for managed/external/ignored classification; absence ⇒ UNMANAGED (safe). | TM had no persistent answer to "what's live vs. desired vs. unmanaged"; the add/remove diff was computed transiently and discarded. | New table keyed on full hostname; `reconcile()` classifies every hostname into one of MANAGED/MISSING/EXTERNAL_OK/ORPHANED/IGNORED/UNMANAGED. Drives `GetDrift` + adopt/ignore/prune. | Revisit if a non-hostname key is ever needed (e.g. path-based ingress). |
| 2026-06-21 | **External routes via a `source` field, not a new tier.** | Tiering is an exposure-policy axis; provenance (scenario vs external) is orthogonal. A `TIER_EXTERNAL` would conflate the two. | `Route.source ∈ {scenario, external}` + `service_target`; external routes skip the scenario/fixed-UI-port rule and forward to an arbitrary `http(s)` target. Full CRUD across proto/API/CLI/UI. | Revisit if external routes need their own tiering/lease semantics. |
| 2026-06-21 | **Adopt resolves scenario vs external by the live target.** | An unmanaged hostname may belong to a known scenario or be foreign. | `AdoptIngress` creates a scenario route when `--scenario` is given and the live service is `http://localhost:<port>` (port parsed from live); otherwise an external route pointing at `--target` or the live service. Ledger records MANAGED/EXTERNAL. No DNS writes. | Revisit if subdomain-based scenario auto-mapping is wanted (deliberately avoided to prevent coincidental mis-mapping). |
| 2026-06-21 | **Recovery scheduler is opt-OUT (default-on)**, env `TUNNEL_MANAGER_RECOVERY_SCHEDULER_DISABLED`. | The opt-in `*_ENABLED` form was a transitional guard during the cloudflared handover from `vrooli-autoheal`; it left actuation dark (recovery never fired) while probes were already opt-out. Nobody runs a tunnel control plane wanting it to watch the tunnel die without acting. | Recovery evaluates `/ready` every minute by default; symmetric with the probe/exposure schedulers. Old opt-in env name **deleted** (greenfield, no alias). | Revisit only if a deployment needs recovery off by default (then flip the env per-host). |
| 2026-08-18 | **Tunnel-presence self-gate on Vrooli-managed cloudflared resource registration**, checked at the top of every `Evaluate()`. | Default-on recovery must not flap on a tunnel-less host. Gating on live `/ready` would disable recovery exactly when it's needed (ready is false during an outage); resource presence is stable regardless of running state. | `UnitPresence` is implemented by the control-plane lifecycle seam: `vrooli resource status cloudflared --format json`; when absent the engine logs `recovery dormant`, stays idle, counts no failure, and attempts no restart. Picked up on the next tick when the resource is registered. | Revisit if a future supervisor exposes a different resource-status contract. |
| 2026-08-18 | **Managed resource lifecycle is the sole cloudflared restart owner.** | Tunnel-manager must not embed platform-specific service-manager or privilege mechanics. | `executeRecovery` requests `vrooli resource restart cloudflared`; the control plane owns platform-specific restart, authorization, and start-limit behavior. Recovery retains threshold, backoff, circuit-breaker, and readiness polling. | Revisit only if the control-plane resource contract changes. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| 2026-06-21 | **Auto-recovery scheduler opt-in (`TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED`, default-off)** — the 2026-06-18 "LIVE from day one" decision was wired as opt-in pending the autoheal handover. | Opt-out (default-on) `TUNNEL_MANAGER_RECOVERY_SCHEDULER_DISABLED` + managed cloudflared resource lifecycle. | The detection engine was always live; the background driver and restart ownership were later moved behind the control plane. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
