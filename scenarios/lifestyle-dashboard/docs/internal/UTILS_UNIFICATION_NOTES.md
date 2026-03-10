# Utils Unification Notes

## Last Updated
2026-03-10

## Summary

The lifestyle-dashboard utility structure follows screaming architecture principles. Utilities are well-organized into appropriate tiers with minimal duplication. One consolidation was performed during the Phase 14 audit.

## Architecture Tier Classification

### UI (`ui/src/`)

| Tier | Location | Description | Status |
|------|----------|-------------|--------|
| **Core** | `lib/format.ts` | Pure formatting utilities (dates, times, bytes) | ✅ Clean |
| **Framework** | `lib/utils.ts` | Tailwind class utilities (CVA integration) | ✅ Clean |
| **Domain** | `lib/api.ts` | API client with typed responses, error handling | ✅ Clean |
| **Testing** | `consts/selectors.ts` | Test ID constants and selector generators | ✅ Clean |

### API (`api/`)

| Tier | Location | Description | Status |
|------|----------|-------------|--------|
| **Core** | `internal/clock/` | Time abstraction seam | ✅ Clean |
| **Core** | `internal/testutil/` | Test database helpers, mocks | ✅ Clean |
| **Domain** | `domain/` | Business types, schema | ✅ Clean |
| **Domain** | `config/` | Configuration with env overrides | ✅ Clean |
| **Domain** | `errors/` | Structured error categories | ✅ Clean |
| **Data** | `repository/` | SQLite implementations | ✅ Clean |
| **HTTP** | `handlers/` | Request/response handling | ✅ Clean |

## Consolidations Performed

### 1. Date/Size Formatting (Phase 14)

**Before:**
- `SettingsPage.tsx` had inline `formatBytes()` and `formatDate()` functions
- `lib/format.ts` had `formatRelativeTime()`, `formatShortDate()`, `formatDateTime()`

**After:**
- All formatting functions consolidated into `lib/format.ts`
- `SettingsPage.tsx` imports from shared location
- `formatDate()` enhanced with fallback parameter for null safety

**Files changed:**
- `ui/src/lib/format.ts` - Added `formatBytes()`, `formatDate()`
- `ui/src/pages/SettingsPage.tsx` - Removed inline functions, added import

## Duplications Analyzed (No Action Needed)

### formatDayLabel (TimelineChart.tsx)

This formatting function remains inline because:
1. Takes chart-specific `TrendPeriod` parameter (7/30/90)
2. Output varies by period length (weekday vs day vs month)
3. Single call site, chart-specific logic

### toLocaleDateString calls

Various components call `toLocaleDateString()` directly for one-off formatting. These are:
1. Too specific to extract (e.g., BriefsPage date picker max)
2. Single-use cases that don't warrant abstraction

## Dependency Direction Compliance

```
pages -> components/dashboard -> lib/api -> @vrooli/api-base
pages -> lib/format (core utilities)
pages -> components/ui (framework utilities)
components/dashboard -> lib/format
lib/format X-> React (pure)
lib/utils X-> app code (pure)
```

## Testing Seams

| Concern | Seam | Location | Status |
|---------|------|----------|--------|
| Time | Clock interface | `api/internal/clock/` | ✅ Ready |
| Database | Repository interfaces | `api/repository/interfaces.go` | ✅ Ready |
| HTTP | Handler constructors | `api/handlers/*.go` | ✅ Ready |
| Mocks | Builder pattern | `api/internal/testutil/mocks.go` | ✅ Ready |

UI utilities (`lib/format.ts`) are pure functions that don't require seams.

## Recommendations for Future Work

1. **None critical** - The utility structure is clean
2. **Consider**: If more byte/size formatting needs arise, could add `formatKB()`, `formatMB()` helpers
3. **Consider**: If more date patterns emerge, could add `formatISODate()` for consistency

## Notes

- `lib/utils.ts` uses `clsx` + `tailwind-merge` pattern standard in shadcn/ui
- `consts/selectors.ts` uses template-based selector generation for type safety
- Go packages follow standard Go project layout with `internal/` for private utilities
