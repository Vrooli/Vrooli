# Control Model — Ecosystem Manager

This document is the **canonical mental model** for how Ecosystem
Manager improves a scenario or resource. Read it before
[`ARCHITECTURE.md`](ARCHITECTURE.md) or any auto-steer code: the
architecture is the *implementation surface*, but this document is the
*intent* that surface is converging toward.

It describes a target design. Where today's code differs, that is
called out explicitly in [Current Implementation Status](#current-implementation-status)
and tracked in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). The
docs are intentionally written ahead of the implementation so the model
and its vocabulary are pinned before code hardens around a weaker shape.

## Purpose Of This Document

Use this document to answer:

- What *is* Ecosystem Manager, conceptually — a scheduler, or a controller?
- How should it decide which skill to run against a target next?
- When should an improvement run stop?
- How do `development-toolchain-validator` (DTV), `test-genie`, and the
  steer skills fit into one loop?
- Why is a "profile" an objective function and not a script?

System shape and surfaces live in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Capability ownership lives in [`DOMAINS.md`](DOMAINS.md). The vocabulary
index lives in [`GLOSSARY.md`](GLOSSARY.md). The runtime state machine
lives in [`FLOWS.md`](FLOWS.md).

## The Core Reframe

Ecosystem Manager is a **closed-loop controller**, not an open-loop
schedule.

The distinction is the entire point of this document:

| | Open-loop schedule (legacy framing) | Closed-loop controller (target) |
|---|---|---|
| Plan | Fixed phase order chosen *before* looking at the target | Recomputed each iteration *from* the target's state |
| Metrics role | Decide *when* to leave a fixed step | Decide *what step* is right next |
| Human input | Supplies the plan | Supplies the objective |
| Intelligence lives in | Termination only | Diagnosis **and** selection |

The legacy auto-steer behavior — march a profile's phases
`progress → ux → refactor → test` in order, with metric thresholds as
exit gates — is an open-loop schedule. All of its intelligence is in
*termination* ("have we done enough of this fixed step?"); none is in
*selection* ("what is the right step?"). The human pre-commits to a plan
before the system has looked at the target. That is why it does not feel
intelligent.

A controller instead runs the classic sense → decide → act → measure →
learn loop, the same shape as a thermostat or a PID controller, with the
target's measured state — not a fixed schedule — driving every decision.

## The Control Loop

```
        ┌─────────────────────────────────────────────────────────┐
        │                                                         │
        ▼                                                         │
  ┌───────────┐    ┌──────────────┐    ┌───────────┐    ┌────────────┐
  │ DIAGNOSE  │───▶│   SELECT     │───▶│  EXECUTE  │───▶│  MEASURE   │
  │ state     │    │ skill that   │    │ skill via │    │ re-audit,  │
  │ = open    │    │ best closes  │    │ agent-mgr │    │ diff the   │
  │ findings  │    │ the biggest  │    │ (sandbox) │    │ findings   │
  │ + gaps    │    │ gap          │    │           │    │ set        │
  └───────────┘    └──────────────┘    └───────────┘    └─────┬──────┘
                          ▲                                   │
                          │      ┌─────────────────────┐      │
                          └──────│ LEARN: credit the    │◀────┘
                                 │ skill for the delta  │
                                 │ (effectiveness table)│
                                 └─────────────────────┘
```

Today's code already owns **EXECUTE** (agent-manager integration) and
**MEASURE** (the metrics collectors). The three boxes that make it a
controller — **DIAGNOSE**, **SELECT**, and **LEARN** — are the work this
model defines.

## State: The Findings Vector

The controller's state is **the set of open `test-genie` findings** for
the target, not a handful of coarse scalars.

Coarse metrics (`accessibility_score = 83`, `unit_test_coverage = 61`)
are a weak state: they tell you a number moved but not *what to do*. The
rich state is the structured audit — each finding carries a **dimension**
(standards, tests, structure, security, visual, …), a **severity**, and
a **location**. "12 open findings, clustered in `standards` and `tests`"
is directly actionable; "tidiness = 83" is not.

The `scenario-improvement-campaign` skill already ranks a `test-genie`
audit into findings. The controller consumes that ranked findings set as
its state vector. The legacy scalar metrics (collected by the auto-steer
metrics collectors — see [`DOMAINS.md`](DOMAINS.md)) remain useful as
**gap measurements** for objective/termination math, but they are not
the primary state.

## The Skill → Dimension Capability Map

This is the **linchpin** of selection, and the piece that is currently
*implicit* (buried in each profile's hand-authored phase order). Make it
**explicit and declared**: every steer skill declares which finding
dimensions it targets.

| Skill (example) | Targets dimensions |
|---|---|
| `ux` | accessibility, visual |
| `test` | coverage, test-quality |
| `screaming-architecture` / `refactor` | structure, cycles, duplication |
| `security-steer` | vulnerability, input-validation |
| `progress` | operational-targets (PRD completion) |
| `documentation-health` | docs coverage, reference integrity |

Once the findings state and this map both exist, selection becomes
mechanical: *the heaviest open cluster is `standards`; the skill that
targets `standards` is X; run X.* There is already a partial notion of
skill "focus" — this formalizes it into a declared contract.

## Selection Policy

Selection is staged. Ship the simplest version first; it captures most
of the felt intelligence.

- **v0 — greedy diagnosis (ship first).** Bucket open findings by
  dimension, score each bucket by `severity × count`, pick the skill
  whose declared targets cover the heaviest bucket. Deterministic, fully
  explainable, no learning required — and already an order of magnitude
  smarter than a fixed profile order.
- **v1 — effectiveness-weighted (contextual bandit).** Track per-`(skill,
  dimension)` history: did running this skill actually reduce findings in
  that dimension, and at what token cost? Weight selection by **expected
  reduction per token**. This is what kills the "`ux` ran five times and
  accessibility didn't move — stop picking it" failure.
- **v2 (shipped) — DTV-primed priors.** Seed the v1 priors from DTV so a
  fresh target does not learn from zero. See
  [DTV As Gate And Prior](#development-toolchain-validator-as-gate-and-prior).

## The Effectiveness Table

The learning substrate is a per-`(skill, dimension)` table. It records
two genuinely different properties — do not conflate them:

| Property | Question | Source |
|---|---|---|
| **Efficacy** | When this skill targets dimension Y, how many findings does it close per token? | **Learned at runtime** from real improvement runs — cannot be primed by DTV |
| **Trust + cost + stability** | Does this skill run cleanly, behave predictably, converge, and at what token cost? | **Primed by DTV**, refined at runtime |

The split matters because DTV validates skills against *pristine*
goldens, which have **zero findings** — so DTV can never observe "skill X
closed 8 findings." Efficacy is learned live; DTV supplies the trust and
cost priors and the eligibility gate.

**Credit assignment** closes the loop: after each run, re-audit, diff the
findings set into *closed / introduced / untouched*, attribute that delta
to the skill that ran, update the table. A skill that introduces more
than it closes earns negative credit and self-deprioritizes — which is
also the runtime regression catcher.

## Profiles Are Objective Functions

In the controller model, a **profile is an objective function, not a
script.**

Legacy profiles are ordered phase lists (`progress → ux → refactor →
test`). In the target model, "balanced" vs "production-ready" vs
"rapid-mvp" become different **weightings of the gap vector**, different
**target thresholds**, and different **budgets** — not different
hard-coded sequences. A profile answers *"what does done mean, and what
do I care about most?"*, and the controller derives the path.

This reframe is the consolidation point with the
`scenario-improvement-campaign` skill: the campaign/findings layer
produces the state, auto-steer is the controller, and the effectiveness
table is the memory — one flywheel instead of two overlapping systems.

## Development Toolchain Validator As Gate And Prior

DTV answers a narrow, precise question: *"Is this skill/tool currently
fit to run unattended, how much should we trust it, what does it cost,
and does it converge?"* It does **not** answer "how good is this skill at
fixing dimension Y on a broken target" — that is efficacy, learned live.

DTV plays three roles in this controller:

1. **Eligibility gate.** A skill that is currently DTV-red (failing
   validation — actively buggy or incoherent) is **not eligible for the
   autonomous fleet** while a healthier skill exists for the dimension. You
   never turn an unattended loop loose applying a skill you already know is
   broken; that just scales damage. But the gate never *stalls* the loop: when
   DTV is unreachable or **every** candidate for the heaviest dimension is red,
   the controller follows the **proceed-cap-flag** policy — it proceeds with the
   least-bad (highest-trust) skill, halves the remaining iteration budget once,
   and prominently flags the iteration for review (`GateDegradedCause`). UNKNOWN
   fitness (no DTV data) fails open to permissive P1 selection, also flagged.
2. **Trust + cost priors.** Among DTV-green skills, run-failure rate
   becomes a trust weight, token/duration history becomes the per-token
   denominator, and convergence becomes a stability signal — the v2
   priors for the effectiveness table.
3. **Thrashing prevention (Layer 1).** See below.

## Thrashing Defense

Thrashing has two distinct flavors, and they need different defenses.

- **Flavor 1 — intrinsic, single-skill non-convergence.** A skill that,
  run twice, undoes or re-edits its own work and never settles. This is a
  property *of the skill*. **DTV catches it perfectly** — its empty-diff
  convergence target literally tests "does this skill, run again,
  stabilize?" on a fixed golden, *before* the skill is ever turned loose
  on a real target.
- **Flavor 2 — inter-skill oscillation on a live target.** Skill A fixes
  structure but introduces a lint finding; skill B fixes lint but
  disturbs structure; back to A. This is emergent and target-specific.
  **DTV cannot catch it** — DTV validates skills in isolation against a
  green golden and never runs A→B→A on a broken target. It must be caught
  at runtime by the controller.

Defense is therefore three layers:

| Layer | Mechanism | Catches |
|---|---|---|
| 1 — Prevention | DTV convergence gate (pre-fleet) | Flavor 1 (intrinsic) |
| 2 — Runtime detection | findings-fingerprint cycle detection; net-progress window (`closed − introduced ≈ 0` over K iterations → halt); credit-table penalties; hysteresis (don't immediately re-pick a skill that just regressed its own dimension) | Flavor 2 (emergent) |
| 3 — Hard backstop | max-iterations / max-tokens / max-wallclock per target (exists today) | Everything, bluntly |

Layer 2's credit mechanism is the *same* effectiveness table that powers
selection — the machinery that makes the controller smart also damps
thrashing for free. A **findings-fingerprint** (hash the open-findings
set each iteration; a recurring hash is a provable state cycle) is the
cheap, precise version of "detect circular behavior programmatically,"
catching a cycle in one repeat instead of burning the whole budget down
to Layer 3. Where agent-manager captures per-run diffs, a **regression
veto** (roll back an iteration whose net weighted findings went up) turns
a bad pick into a wasted pick instead of a setback.

## Termination

Termination is global and gradient-based, not per-phase. Stop when:

- no finding above the severity threshold remains **and** the gap metrics
  hit their objective targets, **or**
- marginal improvement per iteration falls below a floor (diminishing
  returns — this is what prevents "grinding uselessly"), **or**
- the budget is exhausted (Layer-3 backstop).

The diminishing-returns stop and the Layer-2 net-progress thrashing
detector are the same measurement viewed two ways.

## Transparency

Half of "it doesn't feel intelligent" is a *visibility* problem, not an
algorithm problem. Even today's metric-driven decisions are invisible. A
glass-box controller feels intelligent at v0; a black box feels dumb at
v2. Every selection surfaces its decision trace, e.g.:

> Picked `test` — 12 open coverage findings (heaviest cluster), expected
> −8, historical effectiveness 0.7, est. cost ~40k tokens.

The **expected −8** is real: each trace entry carries a `PredictedReduction`
computed at SELECT time (the bandit's expected reduction-per-token × the
estimated run tokens × the chosen dimension's weight) alongside the
`RealizedDelta` filled after MEASURE. The UI renders predicted-vs-realized Δ
per iteration plus a running mean-absolute-error calibration indicator, so a
miscalibrated bandit is visible at a glance. When the Layer-1 DTV gate degrades
(DTV unreachable or every candidate red), the iteration is prominently flagged
with its `GateDegradedCause` and the remaining budget is halved (proceed-cap-flag).

## Staged Rollout

1. **v0:** declared skill→dimension map + greedy findings-driven
   selection + global gradient termination + a visible decision trace.
   This converts the system from "schedule with gates" to
   "diagnose-and-target" — the bulk of the felt-intelligence gain.
2. **v1:** the effectiveness table + contextual-bandit selection +
   credit assignment + Layer-2 thrashing detection.
3. **v2 (shipped):** DTV-primed priors and the Layer-1 eligibility gate,
   with the proceed-cap-flag degraded-gate policy. See the Current
   Implementation Status table below.

Fuse with `scenario-improvement-campaign` (consume its ranked findings as
state) rather than building a parallel findings system.

Cost control is non-optional: `test-genie comprehensive` can run many
minutes, so do not full-re-audit every iteration. Use targeted re-audit
(only the dimensions/files a skill touched), a cheap preset in the inner
loop with comprehensive only at the termination gate, or audit every N
iterations. Choose one deliberately — it decides whether the loop is
practical.

## Current Implementation Status

| Capability | Target (this doc) | Today |
|---|---|---|
| Execute via agent-manager | Yes | **Implemented** |
| Measure (metrics collectors) | Yes | **Implemented** (gap metrics; re-audit is primary) |
| State = findings vector | Yes | **Implemented** (v0) — `pkg/findings` bucketed by dimension |
| Skill → dimension map (declared) | Yes | **Implemented** (v0) — declared on skills, `pkg/skillmap` resolver |
| Selection (diagnose → pick) | Yes | **Implemented** (v1) — effectiveness-weighted bandit within the heaviest dimension; greedy when cold (`Selector`) |
| Effectiveness table + learning | Yes | **Implemented** (v1) — `pkg/effectiveness` ledger + credit assignment after every iteration |
| Token-cost capture | Yes | **Implemented** (v1) — `RunCost` from agent-manager run summary → reduction-per-token selection |
| Termination | Global, gradient | **Implemented** (v0) — objective-met / diminishing-returns / budget |
| Thrashing Layer 1 (DTV) | Yes | **Implemented** (P2) — `DTVEligibilityFilter` denies DTV-red skills before ranking; degraded gate (DTV-down / all-red) follows proceed-cap-flag: least-bad skill, budget halved once, flagged; gated by profile `dtv.gate_enabled` |
| Thrashing Layer 2 (runtime) | Yes | **Implemented** (v1) — fingerprint cycle-halt, net-progress window, hysteresis cooldown, regression veto |
| Thrashing Layer 3 (budget cap) | Yes | **Implemented** (max-iterations) |
| DTV priors / eligibility gate | Yes | **Implemented** (P2) — `PriorProvider` seam wires `DTVPriorProvider` (trust/cost/convergence prior from DTV's `GetSkillFitness`); fail-open to uniform when DTV has no data. See `pkg/dtv` + `pkg/autosteer/dtv_selection.go` |
| Decision-trace transparency | Yes | **Implemented** (v1+P2+P4) — trace carries token cost, per-dimension flow, regression/veto/halt, DTV verdict / prior provenance / exclusion reasons, predicted-reduction (P4), and the degraded-gate cause; UI renders predicted-vs-realized Δ + a calibration indicator; read API + CLI + UI |

**v1 selection details:** within the heaviest open dimension, eligible skills are
ranked by `expectedEfficacyPerToken = (n/(n+k))·observed + (k/(n+k))·prior`
(Bayesian shrinkage, `k = 3`). The prior is supplied by a `PriorProvider`: P1
wires `UniformPrior{0}` so `n = 0` reproduces v0 greedy order; P2 wires
`DTVPriorProvider`, which seeds `prior = weight·base·trust·convergenceConfidence`
(trust = DTV pass-rate) for skills DTV has observed and falls back to `0` for
UNKNOWN / thin-evidence skills — so the bandit blend still washes any prior out
with live evidence. Exploration is **epsilon-greedy with `ε = 0.15/(1+iteration)`**,
deterministic per `(task, iteration)` via an FNV-seeded RNG (reproducible for
tests and replay). See `pkg/effectiveness`, `pkg/dtv`, and
`pkg/autosteer/{selector,dtv_selection}.go`.

The implementation roadmap for closing this gap is tracked in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) and
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and surfaces
- [`DOMAINS.md`](DOMAINS.md) — auto-steer, queue, steering, metrics ownership
- [`FLOWS.md`](FLOWS.md) — the runtime control-loop state machine
- [`GLOSSARY.md`](GLOSSARY.md) — controller vocabulary
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — the reframe decision
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — implementation gap and roadmap
