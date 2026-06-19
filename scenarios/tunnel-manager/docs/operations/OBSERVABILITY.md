# Observability — Tunnel Manager

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

> **Status (Phase 1, documentation-first):** No product code exists yet
> beyond the template scaffold. Signals tagged **planned** describe the
> Phase 2 observability surface; only the scaffold lifecycle health
> checks exist today.

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API and dependency reachability | healthy for local development |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| test-genie result | validation | `make test` | scenario correctness evidence | all required phases pass |
| cloudflared `/ready` | health (planned) | cloudflared (via `tunnel` domain) | tunnel daemon readiness | `/ready` reachable |
| HA connections | metric (planned) | cloudflared Prometheus (`127.0.0.1:20241`) | tunnel connection health | degraded if `< 4`; failure at `0` |
| Tunnel RTT | metric (planned) | cloudflared Prometheus | latency to Cloudflare edge | degraded on spike |
| Request errors | metric (planned) | cloudflared Prometheus | tunnel-level error rate | alert on sustained rise |
| Active streams | metric (planned) | cloudflared Prometheus | live request volume | informational |
| Internal/external probe results | health | `probes` domain | per-route end-to-end reachability | route healthy when both probes pass |
| Degraded-mode signal | health (planned) | `tunnel` domain | early-warning before full failure | reported when HA `< 4` or RTT spikes |
| Recovery events | event | `recovery` domain | manual or opt-in background recovery attempts + outcomes | reviewed per incident |

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Structured logs (planned: failure classification, recovery actions). |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| cloudflared logs | systemd journal | `journalctl -u cloudflared` | Tunnel daemon logs; consulted during recovery (not produced by this scenario). |

## Stored Telemetry

All time-series and history are persisted in **SQLite** (DECISIONS:
"SQLite only"), co-located with the manifest under `SQLITE_PATH`. There
is no external metrics store — foundational infra keeps working when
other resources are down.

| Store | Domain / Table | Contents | Retention / Purge |
|---|---|---|---|
| Metrics history | `tunnel` / `metrics` | scraped HA conns, RTT, errors, active streams (time-series) | Bounded retention with periodic purge (window TBD in Phase 2). |
| Probe history | `probes` / `probes` | per-route internal/external probe results, latency, error | Bounded retention with periodic purge (window TBD). |
| Recovery event log | `recovery` / `recovery_events` | attempts, trigger, action, outcome, timestamps (OT-P1-005) | Retained for post-incident review. |

## Metrics

| Metric | Status | Details |
|---|---|---|
| Tunnel health (HA conns / RTT / errors / streams) | planned | Scraped from cloudflared Prometheus; persisted to SQLite (`tunnel` domain). |
| Per-route reachability | active | From probe history (`probes` domain); background probes run unless disabled. |
| Recovery success rate | active when events exist | Derived from the recovery event log; background event production requires `TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED`, while manual recovery always records events. |
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Product activation | deferred | Foundational infra; adoption measured via exposure manifest, not user activation. |
| Performance budgets | deferred | Define in `../internal/PERFORMANCE.md`. |

## Alerts / Health

The scaffold has lifecycle health checks for API and UI. The planned
`health`/`tunnel` domains expose tunnel readiness and degraded-mode
signals. Webhook/notification alerts (Slack/Discord/email) are PRD
**P2** (OT-P2-004) and deferred. `vrooli-autoheal` remains as alert-only
defense-in-depth for cloudflared (RUNBOOK: single-owner relationship).

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| All planned telemetry | No tunnel/probe/recovery signals until Phase 2. | Implement the `tunnel`, `probes`, and `recovery` domains. |
| Per-route analytics (volume/bandwidth) | Cannot attribute usage per route. | PRD OT-P2-007. |
| Cost telemetry | n/a — foundational infra, no hosted unit economics. | Only if a hosted tier is ever pursued. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
