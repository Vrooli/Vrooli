# Execution Modes and Operating Modes

> **Status:** Implemented concept document. Defines the operating-mode framework that governs *what unit of work an operator-and-agent pair operates on at one time*. `item-level` is the default mode; `holistic-loop` and `phased-plan-drain` are registered non-default initiative modes with backend phase runners.

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

Initiative metadata stores `mode` and `acceptance_criteria`; historical records normalize blank mode to `item-level`. New initiatives always start in `item-level`, and mode changes only occur through the operating-mode switch endpoint. Mode changes emit `initiative.mode_changed` events so stats and audit surfaces can observe adoption.

Mode phases are also registered in the prompt catalog by `(mode, phase)`, with stable registry-authored activity purpose tokens and AgentManager profile keys supplied by the operating-mode definition. API startup validates that every profile key referenced by the registry was returned by AgentManager scenario profile reconciliation, and every registry-owned key must stay in the `swarm-manager/` namespace. The backend computes phase action state from the registered graph, transition rules, and completed round history; UI controls render that state rather than deciding sequencing locally. Phase start fails closed when the prompt catalog entry is missing, mismatched, unavailable from prompt-manager, or renders empty content, so repo-writing initiative agents never run with generic fallback prose. The event log has typed operating-mode phase, replan, and backlog-sync events; operating-mode backlog mutations also carry structured source metadata on the affected backlog event so history can reconstruct the initiating mode, phase, round, run, and requester. The stats surface includes a Modes tab for mode usage, phase runs, profile usage, replan/acceptance rates, and backlog reconciliation counts. Replan and acceptance rates are counted from registry-declared phase metrics policy; `operating_mode.replan_needed` remains a timeline signal and does not increment the metric a second time.

Operator-facing mode catalogs are derived from the same backend registry through `GET /api/v1/operating-modes`. UI and CLI mode selectors must consume that read model instead of carrying parallel built-in mode option lists.

Each `Definition` also carries decision-support metadata (`BestFor`, `NotFor`, `Tradeoffs`, `WhenInDoubtPickInstead`) that the picker, details page, and how-to-choose dialog render directly. The when-to-use prose in this document is canonical; the structured fields are short callouts derived from it. The registry validator enforces that all three lists are non-empty for every registered mode and that `WhenInDoubtPickInstead`, when set, references a registered mode and not self.

Future static modes should follow [DOC: ../internal/OPERATING-MODE-AUTHORING.md]. Mode definitions own transitions, result bindings, output contracts, prompt metadata, metrics semantics, capabilities, and decision-support metadata; shared framework code should not gain mode-specific behavior branches for a new methodology.

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

## The modes in detail

### Backlog-item-level mode (current default)

- **Unit of execution:** one backlog item per agent run.
- **Lifecycle:** `backlog → researching → ready → queued → in_progress → completed/failed`.
- **Refinement primitive:** the workshop loop (rounds with 5-dimension readiness scoring; converges to `plan.md`).
- **Execution primitive:** Generator/Improver agent reads `plan.md`, executes, hands back a result.
- **Review primitive:** review round ratifies completion.
- **Operator interaction:** review the workshopped plan, accept or refine, queue, review the result — async between steps. The framework holds enough per-item context to advance work between operator touches.
- **Strengths:** parallelism (many items in flight at once); auditable per-item provenance; bounded blast radius (one item's bugs don't break others); per-item progress is legible without holding the whole initiative in memory.
- **Failure mode:** items coupled by a shared substrate produce broken intermediate states; work whose item shape will shift mid-execution thrashes the item graph; over-fragmented items waste cycles on per-item ceremony.

### Holistic loop mode

- **Unit of execution:** the initiative, taken holistically.
- **Phases:** investigate → plan → execute → review, with execute either advancing to review or looping back to investigate when its result declares `replan_needed=true`.
- **Refinement primitive:** durable mode rounds that operate on the full member-item graph and produce initiative-level artifacts such as `findings.md` and `initiative-plan.md`.
- **Execution primitive:** an agent executes against the initiative-level plan, touching whichever items the work covers; backlog items get marked done through audited mode reconciliation, not as independent execution events.
- **Review primitive:** validate the initiative as a whole against its acceptance criteria.
- **Operator interaction:** review findings, accept/refine the initiative-level plan, react to replan signals as the loop iterates, ratify the final review — async between phases, the same shape as backlog-item mode. What changes is *what the operator is reviewing* (an initiative-level plan + cross-item findings) rather than how present they are. The framework holds context across rounds.
- **Strengths:** correct unit of validation for coupled work; lower replanning cost (one plan, not N plans); the investigation phase is a first-class step instead of being smuggled into per-item workshop; items can shift in scope as the work reveals what they actually are.
- **Failure mode:** loses parallelism; carries higher per-run cost; not appropriate when items are genuinely independent and stable.

See [DOC: docs/guides/holistic-loop-mode.md] for the operator workflow and
[CODE: api/internal/operatingmode/state.go] for enforced transitions.

### Phased plan drain mode

- **Unit of execution:** the initiative, drained by a stable sequential plan and accumulated handoffs.
- **Phases:** prepare_plan → execute_next → classify_progress, then either execute_next, prepare_plan, or review based on the classifier's progress decision.
- **Refinement primitive:** a durable `phased-plan.md` plus round handoffs. The plan is prepared once, then revised only when `classify_progress` decides `replan`.
- **Execution primitive:** each execute round completes the earliest contiguous slice of the plan it can safely finish and hands off state for the next round.
- **Review primitive:** validate the completed initiative against acceptance criteria after progress is classified as complete.
- **Operator interaction:** review the prepared plan, let agents drain execution slices, inspect progress decisions, and approve review when the plan has converged.
- **Strengths:** continuity across long sequential work, less planning churn than a holistic loop, and explicit progress classification between execution slices.
- **Failure mode:** unsuitable when the plan is not stable enough to drain or when parallel independent item execution is more valuable.

See [DOC: docs/guides/phased-plan-drain-mode.md] for the operator workflow and
[CODE: api/internal/operatingmode/backlog_reconciler.go] for audited backlog
reconciliation.

#### The investigation phase has no analog at backlog-item scale

At backlog-item scale, "what is the current state?" is largely answered by the item's spec.json + its workshop history. The agent doesn't need an investigation pass — the workshop *is* the investigation, scoped to one item.

At initiative scale, the *cross-item ground truth* is not available anywhere in the existing primitives. What's in code today vs. what the items still describe vs. what previous handoffs assumed — these can drift across a multi-item initiative, and only a deliberate investigation pass surfaces the drift. Treating this as the workshop loop's first round (and pretending it's a normal round) misrepresents what's happening.

So holistic-loop mode introduces investigation as an explicit first step:

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

## Understanding a mode: the Flow tab

The operating-mode detail page has a **Flow** tab (formerly labeled "Execution",
which overloaded the term with backlog execution records). Flow is the
operator-facing place to *understand* how a phase mode behaves — what an agent is
told to do, what each phase reads, what it emits, and why the next phase is
chosen — without reading raw JSON first.

Flow renders **one shared phase viewer** with four concern tabs —
**Instructions / Reads / Emits / Transition** — and a **data-source control**
that swaps the fill without changing the view. The same viewer backs the
**Phases** tab (in Contract source), so a phase is understood through one surface
everywhere. The three sources:

- **Contract** — the phase's static contract. Instructions shows the agent skill
  template with its `{{VARIABLE}}` slots still unfilled; Reads/Emits/Transition
  show what the phase is *defined* to consume, produce, and route to. No
  initiative data is substituted.
- **Simulation preset** — deterministic, in-memory walks of the phase graph
  against illustrative data. Presets seed different phase outputs to exercise
  real transition branches (clean pass, replan, continue, blocked, non-accepting
  review, backlog reconcile) but never spawn agents, acquire locks, or persist
  initiative state. Presets are defined server-side in
  [CODE: api/internal/operatingmode/simulation.go]; the render endpoint fills the
  prompt lazily per step.
- **Live round** — the actual rounds recorded for a linked initiative. This is
  real data.

The **Instructions** tab is the most explanatory artifact: the literal agent
prompt for the selected source, plus the agent profile that would run it. For
Simulation and Live the prompt is rendered server-side through the *same*
`renderPhasePrompt` seam the real spawn path uses
([CODE: api/internal/operatingmode/prompt.go]) — a Go parity test asserts the
preview is byte-identical to what an agent receives. When the prompt-manager seam
is unavailable the tab degrades to the resolved variable map rather than erroring.

**Reads** shows one card per prompt variable (`MEMBER_ITEMS_JSON`,
`MODE_ARTIFACTS_JSON`, `PRIOR_ROUNDS_JSON`, `ACCEPTANCE_CRITERIA`) — the same
vocabulary in every source. **Emits** shows the declared schema (Contract) or the
actual structured result (Simulation/Live), with raw payloads behind a "View raw
payload" disclosure. **Transition** shows every declared outgoing route
(Contract) or the single fired transition (Simulation/Live), explained from the
same backend guard a live round uses. The full vocabulary is explained in-app via
the Flow tab's **Guide** button. Because presets, prompts, and transitions are
backend-owned, adding a mode or a branch surfaces in Flow without mode-specific UI.

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

The mode-switch is itself an operator-chosen action and is supported as a first-class lifecycle operation. Generic initiative create/update APIs do not mutate mode. Switching into non-default modes handles active item-level executions explicitly; switching out of non-default modes is blocked while a mode round is active.

## Companion: rescoping affordances inside backlog-item mode

A separate but related capability gap is **rescoping inside backlog-item mode**. Even when backlog-item mode is the right choice, the items as initially-scoped may be wrong: too granular, too coarse, or incorrectly partitioned. The initiative-feedback flow today supports most rescoping ops (`add_item`, `update_item`, `change_priority`, `change_status`, `add_edge`, `remove_edge`, `move_initiative`, `archive_item`, `interrupt_in_progress`, `split_item`) but is asymmetric: split is supported, merge/consolidate is not. The UI also gives the operator no surfaced affordances for these — they are discoverable only by the agent reading the skill prompt.

Rescoping affordances handle that asymmetry and the UX gap. They are a *backlog-item-mode improvement*; they do not replace the need for initiative-level mode.

## Implemented decisions

1. Initiative mode is stored on initiative metadata, but public initiative
   create/update APIs do not mutate it; `POST /operating-mode/switch` is the
   single lifecycle boundary.
2. Initiative-scoped modes use an exclusive initiative lock, so only one mode
   round is active at a time.
3. Acceptance criteria live on initiative metadata and are required before
   review phases can start.
4. Holistic-loop artifacts live under `modes/holistic-loop/`, including
   `findings.md`, `initiative-plan.md`, and append-only round envelopes.
5. Phased-plan-drain artifacts live under `modes/phased-plan-drain/`, including
   `phased-plan.md`, `progress.json`, and append-only round envelopes.
6. Backlog item status propagation happens through run-id-validated
   operating-mode reconciliation endpoints with structured audit metadata.
7. Prompt skills are registered by `(mode, phase)` and fail closed if the prompt
   catalog or prompt-manager cannot supply the exact registered skill.
8. Completed phase rounds must satisfy the registry's output contract before
   they can advance the graph: required structured result envelopes, artifacts,
   handoffs, progress decisions, review verdicts, and replan permissions are
   validated before artifacts are written [CODE: api/internal/operatingmode/registry.go].
9. Round-control and backlog-reconciliation operations are scoped to
   non-default modes. `item-level` is never a fallback for refresh, cancel,
   complete-items, or apply-backlog-sync [CODE: api/internal/operatingmode/handler.go].

## Why both modes are first-class

A common temptation is to collapse the modes — to argue that initiative-level mode is "just a bigger workshop loop" or that backlog-item mode is "just initiative-level mode applied to one item." Both collapses are wrong:

- Initiative-level mode has an investigation phase that backlog-item mode lacks. Forcing investigation into a workshop round mis-shapes the workshop.
- Backlog-item mode has parallelism and bounded blast radius that initiative-level mode lacks by design. Forcing initiative-level mode onto independent-item work loses both.
- The two modes operate on different *units of work* (item vs. initiative) with different *validation surfaces* (per-item acceptance vs. system-level acceptance). That difference doesn't reduce to a multiplier on the same loop.

The modes solve different problems for different work shapes. Both are necessary; neither subsumes the other. Recognizing this is the conceptual move that turns "operator stepped outside the harness" from a failure mode into a mode-switch.

## References

- [`docs/concepts/ARCHITECTURE.md`](./ARCHITECTURE.md) — Backlog-item-level mode is the architecture this document builds on; both modes coexist within the same staging-and-review framing.
- [`docs/guides/workshop-workflow.md`](../guides/workshop-workflow.md) — The backlog-item-level workshop loop, readiness model, and plan.md handoff.
- [`docs/guides/holistic-loop-mode.md`](../guides/holistic-loop-mode.md) — Operator workflow for holistic-loop mode.
- [`docs/guides/phased-plan-drain-mode.md`](../guides/phased-plan-drain-mode.md) — Operator workflow for phased-plan-drain mode.
- [CODE: api/internal/operatingmode/registry.go] — registry core, validation, and mode lookup.
- [CODE: api/internal/operatingmode/mode_holistic_loop.go] — holistic-loop mode definition.
- [CODE: api/internal/operatingmode/mode_phased_plan_drain.go] — phased-plan-drain mode definition.
- [DOC: ../internal/OPERATING-MODE-AUTHORING.md] — internal guide for adding future static modes.
- [CODE: api/internal/operatingmode/state.go] — backend phase action state.
- [CODE: api/internal/operatingmode/backlog_reconciler.go] — audited backlog reconciliation.
- [CODE: api/internal/initiatives/service.go] — initiative metadata and lifecycle-only mode update seam.

## Changelog

- **2026-04-30** — Added mode-aware prompt catalog resolution, stable phase activity purposes, typed operating-mode event/stat contracts, and the first Modes stats UI surface.
- **2026-05-01** — Documented the registry-owned authoring architecture: mode definitions own transition rules, result bindings, prompt metadata, metrics semantics, capabilities, and phase purpose tokens.
- **2026-04-28** — Initial document. Authored during walk #5 explicit divergence after the 2026-04-27 sandboxing trap surfaced "items are the right unit of execution and validation" as a load-bearing assumption that doesn't hold for coupled or shifting work.
