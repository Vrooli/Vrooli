# Current seven-phase readiness result

Test Genie run `20261021-005533-403eb81c` completed **FAIL**, 5/7 phases passed. `api`, `storage`, `experience`, `ui-health`, and `proto` passed; `contracts` and `unit` failed. The run was not repeated.

## Exact failure evidence

- `contracts`: cli-health reported two `binding.arg_unmapped` errors in `scenarios/personal-planner/cli/manifest.json`: `measure-unit` and `measure-basis` do not bind to `CreateGoalRequest`'s nested `measure.unit` and `measure.basis` fields. Eight `binding.bind_where_rename_suffices` warnings remain separate advisories.
- `unit`: Unit Health typed detail reports the API workspace's `go test` command failed in `handlers/calendar`. Test `TestCalendarRESTConnectAndWorkspaceShareRoutedTestPoolIdentityAndRevisions` at `api/handlers/calendar/connect_events_test.go:365` could not find `planning_profiles.default` in the routed test pool. The command output says diagnostics are partial; it identifies one failed test and no omitted failure events. CLI Go tests passed. This is consistent with the current actor-owned workspace contract needing its parity assertion to inspect the actor-owned profile while leaving legacy rows separate; the cause is a bounded inference pending the worker's focused correction and review.

## Producer evidence

- Catalog digest: `ac:sha256:26859fed84a9d68b0e8e19e5cbfdcb4e7fff5ea822e54fd68beb6fa531f6c0a3`.
- `findings.report` artifact `artifact_e2a438b6c22161d3b0ba778d8e7df615`: 730,535 bytes, SHA-256 `24617728e281c28e5cd3640a3fabfdbccd64ab2f67c38eac2823684d526a36f0`.
- `unit.log` artifact `artifact_b656e57beff6b69dfe9975d034db8fde`: 15,809 bytes, SHA-256 `5764b900179494ab4f0788a1b19d695cf86b85e614db7b99714cba714f77802d`.
- `contracts.log` artifact `artifact_0995351b4e95772fcee1ea2fd221ea9f`: 7,116 bytes, SHA-256 `9d43a77e92441edf457eac37763dc180e492bb3a43198fc8830e6b2c325a7e5b`.
- Unit `nativeDetail` was decoded with the committed `unit_health.v1.validation.ValidateScenarioResponse` type. Its Unit Health run ID is `uh-20261021-005603-c051e409a8ded7a34f918dbfbd3566f7`; the returned finding and command excerpt are summarized above. Raw producer artifacts remain with Test Genie.

The contract errors and routed-profile assertion are within the approved Planner CLI/actor-workspace scope. Passing maturity advisories are retained in the producer result and do not create new scope or acceptance.
