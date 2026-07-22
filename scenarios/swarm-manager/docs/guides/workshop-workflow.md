# Plan Workshop

Plan Workshop is Swarm Manager's one operator-visible planning loop for a
backlog item or milestone. It replaces readiness-score and finalize-round
decisions. Historical `workshop/round-*.json` files remain readable evidence;
they do not authorize execution or mutate a current workshop session.

## Simplification measure

The Phase 1 baseline counted six distinct operator decisions for an existing
backlog plan: enter the workshop, answer a round, save it, start another round,
finalize, and bind/validate the resulting plan. After cutover the comparable
loop has five or fewer decisions: open Plan Workshop, submit one response,
choose apply or ignore for a validated candidate when one exists, accept the
current canonical plan, and queue the work. Review and reconciliation are
internal continuations rather than separately selected workflows. The count is
therefore non-increasing (6 before, at most 5 after); repeated review is a
deliberate return to the same loop, not an added workflow family.

## Operator flow

1. Open a session with `POST /api/v1/plan-workshops`, naming a `backlog_item`
   (`kind/name`) or `milestone` subject and a typed review packet.
2. Review findings, decision questions, and references to proposals in the
   existing Agent Session proposal store as one packet.
3. Submit exactly one idempotent response at
   `POST /api/v1/plan-workshops/{id}/responses`. Answers are applied directly
   only when deterministic; accepted proposals request one reconciliation.
4. Review the candidate plan in Plan Manager. Candidate preview, structured
   diff, validation, and guarded in-place apply remain Plan Manager authority.
5. Explicitly accept the valid canonical revision at
   `POST /api/v1/backlog/{kind}/{name}/plan-accept`, with an actor and,
   optionally, the hash seen by the operator.
6. Queue the item. Preflight requires the fresh accepted Plan Manager content
   hash and current plan validity, alongside ordinary dependency and policy
   checks.

Research, independent review, and milestone review each retain their native
conclusion or round as historical evidence. Their terminal result also creates
one idempotent finding on the matching Plan Workshop session, with a typed
disposition (`plan_revision`, `plan_authoring`, `follow_up`, `archive`,
`supersede`, or `attention`), rationale, and confidence. The disposition is an
operator recommendation, not a mutation instruction: the operator still uses
the workshop response and existing `work.follow_up` or `work.correct`
transitions for authorized work. A research conclusion is not itself an
implementation plan.

Automatic follow-up creation is denied by default. An operator may explicitly
opt in with `SWARM_MANAGER_AUTO_FOLLOW_UP=true`; even then the bounded policy
allows only one high-confidence proposal for the current execution. It
deduplicates by source proposal, retains the source review and execution link
on the child record, and emits proposal creation/application audit events.
All other follow-up proposals remain an explicit operator decision in Plan
Workshop. Distinct backlog or milestone work is a normal mutation-list
proposal with `spawned_from` provenance, never an implicit execution.

## Authority and safety

| Concern | Owner | Guard |
| --- | --- | --- |
| Findings, questions, responses, and exact-once resolution | Swarm Manager | Immutable subject version and response idempotency key |
| Proposal records | Agent Session proposal store | Session/proposal references; no duplicate workshop store |
| Plan candidate and canonical apply | Plan Manager | Expected base hash, validation, acknowledgment, active-execution rejection |
| Plan acceptance | Swarm Manager operator action | Actor, timestamp, plan content hash, subject version |
| Workflow execution and retries | Agent Manager | Declared workflow plus immutable invocation |

An acceptance becomes stale when its subject contract or canonical plan content
hash changes. It is never synthesized from historical workshop scores.

## API shape

`ReviewPacket` carries findings, decision questions, and proposal references.
Each proposal reference includes both Agent Session ID and proposal ID. A
response must name the session's immutable subject version, an actor, and an
idempotency key. It may answer known questions and accept only proposals in the
packet. Invalid, stale, and duplicate requests are explicit recovery states;
the operator can refresh the session or safely retry the same key.

## Historical migration

Legacy workshop files are read-only history. The one-shot migration records an
observable marker, preserves every `round-*.json` path, and makes an atomic
backup for each item without explicit plan acceptance before recording that
state as archived from active use. Corrupt historical round files are retained
and listed as migration errors; they never disappear silently or block clean
history from rendering. The migration does not synthesize plan acceptance, and
legacy finalize posts are not an alternate plan-reference binding path.
Missing plans continue through `plan.author`; concrete plan validation findings
may use `plan.repair` as an internal continuation. Both remain hidden behind
the single Plan Workshop operator loop.

See [DOC: docs/concepts/TARGET-OPERATING-MODEL.md] for the ownership model and
[DOC: docs/reference/transition-catalog.md] for declared workflows.
