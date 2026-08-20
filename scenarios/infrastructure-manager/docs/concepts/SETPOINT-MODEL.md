# Setpoint Model

## Purpose Of This Document

This is the **single canonical source** for the **setpoint**: the bar every cell is graded against, where it lives, why it lives here, and what structurally prevents this scenario from moving it.

It is the third of four model documents. [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) owns the projections and the cell grid; [`CONDITION-MODEL.md`](CONDITION-MODEL.md) owns band *evaluation*; this document owns the band *values*. [`TRUST-MODEL.md`](TRUST-MODEL.md) owns whether a reading is admissible in the first place.

## What A Setpoint Entry Is

The setpoint is a set of **bars**, one per cell, expressed as data.

| Field | Meaning | Example |
|---|---|---|
| `cell_ref` | The projection and cell this bar grades | `availability/prompt-manager/uptime-30d` |
| `deadband` | The band inside which no finding is raised | `>= 99.5% over 30d` |
| `sustain` | How long an excursion must persist before it is a finding | `24h` |
| `actuator` | The work type that fires when out of band | `runtime-health-finding` |
| `honesty_flag` | `measured` · `estimate` · `aspirational` · `pending-baseline` · `pending-telemetry` | `aspirational` |
| `decision_ref` | The approved decision that set or last changed this bar | `reliability-target-update:2026-07-23-round-2` |

`honesty_flag` is **derived, never hand-maintained**: a cell whose coverage status is `MISSING` is `pending-telemetry` by definition; when a sensor ships it becomes `pending-baseline`; when a baseline is recorded it becomes `measured`. A hand-set honesty flag is an integrity finding.

## Where The Setpoint Lives

**The setpoint is a checked-in declarative file inside this scenario, parsed at query time.**

This replaces the arrangement in which it lived in the team's plan of record at `docs/infra-health/strategy/RELIABILITY_TARGETS.md`. That location was residue from the instrument not existing yet: the same way `meta-optimization-manager` holds its own models rather than leaving them in `docs/meta-optimization/`, the reliability model belongs with the instrument that computes it. The team plan of record keeps its operating model and governance, and points here.

### Why a file and not a table

The obvious alternative — targets in SQLite, editable from the board — was rejected, and the reason is the whole point of the axis:

> **An observer that can write its own reference model is confirming itself.**

`TARGET_MODEL.md` files this as deviation `D6`. Concretely: if this scenario could author a deadband, it could report itself in band by lowering the bar, and no reader could tell the difference from real improvement.

A checked-in file makes that impossible **by construction rather than by policy**:

| Property | Consequence |
|---|---|
| No API path writes it | There is no endpoint to misuse, no permission to get wrong, and no code path to review for D6 |
| Changing a bar is a code change | It goes through diff review like any other change, and the reviewer sees the before and after |
| Git is the audit trail | `decision_ref` and the commit are two independent records of the same change |
| It is read per query, never cached | The board cannot grade itself against a bar the operator has already changed |

The rejected design is not hypothetical — it is the natural thing to build once the board is polished and someone wants to tune a deadband without opening an editor. The prohibition is recorded in [`../internal/DECISIONS.md`](../internal/DECISIONS.md) so it is not relitigated by convenience.

### The half that is not here

The setpoint is only the *bar*. The *space* — which cells exist at all — lives with each control layer in its own `docs/spaces/<projection>-space.md`, read through that owner's `space --projection <p> --json` verb.

> **Owners define the space. The operator defines the bar. Neither can move the other, and this scenario can move neither.**

See [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) § The split that keeps the observer honest for why the halves are separated this way.

## Deadband Discipline

Two rules are inherited verbatim from upstream canon and are enforceable properties, not advice.

**A deadband states the target, not the current reading.** A band set equal to whatever the sensor last observed reports in-band while the defect stands, and can only ever detect *growth* — never the standing problem. `FRAMEWORK_HEALTH.md` § "Deadband rule" records two targets that carried that shape, and both read `ok` for it. A bar whose value equals the current measurement at authoring time is an integrity finding.

**A deadband may state a direction instead of a level.** Some cells band on a trend across cycles rather than an absolute value. A single reading cannot decide those, and they must be reported as `NEEDS_BASELINE` rather than silently passing. See [`CONDITION-MODEL.md`](CONDITION-MODEL.md) § Banding.

## The Update Protocol

Changing a bar is governed, and the asymmetry is deliberate hysteresis: **slow to tighten, evidence to loosen.** It prevents targets flapping with day-to-day noise.

1. **Bootstrap.** The first `measured` reading on a `pending-baseline` cell is recorded immediately — no waiting period. The 30-day rule governs changing *targets*, not recording *reality*.
2. **Tighten.** A bar may tighten only after 30+ consecutive in-band days of `measured` data.
3. **Loosen.** A bar may loosen only after sustained out-of-band `measured` data with a named non-temporary cause.
4. **Approval.** Every change is a `reliability-target-update` decision approved by the operator, and the entry carries its `decision_ref`.
5. **Actuation efficacy.** A finding that creates downstream work names its sensor and expected in-band return; the sensor grades the fix. Mechanized in [`CONDITION-MODEL.md`](CONDITION-MODEL.md) § Actuation efficacy.

### Anti-windup

- A finding approved but unactuated for **3 consecutive heartbeat cycles** must either escalate or trigger a `reliability-target-update` — either the actuator fires harder or the bar was dishonest.
- A capability that ships without its coverage cell and bar being updated in the same cycle is an automatic `framework-meta` finding.

## Setpoint Integrity

The setpoint is parsed, and a parse that succeeds is not the same as a setpoint that is sound. These checks run on every read and produce findings against the *instrument*, never the plant:

| Check | Failure means |
|---|---|
| Every entry resolves to a cell in a live projection | The bar grades something that does not exist |
| Every cell with coverage `NOW` has a bar | A cell is instrumented and ungraded — readings flow and nothing evaluates them |
| No bar equals the current reading at authoring time | Dead deadband |
| Every `honesty_flag` is derived, not hand-set | A hand-set flag can claim `measured` with no measurement |
| Every changed bar carries a `decision_ref` | An ungoverned change to the bar |
| The retention floor covers the longest declared window | A windowed target that cannot be evaluated from stored history |

An unparseable setpoint is a **hard, loud failure**. The board has nothing to measure against and must say so, rather than reporting an empty map as zero targets in band.

## Denominator Confidence

Every ratio carries the confidence of the space it measured against — `AUTHORITATIVE`, `PARTIAL`, or `SKETCH`, plus a rationale.

**Confidence is `SKETCH` today, and the reason is specific.** The authored targets sit at *operation* granularity. The **obligation** list they should derive from — what the infra-health team must be able to do, from which the targets follow — has never been written. `TARGET_MODEL.md` § 4 is direct: *"a denominator built from obligations alone can never rise above `sketch` confidence"*, and conflating obligation with operation is the most common modelling error.

Naming those obligations is what lifts confidence. It is judgment owned by the team and the operator; it is not derivable here, and no amount of implementation raises confidence without it.

## What Stays Judgment

- **Whether the bar is the right bar.** The board reports distance from the bar and the confidence of the space; it never proposes a value.
- **Whether an out-of-band reading deserves work.** Ranking is a recommendation. Repair, defer and retire stay with the team and the operator.

## Current State

Recorded as of 2026-08-19, as data.

| Fact | Value |
|---|---|
| Setpoint location | Migrated from the retired `docs/infra-health/strategy/RELIABILITY_TARGETS.md` into this scenario as checked-in data |
| Entries authored | 14 target kinds at operation granularity, pre-migration |
| Entries carrying a `decision_ref` | 0 — the field does not exist in the prose form |
| Bars whose value may equal the reading at authoring time | unaudited — the dead-deadband check has never run |
| Setpoint confidence | `SKETCH` — no obligation list has been authored |
| Update-protocol enforcement | none; the protocol is prose with no mechanism |

## Governing Principles

- **The bar is data this scenario reads and cannot write.** No API path, no table, no runtime mutation.
- **A deadband states the target, never the current reading.**
- **Honesty flags are derived from coverage status**, never hand-set.
- **Slow to tighten, evidence to loosen.** Hysteresis is the defence against target flapping.
- **An unparseable setpoint fails loudly.** Zero targets is never reported as zero problems.
- **Confidence is published with every ratio.** A `SKETCH` denominator is stated, not rounded away.

## Cross-References

- [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) — projections, cells, and the space half of the denominator.
- [`CONDITION-MODEL.md`](CONDITION-MODEL.md) — how a bar becomes a band verdict.
- [`TRUST-MODEL.md`](TRUST-MODEL.md) — why some readings never reach banding at all.
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — the durable decisions behind this shape.
- `docs/agent-system/TARGET_MODEL.md` § D6 — the deviation this document is built to prevent.
- `docs/infra-health/operating/OPERATING_MODEL.md` — the team's operating rules and routing.
