# BAS Playbook Execution Overhaul Plan

## Context
- Browser Automation Studio (BAS) workflows currently execute according to requirement references surfaced by `testing::phase::expected_validations_for`, so playbook order is opaque and reseeding happens before every workflow via `_testing_playbooks__apply_seeds_if_needed`.
- Integration test phase (which executes BAS workflows, so technicallly it's the e2e tests - but don't worry about the naming) times out because workflow execution is slow. Part of the reason is that each workflow first reseeds.
- Integration logs are noisy because `_testing_playbooks__wait_for_execution` prints a heartbeat every two seconds and `registry.json` stores verbose `validations`/`fixture_details` payloads that make the file >2000 lines, exceeding single-pass read limits for some agents.
- Deterministic seeding is essential: some workflows mutate the seeded demo project, so leaving BAS+Browserless “hot” without resets risks state drift. We want metadata-driven reseeding instead of resetting before every workflow run.
- Contributors need a predictable “story” for workflows: folder structure under `test/playbooks/` should dictate execution order, and the registry should surface that order plus reseed requirements while staying compact enough to read easily.

## Goals
1. Execute workflows depth-first, following the folder hierarchy in `test/playbooks/` (with numeric prefixes like `01-foundation`, `02-builder` to encode ordering).
2. Keep BAS and Browserless running through a batch but reseed automatically whenever a workflow declares that it mutates state, rather than every time
3. Slim down `test/playbooks/registry.json` so agents can read it quickly while still seeing ordering, fixture usage, requirement IDs, and reseed hints.
4. Preserve linkage validation (requirements ↔ playbooks) with updated wording/code to reflect the new execution model.
5. Teach agents (via docs/lint) to use 2-digit numeric prefixes, regenerate the registry, and verify ordering after edits.
6. Update logs to only show the collected step/heartbeat logs if the workflow failed (e.g. timed out, failed assert step)

## Plan of Action

### 1. Registry Schema Redesign
- Define a compact JSON schema for each playbook entry containing:
  - `file`: relative path (already present).
  - `description`: optional short text.
  - `requirements`: array of requirement IDs (replace verbose `validations`).
  - `fixtures`: existing array of fixture slug strings.
  - `resetRequirement`: enum (`none`, `project`, `global`) derived from workflow metadata.
  - `order`: numeric index computed from the depth-first traversal (stays stable even if requirements change).
- Remove heavy fields (`fixture_details`, `validations`, `fixture_requirements`). Document the schema in `test/playbooks/README.md`.
- Update `scripts/scenarios/testing/playbooks/build-registry.mjs` to emit the streamlined payload. Ensure it sorts entries depth-first according to folder prefixes (directories sorted lexicographically so `01-*` precedes `02-*`, etc.) to match order execution.

### 2. Workflow Metadata & Resolver Enhancements
- Extend workflow metadata JSON to allow `"reset": "none|project|global"` (or similar). Default to `project` for safety when unspecified.
- Update `scripts/scenarios/testing/playbooks/resolve-workflow.py` so the resolved JSON includes the reset value even after fixtures are inlined, and so it propagates fixture-level requirements/ordering metadata if needed.
- Add linting to fail when metadata is missing or invalid, ensuring every workflow declares its reset behavior.

### 3. Runner Adjustments for Ordered Execution & Reseeding
- Teach `testing::phase::run_workflow_validations` (alias `run_bas_automation_validations`) to:
  - Read `test/playbooks/registry.json` and iterate entries in `order` sequence instead of relying on requirement traversal.
  - Before each workflow, check the `resetRequirement` flag and run `test/playbooks/__seeds/cleanup.sh` + `apply.sh` whenever a reset is required. Track current state so consecutive `reset=none` workflows reuse the same seeded app.
  - Keep readiness checks (scenario up, Browserless health) at batch boundaries; optionally group workflows by top-level directory to minimize repeated resets when multiple flows share the same data context.
- Continue writing coverage evidence keyed by requirement IDs pulled from the registry entry.

### 4. Linkage Validation Updates
- Update `scripts/scenarios/testing/shell/validate-playbook-linkage.sh` and any related helpers to:
  - Still enforce the three checks (missing workflow file, workflow referencing missing requirement, orphan playbook) but clarify error messages: execution order now comes from the registry, so missing links mean coverage gaps, not ordering issues.
  - Ensure the validator reads the new `requirements` array from `registry.json` (rather than `validations`) when verifying playbook coverage.

### 5. Documentation & Contributor Guidance
- Revise `scenarios/browser-automation-studio/test/playbooks/README.md` to describe the story-based execution model, numeric directory prefixes, metadata reset semantics, and the need to rerun `build-registry.mjs` after edits.
- Add guidance to `docs/testing/guides/ui-automation-with-bas.md` (or a new doc) emphasizing depth-first execution, how to choose prefixes, and how to interpret `registry.json` ordering.
- Possibly add a CI lint (or git hook) that fails when a new directory lacks the `NN-` prefix or when `registry.json` isn’t regenerated after playbook edits.

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
- Document these logging expectations alongside the ordering plan so agents know how to inspect failures without scanning megabytes of logs.

## Dependencies & Risks
- Requires consensus on metadata schema and the naming convention for directories; coordinate with standards maintainers before enforcing lint.
- Existing workflows without metadata will default to `project` resets, which might increase reseed count until authors annotate them properly.
- Need to ensure backward compatibility during rollout (e.g., runner could fall back to the old requirement-driven order if `registry.json` hasn’t been regenerated yet).

## Verification Strategy
- Unit tests for `build-registry.mjs` covering ordering, reset flags, and schema.
- Integration tests for `run_workflow_validations` using a small set of mock playbooks (including different reset modes) to confirm reseeding triggers as expected.
- Update structure/integration phase artifacts to prove logs now show the desired order and reduced noise.
