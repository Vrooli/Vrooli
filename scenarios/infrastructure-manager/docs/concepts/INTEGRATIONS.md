# Integrations — Infrastructure Manager

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

**Every scenario dependency here is a read.** This scenario invokes no
mutating verb on any dependency — no restart, no shelve, no reconcile-and-fix,
no policy change. That is not a current limitation; it is the
"supervise, don't operate" boundary from
`docs/infra-health/operating/OPERATING_MODEL.md` rule 3, and an extension
that breaks it is an architectural defect rather than a feature.

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, `readings`, `focus` | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| Setpoint document | upstream plan of record | yes | `targets` | `docs/infra-health/strategy/RELIABILITY_TARGETS.md` § Sensor map | Unparseable setpoint is a hard, loud failure — the board has nothing to measure against and must say so rather than report an empty map as zero targets. |
| `vrooli-autoheal` | scenario (read-only) | no | `readings`, `supervision` | `actions trends\|uptime\|transitions\|history`, `check history\|shelved\|reconcile`, `incidents latest` | Per-source `UNAVAILABLE` with stated reason. Five targets lose their reading; the rest keep ranking. |
| `vrooli capacity` | control plane (read-only) | no | `readings` | `capacity reconcile`, `capacity recommend` | Per-source `UNAVAILABLE`. Two targets lose their reading. |
| `storage-manager` | scenario (read-only) | no | `readings` | `storage-manager infra-health --json` | Per-source `UNAVAILABLE`. One target loses its reading. |
| `test-genie` | scenario (read-only) | no | `readings` | `test-genie runs cost --window 7d --json` | Per-source `UNAVAILABLE`. One target loses its reading. |
| `system-monitor` | scenario (read-only) | no | `readings` | `system-monitor process-timeline`, `investigations` | Per-source `UNAVAILABLE`. Burst attribution loses its reading. |
| `data-backup-manager` | scenario (read-only) | no | `readings` | `status`, `coverage`, `drills`, `audits` | Per-source `UNAVAILABLE`. Backup targets lose their reading. |
| `agent-manager` | scenario (read-only) | no | `readings` | `runs stats`, `error-patterns` | Per-source `UNAVAILABLE`. Spawn-success target loses its reading. |
| `scenario-dependency-analyzer` | scenario (read-only) | no | `supervision` | core-set closure query | Supervision reconcile degrades to `UNAVAILABLE` — it must never fall back to an enumerated roster. |

## Vrooli Resources

No external Vrooli resources are declared. Storage is embedded SQLite via
`api-core/storage`; every other input is a read against a scenario or the
control plane.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None. | not-applicable | SQLite is embedded; the scenario holds no queue, cache, or vector state. | A reading volume that outgrows embedded SQLite, or a genuine need for cross-host reading history. |

## Scenario Dependencies

All read-only, all optional, all degrading independently. "Optional" here
means *the board still functions without it and says so* — never that the
reading is silently dropped.

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `vrooli-autoheal` | active | The primary sensor source: 5 of 9 currently-sensored targets, plus the check registry that `supervision` reconciles and two of four trust rules. | Read-only CLI/RPC. Never `check shelve`, `unshelve`, `retire`, or any remediation verb. |
| `vrooli capacity` | active | Claim coverage and declared-vs-observed reserve drift. | Read-only. Never a policy-lever change, degrade, preempt, or release. |
| `storage-manager` | active | Device census, growth slope, declared-ceiling coverage. | `infra-health --json`. Read-only; never triggers cleanup. |
| `test-genie` | active | Validation cost and cache reliability aggregate. | `runs cost`. Phase semantics stay with test-genie. |
| `system-monitor` | active | Process attribution over a saturation window, grouped by source scenario; investigation records. | Read-only; never triggers an investigation. |
| `data-backup-manager` | planned | Backup freshness, coverage, and verified-restore drill outcomes — a target row this team has never written despite the sensor already shipping. | Read-only; never runs a backup or a restore. |
| `agent-manager` | planned | Run statistics and durable error patterns, for the spawn-success target. | Read-only. |
| `scenario-dependency-analyzer` | planned | The core-set closure half of the derived should-be-supervised set. | Read-only derivation query, executed per read. |
| `secrets-manager` | blocked | Secrets availability is load-bearing for scenario start and has no target row. The scenario ships a `cli/` directory but **no `cli/manifest.json`**, so it is unbindable and exposes no typed read. | Blocked until it grows a health/status verb and a manifest. Tracked as an instrumentation gap, not compensated for here. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | Every input is local. The scenario makes no outbound network call and needs no credential. | Would require a PRD change; there is no foreseen need. |

## Failure Modes

The governing rule: **a source that cannot be read produces a visible
availability entry with a stated reason — never a zero, never a silently
dropped row, and never a fabricated healthy verdict.** An unreadable
sensor is a finding about the instrument, not evidence about the plant.

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| Setpoint document | missing, unreadable, or unparseable | Hard failure with a stated reason. The board refuses to report a target count it could not read. | `targets` parser tests |
| Setpoint document | parses, but a row lacks a sensor / deadband / actuator | Integrity finding on that target; the rest of the map still loads. | `targets` integrity tests |
| Any sensor source | timeout past `readDeadline` (3s) | That target's reading is `UNAVAILABLE` with the reason verbatim. Other targets keep their readings. | per-source client tests with fakes |
| Any sensor source | process not running | Same as timeout — `UNAVAILABLE`, reason stated. Never reported as in-band. | per-source client tests |
| `scenario-dependency-analyzer` | unreachable | `supervision` reports `UNAVAILABLE`. **Never** falls back to a cached or enumerated member list. | supervision seam tests |
| `vrooli-autoheal` check registry | ghost or saturated checks present | Readings are marked untrusted per [`TRUST-MODEL.md`](TRUST-MODEL.md) and excluded from aggregates; a finding routes to the instrument. | trust verdict tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
