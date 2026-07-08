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
| `ui-health validate scenario ui-health --static-only --json` | 1.16s command wall-clock; metrics: `static-interop` 38ms / 34 findings, `static-freshness` 354ms / 0 findings, `runtime-render` skipped by static-only | Phase 1 baseline command output | 2026-07-07 |
| `ui-health validate scenario ui-health --json` | 3.63s command wall-clock; metrics: `static-interop` 31ms / 34 findings, `static-freshness` 335ms / 0 findings, `runtime-render` 2535ms / 2 findings | Phase 1 baseline command output | 2026-07-07 |
| `ui-health validate scenario swarm-manager --static-only --json` | 6.81s command wall-clock; metrics: `static-interop` 743ms / 133 findings, `static-freshness` 475ms / 1 finding, `runtime-render` skipped by static-only; command exits nonzero because the baseline contains one error finding | Phase 1 baseline command output | 2026-07-07 |
| `ui-health validate scenario swarm-manager --json` | 11.06s command wall-clock; metrics: `static-interop` 704ms / 133 findings, `static-freshness` 464ms / 1 finding, `runtime-render` 3523ms / 2 findings; command exits nonzero because the baseline contains one error finding | Phase 1 baseline command output | 2026-07-07 |
| `ui-health validate scenario ui-health --static-only --json` | 0.99s command wall-clock; findings unchanged by severity class from the baseline static pass | Phase 8 command output | 2026-07-08 |
| `ui-health validate scenario swarm-manager --static-only --json` | 5.65s command wall-clock; metrics: `static-interop` 599ms / 137 findings, `static-freshness` 496ms / 0 findings, `runtime-render` skipped by static-only | Phase 8 command output | 2026-07-08 |
| `ui-health validate scenario swarm-manager --json` | 7.89s command wall-clock; metrics: `static-interop` 561ms / 137 findings, `static-freshness` 477ms / 0 findings, `runtime-render` 2374ms / 2 findings | Phase 8 command output | 2026-07-08 |

### Phase 1 Validation Baseline — 2026-07-07

Reference scenarios:

- Small target: `ui-health`
- Heavy target: `swarm-manager`

The validation API now emits explicit execution-metrics stages for
`static-interop`, `static-freshness`, and `runtime-render`, which makes the
baseline reproducible from the CLI JSON output. The `swarm-manager` baseline
currently fails because it includes `freshness_ui_bundle_stale` at error
severity; that is recorded as baseline state, not introduced by this timing
work.

Small full-run finding code/severity counts:

| Count | Code | Severity |
|---:|---|---|
| 13 | `standard_component_location` | `FINDING_SEVERITY_WARNING` |
| 5 | `standard_unused_custom_component` | `FINDING_SEVERITY_WARNING` |
| 5 | `standard_raw_primitive_overuse` | `FINDING_SEVERITY_WARNING` |
| 4 | `interop_h_screen` | `FINDING_SEVERITY_WARNING` |
| 2 | `runtime_render_ok` | `FINDING_SEVERITY_INFO` |
| 2 | `pwa_service_worker_offline` | `FINDING_SEVERITY_WARNING` |
| 2 | `interop_protective_comments` | `FINDING_SEVERITY_INFO` |
| 2 | `interop_no_scattered_keydown` | `FINDING_SEVERITY_WARNING` |
| 1 | `pwa_manifest_install_fields` | `FINDING_SEVERITY_WARNING` |

Heavy full-run finding code/severity counts:

| Count | Code | Severity |
|---:|---|---|
| 88 | `standard_raw_primitive_overuse` | `FINDING_SEVERITY_WARNING` |
| 16 | `standard_unused_custom_component` | `FINDING_SEVERITY_WARNING` |
| 13 | `standard_component_location` | `FINDING_SEVERITY_WARNING` |
| 7 | `interop_h_screen` | `FINDING_SEVERITY_WARNING` |
| 3 | `interop_banned_scroll` | `FINDING_SEVERITY_INFO` |
| 2 | `runtime_render_ok` | `FINDING_SEVERITY_INFO` |
| 2 | `pwa_service_worker_offline` | `FINDING_SEVERITY_WARNING` |
| 2 | `interop_no_scattered_keydown` | `FINDING_SEVERITY_WARNING` |
| 1 | `template_id_missing` | `FINDING_SEVERITY_WARNING` |
| 1 | `standard_a11y_harness` | `FINDING_SEVERITY_WARNING` |
| 1 | `pwa_manifest_install_fields` | `FINDING_SEVERITY_WARNING` |
| 1 | `freshness_ui_bundle_stale` | `FINDING_SEVERITY_ERROR` |

### Phase 8 Validation Measurement — 2026-07-08

The post-hardening measurements show the intended headroom on the heavy target:
`swarm-manager` full validation completed in 7.89s command wall-clock, with the
runtime group bounded to 2374ms and static interop at 561ms. Static-only
validation completed in 5.65s command wall-clock, with static interop at 599ms.

The current `swarm-manager` finding set differs from the Phase 1 baseline
because the stale-bundle error disappeared after assets were rebuilt and the
workspace's current UI source state has additional component-canon findings.
No new error-severity finding appeared in this pass; full validation emitted two
`runtime_render_ok` info findings.

Server-owned `vrooli scenario test ui-health` run
`20260708-181138-e430e8d8` completed the wait with verdict `FAIL` after 186.4s,
but the failure was outside this plan's ui-health API changes: the findings
report shows broad pre-existing blockers across structure, architecture,
security, docs, business, and other phases. The wait payload also reported
`active=true` alongside `status=failed` / `verdict=FAIL`; this data-shape issue
was filed to scenario-qa as `knw-1783534528042066897`.

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
