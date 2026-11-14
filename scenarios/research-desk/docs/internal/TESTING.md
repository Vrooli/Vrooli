# Testing — Research Desk

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

- **Fixtures.** `api/internal/testutil/home.txtar` holds small excerpts of real
  research-home layouts (one per generation), home path as a placeholder;
  `testutil.NewHome(t)` expands it into a temp folder. No test reads or writes
  `~/.vrooli/research`.
- **Real home.** `make real-home-check` (build tag `realhome`) walks the operator's home
  read-only and asserts invariants: every folder listed, every state in the enum, every
  derived value with a source or a reason, no file unreadable, nothing changed, and turning
  every courier switch on sends no item already there. It skips loudly without a home.
- **Peers.** Web Console, Vrooli Events and Notification Hub are fakes behind
  `launch.Terminal`, `courier.Hub` and `capabilities` port resolvers (`SEAMS.md`).
- **Requirements.** `requirements/01-must-ship/module.json` names the test that proves each.

## Binary startup

`api/main_e2e_test.go` uses `api-core/boottest` to build this API, verify its
health identity in isolated storage, and check shutdown. Keep the expected
service name in the scenario test. Shared process machinery and failure
regressions live in `packages/api-core/boottest`; see its package README section
for configuration and evidence limits. The existing E2E gate runs this test.
