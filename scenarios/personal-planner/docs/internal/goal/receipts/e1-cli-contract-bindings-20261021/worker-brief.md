# E1 bounded repair — Planner CLI manifest argument bindings

- **Outcome:** resolve the current cli-health findings preventing clean command discovery for the approved Planner CLI surface.
- **Authority:** approved E1 journey requires supported CLI parity; current-source Test Genie contracts result and a fresh read-only `cli-health validate scenario personal-planner` both report two errors on goal measure flags and eight warnings for unnecessary explicit binds.
- **Current directive:** name feedback `PP-SDA-MODULE-FRESHNESS-OWNER-ROUTE-20261021` in the worker handoff. Its supported SDA apply is already reconciled; do not use raw Go dependency commands or alter dependencies in this slice.
- **Exact write set:** `scenarios/personal-planner/cli/manifest.json` only. Pre-dispatch SHA-256 `a0ec06115f2f55b18f53c17dcc92d695077932aeff9e5324f581f1a395efaae8` (777 lines); GCT anchor `gcta-bf111f3dbd4d`, checkout head `52097258290d1bff0191319242d4948e214af9e3`. Re-capture exact manifest through the worker baseline route before writing; preserve every other dirty source file and byte.
- **Finding scope:** the two errors are flags `measure-unit`→nested `CreateGoalRequest.measure.unit` and `measure-basis`→`measure.basis`. The eight warnings are redundant binds on kebab-case flags `definition-of-done`, `promised-boundary` (create/revise), `scope-exclusions` (create/revise), `acknowledgment-status`, `local-date`, and `horizon-days`.
- **Required change:** retain existing public kebab-case flags. Resolve the nested measurement fields with the smallest supported manifest binding after inspecting CLI Health's argument resolver and manifest docs; do not change Proto, generated code, runtime CLI code, or flag spellings. Remove only the eight explicit binds CLI Health identifies as redundant, allowing normal argument-name resolution. Preserve all other manifest semantics.
- **Estimate:** 0 runtime lines, 0 test lines, one manifest-only work unit. Stop on any need to widen paths or alter Proto/generated/API code.
- **Gate:** run `cli-health validate scenario personal-planner` once after the edit and retain raw output plus exit code. Confirm both measure fields resolve and no targeted `binding.arg_unmapped` or `binding.bind_where_rename_suffices` findings remain. Do not run the Test Genie seven-phase suite in this slice.
- **Non-goals:** no new CLI behavior, API/proto changes, Test Genie config/threshold changes, product data writes, provider calls, legacy-record mapping, or E1/Q1 acceptance.
- **Handoff:** preserve before/after manifest hashes, exact line delta, focused CLI-health output, and remaining advisory findings. Run worker epoch-check against the complete E1 file with its exact run ID and parent.

## Worker

Parent `5d07e651-eaa6-48ec-89b4-5dc359059193`; read `scenarios/personal-planner/docs/internal/goal/epochs/E1.md` and `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`; use the epoch's last 20 slice-log lines and the cited CLI Health resolver/manifest docs as the resume set. Registered delivery profile `4c19aef1-36a2-4cc5-8961-42648692883b`; model `gpt-6-luna`, medium; workload `planner-I18/E1`.
