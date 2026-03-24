# Fix: Globs Test

## Required Reading

```bash
prompt-manager skill read swarm-manager-backlog-tools
```

## Purpose

Fix the glob validation function `validateGlobs` in the swarm-manager backlog package to properly validate `**` (doublestar/globstar) patterns. Currently, `filepath.Match` is used for validation, which silently accepts `**` patterns but treats them identically to `*` — matching only a single path segment. This means patterns like `api/**` (intended to match all files recursively under `api/`) will pass validation but won't behave as expected when used downstream.

## Problem Statement

### Root Cause

`validateGlobs()` in `scenarios/swarm-manager/api/internal/backlog/types.go:173-186` uses Go's `filepath.Match(g, "")` to check glob syntax validity. However, `filepath.Match` does not support the `**` (doublestar) pattern — it treats `**` as two consecutive `*` wildcards within a single path segment, which is functionally equivalent to a single `*`.

### Evidence

- `filepath.Match("api/**", "api/foo/bar.go")` returns `false` — it cannot match across directory boundaries
- `filepath.Match("api/**", "api/bar.go")` returns `true` — it only matches within one segment
- The `scenarios/handler.go` in the same codebase already uses `github.com/bmatcuk/doublestar/v4` for proper `**` support (line 1214)
- The `doublestar/v4` library is already an indirect dependency in `go.mod`

### Impact

- Users can set `acceptance_allow` / `acceptance_deny` globs with `**` patterns that pass validation
- These patterns silently fail to match deeply nested files when used for acceptance filtering
- This creates a false sense of security — files that should be denied may be allowed, and vice versa

## Scope

**In scope:**
- Fix `validateGlobs()` to use `doublestar.Match` (or `doublestar.ValidatePattern`) for syntax validation
- Add/update unit tests in `types_test.go` to cover `**` patterns
- Ensure the `doublestar` dependency is promoted from indirect to direct in `go.mod`

**Out of scope:**
- Changing how acceptance globs are matched at runtime (in `workspace-sandbox` or `agent-manager`) — that's a separate concern
- Modifying the scenarios handler glob logic (already correct)
- Adding new glob features beyond fixing `**` validation

## Current Technical Context

### Key Files

| File | Role |
|------|------|
| `scenarios/swarm-manager/api/internal/backlog/types.go` | Contains `validateGlobs()` (lines 173-186) |
| `scenarios/swarm-manager/api/internal/backlog/types_test.go` | Contains `TestValidateGlobs` and `TestValidateScope` |
| `scenarios/swarm-manager/api/internal/scenarios/handler.go` | Already uses `doublestar.FilepathGlob` (line 1214) — reference implementation |
| `scenarios/swarm-manager/api/go.mod` | Has `doublestar/v4` as indirect dependency |

### Current `validateGlobs` Implementation

```go
func validateGlobs(globs []string) error {
    for i, g := range globs {
        if strings.TrimSpace(g) == "" {
            return fmt.Errorf("glob[%d]: empty string not allowed", i)
        }
        if filepath.IsAbs(g) {
            return fmt.Errorf("glob[%d]: absolute paths not allowed: %s", i, g)
        }
        if _, err := filepath.Match(g, ""); err != nil {
            return fmt.Errorf("glob[%d]: invalid pattern %q: %w", i, g, err)
        }
    }
    return nil
}
```

## Target End State

- `validateGlobs` uses `doublestar.ValidatePattern` (or `doublestar.Match`) to validate glob syntax, correctly accepting `**` patterns
- Tests cover: valid `**` patterns, invalid syntax, empty strings, absolute paths, and edge cases like `{a,b}` and `[!abc]`
- `doublestar/v4` is a direct dependency

## Implementation Strategy

### Phase 1: Fix validation (single file change)

1. Import `github.com/bmatcuk/doublestar/v4` in `types.go`
2. Replace `filepath.Match(g, "")` with `doublestar.ValidatePattern(g)` (preferred) or `doublestar.Match(g, "")` in `validateGlobs()`
3. Keep the existing empty-string and absolute-path checks (these are domain-specific guards)

### Phase 2: Update tests

1. Add test cases to `TestValidateGlobs` for:
   - `**/*.go` — valid doublestar pattern
   - `api/**/handler.go` — valid mid-path doublestar
   - `{a,b}` — valid alternation (supported by doublestar but not filepath.Match)
   - Patterns with `..` traversal (should still be rejected by the absolute-path check or a new traversal check)
2. Verify existing tests still pass

### Phase 3: Dependency cleanup

1. Run `go mod tidy` to promote `doublestar/v4` from indirect to direct

## Contract Decisions

- The validation function's API (signature, error format) stays the same
- `**` patterns are valid and accepted
- No changes to how globs are matched at runtime — this fix only addresses validation

## Testing Plan

- [ ] Existing `TestValidateGlobs` cases pass
- [ ] New cases: `**/*.go`, `api/**/handler.go`, `{a,b}` accepted as valid
- [ ] Invalid patterns still rejected (empty, absolute, bad syntax)
- [ ] `TestValidateScope` unaffected
- [ ] `go build ./...` succeeds
- [ ] `go test ./internal/backlog/ -run TestValidateGlobs` passes

## Risks + Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `doublestar.ValidatePattern` has different strictness than `filepath.Match` | Low | Review doublestar docs; test edge cases |
| Changing validation could reject previously-accepted patterns | Very Low | `doublestar` is a superset of `filepath.Match` — it accepts everything `filepath.Match` does plus `**` |
| Existing stored globs with bad syntax could fail re-validation | Very Low | Validation only runs on create/update, not on read |

## Non-goals / Prohibited Patterns

- Do not add runtime glob matching to the backlog package — that belongs in workspace-sandbox
- Do not refactor `validateScope` (it works correctly)
- Do not add `..` traversal detection to `validateGlobs` unless it's broken (it's handled by the absolute-path check currently)

## Definition of Done

- [ ] `validateGlobs` uses `doublestar` for pattern validation
- [ ] All existing tests pass
- [ ] New test cases cover `**` patterns
- [ ] `go build ./...` and `go test ./...` pass in the swarm-manager API
