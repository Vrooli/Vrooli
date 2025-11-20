# Requirement Auto-Sync & Drift Detection Plan

## Background
Agents currently rely on the phased testing pipeline plus `scripts/requirements/report.js --mode sync` to keep `requirements/**/*.json` and PRD checklists aligned with reality. Each test phase writes machine-readable evidence to `coverage/phase-results/<phase>.json` (via `testing::phase::end_with_summary`), and vitest coverage goes to `ui/coverage/vitest-requirements.json`. After the suite finishes, `test/run-tests.sh` automatically invokes `report.js --mode sync`, which rewrites requirement files based on the collected evidence.

Missing pieces:
- Requirement files lose the provenance of their last update; `_sync_metadata` (supported in the JSON schema) is unused.
- There is no canonical snapshot showing the last known operational-target/requirement status, nor any comparison to `PRD.md` checklists.
- `vrooli scenario status` cannot warn when PRD checkboxes or requirement files drift from the last verified evidence.

## Current Evidence Flow (for reference)
1. `test/phases/test-*.sh` source `scripts/scenarios/testing/shell/phase-helpers.sh` and call `testing::phase::add_requirement`, which records requirement IDs, statuses, criticality, and evidence text.
2. On exit, `testing::phase::end_with_summary` writes `coverage/phase-results/<phase>.json` with fields like `phase`, `status`, `updated_at`, and an array of requirement entries.
3. Vitest’s `[REQ:ID]` tagging produces `ui/coverage/vitest-requirements.json` with additional evidence.
4. `scripts/requirements/lib/evidence.js` merges phase JSON and Vitest data into `requirementEvidence` and feeds it to `report.js`.
5. `scripts/requirements/lib/sync.js` updates requirement files, but only flips `status`/`validation.status` and `last_synced_at`—no `_sync_metadata`, no link back to the evidence files or commands that produced them.

## Planned Enhancements
1. **Normalize Evidence Inputs for Every Validation Type**
   - *Phased test + automation evidence*: keep `coverage/phase-results/<phase>.json` as the canonical source for `type: test` and `type: automation` validations (Go, Node, Python, CLI, BAS workflows, bundle/memory checks). Ensure the loader records the relative JSON path that produced each record so `_sync_metadata` can reference it later.
   - *Vitest UI coverage*: continue ingesting `ui/coverage/vitest-requirements.json`, but treat it as supplemental evidence that attaches to the matching phase JSON entry (unit) instead of an independent source.
   - *Lighthouse validations*: wire `scripts/requirements/lib/evidence.js` to ingest `coverage/phase-results/lighthouse.json` (written by the Lighthouse runner) so every `type: lighthouse` validation automatically receives pass/fail + score evidence tied to `.vrooli/lighthouse.json` page IDs.
   - *Manual validations*: introduce a first-class manifest (e.g., `coverage/manual-validations/log.jsonl`) that agents update through a CLI (`vrooli scenario requirements manual-log <req> --status passed --notes ... --artifact docs/...`). The manifest should capture `requirement_id`, `validated_at`, `validated_by`, optional artifact paths, and an `expires_at` timestamp. Auto-sync consumes this manifest so `type: manual` validations have structured evidence that can be diffed later. Drift detection can then flag manual validations as stale whenever `now > expires_at` or when the manifest lacks an entry for the current requirement status. Include a pruning pass that removes manifest entries for requirements that no longer exist so deleted IDs do not linger indefinitely.

2. **Embed Source-Aware Metadata in Requirement Files**
   - Extend `syncRequirementFile` so each requirement gains `_sync_metadata` with:
     - `last_updated` + `updated_by` ("auto-sync"),
     - `phase_results` (array of relative JSON paths for every contributing phase, lighthouse, or manual log entry),
     - `tests_run` (commands captured from the suite runner; include multiple entries if different presets validate different requirements),
     - Validation-type hints such as `workflow_id` for BAS entries or `lighthouse_page_id` for Lighthouse checks.
   - Rewrite requirement/validation objects when syncing so `_sync_metadata` values are persisted to disk (not just kept in memory), and add regression tests that run parse → sync → parse loops to ensure the provenance data survives round trips.
   - For manual validations, also persist the manifest reference + `expires_at` so downstream tooling knows when to demand a new run.
   - Derive `last_updated` from the newest contributing evidence timestamp (not "now") and skip file writes entirely when no requirement actually changed so that a no-op `report.js --mode sync` does not rewrite the entire registry.
   - **Parser & schema preservation**: Update `scripts/requirements/lib/parser.js`, `scripts/requirements/schema.json`, and the validator so `_sync_metadata` (plus future provenance fields) survive parse/sync cycles. Add regression tests proving metadata survives parse → sync → parse loops.

3. **Persist a Scenario-Level Sync Snapshot**
 - After `report.js --mode sync`, write `coverage/<scenario>/requirements-sync/latest.json` containing:
   - Sync timestamp and the command(s) captured directly inside `testing::suite::run` (so ad-hoc invocations, CI presets, and custom wrappers still log accurate evidence),
   - A recorded run-manifest entry (timestamp + resolved preset/command string) emitted by the test runner before `report.js --mode sync` executes, so future drift detectors can detect when someone runs sync without re-running tests,
     - Summary metrics (`enrichment.computeSummary` output),
     - Hash/mtime for each `requirements/**/*.json` module,
     - Per-requirement mirrors of `_sync_metadata`, including manual manifest references and Lighthouse scores,
     - Operational-target rollups derived from `prd_ref` + the numbered folders so we can compare against PRD checkboxes deterministically.

4. **Drift Detection in `scenario status`**
 - Enhance `scenario::requirements::quick_check` to load the snapshot and:
     - Compare requirement hashes vs. the snapshot to catch unsynced edits, using canonicalized content (ignore volatile fields such as timestamps/`updated_by`) so noise doesn’t create false drift warnings.
    - Compare the snapshot timestamp with the newest artifact among `coverage/phase-results/*.json`, `ui/coverage/vitest-requirements.json`, Lighthouse reports, or manual-manifest entries, and warn when evidence is older than the requirement files.
    - Compare operational-target rollups to `PRD.md` checklists—parse the PRD through the same routines already implemented in `scenarios/prd-control-tower` so drift warnings match the system-wide validator.
    - When discrepancies are found, provide the exact remediation (`run tests + report.js --mode sync`) so agents don’t try to “fix” drift by editing JSON manually.

5. **Docs + Prompts + Helper Commands**
   - Update improver/generator prompts, requirements README, and the new manual-validation CLI docs to remind agents: “Run the full suite + `scripts/requirements/report.js --mode sync`; never toggle requirement/PRD status manually.” Document that manual validations are a last resort and link to BAS workflow guidance for replacing them.
   - Provide helper commands:
     - `vrooli scenario requirements snapshot <name>` – prints the stored sync summary and highlights stale manual validations.
     - `vrooli scenario requirements manual-log` – appends to the manifest and optionally uploads artifacts/screenshots required by PRD targets.
   - Teach artifact cleanup scripts (e.g., `testing::artifacts::cleanup` and `testing::artifacts::finalize_workspace`) to treat `coverage/<scenario>/requirements-sync/` as reserved so the snapshot isn’t deleted during log rotation/compression.

## Implementation Steps
1. **Evidence Loader Upgrades**
   - Extend `scripts/requirements/lib/evidence.js` so it ingests three sources **and tags each requirement evidence entry with the relative JSON file that produced it** (so `_sync_metadata.phase_results` can reference concrete artifacts):
    1. Existing `coverage/phase-results/*.json` (tests + automation) with relative file tracking **recorded on every evidence record** so `_sync_metadata.phase_results` can reference the precise artifact.
    2. `coverage/phase-results/lighthouse.json` (Lighthouse runner) with page/score metadata.
    3. `ui/coverage/vitest-requirements.json` ingested as part of the unit phase (instead of special handling) so UI evidence follows the same pipeline as other validations, and evidence rollups are keyed by `validation.type` rather than bespoke per-runner logic.
    4. A new manual validation manifest (`coverage/manual-validations/log.jsonl`) produced by the CLI. Keep the default location consistent with current behavior but expose an env override for advanced workflows.
   - Update the requirement schema/docs to describe the manifest reference + manual `expires_at` semantics.

2. **Manual Validation CLI + Manifest**
   - Build `vrooli scenario requirements manual-log` (shell) that appends JSON entries with `requirement_id`, `status`, `validated_at`, `validated_by`, `notes`, `artifact_path`, and computed `expires_at` (configurable default, e.g., 30 days).
   - Teach `report.js --mode sync` to merge manifest entries into `requirementEvidence`, mark manual validations stale when the log is missing or expired, and persist the manifest path in `_sync_metadata`. Include a pruning step so manifest rows referencing removed requirement IDs are automatically dropped.

3. **Schema & `_sync_metadata` Enhancements**
   - Update `scripts/requirements/lib/sync.js` to emit `_sync_metadata` (per requirement + per validation) with `last_updated`, `updated_by`, `phase_results`, `tests_run`, and validation-type specifics (workflow IDs, Lighthouse pages, manual manifest references).
   - Teach `scripts/requirements/lib/parser.js` to preserve existing `_sync_metadata` (and any future metadata fields) during parsing so repeated sync cycles don’t erase provenance.
   - Capture the executed test command(s) inside `testing::suite::run` (rather than only `test/run-tests.sh`) and pass it to `report.js` via env so `_sync_metadata.tests_run` stays accurate for CI, Makefiles, and direct invocations. Provide an explicit env/flag for `report.js --mode sync` (and `vrooli scenario requirements sync`) so manual invocations can still record the command list when tests were run out-of-band.

4. **Scenario Snapshot Writer + PRD Checkbox Sync**
 - Extend `report.js` sync mode to write `coverage/<scenario>/requirements-sync/latest.json` **only after a full, non-partial suite completes** (e.g., all phases required for sync ran). Preserve the list of executed phases + presets inside the snapshot so downstream tools can detect when the most recent run was partial.
     - Summary metrics + operational-target rollups (computed from `prd_ref` + folder layout),
     - Requirement hashes/mtimes for drift detection,
     - Aggregated manual-manifest state (including next required validation date).

  - While writing the snapshot, update `PRD.md` checkboxes whenever the linked operational target switches state: if all mapped requirements roll up to `complete`, flip `[ ]` → `[x]`; if evidence regresses, flip `[x]` back to `[ ]`. This keeps PRD status in lockstep with the registry and prevents manual edits from lingering.

5. **Drift Detection + Guardrails**
   - Enhance `scenario::requirements::quick_check` to load the snapshot, compare hashes, inspect artifact timestamps vs. snapshot timestamp, parse `PRD.md` checkboxes, and flag validation-type issues (expired manual entries, missing Lighthouse records, etc.). Extract the PRD checklist parser from `scenarios/prd-control-tower` into a reusable helper (e.g., `scripts/prd/parser.js`) so CLI commands and other scenarios consume a single implementation.
   - Document that manual validation manifests live under the scenario (no shared edits, since only one agent works on a scenario at a time) and highlight that concurrency handling is intentionally minimal.
   - Enforce PRD + `prd_ref` consistency (e.g., lint that every `OT-P*-###` checklist entry maps to at least one requirement) so drift detection has reliable data to compare.
   - Update prompts/docs to describe the workflow and surface the new helper commands.

## Deliverables
- `_sync_metadata` embedded per requirement with clear provenance.
- Scenario-level sync snapshot stored under `coverage/<scenario>/requirements-sync/latest.json`.
- `vrooli scenario status` shows operational-target drift warnings + exact remediation commands.
- Updated documentation, prompts, and optional lint to keep humans/agents on the paved path.
