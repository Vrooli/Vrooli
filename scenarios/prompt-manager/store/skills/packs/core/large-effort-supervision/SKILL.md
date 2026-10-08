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
  revision: 21
  createdAt: "2026-09-12T00:00:00Z"
  updatedAt: "2026-10-01T15:00:00Z"
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

### 1. Wakes

| Wake | Trigger | Work |
|---|---|---|
| Daily check | Supervisor team heartbeat, once a day | §2 for every active goal. |
| Step-back | An orchestrator's trigger after a `STEP_BACK` or park/switch/shrink entry in `QUEUE.md` | Audit it (§4). |
| Repeated workaround | The same `WORKAROUNDS.md` failure logged twice, or by two goals | Repair it (§3). |
| Spend spike | A goal's weighted tokens per day above twice its 7-day median | Find the cause; steer or repair. |
| Planning finding | An orchestrator's trigger after a planning finding in the `QUEUE.md` Forecast | Audit the plan (§4). |

Event wakes arrive as manual heartbeat triggers from orchestrators; find the
event in the goal home. Never wake on unchanged state, and never start a
judgment while your previous run is still live. A wake with no open workaround and nothing due records a one-line
quiet disposition and ends. With no active goal, stay idle.

### 2. Daily check

For each active goal home:
1. Read `QUEUE.md`, the active epoch's brief and last 20 slice-log lines, new
   `WORKAROUNDS.md` entries and the latest daily qualification result. Never read
   full transcripts to orient.
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
   within a day is a repair target, not a reason to relaunch again.
5. Reconcile open repair handoffs: submitted/admitted, source-applied, receiver
   acknowledged, and verified recovery are separate states. Retain the exact
   receiving owner, feedback ID, receipt, next handling condition/deadline and
   remaining acceptance. A completed supervisor run or persisted feedback does
   not close a blocker. If the receiver is parked, inspect its actual await
   predicate; do not assume new feedback wakes a child/timer wait or bypass a
   denied wake through another identity. Recheck on changed evidence or the
   existing daily/event opportunity, not repeated unchanged triggers.
6. Repair (§3) the open workaround that costs delivery the most, every daily
   check. A report without a repair or a named reason it could not be done is an
   incomplete check.

### 3. Repair shared infrastructure

For each open workaround entry: reproduce the failure once, fix it in the owning
scenario (Agent Manager, Prompt Manager, Program Runtime, Test Genie, Workspace
Sandbox, Git Control Tower, the bridge) or dispatch one bounded repair run through
Agent Manager with its own `--until`, then mark the entry resolved with evidence
so orchestrators stop using the fallback. Source/test success is a repair
handoff, not closure: require receiver acknowledgement plus the original
consumer operation against the qualified changed build, or retain an exact
unverified dependency and next action. Route only what needs a decision,
credential, dependency approval or production effect you do not hold.

### 4. Audits

Audit every 3rd accepted epoch per goal and every step-back. Read the epoch file,
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
