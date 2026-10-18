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
  revision: 23
  createdAt: "2026-09-12T00:00:00Z"
  updatedAt: "2026-10-16T00:00:00Z"
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

Event wakes arrive as manual heartbeat triggers from orchestrators; find the
event in the goal home. Never wake on unchanged state, and never start a
judgment while your previous run is still live. A wake with no open workaround and nothing due records a one-line
quiet disposition and ends. With no active goal, stay idle.

### 2. Daily check

Build the goal set from one bounded Agent Manager owner observation:
`program-runtime library run agent-manager.supervision-observation-read --input page_size=8`.
Read `signals.coverage`, `signals.discovery`, `signals.page_has_more` and every
row's `disposition` before steering. Continue with the returned opaque page token
when needed. Runtime `unknown`, `needs_review`, blocked, stale or unavailable
does not remove an unfinished retained orchestrator/worker obligation. A withdrawn
row is retired. Goal-home contents such as `activeEpoch`, open feedback or queue
findings do not prove native admission or retained ownership. A row without a
retained orchestrator/worker assignment stays visible as admission/completion
unknown and is not selected or dispatched. `authorized_by` presence alone never
decides selection.
If the owner read is unavailable or partial, retain that gap and its continuation;
do not replace it with a prompt-maintained active-goal list.

For each selected goal home:
1. Read the `QUEUE.md` Forecast, `## Handoff` and `## Needs operator`, the
   active epoch's last 20 slice-log lines, new `WORKAROUNDS.md` entries and the
   latest daily qualification result. Never read full transcripts to orient,
   and never rerun gates in a daily check. If nothing changed since your last
   check (no acceptance, slice-log line, workaround or feedback), record a
   one-line disposition and end.
2. Read weighted non-cache tokens:
   `agent-manager run tokens <orchestrator run ID> --children --json` (Sol 10,
   Luna 1). Subscription dollar figures are $0 by construction; unknown usage stays
   unknown.
3. Report per goal: epochs accepted, weighted cost per accepted epoch, parked
   slices, open workarounds, and whether the exit metric moved. Report the
   orchestrator's own tokens against its children's: above about 25% of the
   goal's spend is overhead to repair (wake count, park timeout, gate reruns).
   Check the `QUEUE.md` Forecast: missing or older than the last acceptance, or
   a scaled queue that cannot close the gap with no planning under way, is a
   steering entry (§5).
4. Confirm the orchestrator is running or parked. A repeated heartbeat relaunch
   within a day is a repair target, not a reason to relaunch again. A goal with
   no accepted epoch and no live worker for 24 hours is stalled. Write a steering
   entry that names the cause: the orchestrator must plan a slice (a redesign if
   nothing is deletable), or move remainders that wait on others out of the way.
   If an operator decision is truly needed, check that `## Needs operator` states
   it with options.
5. Reconcile open repair handoffs: submitted/admitted, source-applied, receiver
   acknowledged, and verified recovery are separate states. Retain the exact
   receiving owner, feedback ID, receipt, next handling condition/deadline and
   remaining acceptance. A completed supervisor run or persisted feedback does
   not close a blocker. If the receiver is parked, inspect its actual await
   predicate; do not assume new feedback wakes a child/timer wait or bypass a
   denied wake through another identity. Recheck on changed evidence or the
   existing daily/event opportunity, not repeated unchanged triggers.
6. Start a repair (§3) of the open workaround that costs delivery the most.
   Usually that means dispatching a bounded Luna repair run, not fixing it in
   this session. A report that neither starts a repair nor names why it could
   not is an incomplete check.

### 3. Repair shared infrastructure

For each open workaround entry: reproduce the failure once, fix it in the owning
scenario (Agent Manager, Prompt Manager, Program Runtime, Test Genie, Workspace
Sandbox, Git Control Tower, the bridge) or dispatch one bounded repair run through
Agent Manager (Luna, its own `--until`; prefer this for anything beyond a few
commands), then mark the entry resolved with evidence
so orchestrators stop using the fallback. Source/test success is a repair
handoff, not closure: require receiver acknowledgement plus the original
consumer operation against the qualified changed build, or retain an exact
unverified dependency and next action. Route only what needs a decision,
credential, dependency approval or production effect you do not hold.

### 4. Audits

Audit every 3rd accepted epoch per goal and every step-back, on the event wake
that reports it, never in the daily check. Read the epoch file,
deletion list and changed files, then rerun the exit-gate commands and affected
journeys. Check that the gates match those set at admission (or each amendment
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
