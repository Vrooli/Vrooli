# Glossary — Ecosystem Manager

The shared vocabulary for Ecosystem Manager. Terms marked **(target)**
describe the closed-loop controller model in
[`CONTROL-MODEL.md`](CONTROL-MODEL.md) and may not be fully implemented
yet; terms marked **(current)** describe today's code.

## Purpose Of This Document

Give every agent and contributor one definition per term so the docs,
code, and conversations stay aligned — especially across the shift from
the open-loop schedule to the closed-loop controller.

## Core Terms

**Task** — The unit of work. A task has a `type` (`scenario` or
`resource`), an `operation` (`generator` or `improver`), a target, a
priority, and a status. Owned by the tasks domain.
[CODE: api/pkg/tasks/types.go]

**Operation** — The work mode: `generator` (create new) or `improver`
(drive an existing target toward an objective).

**Type** — The target class: `scenario` or `resource`.

**Queue** — The processor that schedules and executes tasks via
agent-manager. Status is the `queue/<status>/` directory name; transitions
are atomic file moves. [CODE: api/pkg/queue/processor.go]

**Auto-steer** — The improvement control loop: applying steer skills
across iterations and deciding when to advance or stop. The domain the
[control model](CONTROL-MODEL.md) describes. [CODE: api/pkg/autosteer/decision_trace.go]

## Control-Loop Terms

**Closed-loop controller** — The shape of auto-steer (v0 implemented):
diagnose state → select the best skill → execute → measure → terminate,
repeating until an objective is met. Contrast with *open-loop schedule*.
[CODE: api/pkg/autosteer/execution_orchestrator.go]

**Open-loop schedule (retired)** — The legacy auto-steer: run a profile's
fixed phase order with metric thresholds as exit gates. Replaced by the
closed-loop controller (the phase-list schema is deleted).

**Profile** — A named auto-steer *objective function*: per-dimension
weights, target thresholds (`max_open_severity`, `operational_targets_pct`),
an allowed-skill set, and a budget — from which the controller derives the
path. Stored at `profiles/<id>/profile.json`.
[CODE: api/pkg/autosteer/types.go]

**Objective function** — The meaning of a profile: "what does done mean,
and what do I care about most," not a script.

**Dimension** — A canonical improvement axis (`standards`, `tests`,
`structure`, …) that both test-genie findings and skill declarations map
to. The vocabulary SSOT. [CODE: api/pkg/dimensions/dimensions.go]

**Findings vector** — The controller's primary state: open test-genie
findings bucketed by dimension and weighted by severity.
[CODE: api/pkg/findings/audit.go]

**Selection** — The controller's greedy SELECT stage: pick the skill that
best closes the heaviest profile-weighted open dimension.
[CODE: api/pkg/autosteer/selector.go]

**Termination** — Global, gradient-based stop: objective-met,
diminishing-returns, or budget-exhausted (no per-phase gates).
[CODE: api/pkg/autosteer/terminator.go]

**Decision trace** — The per-iteration record of the controller's
reasoning (state → choice → rationale → realized delta), persisted and
surfaced in the UI. [CODE: api/pkg/autosteer/decision_trace.go]

**Metric (current)** — A scalar measurement collected each iteration
(e.g., `operational_targets_percentage`, `accessibility_score`,
`unit_test_coverage`). [CODE: api/pkg/autosteer/metrics.go]

**MetricsSnapshot** — The collected set of metrics for one iteration,
persisted in execution state.

**Finding (target)** — A structured `test-genie` audit result with a
dimension, severity, and location. The controller's state is the *set of
open findings*, a richer signal than scalar metrics.

**Findings vector (target)** — The controller's state: open findings
bucketed by dimension. The thing the loop tries to drive to empty (or to
threshold).

**Dimension (target)** — A category a finding belongs to (standards,
tests, structure, security, visual, …); the axis the skill→dimension map
keys on.

**Skill → dimension capability map (target)** — A declared contract:
which finding dimensions each steer skill targets. The linchpin of
diagnosis-driven selection.

**Effectiveness table** — Per-`(skill, dimension)` memory tracking
*efficacy* (findings closed per token, learned at runtime). Powers both
selection and thrashing damping. Implemented (v1) in `pkg/effectiveness`
(`skill_dimension_effectiveness` table). Trust/cost/stability priors from
DTV are wired: `DTVPriorProvider` seeds the cold start and the bandit blend
washes it out as live evidence accrues.

**Credit assignment** — After each run, diffing the findings set into
closed/introduced (by stable finding ID, bucketed by dimension) and
attributing the delta to the skill that ran, to update the effectiveness
table. Implemented (v1) — `findings.DiffStates` + `effectiveness.Record`.

**Thrashing** — Wasted oscillation. *Flavor 1* is intrinsic single-skill
non-convergence (caught by DTV). *Flavor 2* is inter-skill oscillation on
a live target (caught by runtime detection). See the three-layer defense
in [`CONTROL-MODEL.md`](CONTROL-MODEL.md).

**Findings fingerprint** — A SHA256 hash of the open-findings ID set used
to detect state cycles (thrashing) in one repeat. Implemented (v1) —
computed in `pkg/findings` and compared by the terminator's cycle detector.

## Integration Terms

**Agent-manager** — The scenario that executes every agent run. The
execution boundary; auto-steer never runs an agent directly.

**Development Toolchain Validator (DTV)** — Validates steer skills/tools
against pristine goldens. For the controller it is an *eligibility gate*
(DTV-red skills are barred from the autonomous fleet), a source of
*trust/cost priors*, and *Layer-1 thrashing prevention*. Wired: the
`DTVEligibilityFilter` gates and `DTVPriorProvider` seeds priors via the
`dtv.Client` read seam (fail-open when DTV is unreachable). When the gate
degrades (DTV down or all candidates red) the controller proceeds with the
least-bad skill, halves the remaining budget, and flags the iteration
(proceed-cap-flag policy).

**test-genie** — Produces the structured findings that become the
controller's state. Wired as the state source via `findings.TestGenieRunner`
(the DIAGNOSE/MEASURE audit runner).

**Steer skill** — A prompt-manager skill that focuses an agent on one
kind of improvement (e.g., `ux`, `test`, `refactor`, `progress`).

**Steering mode** — How a task is steered: a saved profile, an ad-hoc
mode queue, a single manual mode, or none. [CODE: api/pkg/steering/manual_provider.go]

## Cross-References

- [`CONTROL-MODEL.md`](CONTROL-MODEL.md) — full controller model
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape
- [`DOMAINS.md`](DOMAINS.md) — capability ownership
- [`FLOWS.md`](FLOWS.md) — control-loop state machine
