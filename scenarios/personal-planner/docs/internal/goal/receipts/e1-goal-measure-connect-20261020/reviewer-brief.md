# Independent review — Planner I18 E1 typed measurable-goal Connect projection

You are the separate read-only acceptance reviewer. Do not edit files, run tests, run generation, or accept E1/Q1. Inspect the exact source and durable evidence listed here. Put the complete verdict, evidence, limits and exact identities in your native final output so the parent can materialize it verbatim.

## Authority and worker identity

Approved outcome: the Planner Goals Connect contract round-trips the already approved typed Goal measure and target date, including explicit unknown progress for measured goals and compatibility for generic unmeasured goals. See the current E1 slice in `scenarios/personal-planner/docs/internal/goal/epochs/E1.md` and accepted contract at `scenarios/personal-planner/docs/internal/goal/CONTRACT.md` §2.

Worker task `89c2cf7a-4ac5-49f1-a6ee-83d956ef215f`, run `7d020d59-2b3a-4a03-9503-4d7975d65f09`, parent `5d07e651-eaa6-48ec-89b4-5dc359059193`, registered profile `prompt-manager/delivery-epoch-worker` (`4c19aef1-36a2-4cc5-8961-42648692883b`), canonical `gpt-6-luna`, medium. Native run result says the task completed. Earlier no-write task/run `3a3e8210-8abf-49f5-bdc9-34be8de2b03e` / `a3d09f56-9b11-4a3f-87aa-a6ba7cf87725` is retained separately and is not the implementation run.

## Required behavior and code scope

Review only these admitted authored paths and their generated Goal owner closure:
- `packages/proto/schemas/personal-planner/v1/goals/goals.proto`
- `scenarios/personal-planner/api/handlers/goals/connect_handler.go`
- `scenarios/personal-planner/api/handlers/goals/adapter.go`
- `scenarios/personal-planner/api/handlers/goals/connect_handler_test.go`
- Captured Planner Proto Goal generated outputs, Planner lock, shared artifact pointer and descriptor in the manifests below.

Verify additive wire compatibility, mapping through existing goal service validation/persistence, measure and target-date readback, explicit unknown measured progress (not fabricated zero), and generic unmeasured-goal compatibility. The focused test should cover one-time USD and recurring monthly USD forms, date, basis/recurrence, unknown progress, and generic goal behavior. No UI/CLI, income-observation, product-data, or E1/Q1 claim is in scope.

## Immutable evidence

- Original baseline: `receipts/e1-goal-measure-connect-20261020/baseline-original/manifest.json`, SHA-256 `00a4a9257ca0055dbbd83c6f3c5af2d42b7733da7e7606f9894e7d3f53467955`.
- Refreshed exact pre-write baseline: `receipts/e1-goal-measure-connect-20261020/baseline-retry/manifest.json`, SHA-256 `c3e27213cad8b93b1e83541623119628a4272af64f01ccca8c33362ff96b11a2`.
- Worker current manifest: `receipts/e1-goal-measure-connect-20261020/current-manifest.json`, SHA-256 `1aeea0cdb22cb6703b5e876439c0ce48b69b2173ed920702305b84d7598394fe`.
- Worker logs: `proto-generate.log`, `proto-verify.log`, `go-test.log`, `diff-check.log`, `epoch-check.log`; handoff `handoff.md`.
- Prior no-write report and exact recovery prompt: `retry-worker-brief.md`.

Worker retained these results: scoped command reported `protogen: publishing scenarios: personal-planner`; `make verify-committed-gen` exited 0; `go test -count=1 ./internal/goals ./handlers/goals` exited 0; scoped `git diff --check` exited 0; worker epoch-check exited 0 with +1,731 runtime and +2,225 test lines versus epoch baseline. Do not infer from these alone that the out-of-scope generated tree was preserved.

## Important unresolved scope evidence

`generation-scope-observation.txt` records 106 generated status entries under other Proto scenarios after the scoped generation command, with mtimes at `2026-10-20T23:00:50Z`. Those paths were not included in the exact retry baseline, so their prior bytes and whether each was already dirty are unverified. The subsequent Proto verify log says it waited 11.335 seconds for generation lock held by PID `2758766`, so concurrent owner generation is a plausible alternative cause. The parent has not restored any output. A single unknown-signal bug report was filed as `knw-1792537585742220131`; this is an observation, not a finding of causality.

Review whether the bounded Planner implementation itself meets its slice gate and state whether the 106-path scope uncertainty prevents bounded acceptance. Separate a product-behavior verdict from the unresolved unrelated-output preservation risk. If evidence does not establish safety, say so and return a non-acceptance verdict or clear blocking condition. Do not claim E1/Q1 acceptance.

In your native final output include: your reviewer task/run/profile/parent identities; worker task/run/profile/parent/status; source and manifest hashes you verified; the exact retained validation evidence and whether you reran it (do not rerun); verdict; specific limits; and whether the unrelated generated scope issue blocks this bounded slice.
