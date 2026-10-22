# Bounded Planner API lifecycle module readiness

Owner/run: personal-planner-delivery-orchestrator; exact parent run `5d07e651-eaa6-48ec-89b4-5dc359059193`.
Epoch: `scenarios/personal-planner/docs/internal/goal/epochs/E1.md`; read the `## Bounded validation — post-fix seven-phase E1 gate` result and the last 20 slice-log entries. Worker protocol: `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`.

## Outcome

Resolve the exact Planner API lifecycle build blocker from Test Genie run `20261021-003312-ed22a0ca`: `go: updates to go.mod needed; to update it: go mod tidy`. This is an enabling, bounded Planner I18 task so the already-approved seven-phase validation and remaining UI/CLI/Connect actor/workspace journey can proceed. It does not change product behavior, legacy mapping, or accepted contract scope.

## Authority and boundaries

- Use the existing `scenario-dependency-analyzer` owner route for all dependency changes (`docs/package-governance.md`). Never run `go mod tidy`, `go get`, another raw package manager, or edit approved dependency registries.
- Allowed source writes: `scenarios/personal-planner/api/go.mod` and `scenarios/personal-planner/api/go.sum` only, and only when an exact missing/incorrect Go requirement is identified and the SDA dry-run describes that same correction. If SDA cannot represent the needed operation, make no source edits and return the exact owner limitation.
- No API/product/runtime/test/generated source edits, no provider calls, no user-data writes, no changes to legacy rows, OAuth/security, gates, profiles or permissions. Do not repeat Test Genie or the seven-phase suite.
- Preserve the failed run and its receipt. Do not retry lifecycle startup until the exact module correction is independently recorded and the owner command succeeds.

## Evidence and self-validation

1. Read the retained Test Genie and lifecycle setup receipt under this directory's sibling `../e1-postfix-sevenphase-20261021/`; verify the current `go.mod`/`go.sum` identities before any proposed change.
2. Diagnose the exact Go module delta through non-mutating inspection and the SDA dry-run route. If no exact dependency change is justified, stop without editing.
3. For any justified change, apply only the SDA-approved command, retain its full stdout/stderr and exit code, then run the narrowest owner-managed Planner API build/setup check that does not start Test Genie or write product data. Do not claim seven-phase readiness unless that owner check succeeds.
4. Record before/after hashes, the exact SDA command and result, focused validation result, and remaining seven-phase gate in a receipt under this directory. Preserve concurrent changes.
5. Append a truthful slice-log line and run `agent-manager effort epoch-check scenarios/personal-planner/docs/internal/goal/epochs/E1.md --runs <your exact run ID> --wake-key 5d07e651-eaa6-48ec-89b4-5dc359059193`.

## Estimate and exit

Estimate: 3 work units; expected source growth 0 runtime / 0 test lines (module metadata only). Exit metric: either the Planner API lifecycle build succeeds after an exact SDA-governed module correction, or the precise SDA/owner limitation is retained with zero source changes. This bounded task does not rerun or accept the seven-phase gate, the original independent judgment, or E1/Q1.
