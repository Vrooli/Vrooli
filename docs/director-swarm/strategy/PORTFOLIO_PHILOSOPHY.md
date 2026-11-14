# Portfolio Philosophy

How `director-swarm` and the operator think about the goal portfolio. This doc is the frame `portfolio-manager` anchors its proposals against; other teams (monetization, meta-optimization) can also read it to align with current director-swarm priorities.

## Ranking criteria

Goals are ranked — and proposed for `active now` status — against these factors, in order:

### 1. Revenue (primary)

Vrooli is pre-revenue. The top priority is closing that gap — specifically, getting **paid apps into users' hands** under the business bundle. Goals that directly advance a paid-delivery path (desktop release, Stripe integration, bundle headliners approaching readiness) outrank everything else.

"Revenue" does NOT mean the literal fastest path to publishing a paid app. See criterion 2.

### 2. Safety, quality, and auditability of what we publish

We will not ship broken software to paying users. Refund churn from premature paid launches is a worse outcome than a later launch. This criterion covers:

- **Well-tested apps** before paid release.
- **Deployment pipeline reliability** — redeployment, rollback, crash-log and feedback capture.
- **Auditability** — being able to trace what changed, why, and whether it worked.
- **Reusable quality capabilities** — every goal that hardens the production pipeline compounds; the fifth paid app ships faster than the first because the tooling exists.

This is not a secondary consideration to revenue; it is a hard co-requirement. A goal that accelerates revenue but degrades the quality pipeline is rejected.

### 3. Meta-optimization and platform self-improvement

Vrooli's agents, scenarios, and tooling must measurably get sharper over time. If agents are frequently erring, not finishing their work, inefficient, or if technical debt accumulates unchecked, revenue work eventually grinds. A sustained portion of portfolio capacity goes to platform self-improvement (agent reliability, tooling, skill/scenario conversion, run-quality) because the downstream leverage is enormous.

This comes after 1 and 2 but is never zero. A healthy portfolio always has at least one active goal from this category.

### The instrument rule (applies across 1–3, not after them)

A goal whose output is an **instrument** — a sensor, a measurement surface, or an actuator that another goal needs in order to be scoreable — inherits the rank of the highest-ranked goal it unblocks, and sequences ahead of it.

An instrument does not get its own priority band; it borrows one. A sensor for a revenue loop ranks at revenue priority. A sensor for a meta-optimization loop stays at meta priority. Stating the rule relationally blocks both failure modes at once: instrument work starving because its payoff is indirect, and instrument work inflating because "everything is an instrument."

**Test for whether the rule applies.** Name the goal the instrument unblocks, and state what is unscoreable without it. If you cannot name one, it is not an instrument — it is a feature, and it ranks on its own merits under 1–3.

This is the portfolio application of `I3` (Enablement) in [`OBJECTIVES.md`](OBJECTIVES.md). That file states the rule as intent; this section is where it binds to ranking. A control loop whose sensor does not exist is open-loop no matter how good its policy is, so an instrument that trails the loop it enables is mis-sequenced rather than merely deprioritized.

## Concurrency

**There is no cap on active goals.** Swarm Manager is the synthesis point for every idea the operator has for Vrooli; goals can accumulate faster than they're implemented, and that's expected. Storing months (or a year+) of goal-level planning is correct use of Swarm Manager.

The operator works the portfolio by moving between active goals based on where attention is best spent in the moment — when one goal is blocked waiting on an agent run, the operator picks up another. Hard caps would fight this workflow.

**Discipline comes from calibration, not caps.** Portfolio decisions carry prediction blocks ([OUTCOMES_CHARTER.md](../evidence/OUTCOMES_CHARTER.md) §"Prediction ledger"), and expected-cost bands keep decision comparisons honest without bounding the goal count.

**Priority ordering is Swarm Manager's job, not this doc's.** Swarm Manager's priority algorithm ranks goals based on its own signals (priority field, dependencies, backlog item state). `portfolio-manager` defers to that ordering for tactical sequencing, and only proposes portfolio-level decisions (`goal-portfolio`) when a goal's **category fit** against the criteria above is what's being adjusted — not its position in the queue.

**How the instrument rule binds without violating that.** The instrument rule is a *category-fit* rule, not a queue position: it says which band an instrument goal borrows, and `portfolio-manager` proposes a `goal-portfolio` decision to place it there. Sequencing then follows from Swarm Manager's own dependency signal, which is the correct mechanism — an instrument that unblocks a goal should be modeled as that goal's dependency, at which point ordering falls out of the existing algorithm rather than from a second ranking authority. If the dependency signal proves too weak to carry it, that is a Swarm Manager change raised through `capability-work`, not a reason to rank goals here.

## Goal vs backlog item

A **goal** is a multi-step effort with a shared outcome and typically multiple backlog items (fix/execute/research) under it, optionally partitioned into milestones. A **backlog item** is a single piece of work. If a proposal is scoped to one well-defined task, it's a backlog item, not a goal. (Entity vocabulary: swarm-manager's glossary and `path:scenarios/swarm-manager/docs/concepts/OPERATOR-JOURNEYS.md`.)

When `portfolio-manager` proposes a `goal-proposal`, the unit-of-work test is: does this warrant multiple dependent backlog items under one shared outcome, or would it fit as a single backlog item under an existing goal? If the latter, propose it as a backlog item.

## What this doc does NOT do

- **Does not rank individual goals.** Swarm Manager does that.
- **Does not enumerate goals.** See [ROADMAP.md](ROADMAP.md).
- **Does not define success metrics.** See [OUTCOMES_CHARTER.md](../evidence/OUTCOMES_CHARTER.md).
- **Does not replace human judgment at the vision walk.** This is a frame for `portfolio-manager` proposals; the operator resolves conflicts between criteria.

## Updating this doc

Changes go through approved decisions with context `goal-portfolio`. Drift-free authoring is more important than frequent updates — a portfolio philosophy that shifts every heartbeat isn't a philosophy. Expect meaningful revisions rarely (every few months, or when a phase boundary is crossed — e.g., first paying customer, default-alive reached), or when the prediction ledger shows a ranking criterion systematically mispredicting — measured miscalibration is the one fast lane to revision.
