# Architecture Seams & Internal Design

## Agent orchestration boundary

Swarm Manager owns the project domain: authorizing starts, building immutable
domain snapshots, checking current frontiers, applying typed outcomes exactly
once, and projecting history. Agent Manager owns programmatic agent execution:
prompt rendering, workflow state, branching, retries, waits, budgets, and the
append-only execution journal.

The only Swarm-to-Agent-Manager programmatic boundary is
`internal/agentmanager.WorkflowService`. Workflow choice is declared in
[the transition registry](../../.vrooli/swarm-transitions/registry.json); the
registry selects `session`, `workflow`, or `deterministic` behavior and
cannot contain prompts or execution mechanics.

## Human sessions

`internal/agentsessions` is the human-conversation boundary. It creates and
continues Agent Manager Runs only for operator-authored conversations. Creatable
session kinds are `meta_orchestration`, `swarm_operations`, and
`workflow_authoring`; historical
session kinds remain readable but cannot be started.

## Workflow application

Each workflow start records its correlation, input digest, workflow revision,
and terminal outcome. Domain adapters validate the current item, milestone,
plan, and evidence frontiers before applying a result. Duplicate terminal
delivery is idempotent; stale results are recorded without mutating the domain.

## Integration truth

`internal/integrationstatus` is the sole source for dependency availability,
freshness, degraded behavior, and transition preflight. Callers consume its
projection rather than independently inferring Agent Manager, Plan Manager,
Test Genie, Git Control Tower, or Prompt Manager health.

## Historical records

Retired operating-mode and agent-operation records are read-only provenance.
They are not executable configuration and have no active HTTP, CLI, UI, or
workflow-start surface.
