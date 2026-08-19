# Setpoint Model

## Purpose Of This Document

This is the **single canonical source** for what this scenario measures against: the **target model**, where the setpoint lives, why this scenario may not author it, and how the five axes stay separate.

Its sibling [`TRUST-MODEL.md`](TRUST-MODEL.md) owns the Trust axis — whether a reading can be believed at all. Read this file first; that one builds on the target model defined here.

The upstream contract this document instantiates is `docs/agent-system/TARGET_MODEL.md`. Read that as the general shape and this as **one worked instance of it**, exactly as `meta-optimization-manager`'s `COVERAGE-MODEL.md` is a worked instance rather than a second contract.

## The Instrument Contract, In One Sentence

> A team is a control loop: it regulates one plant against a setpoint it does not own, using one instrument it does not decide with, and it gets simpler as the system around it gets more capable.
> — `docs/agent-system/TARGET_MODEL.md` § 1

This scenario is the *instrument* half of that sentence for the `infra-health` team. Every design decision below follows from the two clauses in the middle: **a setpoint it does not own**, and **an instrument it does not decide with**.

## The Plant

The plant is the layered platform stack defined in `docs/infra-health/operating/OPERATING_MODEL.md` § "Platform Under Control". It is **not** the codebase, and it is **not** the `*-health` scenario fleet — those are test-genie phase providers validating code, which is `scenario-qa`'s plant.

| Layer | Timescale | This scenario's relationship |
|---|---|---|
| Commissioning — `vrooli setup`, host tools, safeguards | per host change | Observes outcomes; never invokes. Sudo exists only here. |
| Capacity broker — `vrooli capacity` | ms–minutes | Reads claim coverage and reserve drift. Never changes a policy lever. |
| Autoheal — `vrooli-autoheal` | seconds–minutes | Primary sensor source. Reads uptime, restarts, heal outcomes, the check registry. Never shelves, restarts, or reconciles. |
| System-monitor | minutes–hours | Reads process attribution and investigations. |
| Capability owners | per query / run | Reads derived aggregates only, never members. |
| **Infra-health (this instrument's team)** | heartbeats–days | The outermost, slowest loop. This scenario is its board. |

**Supervise, don't operate.** Operating-model rule 3 is absolute for this scenario: no restarts, no policy-lever changes, no degrade or preempt, no privileged mutation. Instrumentation encodes the same idea in tag letters — `FT` is a flow transmitter, `FIC` is a flow indicating *controller*, and the `C` is what confers the right to act. This scenario has no `C`.

## The Target Model

A **target** is one thing the team intends to control. It is the unit of the setpoint, and it is authored upstream.

| Field | Meaning | Owner |
|---|---|---|
| `id` | Stable identifier for the target kind | the setpoint document |
| `sensor` | The exact command that observes it — or an explicitly empty cell | the setpoint document |
| `deadband` | The band inside which no finding is raised | the operator |
| `actuator` | The work type that fires when out of band | the setpoint document |
| `honesty_flag` | `measured` · `estimate` · `aspirational` · `pending-baseline` · `pending-telemetry` | derived from the sensor cell |
| `gap_opened_on` / `gap_open_days` | When an empty sensor cell was declared, and how long it has stood | derived from the gap marker |

### An empty sensor cell is a first-class state

This is the property that makes the setpoint measurable rather than merely aspirational. A target with no sensor is **open-loop and honest about it** — it is not missing data, it is declared blindness. The scenario counts these, dates them, and reports the count as a target of its own (`OT-P0-005`), following the same self-counting discipline as `FRAMEWORK_HEALTH.md`'s open-loop target count.

The two derived date fields are what let an *intentionally visible* hole be told apart from an *overdue* one. Without them, "declared and dated" degrades into "declared once and forgotten."

## Where The Setpoint Lives, And Why Not Here

**The setpoint is `docs/infra-health/strategy/RELIABILITY_TARGETS.md` § Sensor map.** This scenario reads it and never writes it.

> **An observer that writes its own reference model is confirming itself.**

`TARGET_MODEL.md` files the violation as deviation `D6` and calls it an architectural correction rather than a documentation fix. Concretely: if this scenario could author a deadband, it could report itself in band by lowering the bar, and no reader could tell the difference from real improvement.

### The setpoint has two halves with different owners

This is a genuine divergence from `meta-optimization-manager`, where every denominator lives with a capability owner. Here the setpoint splits:

| Half | Owner | Why not the instrument |
|---|---|---|
| **Targets and deadbands** | the infra-health team and the operator, in the plan of record | These *are* operator judgment. They change only through an approved `reliability-target-update` decision at the morning vision walk, under the hysteresis rules in that document's update protocol. |
| **The supervised population** | derived at read time — core-set closure from `scenario-dependency-analyzer` ∪ load-bearing declared capability members | Operating-model rule 6 forbids enumeration anywhere in this team's surfaces. A cached roster here would be the central capability-health scenario that `INSTRUMENTATION_ROADMAP.md` Gap 11 explicitly rejects. |

Both halves being external is what keeps the observer honest in both directions: it cannot lower the bar, and it cannot quietly decide who is being graded.

### Clearing the Gap 11 objection, explicitly

`INSTRUMENTATION_ROADMAP.md` Gap 11 states: *"No central capability-health scenario — that would recreate the roster this PoR forbids."* That objection is correct and this scenario is designed to satisfy it rather than to work around it:

- It holds **no roster**. Every set is a derivation query executed at read time.
- It reads capability owners' **derived aggregates only**, never their members.
- It computes **nothing** about a member's health; that stays with the owner.

What Gap 11 rejects is a scenario that would own membership and grade members. What rule 6 asks for is exactly what an instrument does. `DESIGN.md` and [`ARCHITECTURE.md`](ARCHITECTURE.md) restate this boundary as an extension rule so a future change cannot erode it silently.

## Deadband Discipline

Two rules are inherited verbatim from upstream canon and are enforceable properties of this scenario, not advice:

**A deadband states the target, not the current reading.** A band set equal to whatever the sensor last observed reports in-band while the defect stands, and can only ever detect growth — never the standing problem. `FRAMEWORK_HEALTH.md` § "Deadband rule" records two targets that carried that shape and both read `ok` for it.

**A deadband may state a direction instead of a level.** Some targets band on a trend across cycles rather than an absolute value. A single reading cannot decide those, and the scenario must report them as *needing a prior baseline* rather than silently passing.

## The Five Axes

Coverage alone is a map, and a map says nothing about whether the roads it draws are still open. Each axis answers a question the others structurally cannot express.

| Axis | Question | Shape | Owned by |
|---|---|---|---|
| **Instrument coverage** | How much of the setpoint can we observe at all? | `sensored / authored` targets, plus the dated open-loop set | `targets` domain |
| **Supervision coverage** | Is every element that should be watched actually watched? | two-direction reconcile diff | `supervision` domain |
| **Trust** | Can this reading be believed, or is it instrument fault? | closed-vocabulary verdict per reading | [`TRUST-MODEL.md`](TRUST-MODEL.md) |
| **Condition** | Is the supervised element in band against its deadband? | in-band / out-of-band per target | `readings` domain |
| **Actuation efficacy** | Did the fix actually return the sensor to band? | per-finding re-read verdict | `focus` domain |

**They are never merged.** Folding trust into condition means an instrument defect gets scheduled as plant work. Folding efficacy into condition means a fix that never worked reads as a fix that worked once. Each stays a separate number on one surface.

### The two coverage axes are not the same question

They are adjacent enough to be confused, and conflating them would hide the more fundamental of the two. **Supervision coverage is plant-side**: of the elements that should be watched, how many hold a check? **Instrument coverage is setpoint-side**: of the targets the team has authored, how many can be read at all? A platform could be perfectly supervised and still be mostly unmeasurable, because five of fourteen targets name no sensor.

Instrument coverage is the axis this scenario reports about *itself*, which is why it comes first: every other number on the board is scoped by it. A condition figure computed over nine targets means something different when the reader knows fourteen were authored, and the ratio is the only thing that carries that. It is served by `OT-P0-005` and is the direct analogue of the open-loop target count in `FRAMEWORK_HEALTH.md`, including the property that it counts its own blind spots.

### Actuation efficacy is the axis nothing else in the fleet has

Update-protocol rule 5 of `RELIABILITY_TARGETS.md` already states it exactly:

> A finding that creates downstream work names its sensor and the expected in-band return. The first heartbeat after that work completes re-reads the sensor and records the result on the finding: returned in band, or did not. **A fix that does not move the sensor re-opens the finding — the fix author does not grade the fix; the sensor does.**

Today that is a sentence in a document with no mechanism behind it. Making it mechanical is `OT-P1-003`, and it is the strongest single defence this scenario offers against reliability theatre.

## What Stays Judgment

Explicitly **not** measured here, and not to be turned into a metric later:

- **Whether the target list is the right list.** That is the obligation-naming step, and `TARGET_MODEL.md` § 11 says plainly that the honest way to express uncertainty about it is denominator-confidence, not a score.
- **Whether to repair, defer, adopt, or retire.** The board ranks; the team and operator decide.
- **Platform-code quality and cross-platform debt.** Judgment-shaped, owned by the `platform-code-auditor` lane. See [`DOMAINS.md`](DOMAINS.md) § Deferred Domains.

## Denominator Confidence

Every ratio this scenario reports carries the confidence of the setpoint it measured against — `AUTHORITATIVE`, `PARTIAL`, or `SKETCH`, plus a rationale. The honesty is recursive: a reader sees "X of Y targets in band, against a `PARTIAL` setpoint," so the board structurally cannot imply completeness it has not earned.

**The setpoint's current confidence is `SKETCH`, and the reason is specific.** The 14 authored target kinds are at *operation* granularity. The **obligation** list they should derive from — what the infra-health team must be able to do, from which the targets follow — has never been written. `TARGET_MODEL.md` § 4 is direct about this: *"a denominator built from obligations alone can never rise above `sketch` confidence,"* and conflating obligation with operation is the most common modelling error. Naming those obligations is the work that lifts this scenario's confidence, and it is judgment that must be done by the team, not derived here.

## Current State

Recorded as of 2026-08-17, as data.

| Fact | Value |
|---|---|
| Authored target kinds in the sensor map | 14 |
| Targets with a live sensor | 9 |
| Targets that are open-loop (empty sensor cell) | 5 |
| Distinct sensor sources in use | 4 — `vrooli-autoheal` (5), `vrooli capacity` (2), `storage-manager` (1), `test-genie` (1) |
| Targets proposed for addition | 7 — three of which need no new telemetry, only a target row |
| Setpoint confidence | `SKETCH` — no obligation list has been authored |
| Open-loop count reported anywhere today | none; nothing counts them |
| Team loop status | `paused-manual` since 2026-07-24 |

## Governing Principles

- **The setpoint is owned elsewhere, in both halves.** Deadbands are operator judgment; the population is a derivation. Neither is authored here.
- **An empty sensor cell is a declared state, not missing data.** It is counted, dated, and aged.
- **A deadband states the target, never the current reading.**
- **Axes stay separate.** Instrument coverage, supervision, trust, condition, and efficacy are five numbers, not one score. The first scopes the other four: every figure on the board is read against how much of the setpoint is observable at all.
- **Surfaces, does not decide.** The board ranks candidates and states confidence; every decision and every actuation stays outside this scenario.

## Cross-References

- [`TRUST-MODEL.md`](TRUST-MODEL.md) — the Trust axis in full.
- [`DOMAINS.md`](DOMAINS.md), [`ARCHITECTURE.md`](ARCHITECTURE.md), [`DATA.md`](DATA.md), [`INTEGRATIONS.md`](INTEGRATIONS.md).
- `docs/agent-system/TARGET_MODEL.md` — the instrument contract this document instantiates.
- `docs/infra-health/strategy/RELIABILITY_TARGETS.md` — the setpoint.
- `docs/concepts/RECURSIVE_SELF_IMPROVEMENT.md` § Control topology — how this loop sits beside meta-optimization.
- `scenarios/meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md` — the sibling worked instance of the same upstream contract.
