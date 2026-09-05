# Architecture Seams & Internal Design

## Agent orchestration boundary

Swarm Manager owns the project domain: authorizing starts, building immutable
domain snapshots, checking current frontiers, applying typed outcomes exactly
once, and projecting history. Agent Manager owns programmatic agent execution:
prompt rendering, workflow state, branching, retries, waits, budgets, and the
append-only execution journal.

The only Swarm-to-Agent-Manager programmatic boundary is
`internal/transitionrunner`, which owns the `internal/agentmanager.WorkflowService`
transport. Workflow choice is declared in
[the transition registry](../../.vrooli/swarm-transitions/registry.json); the
registry selects `session`, `workflow`, or `deterministic` behavior and
cannot contain prompts or execution mechanics. Its declared workflow and
deterministic apply actions are verified as a complete dispatch table at boot.

## Human sessions

`internal/agentsessions` is the human-conversation boundary. It creates and
continues Agent Manager Runs only for operator-authored conversations. Creatable
session kinds are `meta_orchestration`, `swarm_operations`, and
`workflow_authoring`; historical
session kinds remain readable but cannot be started.

## Workflow application

Each workflow start records its correlation, input digest, workflow revision,
and terminal outcome in the shared `transitionrun` journal. Domain adapters
provide the immutable input and typed apply operation; the runner validates the
current item, milestone, plan, and evidence frontiers before applying a result.
Duplicate terminal delivery is idempotent; stale results are recorded without
mutating the domain. The shared sweeper resumes results claimed before a crash.

## Integration truth

`internal/integrationstatus` is the sole source for dependency availability,
freshness, degraded behavior, and transition preflight. Callers consume its
projection rather than independently inferring Agent Manager, Plan Manager,
Test Genie, Git Control Tower, or Prompt Manager health.

## Historical records

Retired operating-mode and agent-operation records are read-only provenance.
They are not executable configuration and have no active HTTP, CLI, UI, or
workflow-start surface.

## Stats projection and measures contract

The append-only event log is authoritative. `internal/stats.Engine` rebuilds a
single incremental projection at startup and advances it by watermark on each
Stats read. `GET /api/v1/stats` exposes the coherent, optionally goal-scoped
snapshot consumed by the operator-facing `/stats` lens and `swarm-manager
stats` CLI commands.

`POST /measures/execute`, the `swarm-manager measures` CLI domain, and the
Connect `MeasuresService` provide declared, provenance-bearing programmatic
questions over the same durable history. Measures are not an operator UI
replacement and must converge onto projection-backed shared computations before
any Stats field is retired. A measure result carries `executed_query` and
`computed_at`; Stats owns cross-field consistency, trends, sample context, and
interactive analysis.

`BacklogService.ListItems` and `BacklogService.GetItem` are the typed read
seams for the operator's `backlog list` and `backlog get` commands and the
cross-scenario feedback flow. `BacklogService.DeleteItem` is the idempotent
typed mutation seam for `backlog delete`; it retains the REST operation's
milestone and dependency-reference cleanup. `CreateItem` is
intentionally narrower than the attachment-aware operator create surface:
it deduplicates triage reports and has no attachment transport. `backlog create`
therefore remains locally bound until an attachment-capable,
behavior-equivalent typed contract is designed; rebinding it today would lose
brownfield behavior.
# Unified work activity seam

`GET /api/v1/backlog/{kind}/{name}/work-feed` is a read-only projection over
execution records, workflow activities, review rounds, plan-workshop state,
and entity events. It is deliberately not a persisted aggregate. Live workflow
state remains read-through at `GET /api/v1/execution/{id}/progress`, which
proxies Agent Manager's trace only while the operator needs it.

## Proto API/domain transition

The existing `v1/api` files are the public Connect request/response surfaces;
their matching `v1/domain` files own durable entity models. API contracts import
the corresponding domain model where a response returns that entity. The shared
cross-domain types (`AgentSessionAttribution`, session artifacts/attachments,
`BacklogFile`, `Milestone`, and `PlanRef`) live in `v1/shared` so this boundary
does not create a second ownership path. The remaining API-to-domain imports are
intentional until each public service is folded into its product-named proto
domain; they preserve one canonical model while the CLI migration consumes the
generated Connect contracts.
