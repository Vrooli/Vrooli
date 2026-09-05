---
name: "agent-manager-plan-family-supervision"
description: "Operate Agent Manager plan-family supervision with fixed policy versions, durable cursors, bounded decisions, explicit interventions, and outcome capture."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["agent-manager","plan-family","supervision","policy","cursor"]
  icon: "eye"
  status: "active"
  revision: 2
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["agent-manager", "plan-manager", "program-runtime"]
    commands: ["agent-manager watch", "plan-manager families frontier", "vrooli-memory learning"]
  origin: {kind: "authored"}
---
## Tools focus: Agent Manager Plan-Family Supervision

Agent Manager owns durable watches, cursors, timers, decisions and interventions.
Plan Manager owns the reviewed graph and frontier. The finite evaluator interprets
one bounded batch and has no authority to launch children or rewrite plan state.
Read `agent-manager` for per-attempt recall/capture and `plan-family-orchestration`
for graph judgment. Preserve the existing operator's task authority.

### Choose the next operation

Read rows in order; the first applicable row decides the next step.

| Situation | Step |
|---|---|
| Child launch independence is not established | Read `plan-manager families frontier <family>`; obtain the exact graph review before launch. **[S1]** |
| Authorized child run IDs exist but no cohort watch exists | Run `agent-manager watch create --spec-file <WatchSpec JSON> --idempotency-key <stable-key>`. Use one family execution, parent and exact child subjects. **[S1]** |
| Need current watch state | Run `agent-manager watch get` using its current help. **[S1]** |
| Need to inspect pending evidence without consuming it | Run `agent-manager watch inspect`; inspection never advances the cursor. **[S1]** |
| Need to wait for a change | Run `agent-manager watch wait` once for the current revision. The durable scheduler continues after the caller detaches. **[S1]** |
| Parent can park under the current run contract | Use the Agent Manager parent park/await path from the usage skill. Do not maintain a polling program. **[S1]** |
| A decision recommends intervention | Read `agent-manager watch actions`, then request an authorized `watch action` with exact target, revision, evidence, cooldown and idempotency key. **[S1]** |
| Need to assess an intervention | Read `watch policy-outcomes`, inspect the named decision/action/child evidence, then run `watch policy-assess --outcome-file <assessment JSON>`. Supersede the unassessed outcome. **[S1]** |
| Need offline diagnosis of one finite batch | Run `program-runtime library run agent-manager.supervision-evaluate` with its declared inputs. This does not consume the watch or apply an action. **[S3]** |
| Need to improve monitoring behavior | Read `agent-manager-improve`; do not change policy during this family execution. **[S0]** |

### Read the decision correctly

| Disposition | Decision |
|---|---|
| quiet | No parent intervention is required. Quiet is not task success. |
| signal | Inspect the evidence and recommendation; current typed authority decides actuation. |
| terminal | Read child outcomes; failed children do not become successful because monitoring ended. |
| cursor_reset | Reconcile through owner state. Never invent offsets. |
| unavailable | Retain the prior cursor and reason; wait for the next durable trigger. |

Event count is a batching trigger, not evidence of progress. Missing friction
projections remain unavailable even when the event batch is full or a child is parked.
An AI completion label can request inspection. Only every watched run's
terminal owner state can terminate the cohort.
Event count alone does not justify nudging a productive child. A parked child may
be waiting correctly on server-owned validation. Repeated calibrated friction is
stronger evidence than one failed tool call. A nudge cannot waive a required gate.
Accepted actions reserve their count and cooldown budget. Emergency disable and
watch cancellation also prevent pending delivery.
Nudge delivery remains pending until the owner reaches a safe resumable boundary;
never describe acceptance as delivered guidance or demonstrated benefit.

### Learning and policy promotion

The runtime records decision/input/child linkage with an unassessed outcome.
An assessment records observed class and outcome evidence; missing observation
remains unassessed. Set `completion_impact_observed` only when referenced evidence
measures impact. Omitted or legacy zero impact is unknown. Do not label a successful child as proof of causal benefit.
Use Memory for reusable operating advice and attempt effort. The supervision
outcome ledger is authoritative for classifier assessment and policy experiments.

A candidate uses `watch policy-candidate --policy-file <definition> --supersedes <version>`.
`watch policy-evaluate --version <candidate>` executes that candidate against
retained assessed inputs. It derives rollout evidence from actual candidate
watch decisions; a caller cannot supply a count. Both positive and negative cases
are required. Promotion uses `watch policy-promote --version <candidate>` only
after the owner accepts replay and rollout evidence and authenticates the review.
Rollback uses `watch policy-rollback --active-version <version>`; emergency disable
uses `watch policy-disable --disabled=true --reason <reason>`. These operations
retain their owner authorization checks. Read current help for exact flags.

The evaluator's executable digest is pinned on first use of the policy. Changed
code is refused for that policy; create and validate a new candidate. Recall
cannot alter this identity or expand allowed actions. Effective gateway provider,
model and applied sampling settings are also pinned on the first classified
decision. Missing or changed identity preserves an unavailable decision. Stable fixes move into the
owner operation, and obsolete guidance must then be removed.

### Troubleshooting & Edge Cases

Capture one operator attempt through the `agent-manager` learning contract after
its evidenced outcome. Keep run IDs, policy/digest, watch/decision/action IDs,
cursor disposition and assessed/unknown result; never copy transcript bodies.

- Policy/artifact mismatch: preserve the refusal and use a reviewed candidate;
  do not rebind the active version to changed code.
- Friction projection absent: report unknown; do not infer zero friction.
- Repeated signal after an allowed nudge: escalate with its action/evidence rather
  than silently increasing intervention frequency.
- Assessment lacks a linked decision or child: repair attribution, not the label.
- Replay unavailable or insufficient: no promotion; retain the required sample gap.
