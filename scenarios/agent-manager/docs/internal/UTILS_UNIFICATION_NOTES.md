# Utils Unification Notes

## Last Updated
2026-02-07

## Summary
Consolidated repeated UI display utilities and fully migrated date/time + currency formatting onto shared modules in `agent-manager` to reduce drift across dashboard, stats, run detail, and pricing surfaces. Shared logic now lives in `ui/src/lib/display.ts`, `ui/src/lib/dateTime.ts`, and `ui/src/lib/currency.ts`.

## Duplications
- Status label + badge variant helpers duplicated in:
  - `ui/src/features/stats/components/breakdown/ModelUsageBreakdown.tsx`
  - `ui/src/features/stats/components/breakdown/ToolUsageAnalytics.tsx`
- Unknown-value (`"unknown"`/empty) name formatting duplicated in:
  - `ui/src/features/stats/components/breakdown/ModelUsageBreakdown.tsx`
  - `ui/src/features/stats/components/breakdown/ToolUsageAnalytics.tsx`
  - `ui/src/components/ModelCostComparison.tsx`
- Hyphenated runner-name humanization local logic in:
  - `ui/src/pages/DashboardPage.tsx`
- Date/time formatter logic previously split between:
  - `ui/src/lib/utils.ts`
  - `ui/src/features/stats/utils/formatters.ts`
  - `ui/src/components/dialogs/pricingDisplay.ts`
- Currency formatter logic previously split between:
  - `ui/src/features/stats/utils/formatters.ts`
  - `ui/src/components/RunDetail.tsx`
  - `ui/src/components/dialogs/pricingDisplay.ts`

## Consolidation Candidates
1. Potential extraction of shared time formatting presets for future non-stats chart features

## Notes
- Consolidation kept behavior intact for all updated call sites.
- `ModelCostComparison` still applies its local truncation rule, now layered on shared unknown-value handling.
- Date/time and currency compatibility wrappers were removed from legacy formatter modules (`ui/src/lib/utils.ts`, `ui/src/features/stats/utils/formatters.ts`) after full callsite migration.
- Shared `Intl` formatter instances are now cached in `ui/src/lib/dateTime.ts` and `ui/src/lib/currency.ts` to reduce repeated constructor overhead.
- Added lightweight UI utility unit tests using Node test runner (`ui/tests/lib/*.test.ts`) with `pnpm test`.
