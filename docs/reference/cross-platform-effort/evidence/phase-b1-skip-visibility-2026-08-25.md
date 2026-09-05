# Phase B1 — Skip visibility and budgets (validation pending)

Date: 2026-08-25

## Implemented

- Added `repocontracttest.SkipPlatform` and `SkipPlatformf`. The helper records
  a JSONL object containing `platform`, `arch`, `test`, and `reason` before
  calling `t.Skip`.
- Replaced all 155 measured platform-gated skip calls with the shared helper,
  including the 61 generated scenario e2e gates and the formerly isolated
  `platform-go` and Bridge-agent modules. The latter now declare the local
  `repo-contract-go` test-support module through their own governed Go module
  files.
- Added `.vrooli/skip-budgets.json` with the measured baseline of 155
  platform-gated skips for Linux, macOS, and Windows. The budget policy is
  ratchet-down and an over-budget run cannot qualify as evidence.
- Updated test-genie’s scenario runner to inject the JSONL record path, expose
  `executed`, `skipped`, `total`, `budget`, and `withinBudget`, and reject a run
  whose recorded platform skips exceed the checked-in budget.

## Validation

Passed:

```text
(cd packages/repo-contract-go && go test ./repocontracttest)
(cd scenarios/test-genie/api && go test ./internal/scenarios ./internal/app/httpserver)
go test ./internal/dockerhost ./internal/hostreqkit
go test ./internal/process ./internal/capacity ./internal/resources/manifest ./internal/hostreq ./internal/hostreqcheck ./internal/hostreqrun ./internal/hostreqspec ./internal/app/hygiene ./internal/safeguards/model-policy-drift ./internal/app/contract ./internal/api ./tools/symbolset
(cd packages/platform-go && go test ./...)
(cd scenarios/vrooli-bridge/agent && go test ./internal/...)
go test ./internal/accel ./internal/baselinefloor ./internal/buildinfo ./internal/cliinstall ./internal/cli/scenariocli ./internal/lifecycle -run 'Test(StartWritesOperationRecord|FailedStartWritesFailedOperationRecord|BuildListPortsFallsBackToEnvironment)$' -count=1
```

The test-genie runner has a focused test proving one skip at budget passes and
the same run fails after the budget is lowered below the measured count. A
repository scan now reports 155 `SkipPlatform`/`SkipPlatformf` calls and no
remaining platform-guarded direct `t.Skip` in the measured set.

A server-owned Test Genie run was also attempted after the internal suite
turned green:

```text
scenario: scenario-dependency-analyzer
run: 20260825-162129-9fcff82a
result: FAIL (17 passed, 4 failed, 5m45s)
portability: passed
structure: passed
```

The failed phases were `quality`, `storage`, `workflow`, and
`provider-conformance`. The failures are existing scenario-provider debt: the
quality phase reports a skipped TypeScript type-check, storage reports direct
`*sql.DB` capture and an undeclared retention receipt path, and workflow cannot
resolve the scenario port while also lacking routed test isolation. The run is
therefore retained as non-comparable B1 evidence; it does not justify raising
the skip budget or claiming a green collection.

Plan Manager validation operation
`16cf8680-56f7-4a67-9c2a-19a03d7645b2` ran fresh. Its producer collection diff
returned `not-comparable`: the required React baseline failed on unrelated CLI,
component-test, and unit-health debt; the structure-health member was
provider-unavailable in the diff. A generation-3 baseline recovery reproduced
the React failure with 17/24 phases passing and 7 failing. This is recorded as
validation evidence, not a B1 pass.

The collection handoff path was then hardened in `git-control-tower`: successful
Test Genie reachability probes are shared for 30 seconds during fan-out, failed
probes are never cached, and collection dispatch is detached from the initiating
request with a finite 15-minute server-owned ceiling. The focused baseline
package now passes `go test ./internal/baseline` and includes a regression test
for the cache and retry behavior. A later collection attempt reached all late
members but still ended with provider-unavailable and pre-existing scenario
results; it remains non-comparable evidence rather than a B1 pass.

## Open boundary

The previously failing full internal suite was rerun after the focused
portability repairs and now passes:

```text
go test ./internal/...
```

This closes the Go/internal validation boundary. B1 remains active because the
server-owned Test Genie suite still needs a comparable run with its emitted
per-platform counts and skip-budget verdict recorded as durable evidence. The
budget is still 155 platform-gated skips per host OS and has not been raised.
