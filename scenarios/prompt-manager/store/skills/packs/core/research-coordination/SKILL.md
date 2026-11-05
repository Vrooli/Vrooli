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
  revision: 26
  createdAt: "2026-10-26T00:00:00Z"
  updatedAt: "2026-11-05T00:28:27Z"
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

Workers use `adaptive-research` to develop questions and `local-experimentation` for comparisons. Read registered skills with `prompt-manager skill read`; if unavailable, read the canonical `SKILL.md` in the repository when accessible and record that source. A skill-registry outage is not a human blocker or a reason to stop independent work. An assignment that names a skill is not evidence the worker read it. Architecture design, external app construction and maintained implementation follow the actual commission; research does not impose a universal read-only boundary or approval gate. This skill owns selection across studies, shared constraints, review and synthesis. Execution, credentials, admission, cancellation and host repair remain with their existing owners. No delivery team, private respawn loop or new scheduler is required.

### Phase 1: Commission the work

**Entry:** The operator has commissioned research outcomes and an allowance, or has asked only for a prepared folder.

1. Recall relevant work with `search-hub query "<research intent>" --type library,record,skill,doc`. Use the [research entry point](../../../../../../../docs/internal/research-experimentation/README.md) to locate the commission, applicable topic and current feedback. Read decisive prior evidence selectively; do not send the full history to each investigator.
2. Start from the operator's commission and strongest available evidence. If useful, run `web-console continuity search --query <native-goal-id> --state any` to find prior conversation evidence; read a live owner when readily available. Apply the [cohort lifecycle](references/cohort-lifecycle.md) when deciding whether prior work is superseded: preserve unavailable telemetry as `unknown`, hold only an uncertain dispatch, and use fresh identities and allowance for a new commission. An explicit operator stop plus terminal known worker records can authorize that new commission. `research-supervision-read` is optional evidence collection, not permission to research.
3. Name the outcome, quality standard, authority and stop conditions. Include in-scope implementation and cross-owner repairs; novelty, size or uncommitted files do not create an approval gate. Use the smallest entry structure that carries the current direction ([guidance](references/direction.md#choose-artifacts-and-startup-context)); give investigators its rationale, evidence, freedom and starting points, not a queue of tiny checks. Distinguish a sustained window from a bounded answer. Form open directions from the objective and evidence, and identify operator-selected versus suggested leads. One experiment is a checkpoint, not completion of a sustained study. When the operator asks for several studies or a supervisor goal, use [the repeatable multi-study goal method](references/direction.md#generate-a-multi-study-supervisor-goal-from-ordinary-language) and the general `harness-goal-authoring` skill; do not ask the operator to craft the prompt.
4. If the operator asks only to prepare a folder, follow [prepare without commissioning](references/direction.md#prepare-without-commissioning). Do not start a clock, goal, worker or effect. A proposed allowance is not an active budget. Stop after producing the reviewable handoff; do not run launch or supervision steps until the operator commissions the work.
5. Reuse the [single-chat launch and recovery method](../local-experimentation/references/adaptive.md#start-or-resume-one-research-chat). Each study retains its entry artifact, checkpoint, allowance and effect ownership. Give a coherent direction that can span checks and sessions; investigators choose methods and may revise the hypothesis within scope. Check workspace access and how progress returns. If native goal telemetry is unavailable, supervise from this session or an authorized direct path; do not wait for a native goal read to begin research.
6. For multiple studies, create one new cohort record per fresh commission under the canonical research home's `cohorts/` directory and follow the [cohort lifecycle](references/cohort-lifecycle.md) for fields and state transitions. Link worker records rather than copying their histories. For one study, keep coordination in its existing record. Link the shared operator-attention view and name its supervisor/writer; reconcile real pending requests using [operator attention](references/operator-attention.md).
7. Include startup, workers, supervision and recovery in one current-commission allowance across phases/follow-ups. Start its clock at the first supervisor action after the operator commissions work; individual dispatch timestamps do not reset it. Read back actual role/model/effort and usage through existing owners; catalog defaults alone do not prove launch selection. Agree material notifications and distinguish enforced, cooperative and unmeasured limits.
8. Name the current-session supervisor, how it receives progress, and where it keeps the shared attention list. Ask only for a consequential decision the user alone owns; continue independent work while it is pending. Use [operator attention](references/operator-attention.md) for authority, request delivery and answer reconciliation. If notification is unavailable, surface the pending request in this chat without claiming it was sent. A prompt or native goal is not evidence of active supervision; verify the actual review path before claiming continuous coverage.
   For sustained work, use the current session as supervisor and a visible checkpoint as recovery state. Native goals and Agent Manager are optional aids; neither a failed wake/read nor unavailable infrastructure stops unrelated research. Preserve the allowance and effect ownership in the cohort record.

**Exit/artifact:** The records identify commissioned outcomes, owners, allowances, shared constraints and how to recover context. No launch is claimed from prompt submission alone.

### Phase 2: Advance and challenge

**Entry:** A study returns evidence, reports closure, or encounters a changed condition.

1. Reconcile study states and observations using the [cohort lifecycle](references/cohort-lifecycle.md). Link owner records and update changed facts; do not copy their histories. Agent Manager run status is not proof of a live worker. Workers keep routine progress in their existing checkpoints.
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
| The operator says to stop the research | Stop further dispatch; stop every exact worker through its owner, checkpoint all rows and reconcile effects. If this supervisor runs a native goal, pause that goal after the explicit stop request and read it back. Keep worker and goal states separate; unresolved stop state remains `unknown`, never `stopped`. |
| A new coordinator takes over | Read topic/run guides and the shared checkpoint; reconcile live owners before resuming. Preserve budgets and one writer. |
| Human judgment is pending | Prepare the exact actionable request in the attention view, use a qualified decision owner, and continue independent work. Reconcile the answer and actual resumption. |
| Notification sender is refused or the grant is pending | Preserve the exact refusal/request and show the attention item or approval URL in the current conversation. Mark Notification Hub delivery as pending/unavailable, not sent. Qualify the actual producer/identity path; channel readiness and in-chat visibility do not supply sender authority or delivery. |
| Hard expiry or recovery is required but unsupported | Report that execution route unqualified; continue only independent work whose requirements are met. |
