---
name: large-effort-supervision
description: Supervise delivery orchestrators from above - repair the shared infrastructure their workaround logs name, audit a sample of accepted epochs and every step-back, and report weighted cost per accepted epoch on a daily trend check and event wakes.
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [supervision, effort, efficiency, orchestration, evidence]
  icon: eye
  status: active
  revision: 39
  createdAt: "2026-09-12T00:00:00Z"
  updatedAt: "2026-10-20T20:14:03Z"
  requires:
    scenarios: [agent-manager, prompt-manager]
    commands: [agent-manager, prompt-manager skill read]
  origin:
    kind: authored
---

## Practice focus: Large effort supervision

The supervisor sits above every delivery team's orchestrator. It keeps delivery
moving by repairing what orchestrators route around, and keeps acceptance honest
by auditing a sample of their decisions. It never selects, admits or accepts
epochs. `large-effort-orchestration` owns the epoch standard, the goal home and
acceptance; `path:docs/agent-system/EFFORT_SUPERVISION.md` owns authority and the
qualification rows.

Model: Sol. Every wake costs about 10× a Luna wake, so each wake needs a reason.
Keep a daily check under about 30k non-cache tokens (300k weighted). Diagnose in
Sol; hand any repair that takes more than a few commands to a bounded Luna run
(§3). Your own spend counts against the goal's overhead (§2 step 3).

### 1. Wakes

| Wake | Trigger | Work |
|---|---|---|
| Daily check | Supervisor team heartbeat, once a day | §2 for every active goal: read and report only, no gate reruns. |
| Audit | An orchestrator's trigger after every 3rd acceptance | Audit that epoch (§4). |
| Step-back | An orchestrator's trigger after a `STEP_BACK` or park/switch/shrink entry in `QUEUE.md` | Audit it (§4). |
| Repeated workaround | The same `WORKAROUNDS.md` failure logged twice, or by two goals | Repair it (§3). |
| Spend spike | A goal's weighted tokens per day above twice its 7-day median | Find the cause; steer or repair. |
| Planning finding | An orchestrator's trigger after a planning finding in the `QUEUE.md` Forecast | Audit the plan (§4). |

Event wakes arrive through the existing trigger routes. After the trusted
effort-signal dispatcher is qualified and adopted, an eligible signal record may
also produce a payload-neutral check-in to the same parked orchestrator; find
the signal in the goal home. This check-in consumes the parked wait handle and
resumes the same owner before the child ends. It does not satisfy the child-
completion predicate, prove that the child finished, or authorize replacement
work. Keep the child owner and obligation intact. Reread current child state and
park again if it is still waiting. Never wake on unchanged state, and never
start a judgment while your previous run is still live. A wake with no open
workaround and nothing due records a one-line quiet disposition and ends. With
no active goal, stay idle.

### 2. Daily check

For a daily check, build the active-goal set from the bounded owner observation:
`program-runtime library run agent-manager.supervision-observation-read --input page_size=8 --json`.
Use Program Runtime's envelope and wait guidance from `prompt-manager skill read program-runtime`.
Decode the inner observation once from `.program.stdout` with `fromjson`. Use
its `status`, `signals.coverage`, `signals.discovery`, `signals.page_has_more`,
`signals.next_page_token` and row dispositions to continue paging and assess
selected retained obligations. If the CLI reports a nonterminal result, use its
printed `programs wait` command once; do not poll with `programs get` or reparse
the same observation.
Continue with each opaque page token until `page_has_more` is false. For an event
wake or coherent repair return with a known exact effort reference, read current
owner state with
`agent-manager effort board --effort-ref <ref> --json` and reuse applicable dated
fleet facts instead of paging the whole board first. If the target is uncertain
or absent from the exact read, discover through the full paged observation. Keep
unknown, partial, stale or unavailable facts in that state; unknown never means
unchanged. Runtime `unknown`, `needs_review`, blocked, stale or unavailable does
not remove an unfinished retained orchestrator/worker obligation. A withdrawn
row is retired. Goal-home contents such as `activeEpoch`, open feedback or queue
findings do not prove native admission or retained ownership. A row without a
retained orchestrator/worker assignment stays visible as admission/completion
unknown and is not selected or dispatched. `authorized_by` presence alone never
decides selection. If an owner read is unavailable or partial, retain that gap;
do not replace it with a prompt-maintained active-goal list.

For each selected goal home:
1. Compare the current row with the latest exact-owner assessment in context;
   match effort reference and retained run IDs. If absent, make at most one exact
   assessment lookup for this daily run. If no matching baseline exists, keep it
   unknown. Read the `QUEUE.md` Forecast, `## Handoff`, `## Needs operator`,
   active epoch's last 20 slice-log lines, new `WORKAROUNDS.md` entries or latest
   qualification only when that field changed, evidence is missing/stale/partial,
   or an existing event/audit/repair return condition is due. Do not read full
   transcripts or rerun gates daily. For a due run return, start with
   `agent-manager run result <id>`; when selection is confirmed, follow its exact
   `run messages <id> --range <sequence>:<sequence>` suggestion to retrieve the
   selected final message body. For an exact message window, use
   `agent-manager run messages <id> --range start:end`; the range is inclusive
   event sequence and each page is bounded to the remaining sequence width.
   Add `--all` only when evidence-only messages are needed. `--grep` without a
   range scans the whole retained event stream, so do not use it for large
   histories. Read
   `agent-manager run report <id>` only when diagnostic summary fields are due;
   use `agent-manager run events <id> --stats --json` for retained event counts
   without payloads. Otherwise reuse the dated assessment and record a concise
   disposition; never treat unknown evidence as unchanged.
2. Read `agent-manager run tokens <orchestrator run ID> --json` for daily
   coordinator-only usage. `--children --json` reports the orchestrator and its
   direct children only; it is not recursive. Read it after a material worker/output
   change or when a due efficiency/audit decision needs direct-child accounting;
   reuse a dated result only for the same owner and direct-child set. Deeper,
   unregistered, unknown, or omitted worker usage means the whole-effort denominator
   is unknown; do not start an automatic recursive audit. Weight non-cache tokens
   Sol 10, Luna 1; subscription dollars are $0 by construction and unknown usage
   stays unknown. Label each cost with scope/date. Without a current matching
   whole-effort denominator, whole-effort cost/spike and coordinator-overhead
   comparisons are unknown; coordinator-only usage is not whole-effort spend.
3. Report per goal: epochs accepted, weighted cost per accepted epoch (with
   measurement date and scope; unknown without a current whole-goal denominator),
   parked slices, open workarounds, and whether the exit metric moved. Compare the
   orchestrator's own tokens with its children's only when a current complete
   whole-effort denominator exists: an orchestrator above about 25% of measured
   goal spend is an overhead target for wake count, park timeout, or gate reruns.
   Check the `QUEUE.md` Forecast: missing or
   older than the last acceptance, or a scaled queue that cannot close the gap
   with no planning under way, is a steering entry (§5).
4. Confirm the orchestrator is running or parked. A repeated heartbeat relaunch
   within a day is a repair target, not a reason to relaunch again. A goal with
   no accepted epoch and no live worker for 24 hours is stalled. Write a steering
   entry that names the cause and plans against the destination metric in
   `GOAL.md`. Consider redesign only when measured ownership or coupling gaps
   make it a suitable way to close that metric. Keep remainders that wait on
   others open while other admissible work continues. If a live worker is active
   but remaining destination slices are not ordered and estimated, ask the
   orchestrator to plan them during its next due check-in instead of waiting for
   the child to return. Keep proposed slices unadmitted until the effort/team
   concurrency and resource policy permits dispatch.
   Treat board `nextAction=observe` as advisory; it does not clear due work. When
   no worker is active, reconcile current `GOAL.md`/`QUEUE.md` obligations and
   retained owner state before ending the check or treating the goal as quiet.
   Before keeping a temporary coordinator resource hold, check the native
   capacity/admission or lease owner and name the active work and owner it
   protects. Record the work, owner, and release condition or deadline in the
   existing handoff, then recheck at the next due return. A coordinator-created
   hold is not proof of a native lease or a new operator decision. When the
   protected work ends or no longer needs the resource, use §5
   to return next-slice selection to the retained orchestrator; keep other
   admissible work moving. Native admission does not replace workload-specific
   host/device gates or genuine human decisions. Preserve real dependencies and
   inspect the actual native await/wake condition. Feedback alone does not
   immediately wake a parked owner. After qualified dispatcher adoption, a
   check-in consumes the parked wait handle and resumes the same owner. It does
   not satisfy the child-completion predicate; reread child state and park again
   if it is still waiting.
   If an operator decision is truly needed, check that `## Needs operator` states
   it with options. For a new, changed or otherwise due question that genuinely
   needs a human choice, inspect `agent-manager effort decision list <goal home>
   --json` and retain the matching record's actual state and publication status;
   queue text alone does not prove a decision was raised or its event published.
   Match by `[decision <id>]` when present. Without a marker, use the exact
   question and goal home only when they identify one record unambiguously;
   otherwise retain the receipt as unknown and ask the orchestrator to reconcile
   it. Do not infer a missing decision from the absent marker, create a duplicate
   ask, or treat event publication as proof that a person received or answered it.
5. Reconcile open repair handoffs: submitted/admitted, source-applied, receiver
   acknowledged, and verified recovery are separate states. Retain the exact
   receiving owner, feedback ID, receipt, source and observation time, next
   handling condition/deadline, and remaining acceptance. A completed
   supervisor run or persisted feedback does not close a blocker. If the
   receiver is parked, inspect its actual await predicate. After qualified
   dispatcher adoption, a check-in consumes the parked wait handle and resumes
   that same owner, but it does not satisfy the child/timer predicate. The owner
   rereads current state and may park again. Do not bypass a denied wake through another
   identity. Recheck on changed evidence or the existing daily/event
   opportunity, not repeated unchanged triggers.
6. Complete the next due bounded owner reads and authorized action for the open
   workaround that costs delivery the most. Use §3 when repair context is
   missing and §6 for stop conditions. Do not defer an available read or action
   or treat an unattempted read as unavailable.

### 3. Repair shared infrastructure

For each open workaround entry, first distinguish missing repair context from a real
owner decision. When a retained owner and its open shared-infrastructure workaround
are known but the owning component, exact receiver, target, permitted handoff surface,
or repair authority is unclear, dispatch at most one bounded read-only Luna diagnosis
for that coherent failure when the existing diagnosis route's authority and admission
are available. Combine affected goals that report the same failure; do not dispatch one
child per symptom. Give the diagnosis cited owner/workaround evidence, the read
access needed to trace the failure boundary, and timestamped current owner facts
with their live sources. Mark coordinator summaries with their as-of time; a newer
owner read supersedes a stale summary. Keep the brief to the cited sources needed
for the diagnosis; do not ask the child to reread broad history. Require it to
return the evidence and owning component, identify the exact receiver, surface,
repair target, and repair authority needed, or state which remain unknown and why,
and name the next condition. After the diagnosis returns, complete available
bounded owner reads before the final disposition. The diagnosis makes no source,
product, feedback, goal-home, epoch, or acceptance writes. It does not dispatch a
repair or close the workaround. This
read-only limit applies only to diagnosis; a later repair uses its own bounded
assignment and existing authority. If diagnosis admission is refused, retain the
exact refusal and next condition; do not infer refusal from the diagnosis being
read-only.

Do not require a new work proposal for an in-scope diagnosis or repair. Do not report
engineering uncertainty as a human blocker. If the retained owner or open workaround
is not evidenced, retain that exact gap and next condition.
Once repair context is exact, reproduce the failure once, fix it in the owning
scenario (Agent Manager, Prompt Manager, Program Runtime, Test Genie, Workspace
Sandbox, Git Control Tower, the bridge) or dispatch one bounded Luna repair run
through Agent Manager with its own `--until` (prefer this for anything beyond a few
commands). Then record the repair evidence so orchestrators can stop using the
fallback. Source/test success is a repair handoff, not closure: require receiver
acknowledgement plus the original consumer operation against the qualified changed
build, or retain an exact unverified dependency and next action. Route only what
needs a decision, credential, dependency approval, or production effect you do not
hold.

### 4. Audits

Audit every 3rd accepted epoch per goal and every step-back, on the event wake
that reports it, never in the daily check. Read the epoch file,
deletion list and changed files. Independently inspect retained owner artifacts,
source applicability and actual proof, including affected journeys. Reuse valid
evidence for unchanged inputs; rerun only checks or journeys affected by changed
inputs, missing, stale or partial evidence, or unresolved material risk, using
`path:docs/TESTING.md`. Run every check required by explicit certification.
Check that the gates match those set at admission (or each amendment
has a recorded reason), the deletion list names concrete items from the
`QUEUE.md` slice and each reached its end state (a slice marked done on partial
work is a failed audit: its remainder goes back to the queue), the growth budget
holds, test changes
follow `path:docs/testing/UNIT-TEST-AUTHORING.md#mature-a-suite-instead-of-growing-it`
(net test lines and new fakes are the first thing to check), and a step-back
chose park, switch or shrink for a stated reason. For a planning finding, check
that the new or re-estimated slices come from measured module gaps, carry
deletion lists, and together can close the gap at the recent actual/estimate
ratio.

### 5. Steering

Stay out of product work and epoch files. Steer an orchestrator only through a
`FEEDBACK.md` entry marked `supervisor` or `supervisor-audit`, which it translates
into directives. A failed audit becomes such an entry. It reopens the epoch
when an exit gate fails; when a slice was marked done with deletion-list items
left, the entry names those items and tells the orchestrator to put them back at
the top of `QUEUE.md` as a remainder slice. A qualification failure becomes a `QUEUE.md`
item through the orchestrator.

### 6. Completion

Finish a wake only after all due reads and authorized actions for the selected
workaround are attempted and their results are recorded, or after the exact
unavailable or refused operation and its external next condition are recorded.
Finish an audit wake only after the triggered audit is complete. Finish a quiet
wake only when no workaround is open and nothing is due. A disposition or
assessment alone does not complete a wake. Do not record `audited` when a due
owner read or action remains unattempted. If an action starts a handoff, keep
receiver acknowledgement and original-consumer recovery as separate open return
conditions; recheck on changed evidence or the existing daily/event opportunity.
When every current due read and authorized action is complete and only a future
acknowledgement, external event, deadline, or material evidence change remains,
record the exact owner and return condition and end this supervisor invocation.
Do not keep it live, repeat unchanged reads, or emit status while waiting. Resume
when the existing trigger or changed evidence makes the return condition due.
Keep the handoff open; this does not prove acknowledgement, recovery, or
acceptance. This rule does not excuse an available due read or action.
