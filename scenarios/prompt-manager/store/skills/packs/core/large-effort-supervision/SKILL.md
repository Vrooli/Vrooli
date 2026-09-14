---
name: large-effort-supervision
description: Autonomously supervise accepted efforts against agreed outcomes, resolve consequential friction with freedom of means, and improve supervision itself using attributable cost and outcome evidence.
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [supervision, effort, efficiency, orchestration, evidence]
  icon: eye
  status: active
  revision: 8
  createdAt: "2026-09-12T00:00:00Z"
  updatedAt: "2026-09-14T00:00:00Z"
  requires:
    scenarios: [agent-manager, prompt-manager, program-runtime]
    commands: [agent-manager, prompt-manager skill read, program-runtime]
  origin:
    kind: authored
---

## Practice focus: Large effort supervision

Improve delivery of accepted outcomes while accounting for the full cost and
disruption of supervision itself. The enrolled supervisor is an active delivery
executive with freedom of means: it observes, investigates, coordinates,
repairs, recovers, delegates and improves whatever is reasonably needed within
the accepted effort's owner mandate. It does not wait for human approval for
each action.

Read `path:docs/agent-system/EFFORT_SUPERVISION.md` for ownership, enrollment,
evidence, metrics, authority and qualification. It is the target contract, not
proof of runtime readiness. Read its implementation record before selecting an
unqualified route. Use `prompt-manager skill read agent-manager` for owner
operations and `large-effort-orchestration` for effort recovery.

After the contract and owner route are qualified, a recurring wake with a
supplied compact owner cut reuses that contract and the typed assessment
skeleton. Do not repeat full contract, schema, board or transcript reads for
orientation; open only a bounded owner detail when a specific material decision
needs fresher evidence.

### Scope

Supervise enrolled orchestrators and assess their progress, judgment, delegation,
friction and resource use. Actively manage delivery: coordinate teams, address
blockers, recover eligible runs, repair workspace/runtime integration, delegate
work and improve reusable methods. Preserve the accepted destination and quality
floors. The owner mandate, not a predicted action list, determines whether the
supervisor may act.

### 1. Observe a recoverable evidence cut

Entry: owner discovery or enrollment identifies an effort. Unknown mandate
authority permits scoped observation only. A durable autonomous-supervision
mandate permits action with freedom of means; discovery itself never creates it.

Use owner discovery to include new orchestrations and changes without named-effort
configuration. Discovery grants observation, not an extension of the effort's
mutation authority. Report incomplete manifests and unknown grants explicitly.

Use a supplied compact joined owner cut and prior pending decisions for recurring
wakes. If no compact cut is available or the owner route is not qualified, read
`agent-manager effort board` with `--effort-ref <exact-reference>` where applicable
and use a bounded read of existing owner reports, labeling any missing join.
Reuse a qualified runtime binding across wakes rather than repeating discovery;
the same projection is in Agent Manager's Efforts view. Follow the owner usage
skill for assessment and directive operations. Do not launch a private replacement
scheduler or repeat full board/transcript reads for orientation. Reconcile uncertain
dispatches before requesting effects. Keep unavailable metrics unknown. Read changed
summaries first and open detailed evidence only for a decision it could change or
a bounded independent sample.

For a stopped or repeatedly blocked effort, compare its checkpoint with the
declared resolution-source references on the board. Verify new operator/repair
evidence through the relevant owner before repeating a wait. Discovery hashes
are change signals, not proof of recovery or authority. Supervisor activity must
not substitute for orchestrator runtime evidence.

Exit: current outcomes, waits, evidence gaps, prior interventions and remaining
allowance are recoverable through owner references.

### 2. Select the consequential deviation

Entry: the evidence cut and authority are known. Classify each required outcome
and its owner subjects separately. Apply authority and uncertain-effect restrictions
to their affected scopes, then walk the first applicable row per outcome. Productive
siblings, legitimate waits and unrelated unavailable metrics do not mask an
independently evidenced deviation. Select one consequential admissible action;
retain remaining findings.

An evidenced runtime or capability defect does not require independent product
acceptance evidence before diagnosis. Separate business steering from an existing
infrastructure repair assignment. Reconcile the latest typed `repairLinks` in the
owner cut and deduplicate by canonical `work_ref`; do not create a competing Swarm
item, ledger entry or grant. An `assigned` or `resolved` link carries its assigning
owner, next operation, completion evidence and stopping condition. If the link is
`needs_assignment`, name the assigning owner, next operation and stopping condition
without claiming dispatch. Escalate a missing assignment once, then retain the
named owner wait rather than repeating a bare reconciliation wait.

| Observation | Decision |
|---|---|
| Authority revoked, effort withdrawn or allowance exhausted | Prevent new affected effects through the owner; retain pending evidence |
| An earlier effect may still be active | Reconcile its original identity; do not replace it |
| A consequential repeated failure or acceptance deviation has evidence | State a falsifiable hypothesis and choose investigation, steering or owner repair |
| A required outcome has no measurement | Repair the sensor within authority or retain the unmet evidence requirement |
| Decision-relevant evidence unavailable or stale | Report uncertainty; obtain one bounded owner observation if economically admissible |
| No actionable deviation and a policy-selected independent sample is due | Take the bounded sample under the diagnostic allowance; retain selection basis and result without inventing a deviation |
| Productive work or a legitimate producer wait, with no evidenced deviation, unmet measurement requiring action or independent sample due for this outcome | Leave it running or parked; do not infer failure from elapsed silence |
| No actionable deviation and no independent sample is due | Record quiet disposition and wait for an owner change |

Apply the canonical metrics contract's economic admission test to optional work:
decision relevance alone is insufficient. Compare full expected cost and avoided
loss; use only a bounded experiment when net benefit is unknown. Required gates
and policy-selected samples retain their obligations and allowances.

Compare accepted outcomes, not changed-line or event counts. A necessary
investigation ending in a small repair or a useful negative result can be
efficient. Repeated investigation without changed evidence is a stronger waste
signal. Read `reviewing-agent-run-efficiency` for run interpretation and
`scenario-work-ladder` when contract or repair ownership is uncertain.

Exit: one decision names the affected outcome, supporting evidence, uncertainty
and expected value of obtaining more evidence or acting.

### 3. Investigate or intervene within authority

Entry: an admissible action addresses a material deviation or an independent sample
selected by operating policy. Sampling alone grants no steering or repair authority.
Return sample findings to outcome classification.

When a policy-selected independent sample is admitted, perform that diagnostic
sample even when the effort is stopped or the selected cut is observation-only.
Sampling remains separate from steering authority and does not authorize a
directive.

Use the existing typed investigation operation for a bounded diagnostic question.
Read its program and owning skill together. Diagnosis does not authorize repair.
Prefer the smallest intervention that can satisfy the outcome; use a structural
repair when the cause is duplicated responsibility or a missing shared invariant.
Use `path:docs/agent-system/SWARM_MANAGER_WORK.md` for assignment shape.

Send steering through the owner directive channel to the orchestrator. Retain
revision, evidence, expected result and stable request identity. Read delivery and
acknowledgment separately. Respect a justified challenge or producer wait. Do not
edit the orchestrator's active control files or redirect its workers concurrently.
Coordinate shared repairs with existing assignments before dispatch.

For premature stops, use the recovery contract in
`path:docs/agent-system/EFFORT_SUPERVISION.md#resolution-evidence-and-recovery`
and read `large-effort-orchestration`'s recovery reference. Classify session loss,
missing credentials, provider selection, quota and dispatch uncertainty separately.
An interactive CLI working does not prove its managed run has the same credential
or session roots. A provider catalog entry does not prove authenticated execution.
Preserve the effort's subscription/provider policy; funded metered alternatives
are not implicitly authorized fallback routes.

Choose compatible continuation, owner fresh-run recovery, repair coordination or
a genuine external wait. Reconcile the exact old executor before replacement.
Issue an authorized CONTINUE directive only with resolution evidence and a
falsifiable recovery hypothesis/comparison. Proven missing sessions use the
separately granted RECOVER_FRESH owner action, never an implicit continuation
fallback. Retain its replacement identity and predecessor, and verify that the
original, still-valid effort binding was reconciled without renewing its budget.
Do not use NUDGE as restart permission.
If owner recovery is absent, identify that capability defect and its next owner
action instead of repeatedly classifying the repaired symptom as the blocker.
Never edit an active driver's control files or create a competing scheduler.

Treat a maintenance fence as its recorded owner's hold. Do not reopen another
operation's fence because it prevents dispatch or looks old. Verify its release
condition and owner handoff; a changed revision invalidates prior drain evidence.
Coordinate conflicting maintenance instead of repeatedly closing or reopening it.

Use the recovery completion gate in the contract. Retain the progress condition
and baseline before recovery. After owner delivery, verify new assignment-relevant
evidence and record recovery verification on the original directive. A heartbeat,
tool-call count or successful restart is not proof of useful progress. Keep causal
benefit unknown unless a comparison supports it. A failed verification consumes
the original allowance and does not grant another replacement attempt.

Exit: the action is durably pending, refused with a reason, or delivered with an
assessment condition. No effect exists only in the supervisor's context.

### 4. Assess delivery and the supervision method

Entry: new evidence can assess a prior hypothesis or intervention.

Compare the expected and observed result. Reject a contradicted hypothesis.
Completion after a nudge does not prove causal benefit. Preserve unassessed cases,
false alarms and missed problems. Sample healthy work as well as flagged work.
Charge observation, inference, review, disruption and method repair to the same
accounting boundary as other effort work. Compare like assignments and separate
critical-path delay from summed agent time. Do not convert unknown cost to zero.

Before improving the method, name a recurring cost, expected reuse, comparison
and stopping condition. Use `path:docs/agent-system/LAYERS.md` and
`path:docs/agent-system/PROMOTION_LADDER.md` for ownership and promotion.
Verify adoption and retire the replaced
procedure. Return to delivery after the bounded experiment; do not audit the
method on every wake. Use independent review for consequential policy changes.

Exit: assessed/unknown benefit, full observed cost and the next decision are
retained in the owner assessment. The hypothesis is accepted, rejected or unknown.

### 5. Wait, recover or retire

Entry: no immediately useful admitted action remains.

Checkpoint owner identities and the reopening condition. Use owner events and
the configured recovery heartbeat; suppress overlap for the supervisor's own
assignment, not an active orchestrator being observed. Skip unchanged judgments
unless a policy-selected sample is due. Do not
hold a native goal open to wait. Keep productive authorized work eligible during
supervision outages. On withdrawal or accepted completion, retire the watch and
verify that pending delivery and recurring wakes cannot create new effects.

Exit: a durable owner wait or evidenced retirement. Capture distilled reusable
lessons through the existing Memory contract without duplicating automatic capture.

Retire completed effort watches individually. Keep the standing supervisor serving
other and newly discovered efforts. When none remain active, leave discovery in
cheap idle without agent inference; the next owner change makes a wake eligible.

### Anti-patterns

| Pattern | Consequence | Correction |
|---|---|---|
| Full transcript audits on every wake | Oversight repeats orientation cost | Read changed summaries; investigate a decision-relevant uncertainty |
| Rewarding findings or interventions | Healthy agents get disrupted | Assess accepted outcomes and justified restraint |
| Penalizing small diffs or required tests | Quality and useful investigation fall | Inspect necessity, reuse and actual outcome evidence |
| Optimizing only worker cost | Supervisor overhead disappears | Include every orchestration and improvement layer |
| Program hides another owner's repair loop | Duplicate state and policy grow | Repair the owner and remove the compensation |
| Self-improvement never returns to delivery | Machinery consumes the destination | Bound the experiment and retain rejected hypotheses |

### Output expectations

Produce one bounded assessment with effort/target revision, evidence references,
current outcome standing, uncertainty, selected action or quiet disposition,
observed resource use, pending owner identities and a next wake/assessment condition.
Retain it in the existing supervision owner, not a second issue or approval ledger.
Once the owner accepts the assessment, use the runtime's typed knowledge-link
operation once and finish. A secondary index/link failure must not trigger another
assessment or an unbounded infrastructure investigation. Preserve the owner receipt
and the failed link separately for the owning workflow to repair.

### Troubleshooting & Edge Cases

| Condition | Response |
|---|---|
| Board or metric unavailable | Retain its reason; use one relevant owner read; keep the outcome unverified |
| Family-only watch cannot represent this effort | Use qualified effort supervision when available; never invent a family |
| Swarm asks for a human interaction on already-authorized work | Apply the qualified fallback doctrine; preserve actual authority and reduced guarantees |
| Directive accepted but not delivered | Inspect the owner delivery state; wait for a cooperative boundary |
| Guidance contradicted by fresh evidence | Supersede or withdraw through the owner; retain the old assessment |
| Supervisor method appears cheaper but quality evidence is missing | Retain the candidate as unqualified; preserve quality floors |
| Runtime route exists only in documentation | Report the capability gap and repair its owner within scope before relying on it |
