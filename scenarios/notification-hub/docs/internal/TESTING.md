# Testing — Notification Hub

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

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/api/health.test.ts](../../ui/src/api/health.test.ts)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## Settings and Reach journeys (BAS)

The cases under `bas/cases/e21`, `e22` and `e23` run in the Hub's routed test
mode (`?vrooli_test_mode=1`). They use no fixture modes. The leased test pool
starts with the fixed test recipient and one declared preference source
(`seedTestPreferenceCatalog` in `api/main.go`); a case that needs devices,
channels, quiet hours or an escalation order sets them up through the Hub's own
API with the `bas/actions/seed-owner.json` subflow (`profile` is `empty` or
`reach`), then checks the result with a read from the same API.

Failure paths (a read that fails after a write, a partial save, an unclear
response) are tested in `ui/src/features/settings` and `ui/src/features/reach`
with the in-memory hub in `features/settings/mocks/fakeHub.ts`, which can fail
or lose any call. The server has no fault mode for them.

Overnight quiet-window rules are tested at the service in
`api/internal/hub/quiet_internal_test.go` (a table of weekday and clock cases)
and `quiet_test.go` (what is held, what is released, critical override).
