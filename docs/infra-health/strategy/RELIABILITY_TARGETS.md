# Reliability Targets

Targets the `infra-health` team measures itself against. Current state is captured separately from target so the gap is visible. Most current-state numbers will start as `pending-telemetry` and move to `measured` as the instrumentation roadmap closes.

This file is **not** a service-level commitment to external users. It is the team's internal yardstick for "is the platform getting more reliable over time."

## Honesty flags

Every number carries one:

- **`measured`** — read directly from autoheal / system-monitor / lifecycle / capacity stores
- **`estimate`** — derived from logs or process records (no canonical store yet)
- **`aspirational`** — the target we're tracking toward
- **`pending-baseline`** — the sensor exists but no baseline has been recorded yet; the first resume-protocol scan records it (bootstrap clause, no waiting period)
- **`pending-telemetry`** — no sensor exists yet; see [`../evidence/INSTRUMENTATION_ROADMAP.md`](../evidence/INSTRUMENTATION_ROADMAP.md)

`pending-telemetry` is derived, not hand-maintained: a target kind whose sensor cell in the sensor map below is empty is `pending-telemetry` by definition. When a sensor ships, the flag moves to `pending-baseline`; when a baseline is recorded, to `measured`.

## Sensor map

Every target kind names its **sensor** (the exact command that observes it), its **deadband** (the error band inside which no finding is raised), and its **actuator** (the decision context that fires when out of band). The scanner reads the sensor named here; it does not re-derive how to measure a target each heartbeat. Deadband values are initial operator choices — the contrarian's hysteresis rules govern changing them.

| Target kind | Sensor | Deadband (no finding while…) | Actuator |
|---|---|---|---|
| Scenario / resource uptime | `vrooli-autoheal actions trends --json` — per-check `uptimePercent`; checks map 1:1 to scenarios/resources. Never the event-weighted `actions uptime` aggregate (see unit rule below) | within 0.5pp below target, or excursion not sustained ≥24h; ghost/saturated/shelved checks excluded (see sensor integrity) | `runtime-health-finding` |
| Alarm-channel flood | `vrooli-autoheal actions uptime --json` — the event-weighted aggregate, re-purposed as what it actually measures: `criticalEvents` per 24h across the whole check registry | criticalEvents ≤ 500 / 24h (initial operator choice) | `runtime-health-finding` |
| Restart frequency | `vrooli-autoheal actions transitions --json` | target exceeded in fewer than 2 consecutive windows | `runtime-health-finding` |
| Cold-start / start / restart latency | — (roadmap Gap 2: `vrooli scenario stats`) | — | `instrumentation-gap` |
| Setup end-to-end time | — (roadmap Gap 6) | — | `instrumentation-gap` |
| Heal success rate | `vrooli-autoheal actions history --json` (outcome field; confirm field set on resume — Gap 7) | ≥ 92% (target 95%) | `runtime-health-finding` |
| Heal-loop incidence | `vrooli-autoheal check history <check>` (repeat pattern within 24h) | single-scenario, single-day occurrence | `runtime-health-finding` |
| Capacity claim coverage | `vrooli capacity reconcile --json` — UNCLAIMED / OVER_CLAIM rows are the error signal | one-off UNCLAIMED (same owner unclaimed across 2+ heartbeats is out of band) | `runtime-health-finding` |
| Declared-vs-observed usage drift | `vrooli capacity recommend --json` (granted reserve vs observed peak) | reserve ≤ 2× observed peak, or excursion not sustained ≥7d | `runtime-health-finding` or `reliability-target-update` |
| Supervised-set coverage | — (roadmap Gap 10 extension: `vrooli-autoheal check reconcile` both directions; interim estimate = manual diff of `check list` vs derived should-be-supervised set) | zero unsupervised members of the derived set | `runtime-health-finding` |
| Capability availability | — (roadmap Gap 11: per-owner derived aggregates — availability/coverage history queryable from each capability owner) | per-owner, once baselined | `instrumentation-gap` until Gap 11 ships; then `runtime-health-finding`, with repeated unabsorbed degradation escalating per operating-model rule 1 |
| Burst attribution | — (roadmap Gap 12: sustained host CPU/RAM saturation → owning scenario/run) | — | `instrumentation-gap` until shipped |
| Storage growth-slope | `storage-manager infra-health --json` — closed device census, growth slope, and `declared_ceiling_measured_coverage` | measured bytes under a declared ceiling; investigate sustained positive slope or falling coverage | `runtime-health-finding` |

## Sensor integrity

Alarm hygiene follows **ISA-18.2 / EEMUA 191** alarm-management discipline — imported concepts are stale alarms, chattering, flood, and shelving only; the full alarm-priority taxonomy and KPI suite are deliberately out of scope. A reading is evidence only if its check passes these rules:

- **Ghost check.** Every check must map to an existing plant element (scenario, resource, or host surface). A check whose target no longer exists is retired via a `runtime-health-finding`; its events never count as downtime and are excluded from every aggregate.
- **Saturated check.** A check pinned at a single status for a full 24h window carries no information — the *transition* is the signal, the repeat event is not. A saturated check converts to exactly one durable incident plus finding, then is shelved or retired; while saturated it stops contributing to uptime and flood aggregates.
- **Shelving with expiry.** A deliberately-stopped scenario (paused teams, decommissioning, maintenance) is shelved so its check does not count as degradation. Every shelf names a reason and an expiry; permanent suppression is prohibited, and an expired shelf reverts to live alarming. Until roadmap Gap 10 ships a shelve verb, shelves are recorded as rows in the team's runtime lessons artifact.
- **Unit rule.** The event-weighted `actions uptime` aggregate is never evidence for a per-scenario claim; it is the alarm-flood sensor only. Per-scenario evidence comes from per-check `actions trends`.

A degraded alarm channel is itself a supervised signal — the alarm-flood target below — not a reason to silently distrust all sensors.

## Critical scenarios

Critical means: autoheal restarts it if it dies, and downtime materially affects the operator's day. Initial set comes from autoheal's configured critical-scenario list.

| Scenario | Uptime target | Restart-frequency target | Cold-start latency target | Current state | Notes |
|---|---|---|---|---|---|
| `prompt-manager` | 99.5% / 30d (`aspirational`) | ≤ 2 / 7d (`aspirational`) | ≤ 10s (`aspirational`) | `pending-baseline`¹ | Source of truth for all team configs; downtime blocks every other team |
| `swarm-manager` | 99.5% / 30d (`aspirational`) | ≤ 2 / 7d (`aspirational`) | ≤ 10s (`aspirational`) | `pending-baseline`¹ | Backlog and initiative state |
| `agent-manager` | 99% / 30d (`aspirational`) | ≤ 5 / 7d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-baseline`¹ | Higher restart tolerance because of sandbox lifecycle churn |
| `vrooli-autoheal` | 99.9% / 30d (`aspirational`) | ≤ 1 / 30d (`aspirational`) | ≤ 5s (`aspirational`) | `pending-baseline`¹ | Tighter target because autoheal restarts everything else |
| `system-monitor` | 99% / 30d (`aspirational`) | ≤ 5 / 7d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-baseline`¹ | |
| `app-monitor` | 99% / 30d (`aspirational`) | ≤ 5 / 7d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-baseline`¹ | Cloudflared tunnel surface |

¹ Uptime and restart-frequency sensors exist (see sensor map); cold-start latency remains `pending-telemetry` (roadmap Gap 2).

## Resource targets

Resources are upstream of the scenarios that depend on them. Outage propagates.

| Resource | Uptime target | Recovery-time target | Current state | Notes |
|---|---|---|---|---|
| `postgres` | 99.9% / 30d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-baseline`² | Most scenarios depend on it |
| `redis` | 99.5% / 30d (`aspirational`) | ≤ 15s (`aspirational`) | `pending-baseline`² | |
| `qdrant` | 99% / 30d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-baseline`² | Vector store; degraded mode is acceptable in scenarios that fall back to keyword search |
| `ollama` | 95% / 30d (`aspirational`) | ≤ 60s (`aspirational`) | `pending-baseline`² | Lower target because cold model loads are expected |
| `cloudflared` | 99.5% / 30d (`aspirational`) | ≤ 30s (`aspirational`) | `pending-baseline`² | Critical for app-monitor public surface |

² Uptime sensor exists (see sensor map); recovery time is derivable from `vrooli-autoheal actions transitions` timestamps as an `estimate` until roadmap Gap 2 ships.

## Platform component targets

| Component | Target | Current state | Notes |
|---|---|---|---|
| `vrooli setup` end-to-end (clean machine) | ≤ 15 minutes from `make setup` to all critical scenarios green (`aspirational`) | `pending-telemetry` | Tier-1 install reproducibility |
| `vrooli scenario start <name>` median latency | ≤ 10s for non-resource scenarios (`aspirational`) | `pending-telemetry` | Excludes first-time builds |
| `vrooli scenario restart <name>` median latency | ≤ 15s for non-resource scenarios (`aspirational`) | `pending-telemetry` | |
| Heal success rate | ≥ 95% of triggered heals succeed without operator intervention (`aspirational`) | `pending-baseline` | Sensor: `vrooli-autoheal actions history` (see sensor map) |
| Heal-loop incidence | ≤ 1% of healed scenarios require ≥3 restarts in 24h (`aspirational`) | `pending-baseline` | High incidence indicates heal is masking root cause. Sensor: `vrooli-autoheal check history` |
| Alarm-channel flood | ≤ 500 critical events / 24h across the check registry (`aspirational`) | `measured` — 1,058 / 24h on 2026-07-24 (was 4,624 on 2026-07-23), still out of band but dominated by 24h-window tail from the 07-23 fixes | Sensor: `vrooli-autoheal actions uptime` (see sensor map). Decomposition + fix ledger in runtime lessons; steady-state residual is reboot-gated `host-*`/`system-mce-recent` saturation + `system-pm-runtime-hog` |

## Capacity arbitration targets

The capacity broker (`vrooli capacity`) arbitrates GPU/RAM/CPU between claimants: claims, priorities, degrade-before-preempt, idle unload, observed-usage tracking. These targets **supervise** the broker — the broker operates; infra-health checks that its coverage and honesty hold. Infra-health never changes policy levers or actuates degrade/preempt directly.

| Target | Value | Current state | Notes |
|---|---|---|---|
| GPU claim coverage | 100% of GPU-resident Vrooli processes hold an active claim (`aspirational`) | `pending-baseline` | Sensor: `vrooli capacity reconcile` — UNCLAIMED / OVER_CLAIM rows are the error signal |
| Declared-vs-observed honesty | granted reserve ≤ 2× observed peak per resident claim (`aspirational`) | `pending-baseline` | Sensor: `vrooli capacity recommend` — sustained over-reservation starves other claimants |
| Enforcement posture | `advisory` until an operator decision graduates it, with evidence (`measured` — policy lever) | `measured` | Premature `enforce=on` is the hazard, not the goal; graduation is a `reliability-target-update` decision |

## Supervised-set coverage

The autoheal check registry must cover the platform's load-bearing set. The ghost rule (sensor integrity above) catches checks whose plant element no longer exists; this target catches the symmetric failure — a plant element that exists and matters but holds no check. Both are directions of one reconcile diff.

**The should-be-supervised set is defined by derivation, never enumeration** (operating-model rule 6). It is computed fresh at reconcile time as:

- the core-set closure — `coreset.CoreSeedScenarios()` seed ∪ transitive Required-closure, as computed by scenario-dependency-analyzer; **plus**
- load-bearing declared capability members — scenarios whose own `.vrooli/` declaration marks them load-bearing for a capability owner (today: `test-genie.json` `policy.providerReadiness: required_when_applicable`; `search.json` gains an equivalent distinction via the Gap 11 contract note).

No list of member names appears in this file or anywhere in this PoR; a target or finding that enumerates members instead of naming the derivation is an automatic contrarian challenge.

| Target | Value | Current state | Notes |
|---|---|---|---|
| Supervised-set coverage | zero unsupervised members of the derived should-be-supervised set (`aspirational`) | `estimate` — 2026-07-24 manual diff of `check list` vs the derived should-be-supervised set: ~55 scenarios running, ~25 supervised, ~30 unsupervised; the unsupervised set includes core capability owners and their load-bearing members — the derivation (running set minus the seeded supervised-config list), not any roster, defines membership | Sensor: Gap 10 extension (`check reconcile`, both directions). Finding recorded in runtime lessons 2026-07-24; the supervised set today is a flat config list seeded once from the core seed and has drifted |

## Capability availability targets

Capability owners (search-hub, test-genie, prompt-manager, meta-optimization-manager, …) run their own scan → validate → aggregate loops over self-declared members. These targets supervise the owners' **derived aggregates** — infra-health checks the machinery works and the aggregate stays in band; it never tracks individual members, whose performance deadbands live in their own declarations (e.g. `search.json` `performance` block).

| Capability owner | Aggregate supervised | Target | Current state | Notes |
|---|---|---|---|---|
| search-hub | provider coverage % and degraded-member count over currently-registered providers | in band per owner-declared thresholds (`aspirational`) | `pending-telemetry` | Availability state is in-memory only today (circuit breaker + on-demand status probe) — Gap 11 |
| test-genie | phase runnability across the declared phase catalog (run / run_degraded / skip rates) | in band (`aspirational`) | `pending-telemetry` | The per-phase provider-readiness gate exists and is the model contract; a queryable aggregate/history surface is Gap 11 |
| meta-optimization-manager | projection availability (Answer / Validate / Guide owners reachable) | in band (`aspirational`) | `pending-telemetry` | Projections degrade honestly to UNAVAILABLE today but no history is kept — Gap 11 |

**Scope boundary.** Search-performance optimization, embedding centralization, provider-less search availability, and any other capability-architecture evolution are roadmap work for the capability owner or the meta-optimization team. Infra-health's entire role here is supplying the measured out-of-band aggregate that justifies such work — it proposes no architecture and names no solution.

## Update protocol

1. **Bootstrap.** The first `measured` reading on a `pending-baseline` row is recorded immediately — no waiting period. The 30-day rule governs changing *targets*, not recording *reality*.
2. **Tighten.** A target may tighten only after 30+ consecutive in-band days of `measured` data.
3. **Loosen.** A target may loosen only after sustained out-of-band `measured` data with a named non-temporary cause.
4. **Approval.** Every target change is a `reliability-target-update` decision approved by the operator at the morning vision walk, citing the decision id in the change line below.
5. **Actuation efficacy.** A finding that creates downstream work names its sensor and the expected in-band return. The first heartbeat after that work completes re-reads the sensor and records the result on the finding: returned in band, or did not. A fix that does not move the sensor re-opens the finding — the fix author does not grade the fix; the sensor does.

The tighten/loosen asymmetry is deliberate hysteresis: slow to tighten, evidence to loosen. It prevents targets from flapping with day-to-day noise. Deadbands in the sensor map follow the same protocol.

**Anti-windup rules:**

- A finding approved but unactuated (no downstream work created or completed) for 3 consecutive heartbeat cycles must either escalate or trigger a `reliability-target-update` — either the actuator fires harder or the setpoint was dishonest.
- A capability that ships without its [instrumentation roadmap](../evidence/INSTRUMENTATION_ROADMAP.md) gap entry and sensor-map cell being updated in the same cycle is an automatic `framework-meta` finding.

## Change log

- `2026-08-03` — Storage growth-slope shipped via storage-manager's closed device census and declared-ceiling coverage metric; Gap 12's storage half is now measured.

- `2026-07-24` (round 4) — Resume-readiness fixes (operator session, teams paused): de-enumerated the supervised-set coverage current-state cell — the unsupervised set is now named by derivation (running set minus the seeded supervised-config list) and counts, never a member roster, per the § Supervised-set coverage no-enumeration rule; changed the capability-availability sensor-map actuator to `instrumentation-gap` until Gap 11 ships (mirroring the burst-attribution / storage growth-slope rows), reverting to `runtime-health-finding` with escalation once the aggregate persists.
- `2026-07-24` (round 3) — Cascade completion (operator session, teams paused): added the supervised-set coverage target (symmetric twin of the ghost rule; should-be-supervised set defined by derivation — core-set closure ∪ load-bearing declared capability members — never enumeration) and capability availability targets (per-owner derived aggregates for search-hub / test-genie / meta-optimization-manager, all `pending-telemetry` pending Gap 11), with the contract-not-roster scope boundary. Alarm-flood current state updated to 1,058/24h (2026-07-24 re-read, tail-dominated). New sensor-map rows: supervised-set coverage, capability availability, burst attribution, storage growth-slope.
- `2026-07-23` (round 2) — Sensor-integrity hardening (operator session, teams paused): adopted ISA-18.2/EEMUA 191 alarm-management discipline (ghost / saturated / shelving-with-expiry / unit rule), moved the per-scenario uptime sensor from event-weighted `actions uptime` to per-check `actions trends`, re-purposed the aggregate as a new alarm-channel flood target (first `measured` baseline 4,624 crit/24h via bootstrap clause), and added the actuation-efficacy update-protocol rule.
- `2026-07-23` — Operator-curated control-loop hardening (operator session, teams paused): added sensor map (sensor / deadband / actuator per target kind), `pending-baseline` honesty flag, capacity arbitration targets, bootstrap clause, hysteresis-based update protocol, and anti-windup rules. Rows moved `pending-telemetry` → `pending-baseline` where sensors shipped (`vrooli-autoheal check history` / `actions uptime|transitions|history`, `vrooli capacity reconcile|recommend`).
- `2026-04-28` — File created with initial aspirational targets. All current-state values are `pending-telemetry` until INSTRUMENTATION_ROADMAP closes.
