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

## Current Reality (What Exists Today)

Key code anchors:

- Workflow execution entrypoint: `scenarios/browser-automation-studio/api/services/workflow/executions.go`
- Adhoc execution: `scenarios/browser-automation-studio/api/services/workflow/adhoc_execution.go`
- Compiler supports both v1 and v2 nodes: `scenarios/browser-automation-studio/api/automation/compiler/compiler.go`
- Engine is Playwright-only: `scenarios/browser-automation-studio/api/automation/engine/README.md`
- Subflow inline-def MIGRATION gap: `scenarios/browser-automation-studio/api/automation/executor/flow_executor.go`
- Websocket sink emits proto but attaches legacy payload: `scenarios/browser-automation-studio/api/automation/events/ws_sink.go`
- UI normalizes workflow response by converting to v1 flow to get nodes/edges reliably: `scenarios/browser-automation-studio/ui/src/stores/workflowStore.ts`

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
   - Any non-browser “control” steps required at runtime (loops/branching/set-variable/subflow dispatch) have a proto representation so execution does not depend on legacy step type strings (e.g. `"loop"`, `"setVariable"`).
   - If the chosen design is “loops are graph cycles” (no explicit loop node), then the legacy loop node type is removed entirely.

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

### Checklist

- [ ] Decide and record the v2-only definition (use the “Definition” section above; edit if needed).
- [ ] Add a “v1 usage report” script/command sequence (no new deps) to run locally and in CI:
  - [ ] `rg -n "\"type\"\\s*:\\s*\"[a-z]+\"|\"data\"\\s*:\\s*\\{" scenarios/browser-automation-studio` (heuristic for v1 nodes)
  - [ ] `rg -n "workflowDefinition\"\\s*:" scenarios/browser-automation-studio` (inline subflows)
  - [ ] `rg -n "legacy_payload" scenarios/browser-automation-studio` (websocket compatibility)
  - [ ] `rg -n "\"type\"\\s*:\\s*\"(loop|setVariable|workflowCall)\"|isSetVariableStep\\(|StepLoop|workflowcall" scenarios/browser-automation-studio/api` (stringly-typed control flow)
  - [ ] `rg -n "snakeToCamel\\(|v2ActionTypeToStepType\\(" scenarios/browser-automation-studio/api` (v2→v1 normalization helpers)
- [ ] Capture current counts in this file under “Progress Snapshots”.

### Acceptance Criteria

- We can answer, with evidence: “Which workflows/paths still require v1 today?”

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio` (baseline run; record pass rate and top failure categories).

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

## Phase 3 — Subflows: Remove Inline `workflowDefinition` (Or Fully Convert It)

### Why This Matters

Inline `workflowDefinition` keeps a map-shaped legacy workflow embedded in params. The executor currently calls out a MIGRATION gap:
`scenarios/browser-automation-studio/api/automation/executor/flow_executor.go` (inline def conversion skipped).

If we don’t address this, we cannot delete legacy “map workflow” behavior safely.

### Two Options (Pick One)

**Option A (Recommended): Ban inline `workflowDefinition`**

- Treat subflows as references only: `workflowId` / `workflowPath`.
- Inline defs are converted into real workflows on disk by tooling (e.g., playbook scaffolding).

**Option B: Implement full inline-def proto conversion**

- Convert the inline map into `basworkflows.WorkflowDefinitionV2` reliably (including nested nodes/edges/settings).
- This is higher risk and tends to perpetuate “map-shaped” compatibility.

**Option C (Recommended when tests/tooling need fixtures): Introduce a SubflowResolver seam**

- Keep runtime subflows as references only (`workflowId`/`workflowPath`).
- Add a resolver seam that can materialize a referenced workflow definition for:
  - Production (project file store / workflow service)
  - Tests (fixture registry) without embedding full inline `workflowDefinition` maps into the workflow JSON.
- This avoids coupling runtime compilation to the test runner’s pre-processing shape.

### Checklist (Option A)

- [ ] Validator: enforce subflow references are `workflowId`/`workflowPath` (and/or explicit reference type); reject `workflowDefinition`.
- [ ] Update playbooks/test tooling to never emit inline `workflowDefinition` (prefer `workflowPath` to files under `bas/actions/...`).
- [ ] Remove inline workflowDefinition parsing/execution paths.

### Checklist (Option C)

- [ ] Define a `SubflowResolver` seam (compiler/executor boundary) that resolves `workflowId`/`workflowPath` to a `WorkflowDefinitionV2`.
- [ ] Production implementation resolves via the workflow/project file store (filesystem source of truth).
- [ ] Test/tooling implementation resolves fixture references without embedding inline `workflowDefinition` maps into workflow JSON.
- [ ] Remove any dependency on “inline workflowDefinition takes precedence” behaviors.

### Acceptance Criteria

- No workflows/executions rely on inline `workflowDefinition`.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 4 — Collapse Workflow Shape Compatibility in Compiler (Delete v1 Workflow Parsing)

### Goal

Remove v1 acceptance in the compiler once all persisted workflows are v2 and subflows are references.

### Checklist

- [ ] Remove v1 node fields support (`rawNode.Type`, `rawNode.Data`) and v1 edge handle fields once no longer needed.
- [ ] Compiler operates on `WorkflowDefinitionV2` as the canonical input (no “protojson → map → legacy structs → step strings” funnel).
- [ ] Remove v2→v1 step type mapping (`v2ActionTypeToStepType`) only after Phase 6 (execution consumes proto `action` natively).
- [ ] Tighten schema validation so bad shapes fail early and clearly.

### Acceptance Criteria

- Compiler rejects v1 workflow JSON (with explicit error).

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 5 — Proto Completeness For Workflow Control-Flow (Unblock True “V2-Only”)

### Why This Exists

We cannot reach true v2-only if execution still depends on legacy step type strings for control flow (e.g. `"loop"`, `"setVariable"`), because those are not represented by `ActionDefinition` today. Before we can cut over to proto `action` everywhere, we must decide and encode the workflow control model.

### Checklist

- [ ] Decide the control-flow model (document in this plan):
  - [ ] **Option 1:** Loops are explicit workflow nodes (add proto support, e.g. `ACTION_TYPE_LOOP` + `LoopParams` / `ACTION_TYPE_SET_VARIABLE` + params).
  - [ ] **Option 2:** Loops are pure graph cycles (delete explicit loop node type; executor enforces step budget + optional cycle guards).
  - [ ] **Option 3:** Control-flow nodes are a separate proto domain (not `ActionDefinition`) but still fully typed and shared (UI/API/driver).
- [ ] Update proto schemas to cover the chosen control-flow model and regenerate code.
- [ ] Update validator rules accordingly (so control-flow nodes are validated with the same rigor as browser actions).

### Acceptance Criteria

- Control-flow execution no longer depends on legacy step type strings.
- The chosen control-flow semantics are representable and validated in proto.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 6 — Cut Over Execution To Proto `action` (Delete v2→v1 Normalization)

### Why This Is the Core Cutover

We are not truly v2-only if the compiler/executor/driver still depend on v1-ish `type` strings and camelCase params. The driver contract already supports a typed `action` field; this phase makes it the single execution path.

### Checklist

- [ ] Compiler emits `CompiledInstruction.action` / `PlanStep.action` as the canonical execution payload.
  - [ ] Stop using `snakeToCamel` and `v2ActionTypeToStepType` on the execution-critical path.
  - [ ] Stop inlining/consuming map-shaped `workflowDefinition` during compilation.
- [ ] Executor routes by `ActionDefinition.type` (and typed params) instead of `step.Type` strings.
- [ ] Playwright-driver consumes `CompiledInstruction.action` and no longer requires `instruction.type` for dispatch.

### Acceptance Criteria

- Execution succeeds when `type/params` are unset but `action` is present.
- No execution path requires camelCase-normalized param keys.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 7 — Remove Legacy Execution/Streaming Compatibility

### Checklist

- [ ] Execution parameters:
  - [ ] Remove legacy `variables map<string,string>` support once all callers use `initial_store/initial_params/env`.
  - [ ] Ensure error messages guide callers to correct fields.
- [ ] Websocket streaming:
  - [ ] Move UI/CLI consumers to consume the proto-first timeline shape (`timeline_entry` / `TimelineStreamMessage`) end-to-end.
  - [ ] Remove `legacy_payload` attachment once UI/CLI consumers no longer require it.

### Acceptance Criteria

- No legacy fields exist in API surface area or websocket payloads.

### Test Gate

- [ ] `vrooli scenario test browser-automation-studio`

---

## Phase 8 — Delete v1 Code + Lock the Door (Prevent Regression)

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
