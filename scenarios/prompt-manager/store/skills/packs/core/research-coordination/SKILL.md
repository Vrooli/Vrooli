---
name: "research-coordination"
description: "Coordinate sustained research: curate evidence-based directions, give investigators room to develop them, challenge weak returns, and reconcile outcomes, authority and cost."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["research", "experiment", "coordination", "supervision"]
  icon: "compass"
  status: "active"
  revision: 14
  createdAt: "2026-10-26T00:00:00Z"
  updatedAt: "2026-11-03T16:01:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["search-hub query", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Practice focus: Research coordination

Keep research advancing toward the operator's decisions. Match continuation to the operator's commission: a sustained research window calls for successive useful checks, while a bounded question may finish early when its consequential frontier is exhausted.

### Scope

Use for a broad investigation needing synthesis or several commissioned studies sharing decisions or resources. Use `adaptive-research` directly for one bounded investigation. The same chat may investigate and coordinate; a separate supervisor or worker is not required. Start additional agents only when the operator or applicable instructions authorize them.

Workers use `adaptive-research` to develop questions and `local-experimentation` for comparisons. Architecture design, external app construction and maintained implementation follow the actual commission; research does not impose a universal read-only boundary or approval gate. This skill owns selection across studies, shared constraints, review and synthesis. Execution, credentials, admission, cancellation and host repair remain with their existing owners. No delivery team, private respawn loop or new scheduler is required.

### Phase 1: Commission the work

**Entry:** The operator has commissioned research outcomes and an allowance, or has asked only for a prepared folder.

1. Recall relevant work with `search-hub query "<research intent>" --type library,record,skill,doc`. Use the [research entry point](../../../../../../../docs/internal/research-experimentation/README.md) to locate the commission, applicable topic and current feedback. Read decisive prior evidence selectively; do not send the full history to each investigator.
2. Apply commission precedence before reading old cohort status: a fresh explicit operator commission supersedes earlier stopped, completed, or merely still-marked-active commissions. Create a new cohort record and, when native goals are used, a distinct new goal. Record the new identifiers and actual dispatch time, then read them back and confirm the goal is the one being supervised. A cohort file alone does not prove a fresh native goal or active allowance. Use prior records as evidence only; never inherit their clock or resume them because their goal/thread still appears active. Resume an earlier commission only when the operator explicitly says to resume/continue it and its owner confirms it is still resumable with allowance remaining. A fresh commission may continue the same topic, but it gets a new start/deadline and a new record. Pass only the fresh effort reference to `research-supervision-read`; never point a fresh commission's supervision read at the predecessor.
3. Name the outcome, quality standard, authority and stop conditions. Include in-scope implementation and cross-owner repairs; novelty, size or uncommitted files do not create an approval gate. Use the smallest entry structure that carries the current direction ([guidance](references/direction.md#choose-artifacts-and-startup-context)); give investigators its rationale, evidence, freedom and starting points, not a queue of tiny checks. Distinguish a sustained window from a bounded answer. Form open directions from the objective and evidence, and identify operator-selected versus suggested leads. One experiment is a checkpoint, not completion of a sustained study.
4. If the operator asks only to prepare a folder, follow [prepare without commissioning](references/direction.md#prepare-without-commissioning). Do not start a clock, goal, worker or effect. A proposed allowance is not an active budget. Stop after producing the reviewable handoff; do not run launch or supervision steps until the operator commissions the work.
5. Reuse the [single-chat launch and recovery method](../local-experimentation/references/adaptive.md#start-or-resume-one-research-chat). Each study retains its named entry artifact, checkpoint, current commission's allowance and effect ownership. Commission a coherent direction that can span several checks and sessions. Starting checks guide entry; investigators choose subsequent methods and may revise the hypothesis within scope. Verify receiving workspace access and actual goal state when dispatching.
6. For multiple studies, use one new cohort record for each fresh commission under the canonical research home's `cohorts/` directory. Record study/thread identities, topic/run paths, actual dispatch time, deadline, shared resource order, next decision triggers and the shared-summary writer. Record observed worker start, end, exit reason, successor and uncovered intervals when available; distinguish unknown from idle. Link worker records; do not copy their histories. For one study, keep coordination in its existing record. Link the shared operator-attention view and name its supervisor/writer; reconcile real pending requests using [operator attention](references/operator-attention.md).
7. Include workers, supervision, preparation and recovery in one current-commission allowance across phases/follow-ups. Read back actual role/model/effort and usage units through existing owners; catalog defaults alone do not prove launch selection. Agree material notifications and distinguish enforced, cooperative and unmeasured limits.
8. Name who is responsible for active supervision and how that person or agent will actually receive updates and act during the window. When the commission permits operator escalations, qualify both notification producers before dispatch using [operator attention](references/operator-attention.md#resolve-authority-once-at-startup). A `notification-sending` grant permits `notification.send` only. Agent decision asks require the separate recipient-bound `notification-decision-asking` grant for `notification.ask`; direct asks by a human still require verified owner identity. The Agent Manager event route requires a real enrolled effort and its matching goal home. Never substitute a push notification, in-chat question or invented effort for a decision ask. Present any owner-generated approval URL in the current conversation; do not wait until the first decision. A ready attention item is not a delivered notification or ask. After the qualified route is used, verify its ask/notification record and receipt. Every decision includes a built-in Other answer with a required free-text note. Source-bound decisions keep their owner-defined typed options, and Other is always a non-approval response. Use genuine choices and a recommendation with rationale when evidence supports one; informational notices do not invent choices. Recipients can dismiss without answering, and requesters can withdraw their direct asks; neither outcome is consent. Include the complete operator-actions file path and a short summary of other actionable items in every notice or ask. If no qualified producer exists, report the exact blocker and keep the decision pending. Surface an unsupported policy as an owner integration gap. Verify that the supervisor is running and can observe and steer the worker; a skill, native goal, or submitted prompt is not a supervisor. If continuous supervision or recovery is not available, disclose that at launch and do not claim it is in place.
   For unattended sustained work, exercise the selected owner continuation or wake path and record the observed wake and recovery limits. Record actual identities, review time, quota source and stop rules; a proposed timer or a running imported transcript is not proof of a live supervisor. Preserve the fresh commission's deadline and reconcile effects before any recovery.

**Exit/artifact:** The records identify commissioned outcomes, owners, allowances, shared constraints and how to recover context. No launch is claimed from prompt submission alone.

### Phase 2: Advance and challenge

**Entry:** A study returns evidence, reports closure, or encounters a changed condition.

1. Record the selected studies, research paths, original cutoff, native thread identities and managed/imported run identities. Reconcile their relationships and uncertain effects through the owners. Workers keep routine progress in their existing checkpoints. Missing observations stay unknown.
2. Choose review timing from findings, blockers, pending events and the approaching cutoff. Use supported owner waits or notifications. Increase the interval when work progresses reliably. A prompt timer does not prove wakeup or enforce expiry. See [conditional observations](references/observations.md#identity-wakeup-and-compound-review) for identity, wakeup and compound-read routes.
3. Read the changed checkpoint and decisive evidence. Inspect native state or bounded events when a decision needs them. A healthy process or empty attention list does not prove useful research. Use a fitting owner program for multi-source supervision when it supplies the needed evidence; inspect direct owner details for partial results or consequential decisions. Do not collect unchanged status or build a reporting framework to fill the research window.
4. Review direction using the [direction guidance](references/direction.md#develop-and-reconsider-directions). Apply the worker's [frontier value test](../local-experimentation/references/adaptive.md#choose-the-next-research-question) across studies. Workers choose local checks; the coordinator selects cross-study priorities and challenges pivotal conclusions. Do not turn optional capability work into an unlimited continuation.
   Review early, after the first meaningful blocker or repeated unchanged checkpoints, and at material transitions. Compare the user's intended outcomes with working capabilities delivered, the remaining path to integration, recent evidence, infrastructure friction, staffing gaps and available usage. A busy worker or growing record is not evidence of progress. When the same state is repeatedly confirmed without narrowing the cause, direct one distinguishing test, authorized repair or workaround, or a useful pivot; reassess its effect rather than waiting for the deadline. Record unknown cost and attribution as unknown.
   In a sustained commission, challenge each proposed study closure against the full authorized topic, its prepared brief, and independent alternatives. A failed candidate, unavailable tool, or single bounded result does not exhaust that topic. Consider a changed-condition repair, different tool or scenario, counterexample, transfer check, or another selected topic; select the best useful permitted direction and give the investigator concrete starting points. Record why alternatives cannot proceed before ending the cohort early. Do not invent activity merely to keep an agent busy.
5. Choose the first applicable row. Preserve the earlier evidence and evaluation when directing a follow-up.

| Evidence or condition | Coordination decision |
|---|---|
| The study's closure cutoff has arrived | Start no further research. Close or cancel owned work through its owners and report remaining effects. |
| An effect or its ownership is unresolved | Reconcile with its owner before duplicate or dependent work. Continue independent permitted checks. |
| A proposed check changes the objective, exceeds the remaining allowance, or requires new authority | Retain it as a proposal; do not execute it under the old commission. |
| A route failed but another method or valuable direction fits | Separate method failure from refutation. Commission the permitted direction with starting evidence. Retry an effect only under its declared changed-condition policy. |
| A finding or early closure leaves a consequential permitted check inside the outcome and budget | Record a prospective amendment if scope changes, then update the working brief with the rationale, starting evidence and investigator freedom. Retain failed gates and original clocks. |
| No consequential permitted check fits | Accept bounded closure with the unresolved questions and reasons. Do not restart merely because time remains. |

6. Before treating work as human-blocked, triage the exact dependency with [operator attention](references/operator-attention.md). Keep ready asks, technical prerequisites and future gates distinct. Surface ready requests through the authorized notification owner, reconcile replies and resume dependent work; continue independent useful research.
7. Challenge the evidence most likely to change the decision: replay a join, inspect an original source, test a counterexample or request an authorized independent review. Correct, narrow or reject a conclusion when the challenge defeats it. Label self-review and missing independent review explicitly.
8. Coordinate heavy work through verified cleanup handoffs followed by fresh owner admission. A coordination file or host snapshot grants no capacity. Keep source/offline work moving where independent gates pass.
9. For a blocker, identify its owning layer and attempt an authorized repair, workaround or alternative. An owner contract is an engineering problem unless a human-only decision actually governs it. Separate any genuine human gate from independent work and keep that work staffed. Follow the [shared checkout and integration guidance](../local-experimentation/references/adaptive.md#shared-checkout-repair-and-integration).
10. At material checkpoints, keep the working brief and current view connected to evidence. Generalize repeated operations only after their inputs and decisions settle; use existing program owners and validate missing-source behavior. For native state, inventory, retained usage or browser evidence, read the relevant route in [conditional observations](references/observations.md).

**Exit/artifact:** Each intervention has evidence, a decision and an exact next check or closure reason. The shared frontier reflects cross-study implications, not just a collection of worker status reports.

### Phase 3: Synthesize and improve

**Entry:** The commissioned outcomes close or the allowance reaches its closure margin.

1. Reconcile scientific findings, execution/cleanup, native goal state and discovery visibility separately. Verify owned effects through their owners; retain unresolved cleanup. Reconcile operator requests, delivery/answer state, remaining gates and the actual resumption/pickup owner. An answer does not reset the allowance for its originating commission; a later fresh commission follows Phase 1 precedence and starts its own allowance. Use the worker closure method rather than duplicating its checklist.
2. Update topic findings, feedback dispositions and next questions through the designated writer. Produce one cumulative synthesis linking each decision to evidence and identifying remaining alternatives. Separate proposed integration from implemented behavior.
3. Reconcile cumulative worker/supervision usage including failures and post-goal follow-ups. Separate uncached input, cached input and output, billing and subscription allowance; unknown remains unknown. Measure accepted outcomes/rework alongside usage, not response size alone.
4. Use the [shared persistence/discovery method](../local-experimentation/references/record.md#persist-and-discover) once for the coordination result. Apply `docs/agent-system/PROMOTION_LADDER.md`: judgment stays in skills, repeated operations move to existing owners, and verified replacements remove old workarounds.

**Exit/artifact:** A new chat can find the question, supported conclusions, unresolved frontier, current ownership and exact next commission without rereading the full conversation. Completion does not qualify automatic recovery or hard budget enforcement.

### Anti-patterns

| Mistake | Replacement |
|---|---|
| Delegate once and collect reports at the deadline | Review material findings and challenge remaining consequential questions |
| Make the coordinator choose every worker action | Commission a direction; investigators choose local methods and follow implications |
| Accurate small checks accumulate without advancing the mission | Reconsider the framing, sources and role before extending that trajectory |
| Spend the remaining horizon checking unchanged state | Wait for a real pending event or close with a handoff |
| Repair every discovered platform defect before useful research | Fix authorized blockers at their owners; retain other defects and continue independent work |

### Troubleshooting & Edge Cases

| Observation | Action |
|---|---|
| Chat activity disagrees with native goal state | Preserve both observations; use the launch/recovery method. Manual continuation is not automatic recovery proof. |
| A new coordinator takes over | Read topic/run guides and the shared checkpoint; reconcile live owners before resuming. Preserve budgets and one writer. |
| Human judgment is pending | Prepare the exact actionable request in the attention view, use a qualified decision owner, and continue independent work. Reconcile the answer and actual resumption. |
| Notification sender is refused or the grant is pending | Preserve the exact refusal/request and show the attention item or approval URL in the current conversation. Mark Notification Hub delivery as pending/unavailable, not sent. Qualify the actual producer/identity path; channel readiness and in-chat visibility do not supply sender authority or delivery. |
| Hard expiry or recovery is required but unsupported | Report that execution route unqualified; continue only independent work whose requirements are met. |
