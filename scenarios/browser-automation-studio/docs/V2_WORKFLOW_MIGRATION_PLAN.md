# Browser Automation Studio: V2 Workflow Migration Plan

## Why We’re Doing This (Context + Importance)

BAS currently runs “v2” workflows (proto-native JSON) through a compatibility funnel that still preserves multiple “v1-shaped” seams:

- **Workflow definition shape**: the compiler accepts both v1 (`node.type` + `node.data`) and v2 (`node.action.type` + nested payloads) and normalizes v2 into legacy step types/param names.
- **Executor/params shape**: compilation produces an instruction/param vocabulary that remains largely v1-ish (camelCase param names, “stepType” strings), which keeps v1 translation logic alive.
- **Subflow shape**: inline `workflowDefinition` maps keep a legacy “map-shaped workflow” path alive and currently have explicit MIGRATION gaps.
- **Streaming shape**: websocket messages are proto-shaped but still ship legacy compatibility payloads.
- **Resolution layers**: multiple token/selector compatibility behaviors exist to support different entrypoints (UI, CLI, test runner).

This makes the system harder to evolve:

- Every new feature requires changes in multiple formats and shims.
- Bugs appear as “something got in the way” because there isn’t a single canonical data model.
- Deleting legacy code is risky because it’s unclear which consumer still relies on it.

**The goal** is to make the workflow system **v2-only** end-to-end so that:

- There is one canonical workflow model (proto-native JSON).
- Execution is deterministic and easier to validate/secure (schema validation is real, not “best effort + translation”).
- The executor, compiler, UI, and test runner can stop carrying compatibility glue.
- Removing v1 reduces surface area for subtle bugs and breaks “two systems drift” over time.

## Ideal State (North Star)

This migration plan targets the ideal BAS state:

- **Core data types**
  - **Workflows**: directed graph of nodes/edges that runs Playwright; supports cycles (loops), assertions, and reusable subflows.
  - **Recordings**: a flat list of actions using the same `ActionDefinition` type; conceptually “a user did X” vs “an agent repeats X”.
  - **Projects**: an index + storage container for workflows and artifacts (storage mechanism flexible).
  - **Executions**: persisted results of running a workflow; should be replayable and ideally share timeline entry structure with recordings plus output/styling.
- **Storage principles**
  - Files are the source of truth; database stores only indexes and queryable metadata.
  - Proto is the shared type system (UI/API/playwright-driver) for everything except minimal DB index types.
  - Exactly one hydration/persistence layer assembles proto-compliant domain objects from DB indexes + files.
- **Architecture principles**
  - Clear boundaries and seams; compatibility logic is isolated (anti-corruption layer) and deletable.
  - Tests can validate contracts without spinning up the world (compiler/executor/driver seams remain strong).

## Current Reality (What Exists Today)

Key code anchors:

- Workflow execution entrypoint: `scenarios/browser-automation-studio/api/services/workflow/executions.go`
- Adhoc execution: `scenarios/browser-automation-studio/api/services/workflow/adhoc_execution.go`
- Compiler supports both v1 and v2 nodes: `scenarios/browser-automation-studio/api/automation/compiler/compiler.go`
- Engine is Playwright-only: `scenarios/browser-automation-studio/api/automation/engine/README.md`
- Subflow inline-def MIGRATION gap: `scenarios/browser-automation-studio/api/automation/executor/flow_executor.go`
- Websocket sink emits proto but attaches legacy payload: `scenarios/browser-automation-studio/api/automation/events/ws_sink.go`
- UI normalizes workflow response by converting to v1 flow to get nodes/edges reliably: `scenarios/browser-automation-studio/ui/src/stores/workflowStore.ts`

## Canonical Domain Model + File Formats (Make It Explicit)

We need to remove ambiguity around “files are the source of truth” by explicitly documenting the canonical shapes **per boundary**. “V2-only” is not just “store v2 JSON”; it means every boundary has a single, enforceable contract.

### Filesystem Source Of Truth (Canonical On-Disk Formats)

1. **Project workflow file (persisted workflows; source of truth)**
   - Stored on disk as **protojson `browser_automation_studio.v1.WorkflowSummary`**.
   - `WorkflowSummary.flow_definition` is the canonical workflow graph (`WorkflowDefinitionV2`).
   - Written by `WriteWorkflowSummaryFile` and read by `ReadWorkflowSummaryFile`.

2. **Fixture/workflow-definition file (reusable snippets; referenced by `workflow_path`)**
   - Stored under `scenarios/browser-automation-studio/bas/actions/*.json` as **protojson `browser_automation_studio.v1.WorkflowDefinitionV2`**.
   - These are not persisted workflows; they are reusable definitions referenced by subflow calls or test tooling.

3. **Execution/recording timeline artifact (replay source of truth)**
   - Stored on disk as **protojson `browser_automation_studio.v1.ExecutionTimeline`** (unified `TimelineEntry` list).
   - This is the replay substrate for *both* recordings and executions (recording = user-driven timeline; execution = agent-driven timeline).

**Policy (target):**

- Persisted workflows always store `WorkflowSummary` protojson (with `flow_definition` v2).
- Reusable playbook snippets always store `WorkflowDefinitionV2` protojson (referenced by `workflow_path`).
- Executions/recordings persist timeline artifacts as `ExecutionTimeline` protojson (unified `TimelineEntry`).
- Subflows reference workflows by `workflow_id` or `workflow_path` only (no inline embedded `workflowDefinition` maps).

### Canonical Shape Matrix (Hard Invariants Per Boundary)

This is the “screaming architecture” contract. Each row is a boundary; if a row is ambiguous, migration will stall again.

| Boundary | Canonical shape (target) | Notes / anti-corruption allowed |
|---|---|---|
| **DB** | Index-only rows (`ProjectIndex`, `WorkflowIndex`, `ExecutionIndex`) | No workflow definitions, no timelines. DB is query/index only. |
| **Filesystem (workflows)** | `WorkflowSummary` protojson (`flow_definition: WorkflowDefinitionV2`) | Protojson MUST be written with proto field names (snake_case) for stability. |
| **Filesystem (fixtures)** | `WorkflowDefinitionV2` protojson | Referenced via `workflow_path` from within a project. |
| **Filesystem (timeline)** | `ExecutionTimeline` protojson (unified `TimelineEntry`) | Powers replay/export; same structure for recording vs execution. |
| **API (create/update)** | `WorkflowDefinitionV2` protojson on write | Legacy/v1 payloads are accepted *only* inside one compat module and immediately converted. |
| **Executor/Compiler internal** | Typed protos (`WorkflowDefinitionV2`, `ActionDefinition`, etc.) | `map[string]any` is banned except inside the compat boundary. |
| **Engine ↔ Driver** | `execution.driver.proto` (`CompiledInstruction.action`, `PlanStep.action`) | `type/params` are deprecated and must not be execution-critical. |
| **WebSocket streaming** | `timeline_entry` (protojson `TimelineEntry` / `TimelineStreamMessage`) | `legacy_payload` exists only during a short compatibility window, then is deleted. |

### Workflow Lifecycle Pipeline (Staged, Named Artifacts)

To prevent “resolution/validation is scattered”, BAS adopts an explicit pipeline. Each stage is testable and has a single owner.

1. **Hydrate**: DB index + filesystem → typed domain protos (`WorkflowSummary`, `WorkflowDefinitionV2`, `Project`)
2. **Validate (schema)**: strict proto decode (`DiscardUnknown: false`) + protovalidate rules
3. **Resolve**: tokens/selectors/fixtures → produce a *resolved* `WorkflowDefinitionV2` (no `@fixture/`, no selector indirections, no inline defs)
4. **Validate (resolved)**: enforce “execution-ready” constraints (no unresolved placeholders)
5. **Compile**: `WorkflowDefinitionV2` → `ExecutionPlan` where `action` fields are sufficient
6. **Execute**: driver dispatch uses `ActionDefinition` only
7. **Record**: driver `StepOutcome` → persisted `TimelineEntry` (timeline artifacts) + DB status index
8. **Stream**: push `timeline_entry` events; UI consumes it directly

## Definition: “V2-Only” (Target End State)

We are “v2-only” when all are true:

1. **Persisted workflow definitions are proto-native**
   - Stored workflow definitions and versions use v2 nodes/edges only (`nodes[].action.*`).
   - API rejects or auto-upgrades v1 workflow definitions on write (no “store v1”).

2. **Execution compilation uses the existing proto driver contract**
   - Compiled plans and instructions primarily use the proto fields:
     - `browser_automation_studio.v1.CompiledInstruction.action`
     - `browser_automation_studio.v1.PlanStep.action`
   - Deprecated `type`/`params` are transitional only and are not required for execution.

3. **Subflows are v2-native references**
   - Subflows are referenced by `workflow_id` and/or `workflow_path` (proto `ACTION_TYPE_SUBFLOW`), never by inline definitions.
   - Inline `workflowDefinition` maps are rejected at validation time.

4. **Streaming contract is singular**
   - Websocket event payloads are contract/proto-first (prefer the `timeline_entry` / `TimelineStreamMessage` shape).
   - Legacy websocket `legacy_payload` is removed once no consumers use it.

5. **There is one canonical token resolution strategy**
   - Selector/token/fixture resolution happens in exactly one place (the “Resolve” stage), not “sometimes here, sometimes there”.

6. **Control-flow semantics are proto-modeled (no stringly-typed execution control)**
   - Any non-browser “control” semantics required at runtime (loops/cycles, branching, variable writes, subflow dispatch) are representable in proto:
     - either as `ActionDefinition` variants, OR
     - as a separate, typed “workflow control” domain that is still shared across UI/API/driver.
   - Edge conditions/branching are typed and validated (not just UI labels).
   - Variable semantics are explicit (namespaces, read/write rules), so execution is deterministic.

7. **One hydration/persistence layer**
   - The filesystem protojson is the source of truth; the database remains an index only.
   - Workflow/project/execution domain objects are assembled via a single hydration layer (no parallel “sometimes map, sometimes proto” codepaths).

## Strategy: Ratchet Migration (Stop Bleeding → Migrate → Delete)

We do this in phases that monotonically reduce v1 reliance. Each phase has:

- Explicit checklist items
- Acceptance criteria
- Test gate (`vrooli scenario test browser-automation-studio`)
- “Hard gates” (behavioral + grep) that prevent regression

## Phase Ordering (Updated)

This plan previously placed “migrate stored workflows” before we had locked down control-flow semantics and before subflows were references-only. In practice, those two items are the highest-leverage sources of drift and repeated rewrites.

**Revised ordering principles:**

1. **Stop creating new legacy shapes** (UI/API “v2 on write”) so the backlog stops growing.
2. **Decide semantics early** (loops/branching/variables) so we migrate *into the final model*, not into an intermediate.
3. **Eliminate inline subflows early** because they force map-shaped embedding and block compiler/executor deletions.
4. **Make “Resolve” a named pipeline** with a single owner, then remove Python from critical paths.
5. Only then **migrate stored assets** and **delete compat** safely.

---

## Phase 0 — Baseline + Inventory (Make the Hidden Dependencies Visible)

### Deliverables

- A reproducible report of where v1 is still used:
  - Workflows stored in v1 shape (if any)
  - Subflows using inline `workflowDefinition`
  - Clients relying on websocket `legacy_payload`
  - Runtime dependencies on legacy step type strings (e.g. `"loop"`, `"workflowCall"`, `"setVariable"`)
  - Any remaining test runner preprocess steps that force a particular shape (fixtures/selectors/token resolution)
  - Any remaining map-shaped workflow compilation/execution paths on hot paths (e.g. `map[string]any` workflowDefinition ingestion)
- A written, explicit statement of canonical storage formats:
  - Persisted workflows: protojson `WorkflowSummary` on disk (contains `flow_definition` v2)
  - Fixtures: protojson `WorkflowDefinitionV2` referenced by `workflow_path`
- An inventory of:
  - Persisted workflow files + `.bas/versions` snapshots
  - Fixture workflow files under `bas/actions/*.json`
  - Subflow targets used in the repository (`workflow_id` vs `workflow_path` vs inline maps)
  - Timeline artifacts on disk (`timeline.proto.json` / `ExecutionTimeline` protojson), and which endpoints write/read them

### Checklist

- [ ] Decide and record the v2-only definition (use the “Definition” section above; edit if needed).
- [ ] Confirm and document canonical on-disk formats (WorkflowSummary vs WorkflowDefinitionV2) in this plan (see section above).
- [ ] Confirm the “Canonical Shape Matrix” rows are correct (owners + anti-corruption allowances).
- [ ] Inventory workflows and fixtures that exist today:
  - [ ] Persisted workflows (project store files + `.bas/versions` snapshots).
  - [ ] Fixture workflows (`bas/actions/*.json`) that are referenced by `workflow_path`.
- [ ] Decide the policy for fixture resolution:
  - [ ] Allowed base roots (project root only).
  - [ ] Path traversal rules (reject absolute paths, `.` and `..` segments).
- [ ] Confirm subflow target contract is already proto-modeled (`ACTION_TYPE_SUBFLOW` with `workflow_id|workflow_path`).
- [ ] Add a “v1 usage report” command sequence (no new deps) to run locally and in CI (use the commands below).
- [ ] Capture current counts in this file under “Progress Snapshots”.

### Acceptance Criteria

- We can answer, with evidence: “Which workflows/paths still require v1 today?”

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio` (baseline run; record pass rate and top failure categories).

### Automation: Progress Snapshot Commands (Copy/Paste)

Use these commands to create objective “ratchet” checkpoints:

- [ ] `rg -n "\"type\"\\s*:" scenarios/browser-automation-studio/bas scenarios/browser-automation-studio/api scenarios/browser-automation-studio/ui` (heuristic for legacy `"type"` usage)
- [ ] `rg -n "\"data\"\\s*:" scenarios/browser-automation-studio/api scenarios/browser-automation-studio/ui` (heuristic for v1 `node.data` usage)
- [ ] `rg -n "workflowDefinition\"\\s*:" scenarios/browser-automation-studio` (inline subflows)
- [ ] `rg -n "legacy_payload" scenarios/browser-automation-studio` (websocket compatibility)
- [ ] `rg -n "snakeToCamel\\(|v2ActionTypeToStepType\\(" scenarios/browser-automation-studio/api` (v2→v1 normalization helpers)
- [ ] `rg -n "workflowcall|setvariable|strings\\.EqualFold\\(.*step\\.Type" scenarios/browser-automation-studio/api/automation` (stringly control-flow routing)
- [ ] `rg -n "DiscardUnknown:\\s*true" scenarios/browser-automation-studio/api` (protojson leniency; should end up compat-only)

---

## Phase 0.5 — Collapse Boundaries (One Compat Module + One Resolution Module)

### Goal

Stop the “compat logic scattered everywhere” failure mode by forcing all legacy acceptance into one deletable place, and forcing all resolution into one staged pipeline.

### Checklist

- [ ] **Compat boundary**: define a single “legacy/compat” module that owns:
  - [ ] v1 workflow shapes (`node.type` + `node.data`) → `WorkflowDefinitionV2`
  - [ ] any remaining “map-shaped protojson wrappers” needed by tooling (e.g., JsonValue wrappers)
  - [ ] any temporary request-shape normalization needed for old callers
- [ ] **Ban compat elsewhere**:
  - [ ] compiler/executor must not contain “proto → map → legacy step strings” normalization on execution-critical paths
  - [ ] handlers must not do bespoke proto-shape patching beyond compat boundary
- [ ] **Resolution boundary**: define a single “Resolve” stage owner (module/service) that:
  - [ ] resolves selectors, tokens, and fixtures/subflow references
  - [ ] outputs an execution-ready `WorkflowDefinitionV2` (no unresolved placeholders, no inline defs)
- [ ] **Subflow resolver seam already exists**: document and standardize it:
  - [ ] Use the existing `executor.WorkflowResolver` implementation: `scenarios/browser-automation-studio/api/services/workflow/workflow_resolver.go`
  - [ ] Explicitly forbid “Python inlines nested workflowDefinition maps” as a long-term dependency

### Acceptance Criteria

- There is exactly one directory/package to delete when dropping legacy shapes.
- “Resolve” is a named step with a single owner and test seam.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 1 — Stop Creating New v1 (Stop the Bleeding)

### Goal

From this point on, any newly saved workflow is v2-native and stays v2-native. This prevents the backlog from growing while we migrate existing assets.

### Checklist (UI/API)

- [ ] UI: always *save* v2 proto-native definitions; if a v1 workflow is loaded, auto-upgrade it in memory and persist v2 on next save.
  - Context: UI currently builds `v1Definition` during normalization: `scenarios/browser-automation-studio/ui/src/stores/workflowStore.ts`.
- [ ] API: enforce “v2 on write”
  - [ ] On workflow create/update/version restore endpoints: reject v1-shaped definitions OR accept and immediately convert to v2 before persisting.
  - [ ] Add explicit error messages so failures are actionable (“workflow nodes must use action.type; v1 node.type is no longer accepted”).

### Acceptance Criteria

- Any workflow saved through UI/API persists as v2 proto-native JSON.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio` (should not regress; record deltas).

---

## Phase 2 — Control Flow + Branching + Variables (Decide Early; Make It Proto-Real)

### Goal

Lock down the semantics that determine the workflow data model. This must happen **before** mass migration of stored workflows/versions; otherwise we risk migrating into a model that immediately changes again.

### Checklist

- [ ] Decide the control model (document here with examples):
  - [ ] **Loops/cycles** (pick one and commit):
    - [ ] Allow graph cycles in `WorkflowDefinitionV2` and define deterministic safeguards (execution budget, cycle guards), **OR**
    - [ ] Keep an explicit loop node, but make it fully proto-real and avoid stringly dispatch.
  - [ ] **Branching**:
    - [ ] replace “edge label drives routing” with typed edge conditions (even a minimal model)
    - [ ] define deterministic routing rules (default edge, precedence, error on ambiguity)
  - [ ] **Variables**:
    - [ ] standardize namespace semantics (`@store/`, `@params/`, `@env/`)
    - [ ] define subflow param passing rules and store merge-back rules (store merges, params/env do not)
- [ ] Update proto schemas + regenerate code once the model is selected.
- [ ] Update validators to enforce the chosen semantics (strict, actionable errors).

### Acceptance Criteria

- The “north star” loop model is representable and validated in proto.
- Branching decisions are deterministic and validated (not “label conventions”).
- Variable semantics are explicit and consistent across workflow + subflows.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 3 — Subflows: References Only (Hard Precondition)

### Why This Matters

Inline `workflowDefinition` keeps a map-shaped legacy workflow embedded in params. The executor currently calls out a MIGRATION gap:
`scenarios/browser-automation-studio/api/automation/executor/flow_executor.go` (inline def conversion skipped).

If we don’t address this, we cannot delete legacy “map workflow” behavior safely.

### Decision

Pick the references-only approach and standardize it across *all* producers/consumers:

- Runtime subflows are **references only** (`workflow_id` / `workflow_path`) via `ACTION_TYPE_SUBFLOW`.
- Resolution is performed by a resolver seam, not by embedding workflow maps into JSON trees.
- The existing production resolver seam is already in place: `scenarios/browser-automation-studio/api/services/workflow/workflow_resolver.go`.

### Checklist

- [ ] Enforce the proto contract:
  - [ ] `ActionDefinition.subflow.target` uses `workflow_id` or `workflow_path` (plus optional `workflow_version`)
  - [ ] `ActionDefinition.subflow.args` uses `common.v1.JsonValue` (no ad-hoc maps)
- [ ] Delete inline-definition dependencies end-to-end:
  - [ ] remove/disable any tooling that inlines nested `workflowDefinition` maps (replace with references)
  - [ ] validator rejects any `workflowDefinition` keys anywhere in workflow JSON (not just at the top level)
  - [ ] executor rejects inline subflow definitions (explicit error) until the key no longer exists in inputs
- [ ] Confirm fixture resolution policy (security + determinism):
  - [ ] allow only project-root-relative paths
  - [ ] reject absolute paths and traversal segments
- [ ] Add a hard grep gate for regression:
  - [ ] `rg -n "workflowDefinition\"\\s*:" scenarios/browser-automation-studio` must be 0 (outside docs/tests explicitly allowed)

### Acceptance Criteria

- No workflows/executions rely on inline `workflowDefinition`.
- Subflows work for both `workflow_id` and `workflow_path` without any inline expansion step.
- Compiler/executor deletion work is now unblocked (no map-embedded workflows).

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 4 — Resolution Pipeline (One “Resolve” Owner; Remove Python From Critical Paths)

### Goal

Replace the current “sometimes Python, sometimes Go” pre-processing with a **single, named Resolve pipeline** that:

- accepts a stored `WorkflowDefinitionV2`,
- performs deterministic resolution (selectors, tokens, fixtures/subflows),
- outputs an execution-ready `WorkflowDefinitionV2` with **zero unresolved placeholders**.

### Resolve Pipeline (Make Explicit)

Define these steps and make them individually testable (unit tests; no browser required):

1. **Path resolution**: resolve `workflow_path` references with traversal protections (project-root relative only).
2. **Selector resolution**: resolve `@selector/...` manifest references (same behavior for UI/CLI/tests).
3. **Token/template resolution**: resolve `${var}` / token forms with explicit namespaces and rules.
4. **Defaults/materialization**: apply defaults so downstream compile/execution is deterministic (no “missing means X sometimes”).
5. **Validation**: run strict proto + semantic validation on the resolved definition.

### Checklist

- [ ] Choose a single owning module/service for Resolve and document the API:
  - [ ] Input: `WorkflowSummary` + `WorkflowDefinitionV2`
  - [ ] Output: fully resolved `WorkflowDefinitionV2`
  - [ ] Errors: actionable pointers (JSON pointers / proto field paths)
- [ ] Ensure tests and runners use the Resolve API, not Python pre-processing.
- [ ] Remove/retire the Python path from the critical execution path (it may remain as an optional dev helper temporarily, but must not be required for correctness).
- [ ] Add hard gates:
  - [ ] “Python resolver invoked” count must be 0 in scenario tests (once cutover is complete).
  - [ ] No production code relies on `workflowDefinition` inline maps (already Phase 3).

### Checklist

- [ ] `vrooli scenario test browser-automation-studio`

### Acceptance Criteria

- “Resolve” is the single owner for selector/token/subflow resolution across UI/API/tests.
- Removing Python no longer changes runtime correctness (only tooling ergonomics at most).

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 5 — Migrate Existing Stored Workflows + Versions (After Semantics + Subflows)

### Goal

Eliminate legacy/v1 workflow definitions already stored (including version history) *into the now-final semantics and subflow model*.

### Checklist

- [ ] Identify storage locations of workflow definitions/versions used by BAS (filesystem + protojson snapshots, including `.bas/versions`).
- [ ] Prefer leveraging existing “read + normalize + write back” behavior where possible (protojson is the source of truth on disk).
  - [ ] Batch-walk project workflow files, load via the workflow proto reader, and rewrite any file flagged as needing normalization.
  - [ ] Ensure version snapshots under `.bas/versions/<workflowID>/*.json` are also migrated/normalized (or regenerated).
- [ ] Add a one-time migration command (CLI or admin endpoint) to normalize a project/workflow tree (no new deps).
- [ ] Decide policy: preserve old v1 versions as historical artifacts vs upgrade all versions.
  - Recommended: upgrade all versions to v2 so compiler/executor can drop v1 entirely later.

### Checklist

- [ ] `vrooli scenario test browser-automation-studio`

### Acceptance Criteria

- “v1 usage report” shows zero persisted v1 workflows/versions.
- No migrated workflow reintroduces inline subflow defs (Phase 3) or unresolved tokens (Phase 4).

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 6 — Compiler Core Replacement (Proto-Native Compiler; No Proto→JSON→V1 Funnel)

### Goal

Make the compiler operate on typed protos end-to-end. Eliminate the `rawNode`/map-shaped parsing and the conversion tax (`v2ActionTypeToStepType`, `snakeToCamel`) on execution-critical paths.

### Checklist

- [ ] Compiler input is typed `WorkflowDefinitionV2` (proto), not `map[string]any` / `rawNode`.
- [ ] Compiler output populates:
  - [ ] `contracts.CompiledInstruction.action`
  - [ ] `contracts.PlanStep.action`
  - [ ] `type/params` may remain temporarily for diagnostics only (non-required)
- [ ] If cycles are allowed (Phase 2), compiler supports them deterministically (budget + ordering rules).
- [ ] Delete/disable the proto→JSON→`rawNode` route and any remaining v1-shape parsing inside the compiler.
- [ ] Add grep gates tied to this phase:
  - [ ] `rg -n "v2ActionTypeToStepType\\(|snakeToCamel\\(" scenarios/browser-automation-studio/api/automation/compiler` must be 0.

### Acceptance Criteria

- Compiler is proto-native; no compiler codepath depends on legacy step type strings or camelCase param normalization.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 7 — Cut Over Execution To Proto `action` (Make `action` Sufficient)

### Goal

Make `action` required/sufficient end-to-end for execution. From this point on, execution correctness must not depend on `type/params`.

### Checklist

- [ ] Execution plans require `ActionDefinition` on each step.
- [ ] Executor routes by `ActionDefinition.type` (typed params), not `step.Type` strings.
- [ ] Playwright-driver dispatch uses `instruction.action` as the primary contract.
- [ ] `type/params` remain only as transitional diagnostics (optional) and must not change behavior.

### Acceptance Criteria

- Execution succeeds when `type/params` are unset but `action` is present.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 8 — Streaming Contract: UI Uses `timeline_entry`, Then Delete `legacy_payload`

### Goal

Stop paying the streaming compatibility tax. The server already emits a proto-first `timeline_entry`, but still attaches `legacy_payload`; the UI still reads it.

### Checklist

- [ ] UI: consume `timeline_entry` / `TimelineStreamMessage` as the primary data source for step timeline UX.
- [ ] Remove UI fallback dependency on `legacy_payload` once the proto-first path is complete.
- [ ] API/WS: stop attaching `legacy_payload` after consumers are migrated.
- [ ] Add a hard grep gate for regression:
  - [ ] `rg -n "legacy_payload" scenarios/browser-automation-studio` must only match docs/tests (or be 0 once deleted)

### Acceptance Criteria

- WebSocket consumers do not require `legacy_payload`.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 9 — Recording/Execution Unification (Timeline As Shared Contract)

### Goal

Make recordings and executions share the same core timeline structures so replay/export/UI are powered by one contract:

- **Recording:** “user did X” timeline entries with `ActionDefinition` + minimal context.
- **Execution:** “agent did X” timeline entries with the same core structure plus outcomes/artifacts/styling.

### Checklist

- [ ] Define the canonical proto types used by both:
  - [ ] recording timeline entries (flat list)
  - [ ] execution timeline entries (list + status/progress)
- [ ] Ensure recording ingestion produces the same timeline-entry contract used by execution streaming/persistence.
- [ ] Ensure UI replay consumes the unified timeline model (not legacy payloads, not bespoke recording shapes).

### Acceptance Criteria

- Replays (imported recordings and executed workflows) are powered by the same timeline entry model and rendering codepaths.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 10 — One Hydration/Persistence Layer (Single Owner; Single Anti-Corruption Boundary)

### Goal

Create a single, explicit hydration/persistence layer that:

- reads protojson from disk,
- uses DB only for indexes,
- and isolates all compatibility logic in one deletable “legacy/compat” module.

### Checklist

- [ ] Identify all current normalization/compat logic that mutates workflow shapes (UI save/load, API validation compat, protoconv helpers, compiler legacy shims).
- [ ] Move remaining “compat mutations” behind one module boundary (anti-corruption layer):
  - [ ] Legacy JSON payloads → `WorkflowDefinitionV2` conversion.
  - [ ] Any “protojson compat wrappers” become a single utility, not scattered.
- [ ] Enforce: internal layers operate on typed proto messages, not `map[string]any`, except inside the compat boundary.
- [ ] Lock ownership and boundaries:
  - [ ] All file IO + protojson parsing for workflows/executions lives in the hydration layer.
  - [ ] Database code stores only index/metadata types; domain types stay proto.

### Acceptance Criteria

- There is one place to delete when dropping legacy payload shapes.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 11 — Protojson Strictness Ratchet (From Lenient → Strict)

### Goal

Once data is migrated and compat is isolated, flip from “best-effort parsing” to “contract enforced”:

- canonical on-disk protojson is strict,
- unknown fields fail,
- schema/semantic validation is mandatory.

### Checklist

- [ ] Restrict `protojson.UnmarshalOptions{DiscardUnknown:true}` to compat-only parsing paths.
- [ ] Canonical readers for WorkflowSummary/WorkflowDefinitionV2/ExecutionTimeline use strict parsing (no silent discard).
- [ ] Add contract tests that assert v1 shapes and unknown fields are rejected.

### Acceptance Criteria

- Unknown fields and v1-shaped payloads cannot silently pass through canonical readers.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 12 — Remove Deprecated Surface Area (Lock In Canonical Contracts)

### Checklist

- [ ] Execution parameters:
  - [ ] Remove legacy `variables map<string,string>` support once all callers use `initial_store/initial_params/env`.
  - [ ] Ensure error messages guide callers to correct fields.
- [ ] Driver/executor contracts:
  - [ ] Reserve/remove deprecated `type/params` once `action` is sufficient end-to-end.
- [ ] Delete any remaining request-shape normalization that exists outside the compat boundary.

### Acceptance Criteria

- No legacy fields exist in API surface area or websocket payloads.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 13 — Delete v1 Code + Lock the Door (Prevent Regression)

### Checklist

- [ ] Delete remaining v1 codepaths:
  - [ ] v1 workflow parsing branches in compiler
  - [ ] v1 param normalization helpers in compiler/executor
  - [ ] legacy streaming fields
  - [ ] any “upgrade legacy fields for backward compatibility” blocks no longer needed
- [ ] Add explicit contract tests that fail if v1 shapes are accepted.
- [ ] Add a CI lint check that fails if v1 patterns are reintroduced (simple `rg` gates are fine).

### Acceptance Criteria

- v1 workflow JSON cannot be persisted or executed.
- v1 patterns are prevented from reappearing.

---

## Progress Snapshots (Fill In As We Go)

Record these at the end of each phase:

- Date:
- Phase completed:
- `v1 node` count (heuristic):
- `inline workflowDefinition` count:
- `legacy_payload` references count:
- Scenario test pass rate and top failures:

### Snapshot: 2025-12-15

- Phase completed: Phase 0 baseline started + Phase 3 started (subflows references-only enforced in executor/compiler/validator) + Phase 7 started (driver/executor tolerate action-only) + Phase 8 completed (WS proto-only; no legacy_payload)
- `v1 node` count (heuristic): 1424 (`rg -n "\"type\"\\s*:" bas api ui | wc -l`)
- `inline workflowDefinition` count: 2 (tests only) (`rg -n "workflowDefinition\"\\s*:" scenarios/browser-automation-studio | wc -l`)
- `legacy_payload` references count: 19 (docs + tests) (`rg -n "legacy_payload" scenarios/browser-automation-studio | wc -l`)
- Notes: validator now rejects `workflowDefinition` keys anywhere (`WF_FORBIDDEN_WORKFLOW_DEFINITION`)
- Notes: Resolve stage now rewrites subflow `workflow_path` → `workflow_id` pre-compile (both root + nested subflows)
- Notes: WS sink now emits strict protojson `TimelineStreamMessage` only (step started included); UI no longer reads `legacy_payload`
- Notes: playwright-driver now derives handler routing from typed `ActionDefinition` (accepts action-only instructions; legacy `type/params` optional)
- Notes: executor preflight capability analysis now derives requirements from typed `ActionDefinition` when legacy `type/params` are absent (action-only no longer breaks capability detection due to missing string step type)
- Notes: Go executor now treats `CompiledInstruction.Action` as primary for control-flow decisions (subflow / set_variable / navigate) so action-only instructions no longer rely on `instruction.Type`
- Notes: project file write endpoint (`/projects/{id}/files/write`) is proto-first for `flow_definition` (strict v2 protojson parse; v1 nodes/edges conversion only when payload looks legacy) via `BuildFlowDefinitionV2ForWrite`
- Notes: remaining handler-level v1→v2 conversions now route through the same compat boundary helper (`/workflows/{id}/modify`, record→workflow generation)
- Notes: AI workflow generation now uses strict v2 protojson + validation, with v1 nodes/edges fallback routed through the same compat boundary helper (`BuildFlowDefinitionV2ForWrite`)
- Notes: workflow_path resolution for action definitions (`actions/*` / `bas/actions/*`) no longer requires the calling workflow to exist in the DB index (supports ad-hoc + playbook/case workflows)
- Notes: selector manifest loading now resolves from scenario-root absolute paths (no longer depends on process working directory)
- Notes: workflow validation endpoints (`/workflows/validate`, `/workflows/validate-resolved`) now validate v2 via the same v2 write compat boundary and emit real proto `WorkflowValidationResult` issues (v1 validator is no longer authoritative for v2)
- Notes: compiler now rejects legacy v1-shaped nodes (missing `action`) instead of silently accepting `node.type/node.data`
- Notes: executor plan compilation no longer tries to synthesize `ActionDefinition` from legacy `type/params` (removed `V1DataToActionDefinition` fallbacks in the contract plan compiler + plan graph helpers); action comes only from v2 workflow nodes
- Notes: automation compiler now marshals flow_definition using proto field names (`use_proto_names: true`) to reduce casing drift at internal boundaries
- Notes: fixed proto contract violation in BAS fixtures by removing `SelectParams.label` when `SelectParams.value` is set (SelectParams oneof)
- Validation: `go test ./...` passes in `scenarios/browser-automation-studio/api`
- Scenario test pass rate and top failures: `vrooli scenario test browser-automation-studio` still fails on standards (HIGH+ missing PRD content/targets) and performance (Lighthouse mobile-dashboard perf < 70); playbooks is past the earlier v2 proto contract failure and path/manifest issues, but currently failing with `workflow failed: not found` during wait (latest logs in `scenarios/browser-automation-studio/coverage/logs/20251215-221248/`)

### Snapshot: 2025-12-16

- Phase completed: Phase 7 continued (action-only runtime is now the default); Phase 1 “stop bleeding” continued (eliminate v1 `type/params` reliance on hot paths)
- `v1 node` count (heuristic): 1431 (`rg -n "\"type\"\\s*:" bas api ui | wc -l`)
- `inline workflowDefinition` count: 2 (tests only) (`rg -n "workflowDefinition\"\\s*:" scenarios/browser-automation-studio | wc -l`)
- `legacy_payload` references count: 19 (docs + tests) (`rg -n "legacy_payload" scenarios/browser-automation-studio | wc -l`)
- Notes: executor loop/subflow handling now prefers typed `ActionDefinition` params end-to-end (loop execution runs from `LoopParams`; subflow parsing consumes `SubflowParams` including args; legacy map parsing remains only as fallback)
- Notes: contract plan compiler now emits action-only instructions/graph steps (deprecated `type/params` are cleared when `action` is present); graph→instruction conversion also clears legacy fields when action is present
- Notes: entry-probe “wait for selector” is now sent to the driver as a typed `WAIT` action (no legacy `type/params`)
- Notes: removed `WorkflowNodeV2 → legacy type/params` shaper (`automation/workflow/v2_execution.go` no longer populates deprecated fields)
- Notes: added regression tests for action-first context propagation and graph execution (`automation/executor/action_first_test.go`)
- Validation: `go test ./...` passes in `scenarios/browser-automation-studio/api`
- Scenario test pass rate and top failures: still fails on standards (HIGH+ missing PRD content/targets) and performance (Lighthouse mobile-dashboard perf < 70); playbooks still failing with `workflow failed: not found` during wait (latest logs in `scenarios/browser-automation-studio/coverage/logs/20251216-001732/`)

---

## “If We Get Stuck Again” (Triage Framework)

When an attempted deletion breaks something, categorize the break:

1. **Persisted data dependency** (old workflows in v1 shape)
2. **Runtime dependency** (subflow inline def, param normalization, token resolution)
3. **Consumer dependency** (UI/CLI expecting legacy fields on websocket or APIs)

Then back up to the earliest phase that guarantees the dependency is removed, and proceed again.
