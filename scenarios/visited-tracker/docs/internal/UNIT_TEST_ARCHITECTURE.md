# Visited Tracker Unit Testing Architecture

## Last Updated
2026-02-04

## Test Organization Status
- [x] Go tests co-located with source files (`api/*_test.go`, `cli/*_test.go`)
- [x] Consistent Go test naming (`*_test.go`)
- [ ] Dedicated `internal/testutil` package (current helpers live in `api/test_helpers.go`)

## Mock Organization Status
- [ ] Centralized mock package (mocks are inline or absent)
- [ ] Mock factory/builder patterns

## Testability Status
- [ ] Systematic dependency injection for filesystem/time/logging
- [x] Seam for glob expansion via `PathMatcher` (`api/targets.go`)
- [x] Visit payloads round-trip cleanly via `VisitFiles` marshal/unmarshal (`api/types.go`)

## Infrastructure Status
- [x] Temp-directory setup helpers exist (`api/test_helpers.go`)
- [ ] Structured test utilities folder

## Issues Found
1. `scenarios/visited-tracker/api/test_helpers.go` - Helpers are monolithic; consider extracting to `api/internal/testutil/` for reuse and clarity.
2. CLI helper tests are local; consider a shared CLI test util module if more commands add heavy setup.

## Priority Improvements
1. Introduce `api/internal/testutil/` for shared setup, JSON helpers, and assertions.
2. Add seam-focused tests for `resolveTargetsWithMatcher` using a stub matcher (no filesystem).
