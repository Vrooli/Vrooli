---
name: "research-coordination"
description: "Coordinate a broad research effort or several experiments: commission bounded studies, challenge findings and early closure, select consequential follow-ups, resolve shared blockers, and synthesize evidence without delivery-team machinery."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["research", "experiment", "coordination", "supervision"]
  icon: "compass"
  status: "active"
  revision: 2
  createdAt: "2026-10-26T00:00:00Z"
  updatedAt: "2026-10-27T00:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["search-hub query", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Practice focus: Research coordination

Keep research advancing toward the operator's decisions. Commission useful checks, challenge their conclusions, and direct the next check from the evidence instead of filling a time allowance.

### Scope

Use for a broad investigation needing synthesis or several commissioned studies sharing decisions or resources. Use `adaptive-research` directly for one bounded investigation. The same chat may investigate and coordinate; a separate supervisor or worker is not required. Start additional agents only when the operator or applicable instructions authorize them.

Workers use `adaptive-research` to choose follow-ups and `local-experimentation` for comparisons. This skill owns selection across studies, shared constraints, review and synthesis. Execution, credentials, admission, cancellation and host repair remain with their existing owners. No delivery team, private respawn loop or new scheduler is required.

### Phase 1: Commission the work

**Entry:** The operator has authorized research outcomes and an allowance.

1. Recall relevant work with `search-hub query "<research intent>" --type library,record,skill,doc`. Read the [research entry point](../../../../../../../docs/internal/research-experimentation/README.md), applicable topic guides, prior findings and feedback.
2. Name the user decisions, outcome evaluator and evidence that could change them. Use the [frontier value test](../local-experimentation/references/adaptive.md#choose-the-next-research-question) before scaling. Split independent questions; keep dependent checks together.
3. Reuse the [single-chat launch and recovery method](../local-experimentation/references/adaptive.md#start-or-resume-one-research-chat). Each study retains its own brief, checkpoint, original allowance and effect ownership. Verify receiving workspace access and actual goal state when dispatching.
4. For multiple studies, use one existing cohort record under the canonical research home's `cohorts/` directory, or create one there if none exists. Record study/thread identities, topic/run paths, deadlines, shared resource order, next decision triggers and the shared-summary writer. Link worker records; do not copy their histories. For one study, keep coordination in its existing record.
5. Include workers, supervision, preparation and recovery in one commission allowance across phases/follow-ups. Read back actual role/model/effort and usage units through existing owners; catalog defaults alone do not prove launch selection. Agree material notifications and distinguish enforced, cooperative and unmeasured limits.
6. Name who is responsible for active supervision and how that person or agent will actually receive updates and act during the window. Verify that owner is running and can observe and steer the worker; a skill, native goal, or submitted prompt is not a supervisor. If continuous supervision or recovery is not available, disclose that at launch and do not claim it is in place.

**Exit/artifact:** The records identify commissioned outcomes, owners, allowances, shared constraints and how to recover context. No launch is claimed from prompt submission alone.

### Phase 2: Advance and challenge

**Entry:** A study returns evidence, reports closure, or encounters a changed condition.

1. Read its changed checkpoint and decisive evidence. Reconcile actual chat/goal and live-operation state before intervention using the launch method. Send at most one intervention for the material change; silence alone is not a stall. Use bounded notifications or owner waits rather than an idle observer goal.
2. Apply the worker's [frontier value test](../local-experimentation/references/adaptive.md#choose-the-next-research-question) across studies. Workers choose local checks; the coordinator selects cross-study priorities and challenges pivotal conclusions. Do not turn optional capability work into an unlimited continuation.
   Review early, after the first meaningful blocker or repeated unchanged checkpoints, and at material transitions. Compare the user's intended outcomes with recent decisions, useful evidence, repeated reads, infrastructure friction, and available usage. A busy worker or growing record is not evidence of progress. When the same state is repeatedly confirmed without narrowing the cause, direct one distinguishing test, authorized repair or workaround, or a useful pivot; reassess its effect rather than waiting for the deadline. Record unknown cost and attribution as unknown.
3. Choose the first applicable row. Preserve the earlier evidence and evaluation when directing a follow-up.

| Evidence or condition | Coordination decision |
|---|---|
| The study's closure cutoff has arrived | Start no further research. Close or cancel owned work through its owners and report remaining effects. |
| An effect or its ownership is unresolved | Reconcile with its owner before duplicate or dependent work. Continue independent permitted checks. |
| A proposed check changes the objective, exceeds the remaining allowance, or requires new authority | Retain it as a proposal; do not execute it under the old commission. |
| A route failed but an independent consequential check fits | Direct that check. Retry the failed effect only under its declared changed-condition policy. |
| A finding or early closure leaves a consequential permitted check inside the outcome and budget | Record a prospective amendment if scope changes, then direct one concrete check with expected decision impact. Retain failed gates and original clocks. |
| No consequential permitted check fits | Accept bounded closure with the unresolved questions and reasons. Do not restart merely because time remains. |

4. Challenge the evidence most likely to change the decision: replay a join, inspect an original source, test a counterexample or request an authorized independent review. Correct, narrow or reject a conclusion when the challenge defeats it. Label self-review and missing independent review explicitly.
5. Coordinate heavy work through verified cleanup handoffs followed by fresh owner admission. A coordination file or host snapshot grants no capacity. Keep source/offline work moving where independent gates pass.
6. For recurring blockers, identify the owning layer. Complete an authorized blocking repair or commissioned improvement there with focused validation and shared-writer coordination; otherwise retain the defect and next check through existing intake. Follow the [shared checkout and integration guidance](../local-experimentation/references/adaptive.md#shared-checkout-repair-and-integration). Reuse suitable governed programs for repeated reads. Do not turn the research into an unbounded platform rebuild.
7. At material checkpoints and handoff, check that each run's guide and current state remain navigable, concise and connected to preserved evidence and feedback. The investigator owns its run; direct a proportional cleanup or coordinate a nonconflicting repair when it drifts. For repeated multi-step supervision, discover and reuse a suitable Program Runtime program; when none fits, adapt or author a bounded program and fix its owner or binding gaps when authorized and worthwhile. Test the actual workflow and generalize a proven reusable shape as appropriate, without turning research into a platform rebuild. Large files, broken links or duplicate summaries are review signals, not proof of wasted context or authority to delete evidence. Compare actual usage, completeness and outcomes before claiming savings.

For retained native usage across explicitly selected threads, discover `web-console.native-usage-summary`; pass the existing authorized session, thread-ID list and time bounds. It returns the owner totals and per-thread statuses with unknown pricing; it does not observe live goal state or enforce a deadline. Compare actual evidence and usage before claiming cost savings.

For native thread and goal observations, discover `web-console.native-thread-state` when the serving owner supports it. Supply the authorized session and up to sixteen explicit thread IDs. An `ok` envelope means the reads succeeded, not that the goals succeeded: inspect each goal status, including blocked or budget-limited goals. An unavailable thread or goal read is not an idle or completed goal. This read does not interrupt work, prove owned-effect cleanup, or enforce a deadline. Reconcile material decisions with the native owner; validate the serving version and compare the full supervision workflow before claiming savings.

For a cross-topic file inventory, a Program Runtime session rooted at the research home can call `lib.research_workspace_inventory()`. It groups path aliases and reports bounded size and missing-file signals; historical layouts and deliberately retained long reports need judgment. It does not inspect native thread state or validate evidence. Reclaim the session after use; the absolute-directory workspace is not a security sandbox.

For browser baseline evidence, discover `browser-automation-studio.capture-surface` when one capture fits the question: it returns a durable execution identity and same-execution screenshot and DOM-tree references. Preserve and verify the receipt before comparison; a DOM tree is not a computed accessibility snapshot, and a returned reference is not a visual acceptance verdict or a substitute for a physical-device capture.

**Exit/artifact:** Each intervention has evidence, a decision and an exact next check or closure reason. The shared frontier reflects cross-study implications, not just a collection of worker status reports.

### Phase 3: Synthesize and improve

**Entry:** The commissioned outcomes close or the allowance reaches its closure margin.

1. Reconcile scientific findings, execution/cleanup, native goal state and discovery visibility separately. Verify owned effects through their owners; retain unresolved cleanup. Use the worker closure method rather than duplicating its checklist.
2. Update topic findings, feedback dispositions and next questions through the designated writer. Produce one cumulative synthesis linking each decision to evidence and identifying remaining alternatives. Separate proposed integration from implemented behavior.
3. Reconcile cumulative worker/supervision usage including failures and post-goal follow-ups. Separate uncached input, cached input and output, billing and subscription allowance; unknown remains unknown. Measure accepted outcomes/rework alongside usage, not response size alone.
4. Use the [shared persistence/discovery method](../local-experimentation/references/record.md#persist-and-discover) once for the coordination result. Apply `docs/agent-system/PROMOTION_LADDER.md`: judgment stays in skills, repeated operations move to existing owners, and verified replacements remove old workarounds.

**Exit/artifact:** A new chat can find the question, supported conclusions, unresolved frontier, current ownership and exact next commission without rereading the full conversation. Completion does not qualify automatic recovery or hard budget enforcement.

### Anti-patterns

| Mistake | Replacement |
|---|---|
| Delegate once and collect reports at the deadline | Review material findings and challenge remaining consequential questions |
| Make the coordinator choose every worker action | Workers advance local frontiers; coordinator resolves cross-study gaps |
| Spend the remaining horizon checking unchanged state | Wait for a real pending event or close with a handoff |
| Repair every discovered platform defect before useful research | Fix authorized blockers at their owners; retain other defects and continue independent work |

### Troubleshooting & Edge Cases

| Observation | Action |
|---|---|
| Chat activity disagrees with native goal state | Preserve both observations; use the launch/recovery method. Manual continuation is not automatic recovery proof. |
| A new coordinator takes over | Read topic/run guides and the shared checkpoint; reconcile live owners before resuming. Preserve budgets and one writer. |
| Human judgment is pending | Retain a reviewable artifact and continue other permitted checks; do not invent acceptance or wait idle. |
| Hard expiry or recovery is required but unsupported | Report that execution route unqualified; continue only independent work whose requirements are met. |
