# Globs Test — Implementation Plan

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

## 1. Purpose

Add comprehensive test coverage for glob pattern handling across the swarm-manager backlog system. Currently, glob validation and matching logic exists in multiple packages but has significant testing gaps — particularly for error paths, batch operations, and cross-package consistency.

## 2. Problem Statement

The swarm-manager backlog system uses glob patterns for `acceptance_allow` and `acceptance_deny` fields. The glob-related code spans four packages:

1. **`backlog/types.go`** — `validateGlobs()` uses `filepath.Match` for syntax checking
2. **`pathutil/root.go`** — `ScenariosFromGlobs()` extracts scenario names from patterns
3. **`scenarios/handler.go`** — `resolveGlobPattern()` uses `doublestar` for filesystem matching
4. **`agentmanager/service.go`** — forwards globs to agent-manager proto

Existing tests cover happy paths but miss:
- Invalid glob rejection at HTTP handler level (create, update)
- Batch handler glob validation (zero coverage)
- Edge cases around `filepath.Match` vs `doublestar` semantics (e.g. `**` patterns)

## 3. Scope

### In Scope
- Add negative/error-path tests for glob validation in create and update handlers
- Add glob validation tests for batch create handler
- Add edge-case tests for `validateGlobs()` (e.g. `**` patterns, path traversal attempts)
- Verify consistency between validation and runtime matching semantics

### Out of Scope
- Changing `validateGlobs()` implementation (unless tests reveal a bug)
- Modifying `doublestar` usage in `resolveGlobPattern()`
- Agent-manager side glob matching
- UI/CLI glob handling

## 4. Current Technical Context

### Key Files
| File | Role |
|------|------|
| `scenarios/swarm-manager/api/internal/backlog/types.go:146-161` | `validateGlobs()` — input validation |
| `scenarios/swarm-manager/api/internal/backlog/types_test.go` | `TestValidateGlobs` — 7 unit test cases |
| `scenarios/swarm-manager/api/internal/backlog/handler.go` | Create/Update handlers call `validateGlobs` |
| `scenarios/swarm-manager/api/internal/backlog/handler_test.go:1654-1700` | Happy-path handler glob tests |
| `scenarios/swarm-manager/api/internal/backlog/batch_handler.go:180-185` | BatchCreate calls `validateGlobs` |
| `scenarios/swarm-manager/api/internal/backlog/batch_handler_test.go` | Zero glob-related tests |
| `scenarios/swarm-manager/api/internal/pathutil/root.go:69-91` | `ScenariosFromGlobs()` |
| `scenarios/swarm-manager/api/internal/pathutil/root_test.go:129` | `TestScenariosFromGlobs` — 11 cases |

### Existing Test Coverage
- `TestValidateGlobs`: 7 cases (unit level)
- `TestCreate_WithAcceptanceGlobs`: happy path only
- `TestUpdate_Acceptance`: happy path only
- `TestScenariosFromGlobs`: 11 cases (good coverage)
- Execution service review trigger: 4 test cases
- **BatchCreate globs: ZERO tests**

## 5. Target End State

- Every glob validation code path (create, update, batch-create) has both happy-path and error-path HTTP-level tests
- `validateGlobs` unit tests cover edge cases: `**` patterns, empty strings, absolute paths, path traversal
- Batch handler has glob validation tests matching the create handler's coverage
- All tests pass: `go test ./internal/backlog/... ./internal/pathutil/...`

## 6. Implementation Strategy

### Phase 1: Expand `validateGlobs` Unit Tests
Add edge cases to `TestValidateGlobs` in `types_test.go`:
- `**` double-star patterns (should pass validation)
- Patterns with `..` path traversal
- Very long patterns
- Unicode in patterns

### Phase 2: Add Handler-Level Error Path Tests
In `handler_test.go`:
- `TestCreate_WithInvalidAcceptanceGlobs` — verify 400 response for invalid patterns
- `TestUpdate_WithInvalidAcceptanceGlobs` — verify 400 response for invalid patterns

### Phase 3: Add Batch Handler Glob Tests
In `batch_handler_test.go`:
- `TestBatchCreate_WithAcceptanceGlobs` — happy path
- `TestBatchCreate_WithInvalidAcceptanceGlobs` — error path

## 7. Contract Decisions

- Tests should use the existing test infrastructure (test helpers, mock stores)
- No new dependencies needed
- Test patterns should be realistic (e.g. `scenarios/web-console/**`, not synthetic)

## 8. Testing Plan

Run: `cd scenarios/swarm-manager/api && go test ./internal/backlog/... ./internal/pathutil/... -timeout 300s`

All new tests must pass alongside existing tests with no regressions.

## 9. Rollout/Validation Checklist

- [ ] All new tests pass
- [ ] Existing tests unaffected
- [ ] `go vet ./...` clean
- [ ] `gofumpt` formatted

## 10. Risks + Mitigations

| Risk | Mitigation |
|------|-----------|
| `filepath.Match` vs `doublestar` semantic gap | Document in test comments; add test case showing `**` passes validation |
| Test infrastructure changes | Use existing patterns from handler_test.go |

## 11. Non-goals / Prohibited Patterns

- Do not refactor `validateGlobs` unless a bug is discovered
- Do not add new dependencies
- Do not modify non-test files unless a bug fix is required

## 12. Definition of Done

- [ ] `validateGlobs` has ≥12 test cases covering all edge cases
- [ ] Create handler has invalid-glob error-path test
- [ ] Update handler has invalid-glob error-path test
- [ ] Batch create handler has both happy-path and error-path glob tests
- [ ] All tests pass with `go test ./internal/backlog/... -timeout 300s`
- [ ] Code formatted with `gofumpt`
