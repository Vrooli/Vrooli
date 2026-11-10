# Testing — Device Control

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Gherkin specifications

User journeys are specified as `.feature` files in `api/internal/control/features/` and run by
`TestFeatures` (`go test ./internal/control -run TestFeatures -v`). The runner is
`api/internal/testutil/gherkin` (Feature, Background, Scenario, Scenario Outline with Examples,
regex steps); each scenario gets a fresh world. Add a scenario when behavior reads as
"given a device state, when the owner acts, then the outcome and next step are X". Keep Go tests for
pure functions and fixtures too rich for plain steps. Program contracts have Python tests in
`.vrooli/program-runtime/tests/test_workflows.py` (`python3 -I`).

## Scenario-specific testing

### Desktop acceptance matrix

| Layer | Required proof |
|---|---|
| Unit | readiness mapping, lease epochs, stale input refusal, clipboard bounds |
| Contract | Bridge signaling/revocation and Web Console message compatibility |
| Companion | ScreenCaptureKit, CGEvent, clipboard, permission/no-Aqua diagnostics |
| Live macOS | minimouse one-display stream, pointer/keyboard, clipboard, reconnect, revoke |
| Fault/security | permission removal, GUI logout, ICE failure, stale controller, viewer write refusal |

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)
