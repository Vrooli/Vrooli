# Testing — Vrooli Bridge

How to write tests against this scenario's shape. Read this *before*
your first non-trivial test — the patterns below are load-bearing for
the gates documented in [`SEAMS.md`](SEAMS.md), in `eslint.config.js`,
and in `.github/workflows/test.yml`.

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
- [ui/src/features/health/HealthCard.test.tsx](../../ui/src/features/health/HealthCard.test.tsx)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## Multi-node soak

The acceptance soak is owned by [`scripts/soak.sh`](../../scripts/soak.sh). It
dispatches typed jobs to the Linux and macOS node IDs, restarts the control
plane only through `vrooli scenario restart`, injects randomly selected agent
kills and bounded network partitions, and continuously queries the durable
SQLite run table. The invariant is fail-closed: no queued or running run may
remain past `timeout_seconds + grace`.

Run it only after the operator has supplied both real node IDs, SSH hosts,
agent-kill commands, and partition enter/restore hooks. The default window is
24 hours; `--dry-run` validates the shape without touching a host. The script
writes the window, fault count, late-run count, fault log, and terminal-state
distribution to `docs/internal/SOAK-REPORT.md`. A dry run or a partial window
is evidence that the harness is configured, not evidence that the acceptance
invariant passed.

The shape is mature on purpose: every pattern below was already needed
in workspace-sandbox and got there by accumulating bugs. Starting here
means inheriting those lessons without repeating them.

## TL;DR — the canonical examples

These files are the source of truth. When in doubt, copy their shape:

- **API**: `api/handlers/health/handler_test.go` — table-driven, real
  middleware via `httpx.NewLiveServer`, fake pinger from `mocks/`,
  typed-proto decode via
  `assertx.MustUnmarshalProto[healthv1.Response]` (the wire shape lives
  in `packages/proto/schemas/vrooli-bridge/v1/health/health.proto`;
  assert on typed proto fields, not `map[string]any` chains). For
  endpoints whose wire shape isn't in proto yet, `MustDecodeJSON[T]`
  is the fallback — but adding the proto first is the right move.
- **UI composition**: `ui/src/App.test.tsx` — smoke-only composition
  test. App composes shell + features; feature behaviour belongs beside
  the feature.
- **UI feature**: `ui/src/features/health/HealthCard.test.tsx` —
  `renderWithProviders`, factory data, inline `vi.mock` factory
  closure, cimode assertions, and real-locale assertions.
- **UI a11y**: `ui/src/components/AppShell.a11y.test.tsx` and
  `ui/src/features/health/HealthCard.a11y.test.tsx` — shell and feature
  accessibility are tested at their ownership boundary.
- **CLI**: `cli/app_test.go` — smoke gate (NewApp, --version, --help).
  When domain commands arrive, extend with `clitest.NewAPIServer` +
  `clitest.CaptureStdout` from `cli/internal/testutil/`.

If your test doesn't look like one of those three, ask why before
shipping.
