# Glossary

## Backlog Item
A git-tracked unit of work stored under `ideas/`, `research/`, `fix/`, or `execute/`.

## Backlog Kind
The category of backlog work: `idea`, `research`, `fix`, `execute`.

## Backlog Status
Lifecycle state for a backlog item: `backlog`, `researching`, `ready`, `queued`, `in_progress`, `completed`, `archived`.

## Execution Run
A tracked run record for queued work, with status transitions through pending/scheduled/running/completed/failed/canceled.

## Execution Policy
Default execution behavior for new queue actions (`manual`, `scheduled`, `yolo`) and default delay settings.

## Scenario Archive
Delete operation that preserves selected scenario artifacts by creating an archived backlog item.

## Prompt Manager Team Output
Findings, plans, and recommendations produced by prompt-manager agent teams (Debug, Feature, QA, Refactor) and deposited into swarm-manager as backlog items. Swarm Manager serves as the staging layer where operators review and refine these plans before approving them for execution. Teams write backlog items via the `swarm-manager-recommendations` skill in prompt-manager.

## Governance Modes
- `manual`: operator explicitly starts work.
- `scheduled`: work auto-starts after configured delay.
- `yolo`: work starts immediately.

## Agent-Manager
Execution engine used by swarm-manager to spawn and track autonomous runs.

## Workflow Transition
A versioned Swarm declaration that identifies the subject, prerequisites, Agent Manager workflow, typed input/output contracts, and exactly-once domain apply action.

## Workflow Execution
Agent Manager's durable execution of a declared workflow. It pins an immutable workflow revision and input snapshot while Agent Manager owns prompts, nodes, waits, branches, retries, and the execution journal.

## DOC Links
- [CODE: ui/src/types/domain.ts#BacklogItem]
- [CODE: ui/src/types/domain.ts#ExecutionRecord]
- [CODE: api/internal/scenarios/handler.go]
- [CODE: api/internal/execution/service.go]
