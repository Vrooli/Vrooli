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

## Development review boundary

`internal/development.Reviewer` is a read-only domain service, exposed through
the generated Connect `TransitionService.PreviewDevelopment` contract. The CLI
uses that API; it does not render its own goal or read server target files.
The service has no runner, approval store, shell, or provider dependency.

The server selects the repository root. Candidate paths must be reviewable
repository-relative artifact types. Reads use `os.OpenRoot`, reject non-regular
files and final symlinks, and enforce 512 KiB per file and 64 resolved files.
Requests bound inventories and text; CLI JSON is limited to 128 KiB. Only
hashes, sizes, paths and caller-proposed text are returned, not file contents.

The preview is not a semantic contract validator, target publisher, immutable
snapshot owner, work-item resolver, budget ledger, or authority grant. Its
launch blockers remain explicit even when every field is present. The separate
`development.Service` now owns retained approval and accounting, described below;
adding a second execution loop here is not permitted.

Focused tests cover stable fingerprints, relevant-target/budget/criterion
changes, missing fields, path traversal, escaping symlinks, oversized files,
transport mapping, and CLI rendering. Runner fakes assert that preview never
calls start or apply. These tests do not qualify mandate execution.

### Retained development domain

- `development.Repository`: immutable snapshot reads and atomic aggregate
  compare-and-swap. `SQLiteRepository` uses the existing routed DB; schema lives
  beside its owner and is registered at boot. There is no private data root.
- `development.Service`: approved target revisions, aggregate reservations,
  owner execution binding, known-usage settlement, revocation and evidence-bound
  acceptance. Internal Reserve/BindExecution/Settle methods are not public RPCs.
- `development.WorkValidator`: composition-root callback resolves an existing
  backlog item through its routed data root and refuses incompatible plan work.
- `development.EvidenceResolver`: production adapter still absent. Resolve must
  fetch owner receipts; CurrentRevision must identify the tested product state.
  Agent-submitted status, a readable test file and a passing harness event are
  not implementations of this boundary.
- `development.Coordinator`: fallback owner adapter now reserves before start,
  carries the Agent Manager engagement grant, fences execution/digest binding,
  propagates cancellation, and settles measured terminal usage. It is not yet
  registered as a transition; native-goal parity and provider-owned evidence
  remain explicit launch blockers.
- `DevelopmentService` Connect API and generated CLI/UI consumers: typed review
  and decision transport only. Human decisions use `api-core/authn` and require
  `swarm-manager:write`. The service manifest requires
  `scenario-authenticator`; provider unavailability and missing credentials
  still fail closed.
- `Server.guardDevelopmentWorkShape`: prevents a retained development item from
  reaching an ordinary plan workflow. It is not the future development launcher.

Runtime integration must use the sole transition runner, reserve actual owner
limits before dispatch, reconcile ambiguous starts with the same identity, and
settle measured usage after owner cancellation. The current core tests exercise
these state transitions with explicit owner fixtures, not actual harness runs.

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

## Shared UI and test utility boundaries

The UI owns reusable API/query behavior in `ui/src/lib/` and test-only provider,
browser, storage, and console setup in `ui/src/test-utils/`. New tests should
extend those helpers rather than recreate QueryClient, router, browser-mock, or
console-handling scaffolding locally. Shared chat primitives and session artifact
routing likewise own cross-surface rendering and mapping behavior.

## Inter-scenario contracts and intentional exceptions

Typed generated contracts are the default for structured UI/API and
Swarm-to-Agent-Manager traffic. Clients resolve dependency URLs per request and
propagate context with bounded timeouts. File-content endpoints remain raw or
streamed by design, and the ecosystem client remains JSON-based until a
behavior-equivalent swarm-manager proto exists. These exceptions are contract
decisions, not evidence that the retired audit snapshots are still required.


## Completed execution evidence in independent review

Automatic finalization and manual review both load bounded Agent Manager evidence
before persisting the review round snapshot. The owner reads bind the Swarm
execution, approval and grant, follow exact child attempts, and retain terminal
results, fresh-worker identities, finalization status and aggregate usage.
Run summaries remain reported evidence; lifecycle success does not establish a
test result or applied-change provenance. Owner failures and read/size limits are
explicit unavailable observations. The reader performs no start or wait mutation.

The reader caps one snapshot at 45 seconds, 1024 owner reads and 512 KiB. The independent review binding permits 1 MiB total snapshot input and Agent
Manager bounds the rendered prompt at 2 MiB. The skill inserts each snapshot
once. The read count covers
the ordinary 128-slice shape; large payloads or slow owners can still exceed a
bound. Such a result requires a separate owner evidence inspection, not a
silently truncated passing review.

Self-restart checks remain in the affected scenario inventory as skipped.
Their aggregate is `not_assessable` with external restart, health and review
follow-up; deferral alone never requests an automatic coding fixup. A real
failure or enabled baseline regression still yields `needs_work`. Execution
finalization finishing does not mean that the operator accepted the item.
