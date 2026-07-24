# Glossary

## Backlog Item
The smallest independently reviewable and schedulable work commitment, stored
under its kind directory (`ideas/`, `research/`, `fix/`, `execute/`,
`chore/`). Owns its goal, scope, dependencies, status, supporting context, and
canonical `plan_ref` when executable.

## Backlog Kind
The category of backlog work: `idea`, `research`, `fix`, `execute`, `chore`.
Every kind flows through the same plan-workshop, acceptance, execution, and
review lifecycle.

## Backlog Status
Lifecycle state for a backlog item. Non-terminal states cover intake,
refinement, queueing, execution, and review (`in_review`, `review_pending`);
the terminal states `completed`, `failed`, and `needs_followup` are reachable
only through an explicit operator review decision. The status vocabulary and
its writer policy live in `api/internal/backlogstatus/statuses.go`.

## Next Action
The server-owned, read-only projection of the highest-priority operator step
for one backlog item. It carries a stable action ID, compact and expanded
labels, enabled state, reason, blockers, and a target. It is not a command
endpoint: plan authoring, acceptance, review, retry, archive, and queueing
continue through their existing authorized operations.

The precedence is: archived and terminal states; active execution; review;
missing canonical plan; plan acceptance or quality repair; dependency blockers;
then `run` only when `ProcessPreflight` and dependency evaluation are ready.
The projection reuses the same preflight that gates queueing, so the
recommendation can never disagree with enforcement.

## Goal
An operator intent statement — a few sentences of higher-level outcome — with
priority, target items, and owned milestones. A goal's scope is derived truth:
its targets plus the transitive prerequisite closure of those items. A goal
carries no plan document; the goal's plan is its graph of milestones and
items, and goal planning means proposing graph changes.

## Milestone
An owned sub-object of exactly one goal that partitions the goal's scope and
carries acceptance criteria (its definition of done) over member backlog
items. Milestones are presentation and verification structure — two levels
only (goal → milestone → item), no nesting, no cross-goal sharing. Milestone
review verifies the acceptance criteria against repository evidence when
member items complete.

## Execution Run
A tracked run record for queued work, with status transitions through
pending/scheduled/running/completed/failed/canceled.

## Execution Strategy
A declared way to execute an accepted plan, selected by the operator at queue
time from the strategy registry (e.g. the phased plan drain). Strategies map
to Agent Manager workflow declarations; the registry is the single catalog.

## Execution Policy
Default execution behavior for new queue actions (`manual`, `scheduled`,
`yolo`) and default delay settings, with queue caps and circuit breakers.

## Scenario Archive
Delete operation that preserves selected scenario artifacts by creating an
archived backlog item.

## Agent-Manager
Execution engine used by swarm-manager to spawn and track autonomous runs.
All programmatic agent work runs as declared Agent Manager workflows; human
conversation runs as Agent Sessions over Runs.

## Workflow Transition
A versioned Swarm declaration that identifies the subject, prerequisites,
Agent Manager workflow, typed input/output contracts, and exactly-once domain
apply action.

## Workflow Execution
Agent Manager's durable execution of a declared workflow. It pins an immutable
workflow revision and input snapshot while Agent Manager owns prompts, nodes,
waits, branches, retries, and the execution journal.

## Plan Workshop Session
A durable Swarm Manager aggregate for one backlog item or milestone. It
versions review packets containing findings, decision questions, and references
to Agent Session proposal records; it never owns a second proposal store.

## Plan Acceptance
An explicit operator authorization for one canonical Plan Manager content hash.
It records the actor, timestamp, accepted hash, and subject version. Queueing
requires a fresh acceptance and Plan Manager validity; historical readiness
scores do not substitute for it. Any plan change clears acceptance.

## Candidate Revision
A whole-plan proposed revision stored and validated by Plan Manager. Applying
one retains the canonical plan ID and requires its expected base content hash,
an explicit quality-impact acknowledgment, and no active bound execution.

## Proposal
A typed, operator-decidable mutation suggested by an agent (session, workshop
round, review, or goal workflow): item mutations, milestone operations,
follow-up runs, corrections, or new backlog items. Proposals are the only path
by which agent judgment changes the work ledger, and each applies exactly once
on acceptance.

## DOC Links
- [DOC: OPERATOR-JOURNEYS.md]
- [CODE: api/internal/backlog/types.go]
- [CODE: api/internal/backlogstatus/statuses.go]
- [CODE: api/internal/backlog/next_action.go]
- [CODE: api/internal/goals/model.go]
- [CODE: api/internal/execution/service.go]
