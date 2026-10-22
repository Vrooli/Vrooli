# Planner I18 E1 Connect measurable-goal projection handoff

Baseline: `baseline-retry/manifest.json`, SHA-256 `c3e27213cad8b93b1e83541623119628a4272af64f01ccca8c33362ff96b11a2` (14 admitted files/absence entries, all matched live bytes before the first write). Original baseline preserved unchanged: `baseline-original/manifest.json`, SHA-256 `00a4a9257ca0055dbbd83c6f3c5af2d42b7733da7e7606f9894e7d3f53467955`. Current identities: `current-manifest.json`, SHA-256 `1aeea0cdb22cb6703b5e876439c0ce48b69b2173ed920702305b84d7598394fe`.

## Authored source paths and line inventory

- packages/proto/schemas/personal-planner/v1/goals/goals.proto: sha256=068e0db0e0e8c7b807bb9528f4e80f42c006b2fec456d7293546396b313c385b, lines=93
- scenarios/personal-planner/api/handlers/goals/connect_handler.go: sha256=8d130619830ee747a454dd37ee07769c19ecf9c76786c9cd43d25665cc414077, lines=89
- scenarios/personal-planner/api/handlers/goals/adapter.go: sha256=a5f2372ffa8cf97ae79d05fff79e29f3120e5ebe0ff27e1477d4a8f87f78e8d8, lines=17
- scenarios/personal-planner/api/handlers/goals/connect_handler_test.go: sha256=926def89c2e3b368acaf113591561b5ea8a5c3a1f5016972068af8835af51fc3, lines=80

Net authored runtime delta is +25 lines across the schema and two handlers; the new focused test adds 80 lines. `module.go` context remains byte-identical.

## Generated Proto owner closure

Changed by `make generate SCENARIO=personal-planner`: `packages/proto/gen/.vrooli-proto-artifact.json`, `packages/proto/gen/descriptor/image.binpb`, `packages/proto/gen/go/personal-planner/v1/goals/goals.pb.go`, `packages/proto/gen/manifests/personal-planner.lock.json`, `packages/proto/gen/python/personal_planner/v1/goals/goals_pb2.py`, `packages/proto/gen/python/personal_planner/v1/goals/goals_pb2.pyi`, and `packages/proto/gen/typescript/personal-planner/v1/goals/goals_pb.ts`. The Connect stub and Python `py.typed` marker stayed byte-identical. Post-generation scope audit found 106 generated file/status entries for other Proto scenarios; these are outside the admitted write set and were not in the exact retry snapshot. Current paths, modification times and hashes are retained in `generation-scope-observation.txt`. Their prior bytes are unverified, so no restoration was attempted.

## Validation

`cd packages/proto && make generate SCENARIO=personal-planner` passed. `cd packages/proto && make verify-committed-gen` passed before consumer validation. `cd scenarios/personal-planner/api && go test -count=1 ./internal/goals ./handlers/goals` passed. Scoped `git diff --check` passed. Exact command outputs are retained in `proto-generate.log`, `proto-verify.log`, `go-test.log`, `diff-check.log`, and `epoch-check.log`. The epoch check exited 0 with runtime +1,731 and test +2,225 against the epoch baseline; it also reports a wall-clock review signal and legacy malformed historical slice-log entries.

The focused Connect regression creates and lists one-time USD and recurring monthly USD goals and checks target, basis, recurrence, date and unknown progress; it also confirms generic goals retain absent measure/date and numeric zero progress. No income observations, UI/CLI changes or broader journey are claimed. Independent review, parent mechanical acceptance and the full E1/Q1 gates remain open. Because the owner command touched out-of-scope generated outputs whose pre-run bytes and prior dirty state were not captured, this worker stops without further product edits; that scope side effect remains unresolved.
