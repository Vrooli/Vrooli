# Performance — UI Health

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| UI build | 5-10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |
| Lighthouse accessibility | ≥95 per route | `npx lighthouse <route>` (manual) | target |
| Lighthouse best-practices | ≥90 per route | `npx lighthouse <route>` (manual) | target |
| axe-core violations | 0 on every page test | `pnpm test -- --run *.a11y` | enforced |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet. | n/a | n/a | 2026-05-20 |

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Performance budgets for real product workflows must be defined after
  domains and UX flows are known.

## Regression Procedure

1. Run `make test`.
2. Capture relevant API/UI command timing.
3. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
4. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Accessibility & Responsive Verification

The plan in `UI_REBUILD_PLAN.md` calls for Lighthouse a11y ≥95 /
best-practices ≥90 on every route, plus a mobile-viewport sweep at
360×640.

**Static (already automated):**

- Every page has an `*.a11y.test.tsx` running `axe-core` (run via
  `pnpm test -- --run *.a11y`).
- Selector registry in `ui/src/consts/selectors.ts` keeps every
  data-testid typed and parameter-checked.
- Filter chips on Search / Inventory / Validation expand to
  `min-h-touch` on mobile (`md:min-h-0` shrinks back on desktop) so
  they meet the 44px tap-target rule on phones without bulking up the
  desktop layout.

**Manual capture (do this before any release):**

1. `vrooli scenario start ui-health`
2. Open the UI in a browser. For each of `/`, `/validation`,
   `/search`, `/inventory`, `/reindex`, `/settings`, run:
   `npx lighthouse <url> --only-categories=accessibility,best-practices --view`
3. Record results in the table below.
4. Resize to 360×640 and walk every page. Note any overflowing text,
   missing hit targets, or focus-trap escapes in `PROBLEMS.md`.

### Lighthouse capture log

| Route | Accessibility | Best-practices | Date | Source |
|---|---|---|---|---|
| (not captured) | n/a | n/a | 2026-05-20 | pending manual run |

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
