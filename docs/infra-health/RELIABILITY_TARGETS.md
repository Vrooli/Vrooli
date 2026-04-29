# Reliability Targets

Targets the `infra-health` team measures itself against. Current state is captured separately from target so the gap is visible. Most current-state numbers will start as `pending-telemetry` and move to `measured` as the instrumentation roadmap closes.

This file is **not** a service-level commitment to external users. It is the team's internal yardstick for "is the platform getting more reliable over time."

## Honesty flags

Every number carries one:

- **`measured`** — read directly from autoheal / system-monitor / lifecycle stores
- **`estimate`** — derived from logs or process records (no canonical store yet)
- **`aspirational`** — the target we're tracking toward
- **`pending-telemetry`** — the stat doesn't exist yet; see [INSTRUMENTATION_ROADMAP.md](INSTRUMENTATION_ROADMAP.md)

## Critical scenarios

Critical means: autoheal restarts it if it dies, and downtime materially affects the operator's day. Initial set comes from autoheal's configured critical-scenario list.

| Scenario | Uptime target | Restart-frequency target | Cold-start latency target | Current state | Notes |
|---|---|---|---|---|---|
| `prompt-manager` | 99.5% / 30d (`aspirational`) | ≤ 2 / 7d (`aspirational`) | ≤ 10s (`aspirational`) | `pending-telemetry` | Source of truth for all team configs; downtime blocks every other team |
| `swarm-manager` | 99.5% / 30d (`aspirational`) | ≤ 2 / 7d (`aspirational`) | ≤ 10s (`aspirational`) | `pending-telemetry` | Backlog and initiative state |
| `agent-manager` | 99% / 30d (`aspirational`) | ≤ 5 / 7d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-telemetry` | Higher restart tolerance because of sandbox lifecycle churn |
| `vrooli-autoheal` | 99.9% / 30d (`aspirational`) | ≤ 1 / 30d (`aspirational`) | ≤ 5s (`aspirational`) | `pending-telemetry` | Tighter target because autoheal restarts everything else |
| `system-monitor` | 99% / 30d (`aspirational`) | ≤ 5 / 7d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-telemetry` | |
| `app-monitor` | 99% / 30d (`aspirational`) | ≤ 5 / 7d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-telemetry` | Cloudflared tunnel surface |

## Resource targets

Resources are upstream of the scenarios that depend on them. Outage propagates.

| Resource | Uptime target | Recovery-time target | Current state | Notes |
|---|---|---|---|---|
| `postgres` | 99.9% / 30d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-telemetry` | Most scenarios depend on it |
| `redis` | 99.5% / 30d (`aspirational`) | ≤ 15s (`aspirational`) | `pending-telemetry` | |
| `qdrant` | 99% / 30d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-telemetry` | Vector store; degraded mode is acceptable in scenarios that fall back to keyword search |
| `ollama` | 95% / 30d (`aspirational`) | ≤ 60s (`aspirational`) | `pending-telemetry` | Lower target because cold model loads are expected |
| `cloudflared` | 99.5% / 30d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-telemetry` | Critical for app-monitor public surface |

## Platform component targets

| Component | Target | Current state | Notes |
|---|---|---|---|
| `vrooli setup` end-to-end (clean machine) | ≤ 15 minutes from `make setup` to all critical scenarios green (`aspirational`) | `pending-telemetry` | Tier-1 install reproducibility |
| `vrooli scenario start <name>` median latency | ≤ 10s for non-resource scenarios (`aspirational`) | `pending-telemetry` | Excludes first-time builds |
| `vrooli scenario restart <name>` median latency | ≤ 15s for non-resource scenarios (`aspirational`) | `pending-telemetry` | |
| Heal success rate | ≥ 95% of triggered heals succeed without operator intervention (`aspirational`) | `pending-telemetry` | |
| Heal-loop incidence | ≤ 1% of healed scenarios require ≥3 restarts in 24h (`aspirational`) | `pending-telemetry` | High incidence indicates heal is masking root cause |

## Update protocol

Targets change when:
1. The team has 30+ days of `measured` data showing the current target is consistently met (tighten) or consistently missed for non-temporary reasons (loosen + flag the underlying cause).
2. A `reliability-target-update` decision is approved by the operator at the morning vision walk.

Approved updates cite the decision id in the change line below.

## Change log

- `2026-04-28` — File created with initial aspirational targets. All current-state values are `pending-telemetry` until INSTRUMENTATION_ROADMAP closes.
