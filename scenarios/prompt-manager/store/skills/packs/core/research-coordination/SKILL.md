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

1. Establish a supervisor-owned review procedure. Record the authorized studies, research paths, original cutoff, native thread IDs and any managed or imported run IDs. Resolve their relationships through the owners; record ambiguous, stale and missing observations as unknown. A native goal, imported history and live owner operation are different evidence. Reconcile uncertain effects before replacement or retry. Workers retain routine progress in their existing records; do not require a new event format. Give them a coherent permitted batch and reserve direct contact for a consequential decision, required help or unresolved shared effect.

   If the installed Agent Manager supports it, `agent-manager run import-identity --harness <kind> --session-id <exact-id> --json` resolves imported-run provenance without paging through the run list. An ambiguous match needs reconciliation; `not_observed` only describes the current imported-run database and does not prove source absence or freshness. If unavailable, use other owner evidence and preserve the unknown.
2. Choose the next review time from the work and the available execution owner. Review sooner during setup, a material blocker, uncertain effects or an approaching cutoff; lengthen the interval as work progresses reliably. Use a supported owner wait or park when it can resume the supervisor, and record its actual behavior. If no such route is available, use an available interactive review and disclose the continuity limit. A prompt timer does not enforce a deadline or prove a wakeup. Do not reread unchanged state or send status-only messages merely to fill time.

   For an eligible Agent Manager managed run, its optional `review-timer` park uses an absolute review time and the owning run's verified identity; see `scenarios/agent-manager/docs/internal/TEMPORAL-FLOWS.md` for the command, timeout and recovery limits. Confirm that the installed owner supports it. Attached native sessions need another supported continuation or an explicit continuity limitation.
3. At a review, gather changed native state, bounded run events, friction, receipt availability and checkpoints from the owners that are available. When suited to the selected subjects, discover `program-runtime.research-supervision-read` for a bounded read of explicit native thread and Agent Manager run IDs. Its `attention` list points to explicit blocked or failed states and event backlogs; an empty list is not an all-clear. Its optional per-run event cursors return page counts and a next cursor without event payloads; retain `has_more` and inspect relevant owner events when the counts warrant it. It does not map identities, establish event freshness, observe every event, or decide whether the research is progressing. Use cursors and source watermarks when supported; inspect workspace navigation and cumulative usage less often or when material. Report attention, no attention detected, or unknown with source identity, freshness, truncation and missing coverage. A quiet or empty source does not prove an agent is healthy or that no event occurred. Read the changed checkpoint and decisive evidence; send at most one intervention for a material change.

   Own the review method as part of the commission. When repeated reads or interpretation are materially costly, discover and reuse fitting governed programs; when none fits, create or update a bounded program at its canonical owner using `prompt-manager skill read program-runtime` and its authoring guidance. Compose existing programs when their contracts fit. Give the program explicit subjects, bounded output and honest stale, partial and unavailable states; preserve direct owner reads for decisive evidence. Validate representative normal, blocked and missing-source cases before relying on a change, record its identity and evidence in the coordination record, and compare supervisor and worker usage and outcomes when available. Do not spend the original research allowance on repeated method changes without evidence of a useful improvement, or treat a shorter packet as proof of lower cost or equivalent research quality.
4. Apply the worker's [frontier value test](../local-experimentation/references/adaptive.md#choose-the-next-research-question) across studies. Workers choose local checks; the coordinator selects cross-study priorities and challenges pivotal conclusions. Do not turn optional capability work into an unlimited continuation.
   Review early, after the first meaningful blocker or repeated unchanged checkpoints, and at material transitions. Compare the user's intended outcomes with recent decisions, useful evidence, repeated reads, infrastructure friction, and available usage. A busy worker or growing record is not evidence of progress. When the same state is repeatedly confirmed without narrowing the cause, direct one distinguishing test, authorized repair or workaround, or a useful pivot; reassess its effect rather than waiting for the deadline. Record unknown cost and attribution as unknown.
5. Choose the first applicable row. Preserve the earlier evidence and evaluation when directing a follow-up.

| Evidence or condition | Coordination decision |
|---|---|
| The study's closure cutoff has arrived | Start no further research. Close or cancel owned work through its owners and report remaining effects. |
| An effect or its ownership is unresolved | Reconcile with its owner before duplicate or dependent work. Continue independent permitted checks. |
| A proposed check changes the objective, exceeds the remaining allowance, or requires new authority | Retain it as a proposal; do not execute it under the old commission. |
| A route failed but an independent consequential check fits | Direct that check. Retry the failed effect only under its declared changed-condition policy. |
| A finding or early closure leaves a consequential permitted check inside the outcome and budget | Record a prospective amendment if scope changes, then direct one concrete check with expected decision impact. Retain failed gates and original clocks. |
| No consequential permitted check fits | Accept bounded closure with the unresolved questions and reasons. Do not restart merely because time remains. |

6. Challenge the evidence most likely to change the decision: replay a join, inspect an original source, test a counterexample or request an authorized independent review. Correct, narrow or reject a conclusion when the challenge defeats it. Label self-review and missing independent review explicitly.
7. Coordinate heavy work through verified cleanup handoffs followed by fresh owner admission. A coordination file or host snapshot grants no capacity. Keep source/offline work moving where independent gates pass.
8. For recurring blockers, identify the owning layer. Complete an authorized blocking repair or commissioned improvement there with focused validation and shared-writer coordination; otherwise retain the defect and next check through existing intake. Follow the [shared checkout and integration guidance](../local-experimentation/references/adaptive.md#shared-checkout-repair-and-integration).
9. Evolve the review procedure from observed recurring work. Discover and reuse suitable governed programs; prototype a small bounded read when no existing operation fits. Test the whole review against direct owner reads, including stale or missing sources, errors, evidence and cleanup. Generalize a proven reusable shape and remove replaced steps. On an unavailable program or owner, use a supported alternate read and retain the missing coverage; never make one program a prerequisite. Do not build a private scheduler or reporting framework. Compare actual usage and accepted outcomes before claiming savings. At material checkpoints and handoff, keep each run's guide and current view navigable and connected to preserved evidence. Large files, broken links or duplicate summaries are review signals, not authority to delete evidence.

For retained native usage across explicitly selected threads, discover `web-console.native-usage-summary`; pass the existing authorized session, thread-ID list and time bounds. It returns the owner totals and per-thread statuses with unknown pricing; it does not observe live goal state or enforce a deadline. Compare actual evidence and usage before claiming cost savings.

For a proportional workspace review, discover `program-runtime.research-workspace-review` and pass the explicit authorized absolute research root. It returns bounded presence and size signals, not research quality; partial or unavailable coverage stays unknown. Direct reads remain a fallback. The older promoted `research_workspace_inventory@1` depends on its session workspace and incorrectly advertises `session_id` as a program input; do not use that input as a workspace selector.

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
