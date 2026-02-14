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
Findings, plans, and recommendations produced by prompt-manager teams and persisted as backlog artifacts.

## Governance Modes
- `manual`: operator explicitly starts work.
- `scheduled`: work auto-starts after configured delay.
- `yolo`: work starts immediately.

## Agent-Manager
Execution engine used by swarm-manager to spawn and track autonomous runs.

## DOC Links
- [CODE: ui/src/types/domain.ts#BacklogItem]
- [CODE: ui/src/types/domain.ts#ExecutionRecord]
- [CODE: api/internal/scenarios/handler.go]
- [CODE: api/internal/execution/service.go]
