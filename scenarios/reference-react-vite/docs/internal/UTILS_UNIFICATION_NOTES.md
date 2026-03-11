# Utils Unification Notes

Documentation of utility consolidation efforts following the Utils Unification steer skill patterns.

## Last Updated
2026-03-11

## Summary

This scenario now follows a **single source of truth** pattern for shared types:
- `ui/src/lib/api.ts` is the canonical source for all API-related types
- `ui/src/test-utils/factories.ts` imports and re-exports types from api.ts
- Test utilities no longer duplicate type definitions

## Consolidated Types

### API Types (Canonical Source: `lib/api.ts`)

| Type | Description | Previously Duplicated In |
|------|-------------|--------------------------|
| `HealthResponse` | Health check response shape | `test-utils/factories.ts` |
| `Task` | Task entity type | `test-utils/factories.ts` |
| `Project` | Project entity type | `test-utils/factories.ts` |
| `Note` | Note entity type | `test-utils/factories.ts` |
| `PaginationMeta` | Pagination metadata | `test-utils/factories.ts` (as `Pagination`) |
| `ListResponse<T>` | Generic paginated list response | `test-utils/factories.ts` |

### Issues Fixed

1. **Type Drift**: `factories.ts` had `Project.task_count` which didn't exist in the API
2. **Naming Inconsistency**: `ListMeta` vs `Pagination` - standardized on `PaginationMeta`
3. **Field Mismatch**: `api.ts` had `meta` but API returns `pagination` - fixed to match API

## Utility Classification

Following the screaming architecture utility tier system:

### Core Utilities (`lib/`)
- `utils.ts` - Pure, general utilities (className merging via `cn()`)
- **Framework-agnostic**: No React, no DOM, just pure functions

### Domain Utilities (`lib/`)
- `api.ts` - API client and domain types
- **Business-specific**: Task, Project, Note entities and API operations
- **Single source of truth** for all API-related types

### Test Utilities (`test-utils/`)
- `factories.ts` - Mock data builders with type imports
- `renderWithProviders.tsx` - React test wrapper
- `setup.ts` - Global test configuration
- **Testing scope only**: Never imported by production code

## Dependency Direction

```
pages/ ──────────────────────┐
components/ ─────────────────┼──▶ lib/api.ts (types + API calls)
                             │
test-utils/factories.ts ─────┘──▶ lib/api.ts (types only)
```

**Rule**: All code depends on `lib/api.ts` for types. No duplication allowed.

## Testing Seams

### Mock Data Factories

Factories accept overrides following the builder pattern:

```typescript
createMockTask({ status: 'completed' })
createMockProject({ name: 'Custom Name' })
createMockListResponse([task1, task2], { total: 100 })
```

### Seam Pattern
- Factories use `new Date().toISOString()` for timestamps
- Future: Could inject clock for deterministic testing
- Current: Acceptable since timestamps are typically ignored in assertions

## Audit Findings

### Before Consolidation
- 6 duplicate type definitions across 2 files
- Type drift: `task_count` field existed only in test types
- Naming inconsistency: `ListMeta` vs `Pagination` vs `PaginationMeta`
- Field name mismatch: `meta` vs `pagination` in `ListResponse`

### After Consolidation
- 0 duplicate type definitions
- Single canonical source in `lib/api.ts`
- Consistent naming: `PaginationMeta`
- Field names match actual API response

## Notes

- **No utils/helpers.ts files**: The codebase has no catch-all utility files
- **Screaming domain structure**: Types are colocated with API operations
- **Test isolation**: Test utilities import from production code, never the reverse

## Related Documentation

- [SEAMS.md](./SEAMS.md) - Architectural seams and boundaries
- [UNIT_TEST_ARCHITECTURE.md](./UNIT_TEST_ARCHITECTURE.md) - Test patterns and utilities
- [CODE: ui/src/lib/api.ts] - Canonical API types
- [CODE: ui/src/test-utils/factories.ts] - Mock data factories
