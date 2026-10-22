# Atomic workspace availability correction

Owner/run: personal-planner-delivery-orchestrator; exact parent `5d07e651-eaa6-48ec-89b4-5dc359059193`. Epoch: `scenarios/personal-planner/docs/internal/goal/epochs/E1.md`. Worker protocol: `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`.

## Trigger and outcome

The independent review task/run `5ca075aa-2381-4b2f-8500-e1f032466f21` / `dc76f6b3-cf94-48f6-a4ad-ed78c1aa4b6c` returned FAIL. Exact report: `scenarios/personal-planner/docs/internal/goal/receipts/e1-workspace-owner-isolation-20261021/independent-review.md`.

Make `ReplaceAvailability` atomic with its profile revision update. If a concurrent update invalidates the expected revision after the initial read, or any delete/insert/update fails, no availability rows may be partially replaced. Preserve the concurrent update and return the appropriate error.

## Authority and limits

- This is a same-scope correction under approved Planner I18 authenticated private-workspace ownership. Keep legacy ownerless rows untouched and inaccessible.
- Exact write set: existing `scenarios/personal-planner/api/internal/workspace/sqlite.go` and `sqlite_test.go` only.
- Do not change module/dependency files, profile/identity semantics, schema, Proto/generated files, APIs, UI/CLI, credentials, production data, or unrelated work.
- Estimate +60 runtime/+80 test lines. Current cumulative E1 runtime is +1,811; stop before exceeding +2,031 (hard ceiling +2,100) or if the fix needs wider scope.
- Preserve the prior FAIL receipt and all baseline/current manifests.

## Required behavior and checks

- Perform owner-scoped deletes, inserts, and revision compare-and-increment within one SQL transaction; rollback on every error and commit before reading the result.
- Cover a failure after deletion and at least one successful insert (for example a duplicate-key insert) and assert the prior owner availability and revision remain unchanged.
- Cover a revision change occurring after the initial profile read but before the guarded update; assert the replacement returns conflict and leaves prior availability intact while preserving the winning profile revision.
- Retain two-owner isolation, fail-closed missing/unverified principals, legacy invisibility/preservation, and existing optimistic revision guards.
- Capture admitted-path baseline hashes, run `cd scenarios/personal-planner/api && go test ./internal/workspace -count=1`, retain exact output and current hashes, append one truthful E1 slice-log line, and run `agent-manager effort epoch-check scenarios/personal-planner/docs/internal/goal/epochs/E1.md --runs <your exact run ID> --wake-key 5d07e651-eaa6-48ec-89b4-5dc359059193`. Do not mark E1 accepted.

Exit only with atomic replacement and focused regressions passing, or return the exact failing source/gate evidence without widening scope.
