---
name: "scenario-work-ladder"
description: "Locate the highest broken layer of an existing scenario - contract (W0), obligations (W1), evidence (W2), implementation (W3) - using a runnable gate per rung, then route the repair to the skill that owns that rung. W2 is the L0-L4 traceability ladder and W3 is the R0-R4 maturity ladder, sized so that high rungs get large changes; W0 and W1 are the contract rungs neither one covers."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["practice","scenario","ladder","contract","prd","requirements","routing","methodology"]
  icon: "layers"
  status: "active"
  revision: 7
  createdAt: "2026-07-27T00:00:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "vrooli"]
    commands: ["prompt-manager skill read", "vrooli scenario", "test-genie report"]
  origin:
    kind: "authored"
---
## Practice focus: Scenario Work Ladder

Locate the highest broken layer in an existing scenario's stack, then repair that layer before any layer under it. Work on the layer that is wrong, not the one that is easiest to measure. Each layer is a claim about the layer under it, so a false claim makes everything beneath it unverifiable.

| Rung | Layer | Claim | Gate | Owner of the repair |
|---|---|---|---|---|
| W0 | Contract: `PRD.md` operational targets | what the scenario must be | compare governing outcomes with the PRD (below) | `prd-authoring` |
| W1 | Obligations: `requirements/` registry | these promises satisfy the contract | `business-health validate scenario <name>` | `prd-authoring` |
| W2 | Evidence: validation refs and tests (the L0-L4 traceability ladder) | the promises are kept | `vrooli scenario requirements validate <name>` | `requirements-traceability-steer` |
| W3 | Implementation: code, docs, UI (the R0-R4 ladder below) | the behavior exists and holds | scoped Test Genie phases (`docs/TESTING.md` §"Ordinary iteration versus certification") | `scientific-debugging` for an unknown cause, otherwise the steer skill for the layer |

Two agents running the same gate on the same scenario land on the same rung. `business-health validate` can pass a contract that is consistent and describes the wrong product, so W1 conformance is not W0 truth.

### Choose the route

| Evidence and purpose | Next step |
|---|---|
| Localized defect, expected behavior established | Reproduce, fix (`scientific-debugging` if the cause is unknown), validate under `docs/TESTING.md`. Do not claim unmeasured W0-W2 gates passed. |
| A sensor names a broken layer | Confirm it with that layer's gate |
| Expected behavior, obligations or ownership disputed | Inspect the highest implicated layer; a contract dispute starts at W0 |
| Full readiness or maturity review | Start at W0 and descend, stopping at the first failure |

A defect that crosses durable state or an external effect: trace who stores each state, what commits it, where the side effect happens and what retry evidence exists, then pick the rung from what is missing (outcome W0, promise W1, passing evidence W2, observed violation of a proven obligation W3). Separate state owners alone prove nothing; a valid retry can converge them.

Stop at the first failing gate and run no gate under it. Repair only that rung; work under a broken rung is discarded when the rung above changes. Re-measure the affected gates afterward (restart at W0 when the contract or obligations changed), and expect a repair to reopen a rung under it.

### The W0 gate

Governing-target applicability is in `docs/agent-system/SCENARIO_DEVELOPMENT.md` §"Authorization and change classification". A name match finds context, not authority; no separate Swarm goal is required.

1. Read the supplied approved contract or operator instruction and what it incorporates, plus referenced goals and accepted decisions. For a full review also list goals naming the scenario (`swarm-manager goals list`, then `goals get --name <goal>`, with JSON when the human view omits content).
2. Establish each candidate's applicability and any explicit supersession before treating it as a constraint.
3. Read the PRD targets at every priority and the scenario's decisions document (`docs/internal/DECISIONS.md`), then compare both directions.

| Condition | Disposition |
|---|---|
| An applicable directive requires a capability the PRD omits or contradicts | report the missing or conflicting outcome |
| PRD priority contradicts an explicit release obligation | report the priority conflict; a request alone does not imply P0 |
| An accepted decision supersedes a target that still states the old position | report the stale target with the supersession |
| A matching goal is historical, proposed or unrelated | keep it as context; check incorporated standing constraints |
| Two applicable sources conflict without a recorded resolution | request a decision; never choose by recency |
| No governing target is supplied or discoverable | W0 is unverifiable; do not create a goal to make it pass |
| A required owner read is unavailable | keep the constraint unverified; do not claim the gate passed |

W0 evidence is the compared quotations and their applicability, not a successful list command. When W0 fails, stop that dependency chain and route authorized repair through `prd-authoring`; a target amendment still needs approval. Independent authorized work continues.

### W3: the maturity rungs (R0-R4)

R0-R4 measure conformance to the PRD, so they mean something only once W0-W2 are true. Test Genie reports the lowest unsatisfied rung. Low rungs are hard gates; high rungs are soft, but a reopened lower gate pulls you back.

| Rung | Satisfied when | Change size |
|---|---|---|
| R0 Runnable and green | build passes, no failing tests, no blocker or error findings | small: make it run |
| R1 Safe and standards-clean | no high or critical security, standards clean, deps sane | small to medium hardening |
| R2 Evolvable architecture | no import cycles, structure clean, contracts and docs map present | large: structural refactors |
| R3 Features hardened | coverage, tidiness, UI, visual, performance clean | medium to large: build and harden features |
| R4 Capability progression | operational targets at or above the profile's threshold | large: new capability, missing features |

Walk R0 to R4 and stop at the first unsatisfied rung. At R0 and R1 prefer the smallest change that clears the gate. At R2 to R4 the honest fix is often big (extract the package, build the endpoint, add the whole flow); a lint pass does not close a missing target, and shrinking it into a cosmetic edit is the commonest failure (R3). Closing a rung can reopen a higher one (new R4 work reopens R3 coverage), which is the loop working. Every change obeys `improvement-do-and-dont`.

### Record the rung

For read-only reviews, return the rung and evidence in the report and change nothing. For authorized repair, reuse the active workflow's record, or add a `## Work ladder` section to the scenario's one problems file (`docs/internal/PROBLEMS.md` if present, else `docs/PROBLEMS.md`; never a second one) with the rung, the evidence (quoted or command output), the blocker and the date. Leave existing entries alone. A rung without evidence is not a record.

### Anti-patterns

| Anti-pattern | Why it fails | Better |
|---|---|---|
| Ladder-on-a-lie: climbing R0-R4 against a PRD that contradicts the approved goal | the rungs build what the operator directed you to remove | resolve W0 first |
| Conformance as truth: treating `business-health validate` PASSED as W0 evidence | it checks structure, not intent | compare directives and targets |
| Editing the contract to match drifted code | the contract then describes the code, so nothing can contradict it | decide which side is wrong; the traceability steer's anti-gaming bans apply |
| Rung inversion or shopping: tests for W2 while W1 is open, or picking the rung you prefer | evidence is discarded when the rung above is repaired; the real gap never closes | walk in order, stop at the first unsatisfied rung |
| Gate skipping or widening: declaring a rung satisfied from a document, or running gates under a failed one | documents state intent; gates under a failed rung describe a stack about to change | run the gate, record its output, stop at the first failure |
| Timid substitution: cosmetic work while a higher rung is open | a passing suite reads as progress while the scenario stalls at tidy but incomplete | work at the rung's scale |
| Measuring once | a cleared rung can reopen | re-measure after each change |

Out of scope: scenarios that do not exist yet (`ecosystem-fit`, then `scenario-generation`); resource maturity (`docs/resources/maturity-migration.md`).

### Troubleshooting & Edge Cases

| Situation | Response |
|---|---|
| `goals get` prints a title and empty `Results:`, or a listing omits descriptions | read the JSON; tell an empty result from a failed read |
| A goal defers to another document | read that document too; the goal plus what it incorporates is the contract side |
| No `PRD.md` | report the missing contract; authoring goes to `prd-authoring`, a review stops with the finding |
| `business-health validate` passes on a starter-template PRD | W1 passes and W0 fails; the P0 targets name no capability the goal names |
| A gate command does not exist for this scenario's shape | record the gap and treat the rung as failing, not passing |
