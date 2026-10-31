# Effort supervision

An operator can see the progress, cost, risks and next action of every admitted unfinished
delivery goal from one bounded board. A Sol supervisor keeps those goals moving by
repairing the shared infrastructure their orchestrators route around, and keeps
acceptance honest by auditing a sample of orchestrator decisions.

Method lives in skills: `large-effort-supervision` (the supervisor) and
`large-effort-orchestration` (epochs, the orchestrator, the goal home and
acceptance). This document owns only authority, inputs and qualification.
Open reliability work on the loop is tracked in
[DELIVERY_RELIABILITY.md](DELIVERY_RELIABILITY.md).
The pre-epoch supervision model and its validation history are archived
(gzip) under
`~/.vrooli/plan-artifacts/epoch-based-delivery-orchestration-and-bas-rehabilitation/evidence/archive/`.

Start with the [work entry point](../../.vrooli/work/README.md) for work-model selection,
team dispositions and the commissioning lifecycle. The [Agent Manager owner reference](../../scenarios/agent-manager/docs/reference/effort-supervision.md)
owns command/format details. Agent Manager computes the only pressure clock; the supervisor
reads it. A team queue limit, commissioned worker limit, pause and heartbeat disablement
are separate facts. Time never grants acceptance.

## Authority

- The orchestrator selects, admits and accepts epochs. Independence comes from its
  mechanical gate plus the supervisor's sampled audits. Workers never accept.
- The supervisor may read goal homes and owner state, rerun an epoch's exit gate,
  repair shared infrastructure in its owning scenario, and write `FEEDBACK.md`
  entries. It never edits epoch files, directs workers, or changes a destination,
  acceptance rule, owner grant, secret, production system, paid service or
  unapproved dependency; those need an operator amendment.
- Delivery execution stays with Agent Manager and its existing orchestrator.
  New work outside the approved destination follows Swarm disposition under
  SWARM_MANAGER_WORK.md. In-scope repair does not require a second work proposal;
  this distinction creates no new runtime authority.

## Board and inputs

`agent-manager effort board` (and the Agent Manager Efforts view) shows one row per
goal, derived from owner state, never copied into a second store. Runtime
activity, progress, worker completion and epoch acceptance are separate facts;
missing metrics are unknown, never zero.

The active epoch's `Workers` field contributes worker references only when its
entire value is an explicit list of UUID run IDs (comma/space/semicolon separated, with
optional parenthesized notes). Prose, partial lists and malformed values are
unknown and must not be tokenized into apparent workers. Assignment rows remain
the authoritative runtime-owner observations; a goal-home worker reference does
not establish that a run is active or accepted.

Supervision selection follows unfinished admitted obligations and existing
unresolved receiving owners, including interrupted, review-required, blocked and
unknown-runtime work. Runtime determines handling, not eligibility. Native
admission, retained execution ownership, outcome acceptance and metadata quality
are separate facts; a missing `authorized_by` field on historical discovery is
not proof that its retained worker was never admitted. Prepared unapproved and
retired work remain visible with distinct dispositions and no implicit dispatch.
Use one bounded owner read with explicit partial/pagination/freshness limits.
Repair invalid destination references through the authorized metadata owner;
do not duplicate the board in a prompt-maintained active-goal list.
Board `nextAction=observe` is advisory; it does not clear due queue work or retire
an unfinished owner. When no worker is active, reconcile current goal/queue
obligations against retained owner state before ending the check or treating the
goal as quiet.

The declared team write surfaces must agree with this approved method: bounded
feedback to the owning goal and source repair through an authorized scoped
repair executor. Team-storage permission is not a run-admission grant. Report an
actual denied operation separately from missing assignment wording; do not add
blanket write access or infer runtime authority from the task text.

A parked row/detail must expose its owner await producer/key and deadline;
`children` completes when a newly ended direct child appears since the prior
wake or its timer expires, not when feedback or a named external dependency
changes. After the trusted effort-signal dispatcher is qualified and adopted, an
eligible signal record may issue a payload-neutral check-in to the same parked
orchestrator before the child ends. The check-in consumes the parked wait handle
and resumes the same owner; it does not satisfy the child-completion predicate,
prove worker completion, or authorize replacement work. Keep the child owner and
open obligation intact. The owner rereads current child state and may park again
if it is still waiting. Keep the external dependency/owner and feedback
acknowledgement separate. A crash with `needs_review` is an owner-recovery/manual
review condition, not healthy execution or permission to duplicate the retained
leader. Preserve the original run/task binding through supported reconciliation
and test any recovery against actual provider/slice progress.

Open supervisor handoffs remain open until the receiving owner acknowledges
them and the original consumer operation verifies recovery. Retain source,
receipt, affected goal/feedback ID, last observation, next owner action and
return condition; never close on dispatch, heartbeat or source tests alone.
For a temporary coordinator resource hold, record the exact protected work and
owner with a release condition or deadline in this handoff, and recheck the
supported capacity/admission or lease owner before extending it. A coordinator-
created hold is not proof of a native lease or a new operator decision. If no
active work still needs the resource, remove the hold and return next-slice
selection to the retained orchestrator; keep other admissible work moving.
Native admission does not replace workload-specific host/device gates or
genuine human decisions.
When the supervisor has completed its current due reads and authorized actions
and only a future acknowledgement, external event, deadline, or changed evidence
remains, it records the exact return condition and ends that invocation. It does
not wait synchronously or repeat unchanged reads; the handoff remains open for
the existing return opportunity.

What the supervisor reads per goal is defined in `large-effort-supervision` §2.

## Qualification

Validation scope follows docs/TESTING.md. Evidence lives under
`~/.vrooli/plan-artifacts/epoch-based-delivery-orchestration-and-bas-rehabilitation/evidence/`.

| ID | Required behavior | Evidence | Standing (2026-09-30) |
|---|---|---|---|
| ES-01 | Orchestrator park wakes on child exit, timer and wake-by-key | Agent Manager orchestration tests; live toy run | Partial: child-exit and timer wakes proven (`toy-park-wake/`, `pilot-e0/park-wake-events.txt`); no live wake-by-key receipt |
| ES-02 | Heartbeat skips a live orchestrator and relaunches an ended one with backoff and a cap | Prompt Manager heartbeat tests | Proven by tests: `fix-pass-2026-09-30/heartbeat-liveness-tests.txt` (6/6); no live relaunch-cap receipt |
| ES-03 | Epoch check reports each trigger and emits the friction wake | Fixture epoch files | Partial: trigger report proven (`toy-steering/final-epoch-check.txt`); no live friction-wake receipt |
| ES-04 | Weighted non-cache tokens readable per run and child set | Agent Manager usage tests; live read | Proven: `pilot-e0/tokens.txt` |
| ES-05 | A directive is written and acknowledged by a live worker | Live toy steering loop | Proven: `toy-steering/E1.md` |
| ES-06 | An epoch is accepted on a real goal; the supervisor runs a daily check and a sampled audit | First BAS epochs | Partial: BAS E0 is the accepted pilot (`pilot-e0/final-epoch-check.txt`); no supervisor check or audit has run |

Update a row's standing only with a new receipt path. Teams and heartbeats stay
disabled until the operator starts a goal.
