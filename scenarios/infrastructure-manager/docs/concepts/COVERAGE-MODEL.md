# Coverage Model

## Purpose Of This Document

This is the **single canonical source** for the **Coverage axis**: the model that answers *"how much of the platform's reliability is instrumented at all?"*

It defines the projections, the cell grid, the denominator/numerator split, denominator-confidence, and the open-loop contract. It is the entry point for this scenario's modelling — read it before its three siblings:

- [`CONDITION-MODEL.md`](CONDITION-MODEL.md) — the Condition axis: for the cells that *are* instrumented, is the element in band?
- [`TRUST-MODEL.md`](TRUST-MODEL.md) — the Trust sub-axis: can a given reading be believed at all?
- [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md) — the setpoint: the bar each cell is graded against, and why this scenario cannot move it.

The upstream contract this document instantiates is `docs/agent-system/TARGET_MODEL.md`. Its sibling worked instance is `scenarios/meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md`, and this file follows that model deliberately rather than inventing a second one.

## The Instrument Contract, In One Sentence

> A team is a control loop: it regulates one plant against a setpoint it does not own, using one instrument it does not decide with, and it gets simpler as the system around it gets more capable.
> — `docs/agent-system/TARGET_MODEL.md` § 1

This scenario is the *instrument* half of that sentence for the `infra-health` team. Every decision below follows from the middle clauses: **a setpoint it does not own**, and **an instrument it does not decide with**.

## The Plant

The plant is the layered platform stack defined in `docs/infra-health/operating/OPERATING_MODEL.md` § "Platform Under Control". It is **not** the codebase, and it is **not** the `*-health` scenario fleet — those are test-genie phase providers validating code, which is `scenario-qa`'s plant.

| Layer | Timescale | This scenario's relationship |
|---|---|---|
| Host substrate — kernel, devices, drivers, crash evidence | continuous | The `substrate` projection. Reads kernel/device error signals, machine-check telemetry, crash evidence, accelerator health, kernel/driver coherence, boot integrity and watchdog liveness as ordered severities. Never mutates any of it. |
| Commissioning — `vrooli setup`, host tools, safeguards | per host change | Observes outcomes; never invokes. Sudo exists only here. |
| Capacity broker — `vrooli capacity` | ms–minutes | Reads claim coverage and reserve drift. Never changes a policy lever. |
| Autoheal — `vrooli-autoheal` | seconds–minutes | Primary sensor source. Reads uptime, restarts, heal outcomes, the check registry. Never shelves, restarts, or reconciles. |
| System-monitor | minutes–hours | Reads process attribution and investigations. |
| Capability owners | per query / run | Reads derived aggregates only, never members. |
| **Infra-health (this instrument's team)** | heartbeats–days | The outermost, slowest loop. This scenario is its board. |

**Supervise, don't operate.** Operating-model rule 3 is absolute for this scenario: no restarts, no policy-lever changes, no degrade or preempt, no privileged mutation. Instrumentation encodes the same idea in tag letters — `FT` is a flow transmitter, `FIC` is a flow indicating *controller*, and the `C` is what confers the right to act. This scenario has no `C`.

## The Projections

Platform reliability decomposes into **projections**, each owned by the control layer that holds its ground truth. A projection is a coherent question about the platform; a **cell** is one answerable fact inside it.

| Projection | Question | Owner (denominator) | Numerator source | Leg unit |
|---|---|---|---|---|
| **supervision** | Is the watch itself complete and sound? | `vrooli-autoheal` | `CheckRegistryService.Reconcile` | check |
| **availability** | Do elements stay up to their bar? | `vrooli-autoheal` | `ActionsService.Trends` (per-check) | check |
| **recovery** | When an element falls over, does healing work without looping? | `vrooli-autoheal` | `ActionsService.History` + `Transitions` | heal episode |
| **capacity** | Are resource claims covered and honest? | control plane (`vrooli capacity`) | `capacity reconcile` / `recommend` | claim |
| **headroom** | Is storage growth bounded under declared ceilings? | `storage-manager` | `infra-health` aggregate | device / ceiling |
| **durability** | Is state backed up and provably restorable? | `data-backup-manager` | `status` / `coverage` / `drills` | plan / target |
| **attribution** | When the host saturates, is the cause known? | `system-monitor` | `metrics process-timeline` | saturation window |
| **validation-cost** | Is the validation loop affordable and reliable? | `test-genie` | `runs cost` | phase |
| **agent-throughput** | Can agents be spawned, and do failures cluster? | `agent-manager` | `measures runs stats` / `error-patterns` | run class |
| **commissioning** | Can a clean host reach green, reproducibly and in time? | control plane (`vrooli setup`) | — (open-loop) | host bring-up |
| **substrate** | Is the host, kernel and device layer sound? | `vrooli-autoheal` | `ActionsService.GetPerCheckTrends` (per-check status) | host subsystem |

Eleven projections, authored at **ideal maturity**. That is deliberate and matches how every other denominator in this system is written: the model is authored at full maturity so the distance between it and reality is *measurable* rather than invisible. Current adoption is recorded in [§ Current State](#current-state) as data, never by trimming the model down to what already works.

### Why these are projections and not domains

`supervision` was originally modelled as a domain of this scenario. It is not — it is one projection among eleven, distinguished only by being the projection whose numerator is a two-direction reconcile rather than a scalar read. Modelling it as a domain would have given one reliability dimension a structural privilege the others do not have, and would have made each new projection harder to add than the last. `substrate` was added on 2026-08-20 at the cost of one space document, one owner-map entry and one sensor table, which is the test that framing was right.

### Units do not compose across projections

A projection's leg unit is part of its identity. `availability` is a percentage over a window; `substrate` is an **ordered severity** (`0` OK, `1` WARNING, `2` CRITICAL) at an instant. Before `substrate` existed, host and device faults were reachable only through `availability`, where a GPU falling off the bus, a machine-check storm and a slow scenario restart were all "a dip in an uptime figure" — a unit in which the fault that matters most is the one that shows least. Bars carry their unit and a reading whose unit disagrees with its bar is `NOT_GRADEABLE`, never silently compared.

### Denominator, numerator, confidence

- **Denominator** — the curated *intended* cell set for a projection. It lives **with the owner**, in that owner's `docs/spaces/<projection>-space.md`, and is read through that owner's `space --projection <p> --json` verb. This is the same contract `search-hub`, `test-genie`, `prompt-manager` and `program-runtime` already implement for `meta-optimization-manager`.
- **Numerator** — live *actual* instrumentation, computed by joining the denominator against the owner's live surface. **Never stored.**
- **Coverage** = numerator ÷ denominator, computed live at query time.
- **Denominator-confidence** — how complete we believe the *denominator itself* is: `AUTHORITATIVE | PARTIAL | SKETCH` plus a rationale. Every coverage number carries it. The honesty is **recursive**: a reader sees "X% instrumented against a Y-confidence denominator", so the board can never imply completeness it has not earned.

### The split that keeps the observer honest

This is the one place this scenario deliberately diverges from `meta-optimization-manager`, and the divergence is load-bearing.

In MoM the owner holds the entire denominator, because a capability owner naming its own supply has no incentive to under-declare. Here the denominator has two halves with different incentive structures:

| Half | Lives with | Why there |
|---|---|---|
| **Which cells exist** — the shape of the space, what a leg is, what `NOW` means for this projection | the **owner**, in `docs/spaces/<projection>-space.md` | Only autoheal knows what its check registry can express. An instrument that asserted autoheal's world would go stale every time autoheal changed, and would be a roster in all but name. |
| **The bar each cell is graded against** — deadbands, target percentages, windows, hysteresis | **this scenario**, in a checked-in setpoint file | Deadbands are *operator judgment*, not owner knowledge. An owner that sets its own bar is the supervisor of the platform grading itself — deviation `D6`, one layer down. |

> **Owners define the space. The operator defines the bar. Neither can move the other.**

That property is what makes a coverage number here mean something. See [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md) for how the bar is held so that this scenario cannot move it either.

## The Cell Grid

A **cell** is `(projection, element, dimension)` — the smallest fact a projection can be asked about. Cells carry a status:

| Status | Meaning |
|---|---|
| `NOW` | A sensor exists, the live join is built, and readings flow. |
| `IN-REACH` | A sensor exists but this scenario does not yet join it. The gap is engineering, not instrumentation. |
| `MISSING` | No sensor exists. The cell is **open-loop**: declared blindness, dated, and aged. |

### An open-loop cell is a first-class state

This is the property that makes the model measurable rather than merely aspirational. A cell with no sensor is **not missing data — it is declared blindness.** The scenario counts these, dates them, and reports the count as a headline of its own, following the same self-counting discipline as `FRAMEWORK_HEALTH.md`'s open-loop target count.

Two derived fields let an *intentionally visible* hole be told apart from an *overdue* one:

| Field | Meaning |
|---|---|
| `gap_opened_on` | When the empty cell was first declared |
| `gap_open_days` | How long it has stood |

Without them, "declared and dated" degrades into "declared once and forgotten."

**The instrumentation roadmap is the open-loop set, computed.** The team's hand-maintained roadmap was retired for exactly this reason. The gap list falls out of the cell grid: every `MISSING` cell is a roadmap entry with a date attached, and there is no second list to drift.

### Cell status is never inferred from silence

A cell absent from a live join keeps its authored status. "Could not resolve" is never fabricated as `MISSING`, and `MISSING` is never rendered as `0%` healthy. This is the same per-cell rule MoM applies, and it exists because the failure it prevents — an owner outage reading as a coverage collapse — would make every board reading untrustworthy during exactly the incidents the board exists to surface.

## Setpoint Drift

Coverage claims decay. Two drift checks run against the live fleet and are reported as findings, not silently absorbed:

| Drift | Detection | Why it matters |
|---|---|---|
| **Dead sensor** | A cell names a sensor whose typed operation no longer resolves through `api-core/discovery` or the owner's descriptor. | A cell reading `NOW` against a sensor that no longer exists is the most dangerous state on the board: it claims instrumentation it does not have. |
| **Closable gap** | A cell is `MISSING`, but a shipped verb in the live command surface could already serve it. | An open-loop cell that could have been closed months ago is not honest blindness — it is unowned work wearing honesty's clothes. |

Both are computed against the fleet's live surface rather than a checked-in list, so neither can go stale in the way the thing it checks can.

## Coverage Is Not The Only Axis

The projections are the **coverage axis**: enumerable instrumentation, measured as `now / total` against a curated denominator. Three further axes are independent of it.

| Axis | Shape | Question | Defined in |
|---|---|---|---|
| **Coverage** | `now / total`, denominator-confidence | Is this reliability dimension instrumented *at all*? | this document |
| **Condition** | per-leg band verdict | For instrumented cells, is the element *in band*? | [`CONDITION-MODEL.md`](CONDITION-MODEL.md) |
| **Trust** | closed-vocabulary verdict per reading | Can this reading be *believed*? | [`TRUST-MODEL.md`](TRUST-MODEL.md) |
| **Efficacy** | per-finding re-read verdict | Did the fix actually *move the sensor*? | [`CONDITION-MODEL.md`](CONDITION-MODEL.md) § Actuation efficacy |

**They are never merged into a score.** Folding trust into condition means an instrument defect gets scheduled as plant work. Folding efficacy into condition means a fix that never worked reads as a fix that worked once. Folding condition into coverage means a fully-instrumented, fully-broken platform reads as healthy. Each stays a separate number on one surface.

### Coverage scopes every other number

Coverage comes first because every other figure is read against it. "Seven of eight elements in band" means something different when the reader knows that eight of twenty-six cells are instrumented at all — and the ratio is the only thing that carries that. A condition percentage published without its coverage denominator is the single most misleading number this scenario could emit.

## What Stays Judgment

Explicitly **not** measured here, and not to be turned into a metric later:

- **Whether the projection set is the right set.** That is the obligation-naming step. `TARGET_MODEL.md` § 11 is direct: the honest way to express uncertainty about a denominator is denominator-confidence, not a score.
- **Whether to repair, defer, adopt, or retire.** The board ranks; the team and the operator decide.
- **Platform-code quality and cross-platform debt.** Judgment-shaped, owned by the `platform-code-auditor` lane. See [`DOMAINS.md`](DOMAINS.md) § Deferred Domains.

## Current State

Recorded as of 2026-08-20, as data. This section is the measured distance from the model above and is expected to change; the model is not expected to change with it. Every figure below is readable from the instrument itself — do not hand-maintain a second copy.

| Fact | Value | Read it with |
|---|---|---|
| Projections authored | 11 | `coverage status` |
| Projections with an owner-authored space doc | 11 | `coverage cells` |
| Owners registering `space --projection <p> --json` | 11 of 11. `capacity` and `commissioning` are served by this scenario as interim holder; its verb refuses the other nine. | `<owner> space --projection <p> --json` |
| Projections this scenario reads through the typed verb | 0 — `coverage` still resolves every space by file path through `FileSpaceReader`. The verb is published; the reader has not been switched to it. | — |
| Projections with a live typed reader | 2 of 11 — `availability` and `substrate`, both from `vrooli-autoheal` | `condition status` |
| Setpoint bars | 33 — 27 gradeable with a unit and threshold, 6 declared not-gradeable with stated reasons | `coverage validate` |
| Substrate bars ratified | 4 of 12 (`SB1`, `SB2`, `SB5`, `SB8`) on 2026-08-20 against live host readings. `SB3`, `SB4` and `SB7` stay provisional with per-bar stated defects; the five device-layer bars are provisional pending both operator review and the join their cells need. | `coverage validate` |
| Denominator confidence | `SKETCH` overall; `PARTIAL` on most projections | `coverage status` |
| Open-loop cells | computed and dated | `coverage open-loop` |
| Team loop status | `paused-manual` since 2026-07-24 — no reading is a live baseline until it resumes | `prompt-manager team heartbeat-control infra-health status` |

The load-bearing remaining gap is the same one it has always been: nine of eleven
projections have an authored denominator but no live numerator, so their cells report an
honest `IN-REACH` rather than a reading. That distance is visible in `coverage status`
rather than hidden by trimming the model to what already works.

The second gap is narrower and newer. All eleven denominators are now published through
the typed `space` verb, but this scenario still reads all eleven by file path. Until
`FileSpaceReader` is replaced by a verb-backed reader the contract is one-sided: owners
publish a typed denominator and the only consumer ignores it, so a document that stops
parsing through the verb still reads fine here. Nothing blocks the switch any more — it
is unstarted work, not a dependency.

The third is the one this cycle uncovered. The `substrate` cell set grew from eight cells
to thirteen without a single new sensor being written: every one of `SB9`-`SB13` names a
device-layer sensor that `system-monitor` already ships and emits on a 60s collector, and
that no projection joined. They are `IN-REACH`, not `MISSING`, and the distinction is the
measurement — five closable gaps counted as honest blindness would have been *unowned
work wearing honesty's clothes*. That five cells appeared from one sweep of an existing
scenario is the strongest available evidence that the spaces are still under-declared, and
it is why substrate confidence stays `PARTIAL` after growing by 62%.

## Governing Principles

- **Owners define the space; the operator defines the bar.** Neither can move the other, and this scenario can move neither.
- **An open-loop cell is a declared state, not missing data.** It is counted, dated, and aged.
- **Coverage scopes every other number.** No condition figure is published without the coverage denominator it was computed against.
- **A cell absent from a live join keeps its authored status.** Unresolvable is never fabricated as `MISSING`, and `MISSING` is never rendered as healthy.
- **Axes stay separate.** Coverage, condition, trust and efficacy are four numbers, never one score.
- **Surfaces, does not decide.** The board ranks candidates and states confidence; every decision and every actuation stays outside this scenario.

## Cross-References

- [`CONDITION-MODEL.md`](CONDITION-MODEL.md) — the Condition axis in full.
- [`TRUST-MODEL.md`](TRUST-MODEL.md) — the Trust sub-axis in full.
- [`SETPOINT-MODEL.md`](SETPOINT-MODEL.md) — the setpoint, deadband discipline, and D6 prevention.
- [`DOMAINS.md`](DOMAINS.md), [`ARCHITECTURE.md`](ARCHITECTURE.md), [`DATA.md`](DATA.md), [`INTEGRATIONS.md`](INTEGRATIONS.md).
- `docs/agent-system/TARGET_MODEL.md` — the instrument contract this document instantiates.
- `docs/infra-health/operating/OPERATING_MODEL.md` — the plant's layer map and routing rules.
- `scenarios/meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md` — the sibling worked instance of the same upstream contract.
