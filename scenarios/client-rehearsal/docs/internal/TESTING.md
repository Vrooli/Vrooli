# Testing — Client Workflow Rehearsal

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

Record the behaviors this scenario must prove, deterministic fixtures, required
external services, and any exceptions to the shared harness. Link to real local
tests as they are added. Keep execution receipts with their owning plan.

## Binary startup

`api/main_e2e_test.go` uses `api-core/boottest` to build this API, verify its
health identity in isolated storage, and check shutdown. Keep the expected
service name in the scenario test. Shared process machinery and failure
regressions live in `packages/api-core/boottest`; see its package README section
for configuration and evidence limits. The existing E2E gate runs this test.

## Rehearsal domain tests

`api/internal/rehearsals/` owns focused unit tests for n8n import normalization, credential-shaped value removal, typed OpenAPI/recorded-exchange simulation, Sandbox profile export, seeded fault recovery, and explicit operator review. Test data must be fictional and must never contact a workflow target. The import bound is 512 KiB. Synthetic cross-fixture tests cover graph branches, Switch ports, multi-trigger selection, subworkflow/error-workflow execution, exact-host OpenAPI/recorded-exchange HTTP matching, and the recorded-exchange-to-Workspace-Sandbox profile contract. Sanitized public workflow exports are imported and run by the local generic interpreter in explicit test cases; those are simulator runs, not native n8n import or execution. Credential-bearing built-in app nodes, Code/native protocol nodes, large-payload streaming, and native n8n runtime execution remain open; do not infer coverage from local adapter tests.
