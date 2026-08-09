# Objectives

**Status:** canon. The operator's statement of what Vrooli is for. Owned by `director-swarm`; authored by the operator, never by an agent. Agents may flag drift, propose an objective, or report a coverage gap; they do not write this file.

This is the top layer of the goal hierarchy. Everything below it — outcome categories, team roster, goals, milestones, backlog items — should be derivable from it, and anything that is not derivable from it is either serving an unstated objective or serving none.

## Why this file exists

Before this file, Vrooli's answer to "what is this system for" was reconstructed bottom-up from instrumentation. The outcome categories in [`../evidence/OUTCOMES_CHARTER.md`](../evidence/OUTCOMES_CHARTER.md) are Command Center dashboard ids; the team contribution map was written against those categories; the team roster was written against that map. Intent never entered, so three categories carried `pending-operator-input` for what "good" means and stayed that way — the question is not answerable at that layer. You cannot say what good engineering velocity looks like without knowing what the velocity is for.

The practical symptom: the operator's personal-life domains appeared in exactly one place in the repository — the `lifestyle` bundle in `path:docs/monetization/catalogs/skus/base/lifestyle.md`, as a market segment, gated behind a revenue trigger. A selling condition was standing in for a building decision because no surface asked the building question.

## Vocabulary

`Objective` is a primitive; its definition and its boundary against `Goal` live in `path:docs/agent-system/PRIMITIVES.md` § Objective. Read it there. In short: an objective is durable and qualitative and there are fewer than a dozen; a goal is time-boxed and measurable and there are many per objective.

Every Swarm Manager goal declares a parent objective. An objective with no goals beneath it is unstaffed intent; a goal with no parent objective is unattributed work.

## The objectives

Objective ids are stable and citable. Do not renumber a retired objective's id; mark it retired and leave the id burned.

| # | Objective | Class | Served by | Evidence source |
|---|---|---|---|---|
| `T1` | **Income.** Vrooli sustains itself and its operator financially. | terminal | `team:monetization` (primary), `team:marketing-crew` (supporting) | Command Center `ledger`, `broadcast` |
| `T2` | **Personal agency.** Health, finances, and household run with materially less of the operator's attention. | terminal | *none* (`pending-capability`) | *none* (`pending-capability`) |
| `T3` | **Contribution.** Other operators can run this system, and it outlives any one operator's involvement. | terminal | `team:marketing-crew` (partial — OSS surface only) | Command Center `broadcast` (all metrics `gap`) |
| `I1` | **Capability compounding.** Every scenario built becomes a permanent capability that makes later work cheaper. | instrumental | `team:director-swarm`, `team:scenario-qa` | Command Center `hive`, `forge` |
| `I2` | **Coherence.** The system stays reliable and stays reasonable-about as it grows. | instrumental | `team:infra-health`, `team:meta-optimization` | Command Center `mission-control`; `prompt-manager graph audit` |
| `I3` | **Enablement.** Every supervising loop has the instruments it needs to actually close. | instrumental | *none* (`pending-operator-input` — see §"Enablement has no owner") | the capability ladder derived below |

## Terminal and instrumental

**Terminal** objectives state what the operator wants the world to be like. **Instrumental** objectives state what the system must become in order to serve them. The distinction is not academic bookkeeping — it is the guard against the failure mode this architecture invites.

A self-improving system whose only written objectives are about its own improvement will improve itself indefinitely, because those are the only objectives it can score against. Instrumental objectives must always be justified by a terminal one. `I1`, `I2`, and `I3` are in this file only because `T1`, `T2`, and `T3` are.

**Where the instrumental set is explained in narrative form.** This table is the declaration; [`RECURSIVE_SELF_IMPROVEMENT.md`](../../concepts/RECURSIVE_SELF_IMPROVEMENT.md) is the prose spine that says how `I1`, `I2`, and `I3` are pursued as one loop — the four projections, the maturation gradient, and the control topology. Edit the ids here; edit the story there.

**`T1` is terminal, not instrumental.** Revenue funds the system, which makes monetization look like a supporting function. It is not. "Earn a living from my business" is a thing the operator wants; it only appears self-referential because this operator's business is Vrooli itself. Another operator would run a different business and staff the same team shape against it. `team:marketing-crew` sits on the same side for the same reason.

## Operator-invariant and operator-specific

This file is the **only** operator-specific layer in the agent system. The six teams, all of `path:docs/agent-system/`, the loops, the canon, and the primitives are operator-invariant: another operator adopting Vrooli would keep every one of them unchanged and replace this file.

That property is a design constraint, not an observation. Do not let operator-specific intent leak downward into team canon, and do not let framework mechanics leak upward into this file. If an objective can only be stated by naming a Vrooli-specific mechanism, it is probably a goal.

## The coverage rule

Both directions are required, and both are mechanically checkable. This is what keeps this file from becoming a mission statement.

1. **Downward.** Every objective traces to at least one team, or carries a dated gap marker naming what is missing.
2. **Upward.** Every team, and every outcome category in [`../evidence/OUTCOMES_CHARTER.md`](../evidence/OUTCOMES_CHARTER.md), traces to at least one objective.

A violation in either direction is a finding, not a style note. Downward violations mean stated intent nobody is serving. Upward violations mean effort nobody asked for. The sensor and its deadband are registered in `path:docs/agent-system/FRAMEWORK_HEALTH.md`; the audit that reads them is `prompt-manager skill read agent-system-audit`.

**The team half of both directions is declared and validated.** Each team names the objectives it serves in `team.json::objectivesServed`, and `prompt-manager graph objectives` checks that every id resolves to a row here, that every team this table names declares the objective back, and that no team traces to nothing. Changing the `Served by` column below therefore moves a sensor rather than waiting for someone to notice.

Two constraints on that mechanism, both load-bearing:

- **This file stays prose.** It is the only operator-specific layer in the agent system and must remain readable as a statement of intent. What is declared elsewhere is the *edge*, never the objective; there is no `objectives.json`, and adding one would put the operator's intent behind a schema.
- **Role qualifiers are optional on purpose.** The `Served by` column qualifies some contributions (`primary`, `supporting`, `partial`) and leaves others open. A team may not declare a role that contradicts a qualifier stated here, and the validator does not require one where this table states none — inventing a role to satisfy a schema would assert something the operator did not write.

The outcome-category half of the upward direction is not mechanical: categories are Command Center dashboard ids in the charter, not a store surface, so `agent-system-audit` §"Phase 4" still scores that one by reading.

## Evidence routing

An objective without an evidence source cannot be scored, and an unscoreable objective decays into a slogan. Where the evidence comes from depends on the class:

| Objective class | Evidence comes from | Who owns the instrument |
|---|---|---|
| instrumental | Command Center categories plus the framework sensors in `path:docs/agent-system/FRAMEWORK_HEALTH.md` | `team:infra-health`, `team:meta-optimization` |
| terminal — business | Command Center `ledger`, LPBS revenue surfaces | `team:monetization` |
| terminal — personal | the scenario that serves the objective reports its own measure | that scenario |
| *any objective whose evidence source does not exist* | a capability-ladder entry (below) | `team:director-swarm` |

Command Center is the instrument for instrumental objectives. That is correct and deliberate. It is **not** the universal outcome surface — a terminal objective in a personal-life domain is measured by the scenario that serves it, not by a platform dashboard. Treating Command Center as the only outcome surface is what made `T2` unrepresentable in the first place.

## The capability ladder

The ladder is **derived, not curated**. It is the set of missing evidence sources and missing actuators named above, ordered by which objective they unblock. Nobody maintains it by hand; it falls out of the coverage rule and the evidence routing table, and its current contents are read by running the audit rather than by reading a list here (`OPERATING_GRAPHS.md` §"State belongs to scenarios").

The ladder exists because enabling work has no privileged lane in the portfolio today. A sensor-building initiative competes against feature work on equal footing and loses, because its payoff is indirect. The ranking rule that corrects this: **an instrument ranks ahead of the loop it enables, not beside it.** A control loop whose sensor does not exist is open-loop no matter how well its policy is written — the same rule `path:docs/infra-health/strategy/RELIABILITY_TARGETS.md` already applies within one team, raised to the portfolio.

## Enablement has no owner

`I3` is stated and unowned. That is deliberate and visible rather than quietly absent: the objective is real, the work is real, and no team currently ranks it. The disposition is `pending-operator-input` — a lane inside `team:director-swarm` is the cheapest option that still lets instrument work outrank the loops it unblocks, but establishing it is an operator decision, not an agent's.

Until it resolves, capability-ladder items route through the existing `outcome-gap` work type and are ranked by `portfolio-manager` alongside everything else, which is exactly the defect `I3` names.

## Deferred and unstaffed objectives

`T2` is stated at full weight and staffed by nobody. This is intentional. An unstated objective is an unaudited one — the coverage rule cannot report a hole that was never declared, which is precisely how personal agency stayed invisible while being written down as a sellable bundle.

Deferring the *work* is a legitimate operator decision. The sequencing trigger is `pending-operator-input`. It must be a capability or intent condition, not a revenue condition inherited from the SKU catalog: `lifestyle`'s "≥50 paying subscribers" answers *when do we sell this*, and that is a sound monetization trigger which does not transfer to *when do we build this*. The two questions are independent and only the first currently has an owner.

## What this file does NOT do

- **Does not set thresholds.** "MRR > $X by date Y" belongs in a goal's acceptance criteria or a decision's prediction block, not here. Objectives state direction; goals state amounts.
- **Does not track status.** Live goal status lives in Swarm Manager; measured outcome status lives in Command Center and the charter's sensor map.
- **Does not rank work.** Ranking criteria live in [`PORTFOLIO_PHILOSOPHY.md`](PORTFOLIO_PHILOSOPHY.md); sequencing lives in [`ROADMAP.md`](ROADMAP.md). This file says what the ranking is *for*.
- **Does not replace `VISION.md`.** The vision is the narrative north star and the argument for why any of this matters. This file is the checkable projection of it: few, id-addressable, and joined to the team roster.

## Updating this file

Objectives change by operator decision through the `vision-update` work type, the same gate that governs `path:VISION.md` — they are the same kind of truth at two levels of resolution. Agents raise `outcome-direction` or `capability-work` decisions when the coverage rule reports a hole; they do not resolve it by editing this file.

Adding or retiring an objective obliges a re-check of both coverage directions. An objective retired without re-checking upward coverage leaves teams working toward nothing.
