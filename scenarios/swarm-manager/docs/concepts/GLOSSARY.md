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

## Operating Mode
A reusable, inspectable, testable methodology loop expressed as **data** (a `mode.json` phase graph under `modes/<id>/`) and interpreted by one generic engine. Fifteen ship today. See [EXECUTION-MODES.md](./EXECUTION-MODES.md).

## Operation (Contract)
A provider-neutral, named + versioned declaration of *what* to run against a target — required capabilities, typed inputs/results, standard outcomes, and evidence expectations (`review-round@1.0.0`). It never names a mode, function, or path. Authored under `operation-contracts/`; 15 ship.

## Binding
A data document selecting *which* operating mode (at an exact revision) implements an operation for a scope. Resolved by deterministic precedence — **authorized-invocation > backlog-item override > initiative override > system-default** — failing closed on absence or an invalid/disabled/deleted winner (never falling back to a lower layer). Overrides live in domain storage, never the shipped catalog.

## Workflow Instance
The durable canonical projection that correlates a target's executions, decisions, timers, and terminal state (`workflow-instance.schema.json`). Committed by compare-and-swap with atomic writes.

## Target Kind
The unit of work an operation runs against: `backlog-item`, `initiative`, `plan-execution`, or `scenario`. Each has an adapter supplying target-specific reads, artifact scope, and lock identity.

## Target Capability
A typed capability a target adapter *provides* (`provides-review-artifacts`), from a closed enum. An operation contract's required capabilities are checked against them, failing closed with the missing set.

## Member-Item Strategy
The initiative configuration for running each member item through its own operation instead of a methodology loop. It is **not** a mode: it survives only as the sentinel value `item-level` (or blank) on an initiative's persisted `mode` field; the loader rejects any mode folder claiming that id.

## Transition Policy
A data document selecting which closed-registry **domain action** fires on a given `(state, outcome)`. It can name only registered actions — never code (no `command`/`handler`/`exec`/`path` field exists). Four ship under `policy/`.

## Outcome Classifier
The mechanism that derives an operation's terminal **outcome** (and edge-routing fields) from a completed run's handoff, via the resolution ladder; abstention routes to `needs_attention`. Classification lives on transitions, not in a dedicated agent phase.

## Execution Provenance / Snapshot
The immutable record pinned before a run starts — operation+version, binding layer+owner, mode+revision, compiled-mode digest, prompt-catalog revision+digest, target, caller-input digest, policy revision, and workflow id. Reproduction fails closed on a digest mismatch; a partial provenance can never authorize a run.

## DOC Links
- [CODE: ui/src/types/domain.ts#BacklogItem]
- [CODE: ui/src/types/domain.ts#ExecutionRecord]
- [CODE: api/internal/scenarios/handler.go]
- [CODE: api/internal/execution/service.go]
