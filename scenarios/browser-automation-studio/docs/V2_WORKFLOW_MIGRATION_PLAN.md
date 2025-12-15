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

We need to remove ambiguity around “files are the source of truth” by documenting the **two** on-disk shapes used today:

1. **Project workflow file (source of truth for persisted workflows)**
   - Stored on disk as **protojson `browser_automation_studio.v1.WorkflowSummary`**.
   - `WorkflowSummary.flow_definition` is the canonical workflow graph (`WorkflowDefinitionV2`).
   - Written by `WriteWorkflowSummaryFile` and read by `ReadWorkflowSummaryFile`.

2. **Fixture/workflow-definition file (used by BAS playbooks/tooling)**
   - Stored under `scenarios/browser-automation-studio/bas/actions/*.json` as **protojson `browser_automation_studio.v1.WorkflowDefinitionV2`**.
   - These are not persisted workflows; they are reusable definitions referenced by `workflow_path`.

**Policy (target):**

- Persisted workflows always store `WorkflowSummary` protojson (with `flow_definition` v2).
- Reusable playbook snippets always store `WorkflowDefinitionV2` protojson (referenced by `workflow_path`).
- Subflows reference workflows by `workflow_id` or `workflow_path` only (no inline embedded “workflowDefinition” maps).

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
   - Subflows are referenced by `workflowId` and/or `workflowPath` (or other explicit reference type).
   - Inline `workflowDefinition` maps are removed (or strictly limited to internal tooling with full proto conversion).

4. **Streaming contract is singular**
   - Websocket event payloads are contract/proto-first (prefer the `timeline_entry` / `TimelineStreamMessage` shape).
   - Legacy websocket `legacy_payload` is removed once no consumers use it.

5. **There is one canonical token resolution strategy**
   - Selector/token/fixture resolution happens in exactly one place (or a clearly-defined staged pipeline), not “sometimes here, sometimes there”.

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

### Checklist

- [ ] Decide and record the v2-only definition (use the “Definition” section above; edit if needed).
- [ ] Confirm and document canonical on-disk formats (WorkflowSummary vs WorkflowDefinitionV2) in this plan (see section above).
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

## Phase 2 — Migrate Existing Stored Workflows + Versions

### Goal

Eliminate v1 definitions already stored (especially version history).

### Checklist

- [ ] Identify storage locations of workflow definitions/versions used by BAS (filesystem + protojson snapshots, including `.bas/versions`).
- [ ] Prefer leveraging existing “read + normalize + write back” behavior where possible (protojson is the source of truth on disk).
  - [ ] Batch-walk project workflow files, load via the workflow proto reader, and rewrite any file flagged as needing normalization.
  - [ ] Ensure version snapshots under `.bas/versions/<workflowID>/*.json` are also migrated/normalized (or regenerated).
- [ ] Add a one-time migration command (CLI or admin endpoint) to normalize a project/workflow tree (no new deps).
- [ ] Decide policy: preserve old v1 versions as historical artifacts vs upgrade all versions.
  - Recommended: upgrade all versions to v2 so the executor/compiler can drop v1 entirely later.

### Acceptance Criteria

- “v1 usage report” shows zero persisted v1 workflows/versions.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio` on a repo state with migrated workflows (record).

---

## Phase 3 — Subflows: References Only + Introduce `SubflowResolver` Seam (Delete Inline Maps)

### Why This Matters

Inline `workflowDefinition` keeps a map-shaped legacy workflow embedded in params. The executor currently calls out a MIGRATION gap:
`scenarios/browser-automation-studio/api/automation/executor/flow_executor.go` (inline def conversion skipped).

If we don’t address this, we cannot delete legacy “map workflow” behavior safely.

### Decision

Pick the references-only approach and formalize it as a seam:

- Runtime subflows are **references only** (`workflow_id` / `workflow_path`) via `ACTION_TYPE_SUBFLOW`.
- Resolution is performed by a single injected resolver, not by pre-processing hacks or embedding workflow maps into JSON.

### Checklist

- [ ] Define a `SubflowResolver` seam (compiler/executor boundary) that resolves `workflowId`/`workflowPath` to a `WorkflowDefinitionV2`.
- [ ] Production implementation resolves via the workflow/project file store (filesystem source of truth).
- [ ] Test/tooling implementation resolves fixture references without embedding inline `workflowDefinition` maps into workflow JSON.
- [ ] Remove any dependency on “inline workflowDefinition takes precedence” behaviors.
- [ ] Validator: reject `workflowDefinition` in subflow params and require `workflow_id|workflow_path`.

### Acceptance Criteria

- No workflows/executions rely on inline `workflowDefinition`.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 4 — Streaming Contract: UI Consumes `timeline_entry`, Then Delete `legacy_payload`

### Goal

Stop paying the streaming compatibility tax. The server already emits a proto-first `timeline_entry`, but still attaches `legacy_payload`; the UI still reads it.

### Checklist

- [ ] UI: consume `timeline_entry` / `TimelineStreamMessage` as the primary data source for step timeline UX.
- [ ] Remove UI fallback dependency on `legacy_payload` once the proto-first path is complete.
- [ ] API/WS: stop attaching `legacy_payload` after consumers are migrated.
- [ ] Keep a clear compatibility window (date/phase marker) so we don’t carry this forever.

### Acceptance Criteria

- WebSocket consumers do not require `legacy_payload`.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 5 — One Hydration/Persistence Layer (Isolate + Delete Compat)

### Goal

Create a single, explicit hydration/persistence layer that:

- reads protojson from disk,
- uses DB only for indexes,
- and isolates all compatibility logic in one deletable “legacy/compat” module.

### Checklist

- [ ] Identify all current normalization/compat logic that mutates workflow shapes (UI save/load, API validation compat, protoconv helpers).
- [ ] Move remaining “compat mutations” behind one module boundary (anti-corruption layer):
  - [ ] Legacy JSON payloads → `WorkflowDefinitionV2` conversion.
  - [ ] Any “protojson compat wrappers” (e.g., JsonValue wrapping in subflow args) become a single utility, not scattered.
- [ ] Enforce: internal layers operate on typed proto messages, not `map[string]any`, except inside the compat boundary.

### Acceptance Criteria

- There is one place to delete when we finally drop legacy payload shapes.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 6 — Collapse Workflow Shape Compatibility in Compiler (Delete v1 Workflow Parsing)

### Goal

Remove v1 acceptance in the compiler once all persisted workflows are v2 and subflows are references.

### Checklist

- [ ] Remove v1 node fields support (`rawNode.Type`, `rawNode.Data`) and v1 edge handle fields once no longer needed.
- [ ] Compiler operates on `WorkflowDefinitionV2` as the canonical input (no “protojson → map → legacy structs → step strings” funnel).
- [ ] Remove v2→v1 step type mapping (`v2ActionTypeToStepType`) only after Phase 8 (execution consumes proto `action` natively).
- [ ] Tighten schema validation so bad shapes fail early and clearly.

### Acceptance Criteria

- Compiler rejects v1 workflow JSON (with explicit error).

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 7 — Proto Completeness For Workflow Control-Flow + Branching + Variables

### Why This Exists

We cannot reach true v2-only if execution still depends on legacy step type strings and ad-hoc conventions for control flow and branching. The proto model must be able to represent:

- loops/cycles deterministically,
- branching conditions explicitly (not just edge labels),
- and variable semantics (what can read/write, and where).

### Checklist

- [ ] Decide the control model (document in this plan with examples):
  - [ ] **Cycles/loops**: pure graph cycles vs explicit loop node (if cycles, define step budgets + cycle guards; if node, define typed params).
  - [ ] **Branching**: typed edge condition(s) (not only `label`) so execution routing is deterministic and validated.
  - [ ] **Variables**: confirm and enforce namespace semantics (`@store/`, `@params/`, `@env/`) for reads/writes; define how subflows pass args/params and merge results.
- [ ] Update proto schemas to cover the chosen control-flow + branching model and regenerate code.
- [ ] Update validator rules accordingly (control semantics validated as strictly as browser actions).

### Acceptance Criteria

- Control-flow execution no longer depends on legacy step type strings.
- The chosen control-flow semantics are representable and validated in proto.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 8 — Cut Over Execution To Proto `action` (Delete Stringly Dispatch + Param Normalization)

### Why This Is the Core Cutover

We are not truly v2-only if execution still depends on v1-ish `type` strings, camelCase params, or a “legacy compiler” path. The driver contract already supports a typed `action` field; this phase makes it the single execution path and removes the “two compilers” problem.

### Checklist

- [ ] Plan compilation produces a single canonical contract:
  - [ ] `contracts.CompiledInstruction.action` and `contracts.PlanStep.action` are required and sufficient.
  - [ ] `type/params` are no longer required for execution (may remain temporarily for diagnostics only).
- [ ] Eliminate stringly dispatch:
  - [ ] Executor routes by `ActionDefinition.type` (and typed params) instead of `step.Type` strings.
  - [ ] Playwright-driver dispatch uses `CompiledInstruction.action` only.
- [ ] Eliminate normalization taxes:
  - [ ] Stop using `snakeToCamel` and `v2ActionTypeToStepType` on the execution-critical path.
  - [ ] Remove any remaining “inline workflowDefinition” compilation paths (subflows resolved via seam from Phase 3).

### Acceptance Criteria

- Execution succeeds when `type/params` are unset but `action` is present.
- No execution path requires camelCase-normalized param keys.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 9 — Remove Legacy Execution Compatibility

### Checklist

- [ ] Execution parameters:
  - [ ] Remove legacy `variables map<string,string>` support once all callers use `initial_store/initial_params/env`.
  - [ ] Ensure error messages guide callers to correct fields.

### Acceptance Criteria

- No legacy fields exist in API surface area or websocket payloads.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 10 — Delete v1 Code + Lock the Door (Prevent Regression)

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

---

## “If We Get Stuck Again” (Triage Framework)

When an attempted deletion breaks something, categorize the break:

1. **Persisted data dependency** (old workflows in v1 shape)
2. **Runtime dependency** (subflow inline def, param normalization, token resolution)
3. **Consumer dependency** (UI/CLI expecting legacy fields on websocket or APIs)

Then back up to the earliest phase that guarantees the dependency is removed, and proceed again.
