# Operational Target Coverage

This reference maps the PRD operational targets (regenerated 2026-07-23) to
current documentation and implementation touchpoints. Statuses here are an
audited snapshot for orientation; the earned source of truth is the
`requirements/` registry synced by test evidence. The retired 2026-01 target
set (OT-P0-001..011 idea-backlog era, OT-P05-*) is superseded; its historical
mapping lives in git history.

## P0 Targets

### OT-P0-001 - Backlog work intake
Status: Implemented.
References:
- [DOC: docs/concepts/OPERATOR-JOURNEYS.md]
- [CODE: api/internal/backlog/handler.go]
- [CODE: api/internal/captures/classify.go]
- [CODE: api/internal/proposals/types.go]

### OT-P0-002 - Canonical work package
Status: Implemented.
References:
- [DOC: docs/guides/workshop-workflow.md]
- [CODE: api/internal/backlog/plan_author.go]
- [CODE: api/internal/backlog/plan_candidate.go]

### OT-P0-003 - Plan workshop loop
Status: Implemented.
References:
- [DOC: docs/guides/workshop-workflow.md]
- [CODE: api/internal/planworkshop/service.go]
- [CODE: api/routes_plan_workshop.go]

### OT-P0-004 - Revision-bound acceptance gate
Status: Implemented.
References:
- [DOC: docs/concepts/GLOSSARY.md] (Plan Acceptance)
- [CODE: api/internal/backlog/plan_acceptance.go]
- [CODE: api/internal/execution/preflight.go]

### OT-P0-005 - Mode-selectable execution
Status: Partial — the Run dialog now supports the `sliced` / `goal`
`execution_mode` picker, and goal mode starts one Agent Manager run. A plan-shape
banner remains pending because Plan Manager does not yet store a shape field.
Current status and source links are in
`docs/agent-system/SCENARIO_DEVELOPMENT.md` §Implementation status.
References:
- [DOC: docs/concepts/ARCHITECTURE.md] (Plan-backed execution)
- [CODE: api/internal/execution/service_queue.go]
- [CODE: api/internal/execution/strategies.go]
- [CODE: .vrooli/swarm-transitions/registry.json]

### OT-P0-006 - Phased slice execution
Status: Implemented.
References:
- [CODE: api/internal/execution/phased_plan_workflow.go]
- [CODE: .vrooli/agent-manager/phased-plan-drain.json]

### OT-P0-007 - Independent post-run review
Status: Implemented.
References:
- [CODE: api/internal/execution/finalization.go]
- [CODE: api/internal/review/workflow_apply.go]
- [CODE: .vrooli/agent-manager/independent-review.json]

### OT-P0-008 - Operator-owned terminal decisions
Status: Implemented (enforced invariant).
References:
- [CODE: api/internal/backlogstatus/statuses.go]
- [CODE: api/internal/backlog/review_decide.go]

### OT-P0-009 - Typed follow-up proposals
Status: Implemented — review and execution outcomes persist one typed
follow-up instruction. The ranked inbox exposes it as `dispatch_followup`, and
dispatch deterministically starts a follow-up run, returns work to planning, or
creates typed backlog proposals according to its disposition.
References:
- [CODE: api/internal/proposals/types.go]
- [CODE: api/internal/backlog/followup_dispatch.go]
- [CODE: api/internal/execution/followup_dispatch.go]

### OT-P0-010 - Item next-action projection
Status: Implemented.
References:
- [DOC: docs/concepts/GLOSSARY.md] (Next Action)
- [CODE: api/internal/backlog/next_action.go]

### OT-P0-011 - Goals with milestones
Status: Implemented.
References:
- [DOC: docs/concepts/OPERATOR-JOURNEYS.md] (Journey 2)
- [CODE: api/internal/goals/model.go]
- [CODE: api/internal/goals/service.go]

### OT-P0-012 - Goal planning loop
Status: Partial — launch paths, typed proposal schemas, and apply/validation
logic exist; no production caller invokes the workflow-run apply endpoint, and
the proposal vocabulary cannot yet author new backlog items.
References:
- [CODE: api/internal/goals/handler.go]
- [CODE: api/goal_workflow_adapter.go]
- [CODE: api/goal_mutation_processor.go]
- [CODE: .vrooli/agent-manager/goal-plan.json]

### OT-P0-013 - Milestone review on completion
Status: Partial — auto-trigger, idempotency, and DoD delivery are wired; the
verdict is stranded by the same missing apply caller as OT-P0-012.
References:
- [CODE: api/internal/goals/workflow.go]
- [CODE: api/routes_execution.go]
- [CODE: .vrooli/agent-manager/milestone-review.json]

### OT-P0-014 - Governed workflow execution
Status: Implemented.
References:
- [DOC: docs/concepts/TARGET-OPERATING-MODEL.md]
- [DOC: docs/reference/transition-catalog.md]
- [CODE: api/internal/transitions/registry.go]
- [CODE: api/internal/agentmanager/workflow.go]

### OT-P0-015 - Plan-backed continuity
Status: Partial — plan execution applies `budget_exhausted` terminal results,
and the continuation sweeper can create one bounded child for eligible
`until-allowance` items. It also handles involuntary interruptions, validates
remaining allowance, and records halt/no-progress brakes. End-to-end owner
qualification and production evidence remain open; children currently use a
fresh run, while same-session resume remains unimplemented. This status does
not certify a live continuation.
References:
- [DOC: docs/concepts/ARCHITECTURE.md] (Continuation)
- [CODE: api/internal/execution/continuation.go]
- [CODE: api/internal/transitionrun/transitionrun.go]

### OT-P0-016 - Evidence-bound finalization
Status: Partial — finalization gathers Test Genie and Git Control Tower
evidence and runs the independent review; the evidence-bound acceptance
predicate lives in the retired development package and has no production
outcome resolvers.
References:
- [CODE: api/internal/execution/finalization.go]
- [CODE: api/internal/review/workflow_apply.go]

### OT-P0-017 - Run review surface
Status: Partial — the run sheet supports `sliced` / `goal` and shows limits,
continuation and scope policy. The plan-shape banner waits for Plan Manager's
unimplemented `shape` field. Swarm persists `operator_note`, but the run sheet
has no dedicated note input or composed goal-message preview. The Development
contract panel belongs to the retired route.
References:
- [CODE: ui/src/components/backlog/run-sheet.tsx]
- [CODE: ui/src/components/backlog/backlog-form-details-section.tsx]

## P1 Targets

### OT-P1-001 - Goal next-action chaining
Status: Not implemented.
References:
- [DOC: docs/concepts/OPERATOR-JOURNEYS.md] (Journey 2, step 7)
- [CODE: api/internal/backlog/next_action.go] (item-level precedent)

### OT-P1-002 - Goal progress and velocity
Status: Partial — progress rollup, Monte-Carlo ETA bands, and scope-creep
history exist; velocity/trajectory reporting does not.
References:
- [CODE: api/internal/eta/montecarlo.go]
- [CODE: api/internal/goals/service.go]

### OT-P1-003 - Goal-scoped discovery
Status: Partial — workflow and launch path exist; stranded by the missing
apply caller and item-authoring vocabulary (see OT-P0-012).
References:
- [CODE: api/internal/goals/handler.go]
- [CODE: .vrooli/agent-manager/goal-discover.json]

### OT-P1-004 - Swarm management sessions
Status: Partial — session kinds and skills exist (meta_orchestration,
swarm_operations, workflow_authoring, proposals); no single operator-loop
skill drives the full cycle.
References:
- [DOC: docs/internal/AGENT-SESSIONS.md]
- [CODE: api/internal/agentsessions/service.go]

### OT-P1-005 - Goal execution mode
Status: Implemented — `goal` mode starts exactly one Agent Manager goal run and
Swarm finalizes its terminal result. See `api/internal/execution/service_control.go`
and docs/concepts/ARCHITECTURE.md §Plan-backed execution.
References:
- [CODE: api/internal/execution/service_queue.go]

### OT-P1-006 - Live execution progress
Status: Partial — the agent-manager execution-trace client exists; records
collect terminal-only, and no live slice panel renders.
References:
- [CODE: api/internal/agentmanager/workflow.go]

### OT-P1-007 - Unified work activity feed
Status: Partial — activity timeline merges executions and session spawns;
review rounds, workshop runs, and operator decisions are not yet in one feed.
References:
- [CODE: ui/src/components/backlog/activity-surface/]

### OT-P1-008 - Execution policy modes
Status: Implemented.
References:
- [DOC: docs/reference/configuration.md]
- [CODE: api/internal/execution/service_queue.go]
- [CODE: api/internal/settings/]

## P2 Targets

### OT-P2-001 - Portfolio analytics
Status: Not implemented (stats surfaces exist per goal, not cross-goal).

### OT-P2-002 - Green-path auto-accept policies
Status: Not implemented (only the bounded SWARM_MANAGER_AUTO_FOLLOW_UP opt-in
exists; terminal decisions remain operator-only by design).

### OT-P2-003 - Mid-run steering
Status: Not implemented (slice-approval signal and cancel are the only
channels into a running execution).
