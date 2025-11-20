# BAS Playbook Execution Overhaul Plan

## Context
- Browser Automation Studio (BAS) workflows currently execute according to requirement references surfaced by `testing::phase::expected_validations_for`, so playbook order is opaque and reseeding happens before every workflow via `_testing_playbooks__apply_seeds_if_needed`.
- Integration test phase (which executes BAS workflows, so technicallly it's the e2e tests - but don't worry about the naming) times out because workflow execution is slow. Part of the reason is that each workflow first reseeds.
- Integration logs are noisy because `_testing_playbooks__wait_for_execution` prints a heartbeat every two seconds and `registry.json` stores verbose `validations`/`fixture_details` payloads that make the file >2000 lines, exceeding single-pass read limits for some agents.
- Deterministic seeding is essential: some workflows mutate the seeded demo project, so leaving BAS+Browserless “hot” without resets risks state drift. We want metadata-driven reseeding instead of resetting before every workflow run.
- Contributors need a predictable “story” for workflows: folder structure under `test/playbooks/` should dictate execution order, and the registry should surface that order plus reseed requirements while staying compact enough to read easily.

## Goals
1. Execute **every** workflow JSON under `test/playbooks/` (excluding the `__*` fixture/seed folders) depth-first by folder hierarchy (with numeric prefixes like `01-foundation`, `02-builder` encoding order). Requirement coverage becomes an overlay rather than the execution driver.
2. Keep BAS and Browserless running through a batch but reseed automatically whenever a workflow declares that it mutates state, rather than reseeding before every run.
3. Slim down `test/playbooks/registry.json` so agents can read it quickly while still seeing ordering, fixture usage, requirement IDs, and reseed hints.
4. Preserve linkage validation (requirements ↔ playbooks) with updated wording/code to reflect the new execution model and warn—but do not block—when playbooks lack requirement references.
5. Teach agents (via docs/lint) to use 2-digit numeric prefixes, regenerate the registry, and verify ordering after edits. The registry should expose a dotted-path order key that mirrors the folder structure.
6. Update logs to only show the collected step/heartbeat logs if the workflow failed (e.g. timed out, failed assert step).

## Plan of Action

### 1. Registry Schema Redesign
- Define a compact JSON schema for each playbook entry containing:
  - `file`: relative path (already present).
  - `description`: optional short text.
  - `order`: dotted path derived from the lexicographic traversal (e.g., `01.03.002`). The runner can sort on this lexicographically to reproduce depth-first behavior without renumbering large chunks when a file moves.
  - `requirements`: array of requirement IDs gleaned from the requirement registry (replaces verbose `validations`). These IDs fuel coverage reporting when the playbook finishes.
  - `fixtures`: existing array of fixture slug strings.
  - `resetRequirement`: enum (`none`, `project`, `global`) derived from workflow metadata or fixture metadata (highest severity wins).
- Remove heavy fields (`fixture_details`, `validations`, `fixture_requirements`). Document the schema in `test/playbooks/README.md` along with guidance on interpreting `order`.
- Update `scripts/scenarios/testing/playbooks/build-registry.mjs` to emit the streamlined payload. The traversal should walk the directory tree depth-first, skipping `__*` folders, and build the dotted `order` using directory-name prefixes (agents can continue to use `01-*` naming to influence ordering).
- For workflows with no requirement references, set `requirements: []` but keep them in the registry so they run. The runner will warn about orphan coverage while still executing them.
- Keep the registry as a single file for now; the simplified schema should keep it well under prior size limits. If it ever grows again, revisit splitting by top-level directory.

### 2. Workflow Metadata & Resolver Enhancements
- Extend workflow metadata JSON to allow `"reset": "none|project|global"` (or similar). Default to `project` for safety when unspecified until authors annotate each flow.
- Update `scripts/scenarios/testing/playbooks/resolve-workflow.py` so the resolved JSON includes the reset value even after fixtures are inlined. Fixtures/subflows also need a `metadata.reset`, and the resolver must bubble the “most severe” reset up to parents (fixture reset > parent reset, default `project` if neither declare one).
- Remove `metadata.requirement` from workflow definitions. Requirement linkage will live solely in `requirements/` (auto-sync still needs `validation.ref` entries), and the registry builder will attach the referenced requirement IDs to each playbook entry.
- Add linting to fail when reset metadata is missing/invalid and to warn when a playbook has zero requirement references (so authors know it won’t count toward coverage until the `requirements/` registry references it).

### 3. Runner Adjustments for Ordered Execution & Reseeding
- Teach `testing::phase::run_workflow_validations` (alias `run_bas_automation_validations`) to:
  - Load `test/playbooks/registry.json`, iterate entries lexicographically by `order`, and run every workflow JSON outside of `__*` folders. Requirements continue to determine coverage, not execution.
  - Treat `requirements: []` entries as valid—but emit a warning after the run so authors know that flow isn’t counted toward requirement sync.
  - Before each workflow, check the `resetRequirement` flag and run `test/playbooks/__seeds/cleanup.sh` + `apply.sh` whenever a reset is required. Track current state so consecutive `reset=none` workflows reuse the same seeded app.
  - Keep readiness checks (scenario up, Browserless health) at batch boundaries; optionally group workflows by top-level directory to minimize repeated resets when multiple flows share the same data context.
- For coverage, look up the registry entry’s `requirements` array and call `testing::phase::add_requirement` for each ID (dedupe as needed). This replaces the current `testing::phase::expected_validations_for` path, so there’s no fallback to requirement-driven execution.
- When heartbeats/log noise are suppressed, persist failure context (timeline JSON, latest screenshot) into `coverage/automation/` so engineers can debug without reruns.

### 4. Linkage Validation Updates
- Update `scripts/scenarios/testing/shell/validate-playbook-linkage.sh` and related helpers to:
  - Still enforce the three checks (missing workflow file, workflow referenced by requirements but missing on disk, and workflows that reference unknown requirements), but downgrade “orphan playbook” (a playbook not referenced by any requirement) to a warning so the run continues. We still need the warning so contributors hook the workflow into requirements for auto-sync.
  - Ensure the validator reads the new `requirements` array from `registry.json` (instead of `validations`) when verifying the registry matches the requirement graph.
  - Remove expectations around `metadata.requirement`; tooling should now compare requirement files ↔ registry entries exclusively to avoid redundant declarations.
  - Keep fixture validation (fixture IDs, fixture-level requirement references) untouched; fixture metadata still needs to cite requirement IDs when they guarantee coverage.

- Revise `scenarios/browser-automation-studio/test/playbooks/README.md` to describe the story-based execution model, numeric directory prefixes, metadata reset semantics, and the need to rerun `build-registry.mjs` after edits. Explicitly mention that `metadata.requirement` is deprecated and that requirements must reference playbooks via `validation.ref`.
- Add guidance to `docs/testing/guides/ui-automation-with-bas.md` (or a new doc) emphasizing depth-first execution, how to choose prefixes, how to interpret the dotted `order`, and how requirement coverage now flows from the registry rather than workflow metadata.
- Add a lint/structure check (CI and local docs) that fails when top-level directories lack numeric prefixes, when prefixes collide, or when `registry.json` hasn’t been regenerated after playbook edits. Also surface warnings when `requirements: []` in the registry so contributors know to hook up coverage.

### 6. Noise Reduction (Optional but Recommended for Same Workstream)
- While revisiting the runner, gate most workflow output behind a fail condition. Logs should go from (example) this:
"""
🚀 Starting workflow: generated-smoke
   Execution ID: 393e42af-8ecb-4b85-8657-74cd17d08878
   [running] 0% - Initializing
   [running] 80% - screenshot (debug-screenshot-pre-assert)
   [completed] 100% - Completed
✅ Workflow generated-smoke completed in 4s (5 steps, 1/1 assertions passed)
🚀 Starting workflow: ai-modal-switch-to-manual
   Execution ID: b968c65f-e85c-4057-a37c-63568c7a04bb
   [running] 0% - Initializing
   [failed] 64% - assert (assert-modal-closed)
❌ Workflow ai-modal-switch-to-manual failed after 4s (7 steps, 1/2 assertions passed)
{"id":"b968c65f-e85c-4057-a37c-63568c7a04bb","workflow_id":"421243c6-b985-4c8b-b12e-130a9f8627f0","status":"failed","trigger_type":"adhoc","started_at":"2025-11-20T14:11:31.099172Z","completed_at":"2025-11-20T09:11:34.842023Z","last_heartbeat":"2025-11-20T14:11:34.831539Z","error":"Expected selector \"[role=\\\"dialog\\\"] h2\" to be absent but it exists","result":{"steps":7,"success":false},"progress":64,"current_step":"assert (assert-modal-closed)","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}
"""
To this:
"""
✅ Workflow generated-smoke completed in 4s (5 steps, 1/1 assertions passed)
🚀 Starting workflow: ai-modal-switch-to-manual
   Execution ID: b968c65f-e85c-4057-a37c-63568c7a04bb
   [running] 0% - Initializing
   [failed] 64% - assert (assert-modal-closed)
❌ Workflow ai-modal-switch-to-manual failed after 4s (7 steps, 1/2 assertions passed)
{"id":"b968c65f-e85c-4057-a37c-63568c7a04bb","workflow_id":"421243c6-b985-4c8b-b12e-130a9f8627f0","status":"failed","trigger_type":"adhoc","started_at":"2025-11-20T14:11:31.099172Z","completed_at":"2025-11-20T09:11:34.842023Z","last_heartbeat":"2025-11-20T14:11:34.831539Z","error":"Expected selector \"[role=\\\"dialog\\\"] h2\" to be absent but it exists","result":{"steps":7,"success":false},"progress":64,"current_step":"assert (assert-modal-closed)","created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}
"""
This will ensure that only one line of output occurs for successful workflows, while failing workflows give the same detailed information (just after failure instead of during execution, but that's totally fine)
- Document these logging expectations alongside the ordering plan so agents know how to inspect failures without scanning megabytes of logs. Explicitly mention where the timeline/screenshot artifacts will be written when a workflow fails (`coverage/automation/<workflow>.json|png`).

## Dependencies & Risks
- Requires consensus on metadata schema and the naming convention for directories; coordinate with standards maintainers before enforcing lint.
- Existing workflows without metadata will default to `project` resets, which might increase reseed count until authors annotate them properly.
- Removing `metadata.requirement` touches validators, docs, and the BAS API schema (e.g., `validator.go`); plan for a transition window where both formats are accepted so existing workflows don’t fail lint immediately.
- Because execution will no longer fall back to requirement traversal, we must ensure every repo has an up-to-date `registry.json` before enabling the new runner logic (e.g., block the change behind a feature flag or add a preflight check that errors if the registry is missing the new schema version).

## Verification Strategy
- Unit tests for `build-registry.mjs` covering ordering, reset flags, and schema.
- Integration tests for `run_workflow_validations` using a small set of mock playbooks (including different reset modes) to confirm reseeding triggers as expected.
- Update structure/integration phase artifacts to prove logs now show the desired order and reduced noise.
