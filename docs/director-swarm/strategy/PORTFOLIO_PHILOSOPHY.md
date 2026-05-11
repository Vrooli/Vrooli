# Portfolio Philosophy

How `director-swarm` and the operator think about the initiative portfolio. This doc is the frame `portfolio-manager` anchors its proposals against; other teams (monetization, meta-optimization) can also read it to align with current director-swarm priorities.

## Ranking criteria

Initiatives are ranked — and proposed for `active now` status — against these factors, in order:

### 1. Revenue (primary)

Vrooli is pre-revenue. The top priority is closing that gap — specifically, getting **paid apps into users' hands** under the business bundle. Initiatives that directly advance a paid-delivery path (desktop release, Stripe integration, bundle headliners approaching readiness) outrank everything else.

"Revenue" does NOT mean the literal fastest path to publishing a paid app. See criterion 2.

### 2. Safety, quality, and auditability of what we publish

We will not ship broken software to paying users. Refund churn from premature paid launches is a worse outcome than a later launch. This criterion covers:

- **Well-tested apps** before paid release.
- **Deployment pipeline reliability** — redeployment, rollback, crash-log and feedback capture.
- **Auditability** — being able to trace what changed, why, and whether it worked.
- **Reusable quality capabilities** — every initiative that hardens the production pipeline compounds; the fifth paid app ships faster than the first because the tooling exists.

This is not a secondary consideration to revenue; it is a hard co-requirement. An initiative that accelerates revenue but degrades the quality pipeline is rejected.

### 3. Meta-optimization and platform self-improvement

Vrooli's agents, scenarios, and tooling must measurably get sharper over time. If agents are frequently erring, not finishing their work, inefficient, or if technical debt accumulates unchecked, revenue work eventually grinds. A sustained portion of portfolio capacity goes to platform self-improvement (agent reliability, tooling, skill/scenario conversion, run-quality) because the downstream leverage is enormous.

This comes after 1 and 2 but is never zero. A healthy portfolio always has at least one active initiative from this category.

## Concurrency

**There is no cap on active initiatives.** Swarm Manager is the synthesis point for every idea the operator has for Vrooli; initiatives can accumulate faster than they're implemented, and that's expected. Storing months (or a year+) of initiative-level planning is correct use of Swarm Manager.

The operator works the portfolio by moving between active initiatives based on where attention is best spent in the moment — when one initiative is blocked waiting on an agent run, the operator picks up another. Hard caps would fight this workflow.

**Priority ordering is Swarm Manager's job, not this doc's.** Swarm Manager's priority algorithm ranks initiatives based on its own signals (priority field, dependencies, backlog item state). `portfolio-manager` defers to that ordering for tactical sequencing, and only proposes portfolio-level decisions (`initiative-portfolio`) when an initiative's **category fit** against the criteria above is what's being adjusted — not its position in the queue.

## Initiative vs backlog item

An **initiative** is a multi-step effort with a shared outcome and typically multiple backlog items (fix/execute/research) under it. A **backlog item** is a single piece of work. If a proposal is scoped to one well-defined task, it's a backlog item, not an initiative.

When `portfolio-manager` proposes an `initiative-proposal`, the unit-of-work test is: does this warrant multiple dependent backlog items under one shared outcome, or would it fit as a single backlog item under an existing initiative? If the latter, propose it as a backlog item.

## What this doc does NOT do

- **Does not rank individual initiatives.** Swarm Manager does that.
- **Does not enumerate initiatives.** See [ROADMAP.md](ROADMAP.md).
- **Does not define success metrics.** See [OUTCOMES_CHARTER.md](../evidence/OUTCOMES_CHARTER.md).
- **Does not replace human judgment at the vision walk.** This is a frame for `portfolio-manager` proposals; the operator resolves conflicts between criteria.

## Updating this doc

Changes go through approved decisions with context `initiative-portfolio`. Drift-free authoring is more important than frequent updates — a portfolio philosophy that shifts every heartbeat isn't a philosophy. Expect meaningful revisions rarely (every few months, or when a phase boundary is crossed — e.g., first paying customer, default-alive reached).
