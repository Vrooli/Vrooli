# Utils Unification Notes

This document tracks utility consolidation efforts and documents the shared utility architecture.

## Last Updated

2026-03-11

## Summary

The scenario uses a **screaming architecture** for utilities:
- **Core utilities** live in `api/internal/validation/` (pure, general-purpose)
- **Test utilities** live in `api/internal/testutil/` (test-only helpers and factories)
- **Framework utilities** live in `ui/src/lib/` (React/TypeScript specific)
- **Domain-specific logic** stays in domain packages (business rules that need context)

## Architecture Tiers

```
api/
├── internal/
│   ├── validation/    # Core: Pure, general-purpose validators
│   │   ├── validation.go      # Shared validation functions
│   │   └── validation_test.go # 30+ test cases
│   │
│   ├── config/        # Core: Centralized configuration
│   │   └── config.go          # Environment-based config
│   │
│   ├── errors/        # Core: Structured error types
│   │   └── errors.go          # Error categories and constructors
│   │
│   └── testutil/      # Testing: Factories and helpers
│       ├── helpers.go         # Request/assertion helpers
│       └── fixtures.go        # Test data factories
│
├── domain/
│   ├── reference/     # Domain: Reference-specific logic
│   ├── skill/         # Domain: Skill connection logic
│   └── expectation/   # Domain: Expectation/assertion logic
```

## Consolidated Utilities

### internal/validation Package

Created to consolidate duplicated validation patterns across domain services.

| Utility | Previously In | Description |
|---------|--------------|-------------|
| `IsValidSlugFormat` | `reference/service.go` | Validates URL-safe slug format |
| `IsValidSkillIDFormat` | `skill/service.go` | Validates skill ID format |
| `IsValidJSONPath` | `expectation/service.go` | Validates JSONPath expressions |
| `IsLengthInRange` | (new) | Generic length range check |
| `ValidateCommandSafety` | `expectation/service.go` | Checks command for dangerous patterns |
| `IsCommandSafe` | `expectation/service.go` | Convenience wrapper |
| `Truncate` | `cli/app.go` | String truncation (also kept in CLI) |

### Dependency Direction

```
domain packages  →  internal/validation  →  (no dependencies)
domain packages  →  internal/config      →  (os, strconv, strings)
domain packages  →  internal/errors      →  (encoding/json, fmt, net/http)
handlers         →  internal/errors      →  (same as above)
```

**Rule:** Internal packages must not import domain packages.

## What Stays In Domain

Domain packages retain logic that requires domain context:

| Logic | Package | Reason |
|-------|---------|--------|
| Slug existence check | `reference` | Requires repository access |
| Path validation | `reference` | Filesystem check, returns normalized path |
| Connection existence check | `skill` | Requires repository access |
| Expectation type validation | `expectation` | Domain-specific enum |
| Assertion operator validation | `expectation` | Domain-specific enum |

## Consolidation Candidates (Not Yet Implemented)

These were identified but not consolidated due to limited benefit:

1. **ServiceConfig pattern** - Each domain has its own ServiceConfig with similar structure. Not consolidated because:
   - Different domains need different fields
   - Functional options pattern is idiomatic Go
   - Premature abstraction would add complexity

2. **ListOptions pattern** - Each domain has its own ListOptions. Not consolidated because:
   - Domain-specific filter fields differ
   - Limit/Offset could be shared but adds coupling

3. **CLI truncate** - Kept in CLI because:
   - CLI is a separate Go module
   - Adding cross-module dependency for one function is overkill
   - CLI wrapper should be self-contained

## Test Coverage

The validation package has comprehensive tests:

- **Slug format**: 17 test cases (valid/invalid slugs)
- **Skill ID format**: 9 test cases
- **JSONPath**: 14 test cases
- **Length range**: 7 test cases
- **Command safety**: 11 test cases
- **Truncate**: 7 test cases

## Notes

- All regex patterns are compiled at package init time for performance
- Validators are pure functions with no side effects
- Error messages are constructed by callers, not validators (separation of concerns)
- The validation package is framework-agnostic (no dependencies on net/http, etc.)

## Related Documentation

- [SEAMS.md](SEAMS.md) - Integration seams and decision points
- [COHERENCE-NOTES.md](COHERENCE-NOTES.md) - UI architecture notes
- [docs/reference/configuration.md](../reference/configuration.md) - Configuration options
