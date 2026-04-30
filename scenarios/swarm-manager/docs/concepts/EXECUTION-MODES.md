# Execution Modes and Operating Modes

> **Status:** Concept document. Defines the operating-mode framework that governs *what unit of work an operator-and-agent pair operates on at one time*. `item-level` is the default implemented mode. `holistic-loop` and `phased-plan-drain` are registered non-default initiative modes whose execution runners are being implemented in phases; see [`docs/plans/swarm-manager-initiative-operating-mode-implementation.md`](../plans/swarm-manager-initiative-operating-mode-implementation.md).

## TL;DR

Swarm Manager now models execution methodology as an explicit initiative `mode`. A mode is not just a phase list; it owns the unit of work, phase graph, run strategy, artifact policy, prompt routing, AgentManager profile policy, backlog/audit reconciliation policy, metrics policy, locking behavior, and UI workspace surface.

| Mode | Unit of execution | Work shape it fits | When to use |
|------|------------------|--------------------|-------------|
| **item-level** (default) | One backlog item per agent run | Items are properly scoped, independent, and stable through execution | Items are right-sized, loosely coupled, reviewable in isolation |
| **holistic-loop** | The whole initiative through `investigate -> plan -> execute -> review -> replan` | Items are coupled, likely to shift mid-flight, mis-scoped, or only validatable as a system | Per-item execution would leave broken intermediate states; the right item shape is only knowable after investigation/execution |
| **phased-plan-drain** | The whole initiative through a stable sequential plan and accumulated handoffs | A large multi-phase plan should be drained by agents completing the earliest contiguous phase(s) they can fully finish | A plan already exists or can be prepared once, and continuity between handoffs matters more than parallelism |

The modes are not "small vs. big work" and they are not "operator-absent vs. operator-present" — they are different units of execution and validation, chosen based on work shape. Choosing the right mode for an initiative is a property of *how its work is shaped*, not of how much work there is or how much the operator is around to drive it.

## Operating-Mode Contract

The backend registry lives at [CODE: api/internal/operatingmode/registry.go]. It currently registers:

- `item-level` as the default bridge over the existing backlog-item execution/workshop/review flows.
- `holistic-loop` as an initiative-scoped, operator-gated loop using `swarm-manager/deep-work` for investigate/plan/execute and `swarm-manager/analysis` for review.
- `phased-plan-drain` as an initiative-scoped sequential handoff strategy using `swarm-manager/deep-work` for prepare/execute and `swarm-manager/analysis` for classify/review.

Initiative metadata now stores `mode` and `acceptance_criteria`; historical records normalize blank mode to `item-level`. Mode changes emit `initiative.mode_changed` events so stats and audit surfaces can observe adoption.

Mode phases are also registered in the prompt catalog by `(mode, phase)`, with stable Agent Activity purposes and AgentManager profile keys supplied by the operating-mode definition. The event log now has typed operating-mode phase, replan, and backlog-sync events; the stats surface includes a Modes tab for mode usage, phase runs, profile usage, replan/acceptance rates, and backlog reconciliation counts. The shared runner and phase APIs still need to call these primitives.

## Why this distinction exists

Swarm Manager's existing flow assumes **backlog items are the right unit of execution and validation**. Every primitive — items as the unit of execution, the workshop loop's 5-dimension readiness model, plan.md as the per-item execution spec, the review-decide handshake — is shaped to make work atomic enough that a single agent run can pick up *one* item, execute it, and hand back a discrete outcome. That assumption is load-bearing: it is what lets the operator review N initiatives in parallel by triaging discrete per-item decisions, and it is what gives execution bounded blast radius.

That assumption holds when items are correctly scoped, independent, and stable through execution. It breaks in several related shapes of work — most visibly when **backlog items are coupled by the very thing being changed**, but also when items are likely to shift mid-flight as new ground truth emerges, or when items are mis-scoped (too fragmented to be the efficient unit, or partitioned along wrong lines).

### The empirical cue (2026-04-27 sandboxing trap)

The two initiatives `agent-sandbox-audit-foundation` (10/10) and `protected-agent-sandboxing` (7/7) both involved restructuring how agent runs flow through the workspace sandbox — making sandboxing default, routing every run through it, adding lifecycle coordination, schema versioning, etc.

When the operator tried to complete these item-by-item inside swarm-manager, each completed item left the system in an *inconsistent intermediate state* for the items still in flight: changes to sandbox routing affected every other run, including the swarm-manager runs that were trying to execute the next backlog item. Swarm-manager itself stopped working effectively as a result.

The work was completed — but the operator had to step *outside* the swarm-manager harness to do it. The pattern that succeeded was:

1. Investigate both initiatives' actual current state in code (no execution)
2. Generate a consolidated plan covering large chunks of remaining work
3. Stage execution against that consolidated plan
4. Re-investigate after each execution wave; revise the plan; repeat

The work didn't fail because the items were big. It failed because **the items couldn't be validated in isolation** — only the system as a whole could be validated, after the *coupled* work had reached a coherent state.

The operator's continuous presence during the recovery was an artifact of having no in-harness support for this kind of work, not a requirement of the work itself. A structured `investigate → plan → execute → review` loop running inside swarm-manager would absorb the same work with the operator's normal async cadence — review findings between phases, accept/refine the plan, react to replan signals — the same shape of presence as today's backlog-item flow.

The plain reading of this: swarm-manager's backlog-item-level mode is the *right* mode when items are properly scoped, independent, and stable, and the *wrong* mode for tightly-coupled architectural work, work whose item shape will shift during execution, or work where the natural unit of validation crosses item boundaries.

Rather than treat the recovery pattern as "an escape hatch when swarm-manager fails," we recognize it as a **second first-class operating mode** with its own primitives — async-reviewable just like backlog-item mode, but operating on a different unit of work.

## The two modes in detail

### Backlog-item-level mode (current default)

- **Unit of execution:** one backlog item per agent run.
- **Lifecycle:** `backlog → researching → ready → queued → in_progress → completed/failed`.
- **Refinement primitive:** the workshop loop (rounds with 5-dimension readiness scoring; converges to `plan.md`).
- **Execution primitive:** Generator/Improver agent reads `plan.md`, executes, hands back a result.
- **Review primitive:** review round ratifies completion.
- **Operator interaction:** review the workshopped plan, accept or refine, queue, review the result — async between steps. The framework holds enough per-item context to advance work between operator touches.
- **Strengths:** parallelism (many items in flight at once); auditable per-item provenance; bounded blast radius (one item's bugs don't break others); per-item progress is legible without holding the whole initiative in memory.
- **Failure mode:** items coupled by a shared substrate produce broken intermediate states; work whose item shape will shift mid-execution thrashes the item graph; over-fragmented items waste cycles on per-item ceremony.

### Initiative-level mode (target)

- **Unit of execution:** the initiative, taken holistically.
- **Phases:** investigate → plan → execute → review → (replan ↻); item-level follow-ups for genuinely small remaining work after the loop converges.
- **Refinement primitive:** initiative-level workshop rounds (analogous to backlog-item workshop, but operating on the full member-item graph and producing an *initiative-level* plan).
- **Execution primitive:** an agent (or sequence of agents) executes against the initiative-level plan, touching whichever items the work covers; backlog items get marked done as plan milestones land, not as independent execution events.
- **Review primitive:** validate the initiative as a whole against its acceptance criteria.
- **Operator interaction:** review findings, accept/refine the initiative-level plan, react to replan signals as the loop iterates, ratify the final review — async between phases, the same shape as backlog-item mode. What changes is *what the operator is reviewing* (an initiative-level plan + cross-item findings) rather than how present they are. The framework holds context across rounds.
- **Strengths:** correct unit of validation for coupled work; lower replanning cost (one plan, not N plans); the investigation phase is a first-class step instead of being smuggled into per-item workshop; items can shift in scope as the work reveals what they actually are.
- **Failure mode:** loses parallelism; carries higher per-run cost; not appropriate when items are genuinely independent and stable.

#### The investigation phase has no analog at backlog-item scale

At backlog-item scale, "what is the current state?" is largely answered by the item's spec.json + its workshop history. The agent doesn't need an investigation pass — the workshop *is* the investigation, scoped to one item.

At initiative scale, the *cross-item ground truth* is not available anywhere in the existing primitives. What's in code today vs. what the items still describe vs. what previous handoffs assumed — these can drift across a multi-item initiative, and only a deliberate investigation pass surfaces the drift. Treating this as the workshop loop's first round (and pretending it's a normal round) misrepresents what's happening.

So initiative-level mode introduces investigation as an explicit first step:

```
investigate ──▶ plan ──▶ execute ──▶ review ──┐
     ▲                                          │
     └────── replan (with new findings) ────────┘

(item-level follow-ups for small remaining work after convergence)
```

#### Backlog items in initiative-level mode: tracking, not execution

When an initiative runs in initiative-level mode, its member backlog items do not disappear. They survive as **tracking and scope markers**: they record *what* the initiative claims to deliver, they let progress be reported as items-marked-done, they support partial cancellation, and they remain the addressable unit for cross-initiative dependencies.

What changes is that they are no longer *independent execution units*. Their `plan.md` (if present) is informational — supplanted by the initiative-level plan. Their workshop rounds (if any) are historical context, not active refinement loops. Marking items `completed` happens as the initiative's plan milestones cover them, not as standalone execution events.

This is the load-bearing distinction. We keep items as the unit of *visibility* and *scope* while changing the unit of *execution* and *validation*.

## When to use each mode

The right mode is a property of how the work is shaped, not its size. A 10-item initiative composed of independent SKU-level changes is appropriately backlog-item-level. A 3-item initiative whose items all touch the same auth middleware is appropriately initiative-level.

### Use backlog-item-level mode when

- Items are right-sized for one agent run each
- Items are loosely coupled (one item's completion does not invalidate the others' assumptions)
- Items are reviewable in isolation
- Items are stable — their scope and definition won't need to shift as execution proceeds
- Parallelism is valuable (many items in flight at once)

### Use initiative-level mode when

- Items are coupled by a shared substrate that all of them are changing
- Intermediate states (after item N but before item N+M) leave the system inconsistent
- Items are likely to shift mid-execution as new ground truth emerges
- Items are mis-scoped — too fragmented to be the efficient unit, or partitioned along wrong lines
- The natural unit of validation is "does the system as a whole work"
- The right plan can only be authored *after* investigating cross-item ground truth
- The work requires holistic thinking that doesn't decompose cleanly into per-item plans
- Replanning is expected: the first plan will be wrong about something material that only execution will reveal

### Switching modes mid-initiative

The mode is a property of an initiative at a point in time; it is not immutable. An initiative may begin in backlog-item mode, hit the coupled-substrate failure (or any other shape mismatch), and be promoted to initiative-level mode. Conversely, after an initiative-level plan converges, residual independent work may be drained back to backlog-item mode for parallel finish-out.

The mode-switch is itself an operator-chosen action and should be supported as a first-class operation, not a workaround.

## Companion: rescoping affordances inside backlog-item mode

A separate but related capability gap, addressed in Plan A ([`docs/plans/swarm-manager-initiative-feedback-ux.md`](../plans/swarm-manager-initiative-feedback-ux.md)), is **rescoping inside backlog-item mode**. Even when backlog-item mode is the right choice, the items as initially-scoped may be wrong: too granular, too coarse, or incorrectly partitioned. The initiative-feedback flow today supports most rescoping ops (`add_item`, `update_item`, `change_priority`, `change_status`, `add_edge`, `remove_edge`, `move_initiative`, `archive_item`, `interrupt_in_progress`, `split_item`) but is asymmetric: split is supported, merge/consolidate is not. The UI also gives the operator no surfaced affordances for these — they are discoverable only by the agent reading the skill prompt.

Plan A handles that asymmetry and the UX gap. It is a *backlog-item-mode improvement*; it does not replace the need for initiative-level mode.

## Open questions (deferred to Plan B for resolution)

1. **Workshop schema at initiative scale** — does initiative-level workshop reuse the 5-dimension readiness model (`problem_clarity`, `scope_defined`, `approach_solid`, `testable`, `risk_awareness`) verbatim, extend it with cross-item dimensions (e.g., `coupling_understood`, `acceptance_at_system_level`), or replace it?
2. **Plan artifact location** — where does the initiative-level plan live? Candidates: a sibling `plan.md` at the initiative folder; a dedicated `initiative-plan.md`; reuse of the existing `orchestration-summary.md` shape; or a new file. Trade-off is discoverability vs. collision with existing per-item `plan.md`.
3. **Investigation deliverable shape** — does investigation produce a separate `findings.md`, or is it folded into the first plan revision? Both have precedent; tooling implications differ.
4. **Item status propagation** — when initiative-level plan milestones cover items, who marks them `completed`? The agent at execution time, the operator at review time, or a status-derive layer that infers completion from acceptance test pass?
5. **Mode metadata storage** — does `mode: "item-level" | "initiative-level"` live in initiative metadata, or is it implicit (presence of an initiative-level plan = initiative-level mode)?
6. **Mode-switch protocol** — when an initiative is promoted from item-level to initiative-level mid-flight, what happens to in-flight per-item executions? Cancel them? Let them finish? Both have plausible defaults.
7. **Review-loop reuse vs. new** — does initiative-level review use the existing `swarm-manager-initiative-review` skill or a new one? Existing review is decision-oriented (member items needing decide actions); initiative-level review is "did the initiative as-a-whole satisfy its acceptance."
8. **Acceptance-criterion model** — backlog items have `acceptance_allow` / `acceptance_deny` globs. Does an initiative gain its own acceptance criteria (likely yes), and where do they live?
9. **Parallel-agent boundary** — when initiative-level mode is active, can multiple agents run on the same initiative concurrently (e.g., investigation + drafting in parallel) or is the lock single-agent like the existing feedback flow?
10. **Cost / budget surfacing** — initiative-level runs are longer and more expensive than item-level runs. Does the UI surface a different cost prompt before kickoff? (Probably yes; defer to Plan B.)

These are the questions Plan B must answer before implementation begins.

## Why both modes are first-class

A common temptation is to collapse the modes — to argue that initiative-level mode is "just a bigger workshop loop" or that backlog-item mode is "just initiative-level mode applied to one item." Both collapses are wrong:

- Initiative-level mode has an investigation phase that backlog-item mode lacks. Forcing investigation into a workshop round mis-shapes the workshop.
- Backlog-item mode has parallelism and bounded blast radius that initiative-level mode lacks by design. Forcing initiative-level mode onto independent-item work loses both.
- The two modes operate on different *units of work* (item vs. initiative) with different *validation surfaces* (per-item acceptance vs. system-level acceptance). That difference doesn't reduce to a multiplier on the same loop.

The modes solve different problems for different work shapes. Both are necessary; neither subsumes the other. Recognizing this is the conceptual move that turns "operator stepped outside the harness" from a failure mode into a mode-switch.

## References

- [`docs/concepts/ARCHITECTURE.md`](./ARCHITECTURE.md) — Backlog-item-level mode is the architecture this document builds on; both modes coexist within the same staging-and-review framing.
- [`docs/guides/workshop-workflow.md`](../guides/workshop-workflow.md) — The backlog-item-level workshop loop, readiness model, and plan.md handoff. The initiative-level loop is documented as a contrast in this document; its implementation is in Plan B.
- [`docs/plans/swarm-manager-initiative-feedback-ux.md`](../plans/swarm-manager-initiative-feedback-ux.md) — Plan A: rescoping affordances inside backlog-item mode (companion, not replacement).
- [`docs/plans/swarm-manager-initiative-operating-mode-implementation.md`](../plans/swarm-manager-initiative-operating-mode-implementation.md) — Plan B: implementation of initiative-level mode.
- [CODE: api/internal/proposals/types.go] — current ops the feedback skill can propose (10 ops; merge missing, see Plan A).
- [CODE: api/internal/initiatives/service.go] — current initiative model (lightweight grouping; no execution unit yet).

## Changelog

- **2026-04-30** — Added mode-aware prompt catalog resolution, stable phase activity purposes, typed operating-mode event/stat contracts, and the first Modes stats UI surface.
- **2026-04-28** — Initial document. Authored during walk #5 explicit divergence after the 2026-04-27 sandboxing trap surfaced "items are the right unit of execution and validation" as a load-bearing assumption that doesn't hold for coupled or shifting work.
