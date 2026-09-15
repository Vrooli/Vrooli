# Test Genie baseline — 2026-07-26

This document records the immutable regression anchor captured before the
scenario-to-desktop maturity and proto-foundation campaign.

- Test Genie run: `20260726-154231-fc7b1fb3`
- Verdict: `FAIL`
- Completeness: `57/100`, classified `functional_incomplete` at rung `R0`
- Requirements: `11/34` passing
- Operational targets: `3/12` passing
- Evidence tier: `degraded`; business, docs, proto, structure, and unit used
  stale evidence.
- Unit baselines: API `56.2%` against `75%`; CLI `37.0%` against `75%`; UI
  `52.0%` against `85%`; runtime was not a measured surface.
- Known unit findings: `TEST_EXECUTION_FAILURE` and
  `TEST_FLAKE_SUSPECTED` for API and UI.

The immutable Git Control Tower collection
`scenario-to-desktop-maturity-and-proto-foundation-baseline` completed on
2026-07-26 with source fingerprint
`9a74ef3a220d7af87a58ce353e5d1003186bb710`.

## Phase 1 determinism evidence

- The initial uncached API series reproduced the known flake on run 3:
  `TestResumePipeline` and `TestResumePipelineWithStopAfter` timed out after
  two seconds even though their logs show both resumed pipelines completing
  immediately afterward.
- After replacing that arbitrary observation budget with the Go test deadline,
  all 20 uncached full API suite runs exited zero. Logs are retained at
  `/tmp/scenario-to-desktop-api-postfix-1.log` through
  `/tmp/scenario-to-desktop-api-postfix-20.log` for the active execution.
- All 20 uncached full CLI suite runs exited zero. Logs are retained at
  `/tmp/scenario-to-desktop-cli-test-1.log` through
  `/tmp/scenario-to-desktop-cli-test-20.log` for the active execution.
- All 20 uncached full runtime suite runs exited zero. Logs are retained at
  `/tmp/scenario-to-desktop-runtime-test-1.log` through
  `/tmp/scenario-to-desktop-runtime-test-20.log` for the active execution.
- All 20 UI coverage runs exited `1` for the same declared global 85% coverage
  floor. The test execution itself did not flip; this deterministic coverage
  gate remains Phase 13 work. Logs are retained at
  `/tmp/scenario-to-desktop-ui-coverage-1.log` through
  `/tmp/scenario-to-desktop-ui-coverage-20.log` for the active execution.
