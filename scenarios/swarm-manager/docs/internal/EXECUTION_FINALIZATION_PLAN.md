# Swarm Manager Multi-Scenario Execution Finalization Plan

## Document Metadata
- Owner scenario: `swarm-manager`
- Date: 2026-03-30
- Status: Draft implementation plan
- Scope: Post-run finalization redesign for restart + health + multi-scenario review

## Purpose
Unify the end-of-execution flow for eligible Swarm Manager executions so that, once an agent-manager run finishes, Swarm Manager:

1. Resolves the actual affected scenarios.
2. Restarts those scenarios sequentially.
3. Verifies they come back healthy using the standard Vrooli health contract.
4. Retries once on transient restart/health failures and records warnings when that recovery path is used.
5. Runs readiness review for every affected scenario.
6. Aggregates the per-scenario outcomes into one execution result and one follow-up/fixup context.

This plan also replaces the current `0..1 review` assumption across the API, persisted model, and UI with a `0..N scenario finalization results` model.

## Required Prerequisite Skills
Before implementing any phase in this plan, engineers/agents MUST run:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health intent-clarification seam-discovery-and-enforcement utils-unification test
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

These prerequisite commands were executed on 2026-03-30 while preparing this plan.

## Required Reading
Read these files before implementation:

```bash
# Current execution lifecycle
cat scenarios/swarm-manager/api/internal/execution/model.go
cat scenarios/swarm-manager/api/internal/execution/service.go
cat scenarios/swarm-manager/api/internal/execution/handler.go
cat scenarios/swarm-manager/api/internal/execution/review_client.go

# Agent-manager seams
cat scenarios/swarm-manager/api/internal/agentmanager/client.go
cat scenarios/swarm-manager/api/internal/agentmanager/service.go
cat packages/proto/schemas/agent-manager/v1/api/service.proto
cat packages/proto/schemas/agent-manager/v1/domain/run.proto

# Scenario lifecycle + health seams
cat scenarios/swarm-manager/api/internal/scenarios/lifecycle.go
cat scenarios/swarm-manager/api/internal/scenarios/cli_source.go
cat scenarios/swarm-manager/api/internal/scenarios/exec.go
cat cli/commands/scenario/modules/status.sh
cat cli/commands/scenario/modules/lifecycle.sh
cat cli/commands/scenario/validators/health-validator.sh

# Review engine behavior
cat scenarios/git-control-tower/api/review_model.go
cat scenarios/git-control-tower/api/review_handler.go
cat scenarios/test-genie/bas/README.md

# Current UI contract and consumers
cat packages/proto/schemas/swarm-manager/v1/domain/execution.proto
cat scenarios/swarm-manager/ui/src/types/domain.ts
cat scenarios/swarm-manager/ui/src/services/proto-contracts.ts
cat scenarios/swarm-manager/ui/src/pages/BacklogDetailsPage.tsx
cat scenarios/swarm-manager/ui/src/pages/ExecutionDetailsPage.tsx
cat scenarios/swarm-manager/ui/src/components/execution/execution-card.tsx
cat scenarios/swarm-manager/ui/src/components/execution/follow-up-dialog.tsx
```

## Confirmed Product Decisions
- Eligible executions for this unified flow are `process`, `fixup`, `followup`, and `custom`.
- Excluded executions are `research`, `workshop`, `finalize`, and `spec-sync-archive`.
- Actual affected scenarios should come from agent-manager sandbox diff when available.
- `acceptance_allow` is the fallback scope source when exact changed files are unavailable.
- Scenario restarts should be sequential.
- Reviews should run only after all restart/health work has settled.
- Scenario-level reviews should be sequential in v1.
- Health success means the scenario is running and its standard health contract reports healthy/ready through the normal Vrooli mechanism.
- Restart/health should retry once after a bounded wait window, and any recovery should be persisted as a warning.
- Missing or invalid health schema must be surfaced as a warning/failure condition, not silently ignored.
- The UX must move from `0..1 review` to `0..N scenario finalization results`.

## Problem Statement
The current end-of-execution logic is too lossy for multi-scenario work:

- Swarm Manager currently collapses review onto the first scenario derived from `acceptance_allow`.
- The actual changed scope is ignored even though agent-manager exposes per-file diff data for sandboxed runs.
- No scenario restart happens before review, so post-run validation can happen against stale runtime state.
- Swarm Manager currently stores only one `review_job_id`, one `review_result`, and one `review_skip_reason`.
- UI surfaces assume there is at most one review outcome to display and at most one review summary to pass into fixup/follow-up flows.
- Execution progression is currently advanced mostly from `Get` and `List` calls, which is weak for a longer post-run orchestration flow.

This causes three concrete failures:

1. Multi-scenario executions can appear "reviewed" when only one affected scenario was actually checked.
2. Review and follow-up work can proceed against stale runtime state.
3. Health-contract violations can prevent trustworthy validation while remaining under-reported.

## Current Technical Context

### 1. Exact changed-file data is available, but Swarm Manager does not currently consume it
- Agent-manager exposes `GET /api/v1/runs/{run_id}/diff`.
- `RunDiff.files` contains per-file paths and change types.
- `Run.summary` also exposes `files_modified`, `files_created`, and `files_deleted`.
- Swarm Manager's agent-manager client currently wraps `GetRun`, but not `GetRunDiff`.
- Swarm Manager's `RunState` already captures `sandbox_id`, which can be forwarded to review requests later.

### 2. The lifecycle restart seam already exists
- Swarm Manager already has a `scenarios.Lifecycle` interface with CLI-backed `Start`, `Stop`, and `Restart`.
- `vrooli scenario restart <name>` is the canonical restart path.
- The CLI restart path forces setup to run so changed code is rebuilt rather than reusing a stale healthy process.

### 3. The health contract already exists and is stronger than Swarm Manager currently uses
- `vrooli scenario status <name> --json` already returns:
  - `scenario_data.status`
  - `scenario_data.health_status`
  - `diagnostics.health_checks.*.available`
  - `diagnostics.health_checks.*.schema_valid`
  - `diagnostics.health_checks.*.status`
- The CLI health validator enforces required fields such as `status`, `service`, `timestamp`, and `readiness`.
- UI health also requires `api_connectivity`.
- Swarm Manager's current scenario inventory only uses `vrooli scenario list --json`, which loses this health detail.

### 4. Git Control Tower review is singular per scenario
- `ReviewRunRequest` accepts exactly one `scenarioName`.
- Git Control Tower rejects concurrent review runs for the same scenario.
- Within a single scenario review, Git Control Tower runs checks in parallel across `tidiness`, `tests`, and `rules`.
- `expectedPaths` and `sandboxId` are accepted by the review request but are not yet used by the current execution path in Git Control Tower.

### 5. Test Genie can restart scenarios during review-related checks
- Playbooks isolation provisions temporary resources and can restart the scenario under test.
- Normal runs restore the scenario back onto its usual resources afterward.
- This makes scenario-level sequential review the safer default for v1 because it avoids overlapping review-induced restarts across multiple scenarios.

### 6. Swarm Manager's review contract is singular end-to-end
- Persisted record: one `review_job_id`, one `review_result`, one `review_skip_reason`, one `review_started_at`.
- Proto contract: same singular fields.
- UI domain types: one optional `reviewResult`.
- UI consumers: backlog details, execution details, execution cards, follow-up dialog, and supporting contract mappers all assume `0..1`.

### 7. Execution progression currently depends too much on read traffic
- `refreshRunningLocked` advances agent-manager run state and review polling during `Get` and `List`.
- The execution page polls every 6 seconds, but the server-side scheduler only handles scheduled starts.
- A richer finalization flow should not depend on the operator keeping a page open.

## Target End State

### Top-level lifecycle
Use `validating` as the umbrella post-run state for eligible executions, but make its sub-phase explicit in the record.

```text
starting/running/needs_review
  -> completed by agent-manager
  -> validating(scope_detection)
  -> validating(restarting)
  -> validating(health_check)
  -> validating(reviewing)
  -> completed
  -> needs_fixup
  -> failed
```

### Aggregate result rules
- `completed`: every affected scenario restarted successfully and every review result is `ready` or `ready_with_notes`.
- `needs_fixup`: at least one scenario restart/health/review result is not acceptable, but the flow completed cleanly enough to provide actionable feedback.
- `failed`: Swarm Manager itself could not run the orchestration reliably enough to produce actionable per-scenario output.

### Manual operator rerun
The current manual "Trigger Review" action should become "Run Post-Run Checks" in the UI. The backend route may remain for compatibility, but its semantics should become "rerun the full finalization flow" rather than "trigger one singular review".

## Contract Design
Do not add a second parallel singular-review model. Replace the current review-specific fields with one unified finalization object.

### New persisted execution shape
Add a `finalization` object to `execution.Record` and remove the primary dependence on:
- `review_result`
- `review_job_id`
- `review_skip_reason`
- `review_started_at`

### Proposed finalization shape
```text
Finalization
  eligible: bool
  status: pending | running | completed | skipped | failed
  phase: scope_detection | restarting | health_check | reviewing | completed | skipped | failed
  scope_source: sandbox_diff | acceptance_allow | sandbox_diff_plus_acceptance_allow | none
  skip_reason: string
  started_at: string
  completed_at: string
  warnings: []FinalizationWarning
  affected_scenarios: []string
  aggregate_classification: ready | ready_with_notes | needs_work | not_assessable | skipped
  aggregate_summary: string
  scenarios: []ScenarioFinalization
```

```text
ScenarioFinalization
  scenario_name: string
  changed_paths: []string
  restart:
    status: pending | running | completed | failed
    attempts: int
    last_error: string
    started_at: string
    finished_at: string
  health:
    status: pending | running | completed | failed
    scenario_status: string
    health_status: string
    schema_valid: bool
    details: string
    checked_at: string
  review:
    status: pending | running | completed | skipped | failed
    job_id: string
    skip_reason: string
    result: ReviewResult
```

```text
FinalizationWarning
  code: string
  scenario_name: string
  message: string
  retryable: bool
  created_at: string
```

### Design notes
- Keep `ReviewResult` itself unchanged and embed it inside each scenario review result.
- Keep `validating` and `needs_fixup` as top-level execution states to minimize status churn across the rest of the app.
- Do not invent separate "review" and "restart" stores. Persist the whole finalization state on the execution record.

## Scope Resolution Algorithm
Resolve affected scenarios in this order:

1. Try agent-manager run diff.
2. If diff is unavailable or the run has no sandbox, fall back to `acceptance_allow`.
3. If diff exists and contains direct scenario paths plus shared paths, use direct scenario paths and union in `acceptance_allow` scenarios only when shared paths make the scope ambiguous.
4. If neither source yields any scenario, mark finalization `skipped` with a recorded reason and leave the execution `completed`.

### Path rules
- `scenarios/<name>/...` directly maps to scenario `<name>`.
- Paths outside `scenarios/` do not automatically map to a scenario in v1.
- Shared paths such as `packages/`, `resources/`, or repo-level files should generate a warning when they force fallback broadening.
- Add path helpers under `internal/pathutil` rather than re-implementing scenario extraction logic in execution service code.

## Restart And Health Algorithm
For each affected scenario, in deterministic order:

1. Call `Lifecycle.Restart`.
2. Poll `vrooli scenario status <name> --json`.
3. Treat the scenario as ready only when:
   - `scenario_data.status == "running"`, and
   - `scenario_data.health_status == "healthy"` or the equivalent structured health data proves the scenario is healthy, and
   - required health checks are present and schema-valid.
4. If the scenario reaches `running` but health schema is missing or invalid, record a warning and treat the scenario as failed for finalization purposes.
5. If the first restart attempt times out or never becomes healthy, record a warning, perform one additional full restart, and poll again.
6. If the second attempt still fails, stop finalization for that scenario and mark it as failed.

### Default timing
Use constants in v1 rather than new settings:
- Health poll interval: `5s`
- Health wait timeout per attempt: `2m`
- Max restart attempts: `2`

If operator tuning becomes necessary later, lift these constants into the existing settings system as a follow-up.

### Health probe seam
Add a dedicated scenario status/health probe seam in `internal/scenarios` rather than overloading the coarse inventory source. Reuse the CLI JSON contract that already includes schema validation and health summaries.

## Review Algorithm
After all affected scenarios have either:
- restarted and become healthy, or
- failed with a recorded restart/health outcome,

run scenario reviews sequentially for only the scenarios that passed restart/health.

### Review request behavior
For each scenario:
- `ScenarioName`: the concrete scenario being reviewed.
- `ExpectedPaths`: the scenario-specific changed paths when available, else the fallback allowlist.
- `SandboxID`: forward the agent-manager sandbox ID when available, even though current Git Control Tower behavior does not yet consume it.

### Why sequential in v1
- Git Control Tower already parallelizes checks within one scenario review.
- Test Genie can restart the scenario under test during isolated phases.
- Sequential scenario reviews reduce interference when scenarios call each other or share dependencies.

## Background Orchestration
Do not rely on `Get` and `List` polling as the primary engine for this flow.

### Required change
Introduce a server-side active execution processor that advances:
- agent-manager status transitions,
- post-run finalization phases,
- per-scenario review polling,
- timeout handling,
- auto-fixup spawning.

### Expected shape
- Keep `Get` and `List` refresh as a safe catch-up path.
- Move the primary transition loop into a background worker or scheduler tick owned by Swarm Manager.
- Rename/refactor `refreshRunningLocked` into a more general active-execution advancement function instead of letting the method keep accreting unrelated responsibilities.

## Follow-Up And Fixup Semantics
Fixup/follow-up generation must consume the aggregate scenario results, not a single collapsed review.

### Prompt behavior
- Replace `buildReviewFeedback(*ReviewResult)` with an aggregate formatter that serializes:
  - restart/health failures,
  - warnings,
  - every non-green review dimension across all scenarios.
- Preserve scenario names in the prompt so the agent can target the right app.
- Do not bury health-contract violations; they must appear in fixup context the same way review findings do.

### UX behavior
- The follow-up dialog should show scenario-grouped findings.
- Default fixup context should include every scenario that needs work.
- `Fix Review Issues` should be available whenever any scenario finalization result is failing or not assessable.

## UI And API Surface Changes

### API and proto
- Replace singular review fields in `packages/proto/schemas/swarm-manager/v1/domain/execution.proto`.
- Regenerate Go and TypeScript proto outputs.
- Update handler/proto mapping in Swarm Manager API and UI contract mappers.

### UI surfaces that must support `0..N`
- `ui/src/pages/BacklogDetailsPage.tsx`
- `ui/src/pages/ExecutionDetailsPage.tsx`
- `ui/src/components/execution/execution-card.tsx`
- `ui/src/components/execution/follow-up-dialog.tsx`
- `ui/src/services/proto-contracts.ts`
- `ui/src/types/domain.ts`

### Expected rendering pattern
- Show one aggregate badge/summary at the execution level.
- Show a scenario list beneath it with per-scenario restart + health + review status.
- Manual rerun action should operate on the whole finalization flow.
- The backlog details page should no longer imply that one badge equals one reviewed target when multiple scenarios were affected.

## Implementation Phases

## Phase 1 - New Seams And Helpers
### Goals
- Add the missing inputs needed for exact-scope finalization.

### Work items
- Add `GetRunDiff` to `api/internal/agentmanager/client.go`.
- Add a service-level accessor for run diff or changed paths in `api/internal/agentmanager/service.go`.
- Add scenario path helpers to `api/internal/pathutil`.
- Add a dedicated structured scenario status/health probe seam in `api/internal/scenarios`.
- Add tests for diff parsing, path grouping, and health probe parsing.

### Acceptance criteria
- Swarm Manager can fetch sandbox diff paths for a completed run.
- Swarm Manager can fetch structured health/status details for one scenario without parsing human CLI output.

## Phase 2 - Finalization Domain And Background Worker
### Goals
- Replace the singular review state machine with unified finalization orchestration.

### Work items
- Replace singular review fields in the execution model with `finalization`.
- Introduce background advancement for active executions.
- Replace `shouldTriggerReview` with `shouldRunFinalization`.
- Add sequential scope detection, restart, health, and review progression.
- Update timeout and auto-fixup logic to consume aggregate scenario results.

### Acceptance criteria
- A completed eligible execution moves through finalization without any `Get` or `List` calls.
- Multi-scenario executions create one scenario result per affected scenario.
- Restart/health warnings persist on the record.

## Phase 3 - API, Proto, And UI Conversion
### Goals
- Move all consumers to the new `0..N` finalization contract.

### Work items
- Update proto schema and generated outputs.
- Update API mapping code.
- Update domain types and contract mappers in the UI.
- Replace singular review badges/panels with aggregate + per-scenario rendering.
- Rename operator copy from "Trigger Review" to "Run Post-Run Checks" where appropriate.

### Acceptance criteria
- No Swarm Manager UI surface assumes `execution.reviewResult`.
- Backlog details and execution details can render multiple scenario outcomes.
- Manual rerun acts on the full finalization flow.

## Phase 4 - Fixup / Follow-Up Integration
### Goals
- Ensure downstream remediation uses the full multi-scenario result set.

### Work items
- Replace singular review feedback formatter with aggregate formatter.
- Update follow-up dialog defaults and summary UI.
- Update tests for fixup/follow-up prompt generation and dialog behavior.

### Acceptance criteria
- Fixup prompts include all failing scenarios and their relevant findings.
- Follow-up UI groups findings by scenario.

## Testing Plan

### Backend tests
- `agentmanager/client_test.go`: `GetRunDiff`
- `agentmanager/service_test.go`: diff forwarding and sandbox ID propagation
- `execution/service_test.go`:
  - diff-first scope detection
  - `acceptance_allow` fallback
  - shared-path warning behavior
  - sequential restart progression
  - health timeout + retry + warning capture
  - per-scenario sequential review progression
  - aggregate `completed` / `needs_fixup` / `failed` outcomes
  - manual rerun of finalization
  - auto-fixup after aggregate failure
- `execution/handler_followup_test.go`: aggregate feedback propagation
- new scenario probe tests under `api/internal/scenarios`

### UI tests
- `ui/src/services/proto-contracts.ts` mapping tests for finalization payloads
- `ui/src/components/execution/execution-card.test.tsx`
- `ui/src/components/execution/follow-up-dialog.test.tsx`
- page-level tests for backlog/execution detail rendering of multiple scenario results

### Integration validation
- Run Swarm Manager API tests.
- Run relevant UI test suites.
- Exercise at least one real multi-scenario execution end-to-end in a local environment:
  - one scenario under `scenarios/<name>/...`
  - one execution affecting two scenarios
  - one restart retry recovery case
  - one missing/invalid health contract case

## Risks And Mitigations
- Risk: shared package/resource changes do not map cleanly to affected scenarios.
  - Mitigation: start with direct scenario paths + explicit fallback broadening and warnings.
- Risk: finalization becomes stuck if only UI polling drives progress.
  - Mitigation: move active progression into a background worker.
- Risk: health checks are inconsistent across scenarios.
  - Mitigation: rely on the CLI's existing structured contract and treat invalid/missing schema as a first-class result.
- Risk: sequential reviews increase end-to-end latency.
  - Mitigation: Git Control Tower already parallelizes checks per scenario; prefer correctness over concurrency in v1.

## Non-Goals
- No transitive dependency graph engine for shared code in v1.
- No parallel multi-scenario review fan-out in v1.
- No deeper app-specific smoke-test framework beyond the standard scenario status/health contract in v1.
- No separate persistent store for review jobs outside the execution record.

## Definition Of Done
- Exact changed scenarios are derived from sandbox diff when available, with `acceptance_allow` fallback.
- Eligible executions restart affected scenarios before review.
- Restart/health work is sequential, retried once on transient failure, and warnings are persisted.
- Every affected scenario gets its own review result entry.
- Aggregate execution outcome is derived from the full scenario set.
- Fixup/follow-up flows consume aggregate multi-scenario findings.
- UI/API/proto contracts support `0..N` scenario finalization results.
- Post-run orchestration progresses server-side without depending on operator page refreshes.
