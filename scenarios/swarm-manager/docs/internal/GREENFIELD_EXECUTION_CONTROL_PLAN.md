# Swarm Manager Greenfield Execution Control Plan

## Document Metadata
- Owner scenario: `swarm-manager`
- Date: 2026-02-13
- Status: Draft implementation plan
- Scope: Greenfield redesign and implementation pivot

## Greenfield Mandate (Non-Negotiable)
This plan is **greenfield**. Implementation MUST:
- Not include migration code
- Not include backward compatibility shims
- Not include deprecation layers
- Not include dual old/new execution paths
- Not include legacy recommendation-engine support

If existing code conflicts with this direction, remove or replace it directly.

## Required Prerequisite (Skills)
Before implementing any phase in this plan, engineers/agents MUST run:

```bash
prompt-manager skill read api-steer cli-steer interoperability-steer storage-steer seam-discovery-and-enforcement react-coherence
```

This prerequisite was executed on 2026-02-13 while preparing this plan.

## Why We Are Changing Direction
Swarm Manager currently has useful building blocks, but the control model is split:
- Backlog is the right place for explicit work intent and context.
- A separate recommendations subsystem duplicates recommendation responsibilities that should live with Prompt Manager teams.
- Run/execution visibility is not yet the primary control surface.

Target operating model:
- Prompt Manager teams produce findings/recommendations.
- Teams write structured `idea`/`fix`/`execute` backlog items into Swarm Manager instead of editing scenario code directly.
- Swarm Manager is the single execution control plane (manual, delayed auto-start, or YOLO).
- Operators use Swarm Manager UI/CLI to govern change execution.

## Current Problem Statement
### 1) Recommendation overlap and control ambiguity
Current recommendations API/UI can generate and start recommendation runs, but this competes with Prompt Manager team workflows rather than governing them through backlog.

### 2) Archive pathing inconsistency blocks trustworthy archival workflow
Archive-to-backlog logic currently derives a fixed relative path and ignores runtime scenario root, causing real test failures and inconsistent data placement.

### 3) Incomplete run lifecycle as first-class domain
Run IDs/task IDs exist in specific flows, but Swarm Manager does not yet provide a dedicated execution-control domain with complete pending/running/completed/failed visibility and policy-driven scheduling.

### 4) Team outputs are not yet standardized into backlog contracts
Prompt Manager team outputs are not yet enforced through a single structured tool contract that creates backlog items with required provenance and execution metadata.

## Strategic Direction
1. Remove Swarm Manager recommendation feature entirely.
2. Move recommendation generation/research output responsibility to Prompt Manager teams (via using the swarm-manager tool skill).
3. Standardize team output via a Prompt Manager "Swarm Manager tool" skill that leverages the swarm-manager cli (similar to how the visited-tracker tool skill works).
4. Make backlog item creation and enqueue/start paths frictionless.
5. Replace Recommendations UI with an Execution UI centered on agent-manager runs linked to backlog items.

## Target Product Model
### Domain model
- Backlog Item domain remains (`idea`, `research`, `fix`, `execute`).
- Add Execution Run domain as first-class Swarm Manager concern.
- Remove Recommendation domain from Swarm Manager surface and runtime.

### Control modes
- `manual`: operator must start explicitly.
- `scheduled`: auto-start after configured delay window.
- `yolo`: start immediately.

### Source of truth
- Backlog state: filesystem-backed under Swarm Manager scenario root.
- Execution run records: filesystem-backed store in `.vrooli` (greenfield schema).
- Agent-manager: execution engine; Swarm Manager stores authoritative run metadata view for governance.

## Phased Implementation Plan

## Phase 0 - Foundation and Hard Constraints
### Goals
- Lock scope to greenfield replacement.
- Ensure all contributors load required skills.

### Deliverables
- This plan file committed and referenced in execution tickets.
- Implementation checklist template includes skill command as required pre-step.
- Explicit coding policy note in Swarm Manager docs: no legacy/compat path.

### Acceptance criteria
- Every implementation PR references this plan and confirms skill command execution.
- No code added for recommendation backward compatibility.

## Phase 1 - Recommendation Feature Removal (Hard Cut)
### Goals
- Remove recommendation subsystem from Swarm Manager API/CLI/UI/state/proto/storage.

### Work items
- API:
  - Remove recommendations routes and handlers.
  - Remove recommendation storage and generation engine.
- CLI:
  - Remove `recommendations list|refresh|create|update` command group.
- UI:
  - Remove Recommendations page and stores/services/selectors tied to it.
  - Update navigation tabs from 4-tab model to include Execution page in place of Recommendations.
- Proto/contracts:
  - Remove Swarm Manager recommendation API/domain contracts and generated artifacts.
- Settings:
  - Remove recommendation engine configuration fields from settings model.

### Acceptance criteria
- No recommendation endpoints/commands/UI remain.
- Build and tests pass for remaining surfaces.
- Navigation and docs no longer reference recommendation engine behavior.

## Phase 2 - Archive and Backlog Integrity (Must-Fix)
### Goals
- Make archive-to-backlog deterministic and root-correct.
- Ensure archived scenarios become reliable backlog inputs.

### Work items
- Fix archive root derivation to use injected/real scenario root (not hardcoded relative path).
- Enrich archived `spec.json` with provenance fields:
  - `sourceScenarioName`
  - `sourceScenarioPath`
  - `archivedAt`
  - `archivedBy`
  - `archiveReason`
  - `preservedFiles`
  - `preservePresetOrCustom`
- Keep strict path safety and ignored-directory behavior for preserved files.

### Acceptance criteria
- Archive unit tests pass, including real path assertions.
- Tests use proper testing seams and mocks, so that accidentally affecting actual files/folders is impossible
- Archive creates backlog item in canonical `scenarios/swarm-manager/ideas/*` location.
- Archived items are immediately usable by execution pipeline.

## Phase 3 - Prompt Manager Tool Skill and Team Contracts
### Goals
- Standardize all team findings into Swarm Manager backlog items.

### Work items
- In `prompt-manager`, create a new tool skill:
  - Purpose: teach exact Swarm Manager usage patterns.
  - Required output contracts per team:
    - Scenario Feature Team -> `idea` or `execute` (explicit rule set)
    - Scenario QA Team -> `fix` or `execute`
    - Scenario Refactor Team -> `execute` or `fix`
- Skill must require structured fields:
  - `targetScenario`
  - `problemOrOpportunity`
  - `proposedAction`
  - `evidence`
  - `riskLevel`
  - `executionModeHint`
  - `createdByTeam`
  - `sourceRunId`
- Run `prompt-manager skill read skill-authoring-tools skill-principles cli-steer visited-tracker-tools` to learn how to properly implement this skill, and to see an example of another tool skill. It's important that the swarm-manager CLI is a thin wrapper over its API, and by default outputs maximally useful "human" output, per cli-steer instructions
- Agents in the corresponding Teams (Scenario Feature Team, Scenario QA Team, Scenario Refactor Team) have their files updated to prohibit direct scenario code edits.

### Acceptance criteria
- All Agents that are member of the four Prompt Manager teams reference the shared tool skill, if they are meant to create the recommendations (not sure the exact setup, but I suspect the leader team member is the only one that's meant to use the swarm-manager skill and author recommendations).
- Teams create backlog items through Swarm Manager contracts only.
- Backlog items created by the team are execution-ready without manual rewriting/revision.

## Phase 4 - Execution Control Domain (New Core)
### Goals
- Introduce first-class execution run tracking linked to backlog items.

### Work items
- Create execution run model and storage:
  - `execution_id`, `backlog_kind`, `backlog_name`, `task_id`, `run_id`, `status`, `mode`, `scheduled_at`, `started_at`, `finished_at`, `failure_reason`, `started_by`.
- Add API module for execution operations:
  - list/filter runs
  - get run details
  - start run
  - cancel scheduled run (if not started)
  - retry failed run
- Add scheduler worker for `scheduled` mode:
  - deterministic delay handling
  - idempotent start semantics
  - safe recovery on restart
- Sync backlog item state with execution state transitions.

### Acceptance criteria
- Operators can trace every run from backlog item to agent-manager IDs and terminal status.
- Scheduled runs auto-start after delay reliably.
- YOLO starts immediately and is recorded as mode-driven action.

## Phase 5 - Backlog Enqueue/Start UX and CLI Ergonomics
### Goals
- Make enqueueing and starting work effortless.

### Work items
- UI:
  - Add one-click `Queue` / `Start Now` / `Schedule` actions on backlog list and details.
  - Add explicit mode selector and delay input where relevant.
  - Provide clear validation errors and state feedback.
- CLI:
  - Add explicit execution commands:
    - `backlog queue <kind> <name> --mode manual|scheduled|yolo [--delay ...]`
    - `execution list|get|start|retry|cancel`
- API/CLI parity enforced for all execution actions.

### Acceptance criteria
- User can enqueue/start in <= 2 interactions in UI.
- CLI supports complete execution governance without UI dependency.

## Phase 6 - Recommendations Page Replacement with Execution Page
### Goals
- Replace old recommendations surface with execution control and observability.

### Work items
- Build Execution page with sections:
  - Pending/Scheduled
  - Running
  - Completed
  - Failed
- Show links to:
  - backlog item
  - task/run IDs
  - start timestamps/durations
  - failure reasons and retry actions
- Add filtering by scenario, team source, mode, status, time range.

### Acceptance criteria
- Execution page becomes primary operations dashboard for scenario changes.
- No recommendation-page dependencies remain.

## Phase 7 - Storage and Seam Hardening
### Goals
- Ensure storage correctness and seam enforceability under sustained autonomous use.

### Work items
- Normalize all Swarm Manager stores to canonical root (no hidden cwd coupling).
- Ensure atomic writes and corruption-safe recovery for execution records.
- Enforce seam boundaries:
  - handlers orchestrate
  - execution service handles lifecycle logic
  - storage adapters isolated and testable
- Add clear contract tests for API/CLI/UI interoperability.

### Acceptance criteria
- No domain logic hidden in transport layers.
- Execution store survives restart and supports deterministic recovery.
- Contract tests validate end-to-end surface consistency.

## Phase 8 - Documentation, Policy, and Operational Readiness
### Goals
- Ensure human + autonomous teams can operate professionally and consistently.

### Work items
- Update architecture docs to new model:
  - remove recommendation engine references
  - document Prompt Manager team -> backlog -> execution pipeline
- Add runbooks:
  - manual mode operations
  - scheduled mode operations
  - YOLO operations with safeguards
  - failure handling and retry policy
- Update requirements modules to reflect implemented state.

### Acceptance criteria
- Docs match runtime behavior with no stale recommendation references.
- Onboarding instructions let a new operator execute pipeline without tribal knowledge.

## Cross-Phase Engineering Standards
### API and Proto
- Proto-first contracts for new execution APIs.
- Domain-organized API modules only.
- No compatibility/legacy fields retained for removed recommendation flows.

### CLI
- Thin wrappers over API only.
- Full command parity with execution APIs.

### Storage
- Filesystem persistence remains explicit and canonical.
- No hardcoded/implicit relative roots for mutable data.

### React/UI
- Coherent state boundaries (server state via query layer, app state via stores where justified).
- Execution page follows existing UI architecture patterns while replacing recommendations surface completely.

## Testing Strategy by Phase
- Unit tests: handlers/services/storage adapters.
- Contract tests: proto <-> API <-> UI service mapping.
- Integration tests: backlog queue/start/schedule/retry lifecycle.
- UI tests: execution page filters, state transitions, and action flows.
- Scenario-level tests: validate full Prompt Manager team output -> backlog -> execution path.

## Explicit Non-Goals
- No migration of legacy recommendation records.
- No support for old recommendation endpoints/commands/UI.
- No backward compatibility behavior toggles.
- No dual-mode operation retaining old and new control planes.

## Definition of Done
Swarm Manager is complete for this pivot when:
1. Prompt Manager teams write structured backlog items instead of direct code changes.
2. Swarm Manager has no recommendation subsystem.
3. Execution mode policy (`manual|scheduled|yolo`) governs all automated backlog work.
4. Execution page is the operational dashboard for pending/running/completed/failed runs.
5. Archive-to-backlog is path-correct, deterministic, and provenance-rich.
6. API/CLI/UI docs and tests fully reflect the greenfield model.
