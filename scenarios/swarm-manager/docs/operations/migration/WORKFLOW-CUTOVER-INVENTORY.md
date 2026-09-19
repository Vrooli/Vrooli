# Workflow Cutover Inventory

> **Status: historical migration rationale and retirement checklist.** The
> machine-readable [`registry.json`](../../../.vrooli/swarm-transitions/registry.json)
> and its [Active Transition Catalog](../../reference/transition-catalog.md)
> describe current behavior. This document explains why the former
> agent-operation and operating-mode runtime must not return, and records the
> checks required before historical projections are deleted. It does not
> authorize deletion or data migration by itself.

## Goal and boundary

Swarm Manager remains the project-work ledger and operator control plane. Agent
Manager becomes the sole runtime for programmatic agent methodology.

```text
Swarm owns: input snapshots, authorization, domain validation, exactly-once apply
Agent Manager owns: prompts, runs, result contracts, retries, loops, waits, routing
External systems own: plan validity and test/regression evidence
```

The decision rule is in [Target Operating Model](../../concepts/TARGET-OPERATING-MODEL.md):
human-led conversation remains an Agent Session backed by a Run; code-composed
and code-consumed agent work is a declared Workflow. Deterministic domain
actions are neither.

## Migration invariants

- Preserve backlog, initiatives, goals, dependencies, `plan_ref`s, records,
  events, sessions, and historical provenance.
- No active execution changes identity or semantics in place. A new workflow
  execution starts under a new authorized intent; the prior execution remains
  readable history.
- Swarm only applies a terminal workflow result when the workflow revision,
  input/entity snapshot, plan frontier, and required evidence still match.
- Agent Manager must return typed outcomes. Swarm must not retain transcript
  parsing, prompt construction, output classification, or orchestration loops.
- A legacy persisted projection is removed only after its replacement is live,
  historical reads have a defined destination, and an inventory-backed one-shot
  migration has passed reconciliation.
- The existing safety protocol in [RUNBOOK.md](./RUNBOOK.md) applies to every
  persisted-state change: backup, quiescence, staged replacement, identity
  reconciliation, and rollback.

## Ordered program

| Wave | Objective | Why it comes now | Exit evidence |
| --- | --- | --- | --- |
| **A. Canonical contract** | Ratify target concepts, integration roles, transition grammar, and the workflow catalog. | Prevents a mechanical port of modes into differently named workflow files. | Documentation approved; each transition has an owner and a typed boundary. |
| **B. Integration SSOT** | Build one provider for integration availability, freshness, degraded behavior, and workflow-start preflight; project it in Settings/API/CLI. | Workflows cannot be safely selected or explained while dependency truth is scattered. | Agent Manager, Plan Manager, Prompt Manager, Test Genie, Control Tower, and ecosystem dependencies report one consistent status model. |
| **C. Transition SSOT** | Create JSON-authored transition registrations that map a domain transition to Session/Run, Workflow key, or deterministic action. | Makes the decision rule inspectable and validates that no code-only agent pathway remains. | Registration validates schemas, input/output contracts, integration prerequisites, and permitted terminal outcomes. |
| **D. Low-risk workflow pilots** | Complete the existing workshop and plan-drain pilots; add plan author/repair with Plan Manager validation. | Exercises immutable input, result application, bounded repair, approval waits, and evidence without migrating the whole project model. | Production-like Test Genie and Control Tower evidence; exact-once apply and recovery gates pass. |
| **E. Backlog lifecycle cutover** | Move classification, refinement, clarification, finalization, research, review, fixup, follow-up, and retry into workflows. | These are repetitive current agent-method flows with clear item boundaries. | New item work no longer starts operating-mode operations or direct programmatic Runs. |
| **F. Initiative cutover** | Move initiative investigation, plan creation/repair, execution/review/reconciliation loops into workflows. | Initiatives depend on the proven item and evidence contracts, but need their own scope and membership snapshots. | Initiative state is a domain container, never a Swarm-owned agent state machine. |
| **G. Tail workflows and retirement** | Migrate scenario spec sync and remaining special cases; migrate/archive legacy projections; remove obsolete APIs, UI, packages, declarations, tests, and documentation. | Deletion is safe only after normal project work no longer relies on the incumbent runtime. | No programmatic direct-Run spawn or operating-mode/agent-operations runtime remains in the active path. |

The desired removal target is a success measure, not a pre-approved deletion
budget. Each wave records actual removed code and artifacts; only the completed
inventory can establish whether the aggregate reaches the target.

## Transition inventory

### Intake and conversational work

| Current concern | Target transition | Target mechanism | Stored-state disposition | Wave |
| --- | --- | --- | --- | --- |
| Capture classification | `capture.classify` | Declared `swarm-manager/capture-classify` workflow; Swarm applies only a matching terminal typed result. | Keep captures and classification history; persist execution, definition digest, and immutable capture-snapshot version. | E |
| Meta-orchestration conversation | `session.meta_orchestration` | Agent Session / Run. | Retain session/message/proposal model. | A |
| Swarm operations conversation | `session.swarm_operations` | Agent Session / Run. | Retain session and advisory/proposal boundary. | A |
| Historical methodology-authoring conversation | None; historical sessions only | No new session kind. Discussions use `session.meta_orchestration`; code-owned outcomes become declared workflows. | Retain history; do not create a generic chat type. | G |
| Session proposal application | `proposal.apply` | Deterministic Swarm validation and apply. | Retain proposal and artifact provenance. | A |

### Backlog shaping and plan validity

| Current concern | Target transition | Target mechanism | Stored-state disposition | Wave |
| --- | --- | --- | --- | --- |
| Workshop synthesis round | Retired | Replaced by the explicit `plan.workshop.review` and `plan.workshop.reconcile` session workflow pair. | Historical item, decision, and workshop records remain read-only. | Retired |
| Clarification start/continue | Retired | Replaced by one Plan Workshop operator response; open-ended discussion belongs in Agent Sessions. | Historical clarification threads remain read-only. | Retired |
| Delayed workshop auto-advance | Retired | Plan Workshop has no scheduled or automatic continuation. | Historical scheduler records remain read-only evidence only. | Retired |
| Workshop finalization | `plan.author` | Declared `swarm-manager/plan-author` workflow returns a candidate plan; Swarm imports it through Plan Manager, requires a passing render verdict, then binds once. | Preserve old rounds as evidence; persist workflow correlation/provenance and keep `plan_ref` canonical. | E |
| Invalid or stale plan | `plan.repair` | Workflow receives Plan Manager validation findings, repairs the plan, then revalidates under a bounded policy. | No new Swarm readiness-loop state; retain validation evidence and terminal reason. | D |
| Research refinement | Retired | Research review uses the generic Plan Workshop review and reconciliation pair. | Keep research item and supporting artifacts as read-only history. | Retired |
| Research conclusion | `research.conclude` | Declared `swarm-manager/research-conclude` workflow owns bounded conclusion rounds and emits one typed Plan Workshop finding/disposition; the execution-record terminal apply adapter remains the migration boundary. | Preserve conclusion and evidence; replace the execution-mode record correlation with workflow provenance. | E |

### Plan execution, review, and evidence

| Current concern | Target transition | Target mechanism | Stored-state disposition | Wave |
| --- | --- | --- | --- | --- |
| Execute a valid plan | `plan.execute` | `swarm-manager/phased-plan-drain` workflow. | Execution record keeps authorized intent and workflow correlation; Agent Manager owns attempts and loop state. | D → E |
| Slice review | `work.review` | Declared `swarm-manager/independent-review` workflow starts from an immutable execution snapshot and emits one typed Plan Workshop finding/disposition; Swarm retains the review ledger, snapshot validation, exactly-once apply, and operator gate. | Workflow-owned rounds carry pinned workflow provenance and are excluded from legacy polling/recovery; historical operation rounds remain readable. | D complete; E for correction composition |
| Review evidence request | `review.evidence_request` | Declared `swarm-manager/evidence-request` workflow receives an immutable review-thread snapshot; Swarm applies the terminal typed evidence result to that thread exactly once. | Preserve review evidence and messages; new threads pin workflow provenance and do not create an operation correlation. | E in progress |
| Review asks for correction | `work.correct` | Declared `swarm-manager/work-correct` workflow starts from an immutable parent execution/finalization snapshot. Swarm retains only authorization, exact-once terminal apply, and re-review routing. | Keep final evidence and decision; legacy operation executions are historical-read only. | E in progress |
| Test/baseline regression | `work.correct` or explicit attention | Typed Test Genie/Control Tower evidence informs the existing correction or operator-decision path. | Persist evidence and result application; no transcript rediscovery. | E |
| Execution retry | `plan.execute` | A new phased-plan drain binds the canonical Plan Manager execution and resumes its frontier. | Historical execution remains immutable; no in-place retry correlation. | E in progress |
| Post-completion follow-up | `work.follow_up` | Declared `swarm-manager/work-follow-up` workflow starts from an immutable parent execution/finalization snapshot. Swarm applies the typed terminal result exactly once. | Link parent/child work in the ledger and records; no active operation-runner correlation. | E in progress |
| Cancellation, resume, approval | Execution controls | Swarm's execution endpoints authorize Agent Manager workflow cancel/signal operations directly. | Keep control audit with the execution record; no inert registry declaration. | D → E |

### Initiative and scenario work

| Current concern | Target transition | Target mechanism | Stored-state disposition | Wave |
| --- | --- | --- | --- | --- |
| Initiative investigation and work breakdown | `initiative.discover` | Workflow producing typed proposal(s) for plans, member items, dependencies, and decisions. | Initiative remains the goal container; proposals apply through Swarm. | F |
| Initiative plan author/repair | `initiative.plan` | Workflow plus Plan Manager validation. | Preserve initiative `plan_ref`, membership, and acceptance criteria. | F |
| Initiative execution/replanning | `initiative.execute` | Composition of plan execution, evidence, and explicit attention signals—not a Swarm mode engine. | Workflow owns loop state; Swarm owns goal/initiative state. | F |
| Initiative review/reconciliation | `initiative.review` | Declared `swarm-manager/initiative-review` workflow receives an immutable initiative snapshot. Swarm persists the workflow-owned round, applies its typed assessment and Plan Workshop finding/disposition exactly once, releases the initiative lock, and retains the operator decision. | Retain acceptance evidence and decisions; historical operation rounds remain readable. | F in progress |
| Scenario specification sync | `scenario.spec_sync` | Declared `swarm-manager/scenario-spec-sync` workflow with a portable immutable scenario/archive snapshot. Swarm alone applies the typed result, archives, and deletes exactly once. | Preserve scenario records and archive context; active paths have no special operation/mode correlation. | G in progress |

## Current runtime components and expected disposition

| Component | Present responsibility | Target disposition |
| --- | --- | --- |
| `internal/operatingmode/` | Former mode grammar and phase engine. | Retired. The transition registry plus declared Agent Manager workflows are the only active workflow selection contract. |
| `internal/agentops/` | Former operation, binding, policy, provenance, and workflow-instance model. | Retired and deleted. Swarm retains only transition registration; it has no second workflow model. |
| `internal/opsrunner/` | Former persisted workflow runner and scheduler. | Retired and deleted. Agent Manager owns execution, journaling, retry, and control. |
| `internal/opsbridge/` and `internal/opscatalog/` | Former completion routing and catalog glue. | Retired and deleted with the agent-operations runtime. |
| `internal/agentmanager/` | Both direct Run integration and workflow adapters. | Retain Session/Run support and a generic workflow client; replace feature-specific adapters with one typed workflow invocation/apply seam. |
| `internal/execution/` | Mix of essential authorization/apply state and legacy orchestration, polling, finalization, retry/follow-up logic. | Split deliberately: retain authorization, snapshot, control, exactly-once apply, and domain projection; retire agent-method loops and duplicate polling after workflow migration. |
| `internal/review/` | Evidence domain plus agent review-round orchestration. | Retain evidence ownership and operator decisions; move review/fixup flow execution to workflows. |
| `internal/agentactivity/` | Observability for Runs and programmatic activity. | Keep/reduce to a projection that can correlate Session Runs and workflow executions; remove it as an execution authority. |
| `modes/`, `operation-contracts/`, `bindings/`, `policy/` | Former authored data for the incumbent runtime. | Retired and deleted. Historical state remains read-only; declared Agent Manager workflows and Swarm transition registration are authoritative. |

## SSOT requirements before broad migration

### Integration status

One integration-status provider owns checks and projections for:

```text
Agent Manager      workflow catalog and execution
Plan Manager       plan validation and rendering
Prompt Manager     prompt reference resolution
Test Genie         test runs and result freshness
Git Control Tower  baseline and regression verdicts
```

For each integration it returns availability, configured/required state,
freshness, degraded behavior, and affected transition keys. Settings, API, CLI,
and workflow-start preflight consume that same projection. Git Control Tower's
structured health and review-result projections are useful design references;
Wave B must verify the exact provider contract rather than assuming an existing
reusable integration-health component.

### Transition registration

Transition registration is JSON desired state owned by Swarm. It is not a new
workflow engine. Its only job is to make the domain boundary inspectable and
validate the correct runtime choice.

```json
{
  "schemaVersion": "swarm-transition/v1",
  "key": "plan.repair",
  "subject": "backlog-item",
  "kind": "workflow",
  "workflow": { "owner": "swarm-manager", "key": "swarm-manager/plan-repair" },
  "requires": ["plan-manager", "agent-manager"],
  "inputContract": "plan-repair-input/v1",
  "terminalOutcomes": ["ready", "needs_attention", "abstained", "failed"],
  "applyAction": "bind_validated_plan_ref"
}
```

The final schema must also support `session` and `deterministic` kinds. It must
not express prompts, branch conditions, retries, or run mechanics; those remain
inside an Agent Manager workflow declaration.

#### Current registry implementation

`scenarios/swarm-manager/.vrooli/swarm-transitions/registry.json` is the
inspectable source for the target transition catalog. The transport-free loader
at `api/internal/transitions` accepts only `swarm-transition/v1` definitions and
only the `session`, `workflow`, and `deterministic` kinds. It validates required
contracts, terminal outcomes, workflow locators, and integration requirements;
its strict JSON decoder rejects unknown fields. This specifically prevents a
registration from carrying a prompt, branch, retry, loop, or scheduler setting.

The registry currently records all 23 target transitions in this inventory,
including conversational sessions and deterministic proposal/control actions.
It is a Phase 3 foundation, not evidence that each workflow is implemented or
wired: Agent Manager declaration reconciliation, integration preflight, generic
start/control/result application, and deletion of the incumbent runtime remain
required before any registered workflow is active. The registry test is the
coverage gate for the catalog, while later integration tests must prove every
workflow locator exists in Agent Manager and each `applyAction` is registered.

`api/internal/integrationstatus` is the paired Phase 2 provider foundation.
Integration-specific adapters implement its narrow `Checker` seam; the provider
normalizes configured state, availability, freshness, degraded behavior, and a
diagnostic into one projection. Its `Preflight` consumes a transition
registration's `requires` list and fails closed for unknown, unavailable, stale,
unconfigured, or expired dependencies. It is intentionally not yet exposed by
the legacy settings/status handlers: replacing their duplicate health decisions
is a subsequent wiring step, not a claim that it has happened already.

The read-only API projection is registered at `GET /api/v1/integrations`. It
checks the six required scenario integrations through their standard health
endpoints and reports their explicit degradation behavior. The provider derives
each status's `affectedTransitions` from the loaded transition registry, rather
than accepting a second hand-maintained dependency map. Settings, CLI, and the
registered workshop/plan-drain starts consume this projection/provider and do
not perform local health probes. Later workflow adapters must use this same
preflight seam.

The operator CLI now exposes the same projection as `swarm-manager integrations`
(`--json` preserves the raw API payload). It performs no local probes, so its
availability, freshness, diagnostics, and degradation story cannot diverge from
workflow-start preflight.

The existing workshop and phased-plan adapters use
`agentmanager.WorkflowService`'s generic invocation protocol: start with a
workflow key plus immutable input, collect a terminal execution with its pinned
digest/input/output/journal attempts, and send/cancel a durable workflow
operation. Their domain packages retain only snapshot construction and typed
result decoding. New transitions must use this generic seam rather than adding
another feature-specific Agent Manager workflow client.

The active workshop and phased-plan starts now load `swarm-transition/v1` at
server composition, require a matching workflow registration, and call the same
integration-status `Preflight` used by the operator API. This closes the former
gap where a pilot could start Agent Manager directly despite a stale dependency.
Minimal temporary-root test fixtures intentionally omit declarations and do not
install this production guard; malformed registries on a real scenario root
still fail server initialization.

Terminal pilot results are no longer applied by Swarm polling Agent Manager.
Workshop retains a bounded boot-time scan of durable correlations; both workshop
and phased-plan execution use explicit, idempotent domain apply endpoints for
normal and post-restart completion. This keeps Agent Manager authoritative for
terminal workflow state and makes the consumer mutation an auditable command.

`swarm-manager/plan-repair` is the third declared pilot. It receives a bounded
entity snapshot, canonical plan content/frontier, and concrete Plan Manager
findings; it may return a candidate plan, attention, or abstention, but never a
claim of validity. `agent-manager declarations reconcile-scenario --scenario
swarm-manager --validate-only` validates this declaration with the scenario's
real catalog contract. The bounded backlog pilot is exposed at
`POST /api/v1/backlog/{kind}/{name}/plan-repair` and its explicit terminal
apply endpoint. It persists the authorized workflow revision/entity/plan
frontier, runs the same transition-registry integration preflight as the other
pilots, and validates those values again before apply. A ready candidate is
imported as an explicit Plan Manager supersession, rendered with `pass` quality,
and only then has its `plan_ref` moved exactly once. There is no in-place plan
mutation or unvalidated candidate bind.

The declaration has been reconciled into Agent Manager's catalog as revision
`sha256:9af0a0addd934d9c15a35c47e68e7713f9a80d4d3b520109be1a42df5c66762e`.
That digest is operational evidence for this source revision only; a source or
prompt change requires a new reconcile and creates a new pinned revision.

## Pilot acceptance gates

The early cohort must prove the entire boundary before broader deletion begins:

- A valid workflow revision is reconciled and pinned before start.
- The consumer passes a bounded immutable input snapshot and records its digest.
- Typed output is validated by Agent Manager and by the owning authority where
  applicable (for example, Plan Manager validates a plan).
- An invalid plan follows bounded repair and ends in a truthful attention state
  when it cannot be repaired.
- Approval, cancellation, and retry are idempotent and survive a process restart.
- Test Genie and Control Tower evidence is correlated to the execution without
  treating agent prose as validation proof.
- Swarm applies a terminal result once, rejects stale entity/frontier results,
  and preserves readable provenance.
- Historical current-state records remain visible throughout the pilot.

## Required pre-implementation inventory work

Before any wave changes behavior, expand every row above into the machine-usable
inventory fields below and validate them against live code and fixtures:

```text
current entry point
current spawned-run path
input source and sensitivity
output / classification path
persisted files and schema
owner and version correlations
integration dependencies
current UI/CLI/API surfaces
target transition key and workflow declaration
apply action and idempotency key
historical-read strategy
migration / rollback procedure
candidate files and tests for deletion
```

This is the evidence needed to estimate removal accurately and to author the
implementation plan. It is deliberately more rigorous than a code search: a
transition is not eligible for cutover until its domain mutation and persisted
history have an explicit destination.

## References

- [DOC: ../../concepts/TARGET-OPERATING-MODEL.md]
- [DOC: ./RUNBOOK.md]
- [DOC: ./LEGACY-MAPPING.md]
- Retired operating-mode, agent-operation, and operation-runner packages
- [CODE: api/internal/execution/phased_plan_workflow.go]
- [CODE: api/internal/agentmanager/phased_plan_workflow.go]
