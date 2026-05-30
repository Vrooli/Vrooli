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
[control model](CONTROL-MODEL.md) describes. [CODE: api/pkg/autosteer/]

## Control-Loop Terms

**Closed-loop controller (target)** — The intended shape of auto-steer:
diagnose state → select the best skill → execute → measure → learn,
repeating until an objective is met. Contrast with *open-loop schedule*.

**Open-loop schedule (current)** — Today's auto-steer: run a profile's
fixed phase order with metric thresholds as exit gates. Intelligence is
in termination only, not selection.

**Profile** — A named auto-steer configuration. **(current)** an ordered
list of phases with stop conditions and quality gates, stored at
`profiles/<id>/profile.json`. **(target)** an *objective function* — a
weighted gap vector, target thresholds, and a budget — from which the
controller derives the path. [CODE: profiles/balanced/profile.json]

**Objective function (target)** — The reframed meaning of a profile:
"what does done mean, and what do I care about most," not a script.

**Phase (current)** — One steering step in a profile: a skill set
(`skill_ids`), `max_iterations`, and `stop_conditions`.

**Stop condition** — A simple or compound (AND/OR) predicate over metrics
that, when true, ends a phase. [CODE: api/pkg/autosteer/evaluator.go]

**Quality gate** — A condition whose failure can `halt` advancement
regardless of progress. [CODE: api/pkg/autosteer/phase_coordinator.go]

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

**Effectiveness table (target)** — Per-`(skill, dimension)` memory
tracking *efficacy* (findings closed per token, learned at runtime) and
*trust/cost/stability* (primed by DTV). Powers both selection and
thrashing damping.

**Credit assignment (target)** — After each run, diffing the findings set
into closed/introduced/untouched and attributing the delta to the skill
that ran, to update the effectiveness table.

**Thrashing** — Wasted oscillation. *Flavor 1* is intrinsic single-skill
non-convergence (caught by DTV). *Flavor 2* is inter-skill oscillation on
a live target (caught by runtime detection). See the three-layer defense
in [`CONTROL-MODEL.md`](CONTROL-MODEL.md).

**Findings fingerprint (target)** — A hash of the open-findings set used
to detect state cycles (thrashing) in one repeat.

## Integration Terms

**Agent-manager** — The scenario that executes every agent run. The
execution boundary; auto-steer never runs an agent directly.

**Development Toolchain Validator (DTV)** — Validates steer skills/tools
against pristine goldens. For the controller it is an *eligibility gate*
(DTV-red skills are barred from the autonomous fleet), a source of
*trust/cost priors*, and *Layer-1 thrashing prevention*. Not yet wired.

**test-genie** — Produces the structured findings that become the
controller's state. Not yet wired as the state source.

**Steer skill** — A prompt-manager skill that focuses an agent on one
kind of improvement (e.g., `ux`, `test`, `refactor`, `progress`).

**Steering mode** — How a task is steered: a saved profile, an ad-hoc
mode queue, a single manual mode, or none. [CODE: api/pkg/steering/]

## Cross-References

- [`CONTROL-MODEL.md`](CONTROL-MODEL.md) — full controller model
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape
- [`DOMAINS.md`](DOMAINS.md) — capability ownership
- [`FLOWS.md`](FLOWS.md) — control-loop state machine
