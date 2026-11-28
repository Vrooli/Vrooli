# Testing — Experience Manager

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
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## TL;DR — the canonical examples

These files are the source of truth. When in doubt, copy their shape:

- **Validation API**: `api/handlers/validation/connect_handler_test.go`
  and `api/internal/reconcile/reconcile_test.go` — table-driven checks over
  real parser/reconcile behavior, including advisory-vs-strict gate semantics
  and the business-health Matrix calibration invariant.
- **Studio API**: `api/handlers/studio/connect_handler_test.go` — handler
  tests for repository guards, proto mapping, ListSpec/ListEvidence, render,
  compare, promote, and binding suggestion flows.
- **Ratchets**: `api/internal/spec/spec_test.go`,
  `api/internal/assessment/assessment_test.go`, and
  `api/internal/autofix/autofix_test.go` — registry/doc/finding-doc/test-genie
  parity, JSON Schema conformance, fix-rule disjointness, and BAS scaffold
  idempotency.
- **UI composition**: `ui/src/App.test.tsx` — smoke-only composition
  test. App composes shell + features; feature behaviour belongs beside
  the feature.
- **UI cockpit**: `ui/src/pages/ExperiencePages.test.tsx` — behavior tests
  for Fleet, Scenario Explorer, Evidence, Studio, and Findings live-data
  states. This is the canonical shape for page-level query/mutation tests.
- **UI a11y**: `ui/src/layout/AppShell.a11y.test.tsx` — shell
  accessibility is tested at its ownership boundary.
- **CLI**: `cli/app_test.go` — smoke gate (NewApp, --version, --help).
  When domain commands arrive, extend with `clitest.NewAPIServer` +
  `clitest.CaptureStdout` from `cli/internal/testutil/`.

If your test doesn't look like one of those three, ask why before
shipping.

## UI testing

### Canonical UI test pattern

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { AppShell } from "../layout/AppShell";
import { selectors } from "../../consts/selectors";

describe("AppShell rendering (cimode — copy-independent)", () => {
  afterEach(() => { cleanup(); });

  it("renders shell navigation via test ID", () => {
    renderWithProviders(<AppShell />);
    expect(screen.getByTestId(selectors.layout.nav)).toBeInTheDocument();
  });
});
```

### Two-layer test pattern (cimode + real locale)

`test-setup.ts` puts vitest into `cimode` before every test — `t('app.title')`
returns the literal key `"app.title"`. Most tests run there: assertions
go through `selectors.*` (test IDs) and `strings.*` (the typed key
registry), so they survive any wording change in any locale.

A second `describe` block opts into real locales with
`beforeEach(async () => { await setLocale("en"); })` and asserts on the
canonical English copy via raw `en.json` references. These tests *should*
update when canonical English copy changes — that's what they verify.

See `pages/ExperiencePages.test.tsx` for the full page-query pattern and
`i18n/locales.test.ts` for CLDR plural coverage. Keep `App.test.tsx`
smoke-only so deleting a feature does not require rewriting the app
composition test.

### Accessibility tests

Accessibility tests follow the same ownership rule as production UI:

- **Shell**: `components/AppShell.a11y.test.tsx` renders
  `<AppShell>` with stable placeholder children. It covers page layout,
  headings, locale controls, and shell-level semantics.
- **Feature**: `features/<name>/<Name>Card.a11y.test.tsx` renders the
  feature directly, owns its API mocks, waits for each state it scans,
  and calls `expectNoA11yViolations(container)`.
- **App**: `App.test.tsx` stays composition smoke. Do not make it the
  default a11y gate; a full-`App` a11y test couples shell coverage to
  every async feature query and becomes fragile as features change.

Before running axe, wait for the state the test owns. For query-backed pages,
wait for a stable selector or role after the mocked query settles. This keeps
React Query updates inside the awaited test boundary and prevents `act(...)`
warnings.

`test-setup.ts` fails tests that write unexpected `console.error` or
`console.warn` output. If a test intentionally exercises a noisy React
path, suppress that warning locally and assert the user-visible
contract. `ErrorBoundary.test.tsx` is the reference: it suppresses
React's intentional boundary logging while asserting `onError` and the
fallback UI.
