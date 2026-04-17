# Resource And Scenario Dependency Orchestration Greenfield Plan

This plan covers the cleanup and redesign of scenario dependency orchestration so that scenario dependencies and resource dependencies follow one coherent lifecycle contract.

The objective is not just to make startup "work more often." The objective is to make dependency behavior:

- explicit
- consistent
- testable
- transactional
- easy to reason about
- implemented through clear ownership boundaries

This is a **greenfield orchestration cleanup**. It must be treated as a breaking simplification, not as an additive compatibility exercise.

## Greenfield Constraint

This work is explicitly **greenfield**.

That means:

- do **not** preserve conflicting legacy dependency semantics just because some manifests or code paths imply them today
- do **not** add compatibility wrappers that let both "old" and "new" orchestration models coexist indefinitely
- do **not** duplicate resource startup logic inside lifecycle when resource control already exists
- do **not** preserve partial-start behavior that leaks stale processes, stale lock files, or misleading runtime state
- do **not** keep dead fallback paths once the new orchestration flow is in place
- do **not** document contradictory dependency combinations as acceptable patterns

If a behavior is wrong or ambiguous, replace it cleanly and remove the old behavior in the same implementation stream.

## Why This Exists

The current repo already has most of the vocabulary required for a clean dependency model:

- scenario manifests already declare:
  - `enabled`
  - `required`
  - `startup_policy`
  - `degraded_behavior`
- scenario dependency recursion is already implemented
- resource control already has a proper service boundary for:
  - discovery
  - status
  - start
  - stop

But the implementation was split:

- scenario dependencies were actively orchestrated during scenario start
- resource dependencies were mostly treated as env/config injection plus scenario-specific bootstrap assumptions
- startup failure handling was not transactional enough
- stale fixed-port listeners and stale lock files could confuse later runs

Without a focused plan, the likely failure mode is platform drift:

- one set of semantics in schema
- another in lifecycle
- another in resource control
- another in docs
- and another in operator expectations

This plan exists to prevent that split-brain state.

## Key Findings From Investigation

These findings are the factual baseline for the implementation work.

### 1. Scenario dependency recursion already exists

Scenario startup currently walks `dependencies.scenarios` recursively in:

- [internal/lifecycle/dependencies.go](/home/matthalloran8/Vrooli/internal/lifecycle/dependencies.go:1)
- [internal/lifecycle/lifecycle.go](/home/matthalloran8/Vrooli/internal/lifecycle/lifecycle.go:187)

The current runtime behavior is:

- `required: true` with no explicit `startup_policy` behaves like `must_start`
- `required: false` with no explicit `startup_policy` behaves like `ignore`
- explicit `startup_policy` overrides that default

### 2. Resource dependency schema already carries similar fields

The manifest/schema already supports the same core fields for resources:

- [service.schema.json](/home/matthalloran8/Vrooli/.vrooli/schemas/service.schema.json:146)
- [resources.schema.json](/home/matthalloran8/Vrooli/.vrooli/schemas/resources.schema.json:64)

So the main gap is not schema shape. The main gap is orchestration behavior parity.

### 3. Resource dependencies are not orchestrated with the same rigor

Scenarios receive injected resource-derived env vars, but required resources are not consistently started and health-checked before scenario develop/startup continues.

In practice this means:

- API processes get correct `POSTGRES_*`, `DATABASE_URL`, `OLLAMA_*`, `QDRANT_*`, etc.
- if the resource is down, the scenario fails later in its own startup path

This is configuration delivery, not full dependency orchestration.

### 4. Failed starts are not clean enough

The current start path:

- starts dependencies
- allocates ports
- runs setup
- runs develop
- waits for health

But on failure after some steps have launched, cleanup is incomplete. This can leave:

- stale lock files
- misleading process metadata
- stale fixed-port listeners from older runs
- confusing status/health behavior on the next run

Relevant code:

- [internal/lifecycle/lifecycle.go](/home/matthalloran8/Vrooli/internal/lifecycle/lifecycle.go:187)
- [internal/lifecycle/health.go](/home/matthalloran8/Vrooli/internal/lifecycle/health.go:12)
- [internal/ports/ports.go](/home/matthalloran8/Vrooli/internal/ports/ports.go:1)
- [internal/process/process.go](/home/matthalloran8/Vrooli/internal/process/process.go:1)

### 5. Real runtime examples exposed the gap

During live validation:

- `swarm-manager` startup recursed into `prompt-manager`
- `prompt-manager` recursed into `agent-manager`
- `agent-manager` required `workspace-sandbox`
- `workspace-sandbox` failed because required Postgres was not running

Separately:

- `prompt-manager` and `workspace-sandbox` both hit `EADDRINUSE` on fixed UI ports due to stale `node server.js` listeners from older runs
- `agent-inbox` failed at the stricter UI type-check step, which is an expected standards backlog rather than an orchestration bug

## Current Progress Snapshot

The repo has already completed a meaningful portion of this plan.

### Completed Or Mostly Completed

- dependency semantics are now enforced in the manifest layer
- schema descriptions now explain `required`, `startup_policy`, and `degraded_behavior`
- scenario and resource dependencies now share one policy resolver in lifecycle
- resource dependencies are now orchestrated before scenario startup/bootstrap
- failed resource dependencies are surfaced as structured `FailedResources`
- failed health checks now trigger rollback for background runtime state in the main start path
- fixed-port orphan cleanup now exists for lifecycle-managed listeners

Representative files:

- [internal/scenario/scenario.go](/home/matthalloran8/Vrooli/internal/scenario/scenario.go:439)
- [internal/lifecycle/dependencies.go](/home/matthalloran8/Vrooli/internal/lifecycle/dependencies.go:1)
- [internal/lifecycle/lifecycle.go](/home/matthalloran8/Vrooli/internal/lifecycle/lifecycle.go:187)
- [docs/resources/configuration.md](/home/matthalloran8/Vrooli/docs/resources/configuration.md:1)

### Remaining Gaps

The refactor was not fully closed out at the time this snapshot was added. The main gaps were:

- make rollback cover pre-`develop` failures as rigorously as health failures
- make resource orchestration wait for readiness instead of performing only a single post-start status check
- extend smoke/unit coverage for required resources, transitive resource-backed dependencies, and stale fixed-port listeners
- finish cleanup/consolidation so the lifecycle package is easier to reason about and the plan text matches current reality

### Current Closure Status

These gaps have now been addressed in code and targeted tests:

- rollback now starts immediately after environment/lock allocation, so setup/bootstrap failures also clean up runtime state
- resource dependency orchestration now waits with bounded polling for post-start readiness instead of checking status only once
- lifecycle tests now cover:
  - required resource timeout behavior
  - optional `try_start` timeout/degraded behavior
  - setup-phase rollback cleanup
  - transitive scenario -> resource startup
  - fixed-port orphan cleanup ownership filtering

Representative files:

- [internal/lifecycle/lifecycle.go](/home/matthalloran8/Vrooli/internal/lifecycle/lifecycle.go:187)
- [internal/lifecycle/dependencies.go](/home/matthalloran8/Vrooli/internal/lifecycle/dependencies.go:1)
- [internal/lifecycle/lifecycle_test.go](/home/matthalloran8/Vrooli/internal/lifecycle/lifecycle_test.go:358)

## Related Code And Plans

- [Scenario CLI Manifest Greenfield Migration Plan](/home/matthalloran8/Vrooli/docs/plans/scenario-cli-manifest-greenfield-migration-plan.md:1)
- [Scenario Go CLI Standardization Greenfield Plan](/home/matthalloran8/Vrooli/docs/plans/scenario-go-cli-standardization-greenfield-plan.md:1)
- [service.schema.json](/home/matthalloran8/Vrooli/.vrooli/schemas/service.schema.json:1)
- [internal/scenario/scenario.go](/home/matthalloran8/Vrooli/internal/scenario/scenario.go:118)
- [internal/lifecycle/dependencies.go](/home/matthalloran8/Vrooli/internal/lifecycle/dependencies.go:1)
- [internal/resources/control/service.go](/home/matthalloran8/Vrooli/internal/resources/control/service.go:1)
- [internal/resources/resources.go](/home/matthalloran8/Vrooli/internal/resources/resources.go:1)

## Dependency Contract

This is the contract the implementation should enforce and document.

### Shared Fields

Both `dependencies.scenarios.<name>` and `dependencies.resources.<name>` should obey the same core semantics:

- `enabled`
  - whether the dependency is active in this manifest
- `required`
  - whether the dependency is semantically important for correct behavior
- `startup_policy`
  - lifecycle orchestration behavior
- `degraded_behavior`
  - human explanation of what fallback behavior exists when unavailable

### Operational Semantics

- `enabled: false`
  - dependency is skipped entirely

- `required: true` and no explicit `startup_policy`
  - defaults to `must_start`

- `required: false` and no explicit `startup_policy`
  - defaults to `ignore`

- `startup_policy: must_start`
  - orchestration must attempt startup and require health/readiness
  - failure blocks parent startup

- `startup_policy: try_start`
  - orchestration attempts startup
  - failure is recorded as degraded
  - parent startup may continue

- `startup_policy: ignore`
  - orchestration does not attempt startup
  - dependency may still be referenced in docs/config, but does not participate in startup

### Validation Guidance

These combinations should be treated as invalid or at least strongly warned:

- `required: true` with `startup_policy: ignore`
- `required: true` with `startup_policy: try_start` unless the scenario is intentionally designed to degrade

The clean standard is:

- `required` describes importance
- `startup_policy` describes orchestration behavior

## Target End State

At the end of this plan, the repo should converge on the following model.

### Orchestration

- lifecycle owns startup orchestration
- scenario dependency startup and resource dependency startup follow one policy model
- resource startup is delegated through resource control, not reimplemented inside lifecycle

### Transactionality

- failed start rolls back started processes
- stale process records are removed
- stale lock files are removed or corrected
- health failure does not leave a misleading "running" state

### Boundaries

- `internal/scenario`
  - manifest types, defaulting, validation
- `internal/resources/control`
  - canonical resource status/start/stop execution
- `internal/lifecycle`
  - dependency policy evaluation
  - dependency orchestration
  - rollback/cleanup
- `internal/ports` / `internal/process`
  - runtime bookkeeping, listener ownership, stale cleanup

### Documentation

- docs and schema descriptions state one dependency model
- no docs imply that resources are "special" in an undocumented way
- no docs preserve legacy/ambiguous combinations as acceptable alternatives

## Architecture Direction

The cleanest implementation seam is:

1. keep resource driver/status/start logic inside `internal/resources/control`
2. add a lifecycle-facing resource dependency orchestrator in `internal/lifecycle`
3. share dependency-policy interpretation between:
   - scenario dependency executor
   - resource dependency executor

This avoids two bad outcomes:

- lifecycle learning too much about resource drivers
- resource control learning too much about scenario startup recursion

### Recommended Internal Structure

Target direction inside `internal/lifecycle`:

```text
internal/lifecycle/
├── lifecycle.go
├── health.go
├── phases.go
├── setup.go
├── dependency_policy.go
├── scenario_dependencies.go
├── resource_dependencies.go
└── rollback.go
```

This does not need to be exact, but the responsibilities should be this clean.

## Phase 0: Freeze The Dependency Contract

1. Write a short internal dependency semantics note in `docs/`.
2. Define exact meanings for:
   - `enabled`
   - `required`
   - `startup_policy`
   - `degraded_behavior`
3. Define the defaulting rules clearly.
4. Decide which contradictory combinations are:
   - rejected
   - warned
   - allowed only with explicit rationale
5. Explicitly state that this contract applies equally to:
   - scenario dependencies
   - resource dependencies

Acceptance criteria:

- one-page contract exists
- no ambiguity remains around default behavior
- no dual interpretation is left to "tribal knowledge"

## Phase 1: Strengthen Manifest Validation Without Growing Schema

1. Keep schema shape mostly unchanged.
2. Add manifest/runtime validation in `internal/scenario` for dependency combinations.
3. Validate both scenario and resource dependencies with the same rules.
4. Ensure `enabled: false` dominates other fields.
5. Add tests for:
   - defaulting behavior
   - invalid combinations
   - preserved optional combinations

Primary files:

- [internal/scenario/scenario.go](/home/matthalloran8/Vrooli/internal/scenario/scenario.go:118)
- [internal/scenario/scenario_test.go](/home/matthalloran8/Vrooli/internal/scenario/scenario_test.go:440)

Acceptance criteria:

- dependency semantics are enforced in code
- tests describe the contract clearly
- no new manifest fields added unless implementation proves a real missing concept

## Phase 2: Extract Shared Dependency Policy Logic

1. Refactor lifecycle so dependency policy interpretation lives in one place.
2. Create a small internal policy evaluator that takes:
   - dependency metadata
   - caller startup mode (normal vs best-effort)
3. Return a normalized orchestration decision:
   - skip
   - must_start
   - try_start
4. Reuse this for both:
   - scenario dependency walking
   - resource dependency walking

Primary files:

- [internal/lifecycle/dependencies.go](/home/matthalloran8/Vrooli/internal/lifecycle/dependencies.go:1)
- [internal/lifecycle/lifecycle_test.go](/home/matthalloran8/Vrooli/internal/lifecycle/lifecycle_test.go:237)

Acceptance criteria:

- policy is interpreted in exactly one place
- scenario and resource dependencies do not drift semantically

## Phase 3: Add Resource Dependency Orchestration To Lifecycle

1. Add a lifecycle-owned resource dependency orchestrator.
2. It should:
   - inspect `item.Manifest.Dependencies.Resources`
   - normalize policy via the shared evaluator
   - use resource control to query status and start resources
   - require health for `must_start`
   - continue degraded for `try_start`
   - ignore `ignore`
3. Record failed resource dependencies in a structured result, parallel to failed scenario dependencies.
4. Ensure this runs before scenario-specific setup/bootstrap that assumes resource availability.

Primary reuse seam:

- [internal/resources/control/service.go](/home/matthalloran8/Vrooli/internal/resources/control/service.go:1)
- [internal/resources/resources.go](/home/matthalloran8/Vrooli/internal/resources/resources.go:1)

Key design rule:

- lifecycle may orchestrate resource startup
- lifecycle may not duplicate resource driver logic

Acceptance criteria:

- required resources are started and health-checked before scenario startup proceeds
- optional try-start resources produce degraded startup rather than hidden downstream failures
- resource orchestration reuses resource control rather than reimplementing it

## Phase 4: Make Startup Transactional

1. Refactor scenario start/restart so startup is rollback-capable.
2. If failure occurs after any background process has started:
   - stop started process groups
   - remove process records
   - clear or correct locks
   - clear degraded markers if the run never reached a valid running state
3. Ensure dependency failures, health failures, and late setup failures all go through the same cleanup path.

Primary files:

- [internal/lifecycle/lifecycle.go](/home/matthalloran8/Vrooli/internal/lifecycle/lifecycle.go:187)
- [internal/lifecycle/health.go](/home/matthalloran8/Vrooli/internal/lifecycle/health.go:12)
- [internal/process/process.go](/home/matthalloran8/Vrooli/internal/process/process.go:1)
- [internal/ports/ports.go](/home/matthalloran8/Vrooli/internal/ports/ports.go:1)

Acceptance criteria:

- failed start leaves no false-positive running state
- failed start leaves no stale process metadata for dead children
- failed start does not strand new lock files behind

## Phase 5: Tighten Fixed-Port And Orphan Cleanup

1. Before launching a fixed-port step:
   - inspect the port
   - inspect the lock
   - determine whether the owner PID is live
2. If the lock is stale, clean it before launch.
3. If an orphaned Vrooli-owned listener is holding the port, either:
   - clean it automatically when ownership is clear
   - or fail with a precise remediation message
4. Ensure status/diagnostics use the same underlying ownership signals.

This phase should reuse existing primitives rather than invent new ones:

- [internal/ports/ports.go](/home/matthalloran8/Vrooli/internal/ports/ports.go:1)
- `vrooli diagnose-port`

Acceptance criteria:

- fixed-port scenarios do not fail mysteriously because of dead lock owners
- orphaned listeners are surfaced or cleaned consistently
- startup does not silently bind against misleading old UI/API processes

## Phase 6: Separate Resource Availability From Scenario Database Bootstrap

1. Keep `ensureScenarioDatabase(...)` focused on scenario-owned schema/bootstrap work.
2. Do not let it remain the de facto place where missing resources are discovered.
3. Required resource availability must be established before this function runs.
4. After resource dependency orchestration, this bootstrap layer should only do:
   - database existence checks
   - schema.sql bootstrap
   - migration file execution

Primary file:

- [internal/lifecycle/setup.go](/home/matthalloran8/Vrooli/internal/lifecycle/setup.go:1)

Acceptance criteria:

- resource unavailability fails in dependency orchestration, not deep inside scenario bootstrap
- setup code has a narrower, cleaner responsibility

## Phase 7: Docs, Schema Descriptions, And Cleanup

1. Update schema descriptions so the dependency semantics are explicit and aligned with runtime.
2. Update lifecycle docs to describe the new resource orchestration behavior.
3. Remove docs that imply resources are merely env providers during startup.
4. Remove any dead or superseded code paths created obsolete by the refactor.

Acceptance criteria:

- docs match runtime behavior
- no legacy behavior remains documented as equally valid
- no dead orchestration branches remain in code

## Testing Strategy

The quality bar for this work should be enforced through layered tests.

### 1. Manifest And Contract Tests

Add or extend tests in:

- [internal/scenario/scenario_test.go](/home/matthalloran8/Vrooli/internal/scenario/scenario_test.go:440)

Cover:

- scenario dependency defaulting
- resource dependency defaulting
- invalid combinations
- `enabled: false` dominance

### 2. Lifecycle Unit Tests

Extend:

- [internal/lifecycle/lifecycle_test.go](/home/matthalloran8/Vrooli/internal/lifecycle/lifecycle_test.go:237)

Add cases for:

- required resource missing -> startup blocks
- try-start resource missing -> startup continues degraded
- ignored resource -> no orchestration attempt
- mixed scenario + resource dependency graphs
- rollback on health failure
- rollback on dependency failure
- stale lock cleanup behavior

### 3. Lifecycle Smoke Tests

Extend:

- [internal/lifecycle/lifecycle_smoke_test.go](/home/matthalloran8/Vrooli/internal/lifecycle/lifecycle_smoke_test.go:146)

Cover:

- scenario with required resource
- scenario with optional resource
- transitive scenario dependency that requires a resource
- fixed-port scenario with stale/orphaned listener

### 4. Resource Control Tests

Extend:

- [internal/resources/control/service_test.go](/home/matthalloran8/Vrooli/internal/resources/control/service_test.go:1)

Cover:

- status classification is stable enough for lifecycle orchestration
- lifecycle-facing start flow can rely on resource control semantics
- `must_start` and `try_start` orchestration decisions can consume resource control cleanly

### 5. CLI Regression Validation

Extend:

- [internal/resources/resource_cli_compat_test.go](/home/matthalloran8/Vrooli/internal/resources/resource_cli_compat_test.go:1)

Keep a lightweight regression check for fresh build/install behavior of:

- scenario CLIs using `cli-core`
- resource CLIs using manifest-native control

## Manual Validation Matrix

Before considering the rollout complete, validate at least:

- `workspace-sandbox`
  - required Postgres path
  - fixed UI port behavior
- fresh scenario/resource CLI build remains healthy enough to support validation flows
- `agent-manager`
  - transitive dependency on `workspace-sandbox`
- `prompt-manager`
  - required Postgres
  - optional `agent-manager`
  - optional `qdrant` / `ollama`
- `swarm-manager`
  - transitive scenario dependency graph
- `agent-inbox`
  - ensure stricter UI checks remain the only blocker when applicable

Manual scenarios to cover:

- required resource stopped
- required resource healthy
- optional try-start resource stopped
- stale fixed-port listener present
- clean start after failed start
- fresh CLI build/install from clean state

## Definition Of Done

This work is done only when all of the following are true:

- there is exactly one dependency policy model in the codebase
- scenario and resource dependencies use the same semantics
- resource orchestration reuses resource control rather than duplicating driver logic
- failed starts are transactional enough that they do not leave stale runtime debris
- fixed-port startup is robust against stale locks and orphan listeners
- tests cover the dependency contract at manifest, lifecycle, and control seams
- dead fallback/compatibility code is removed
- docs and schema descriptions match the runtime behavior

## Review Checklist

Any implementation of this plan should be reviewed against these questions:

- Is dependency policy interpreted in exactly one place?
- Does lifecycle orchestrate resources through resource control rather than bypassing it?
- Can a failed start leak stale processes, stale locks, or misleading running state?
- Are contradictory dependency configurations rejected or clearly surfaced?
- Did the change delete old logic instead of layering over it?
- Can a new engineer understand the orchestration flow by opening a small, obvious set of files?
- Do tests prove behavior at clean seams rather than only through broad end-to-end runs?

If any of those answers is "no", the implementation is likely increasing debt.

## Recommended Execution Order

1. Freeze the dependency contract in docs.
2. Add manifest validation and defaulting tests.
3. Extract shared dependency policy interpretation.
4. Implement lifecycle resource dependency orchestration via resource control.
5. Refactor startup into a transactional flow with rollback.
6. Tighten stale-port/orphan handling.
7. Fix the `cli-core` consumer module contract.
8. Run unit, smoke, and manual validation matrix.

## Main Risk

The dangerous failure mode is half-migration:

- scenario dependency semantics stay in one place
- resource dependency semantics get bolted on differently
- setup still performs implicit infrastructure discovery
- startup rollback remains partial
- old and new paths both linger

That would preserve hidden coupling and make future cleanup harder.

The safe path is:

- one dependency contract
- one orchestration policy model
- one resource-control seam
- one startup/rollback path
- one test suite that defines the behavior
