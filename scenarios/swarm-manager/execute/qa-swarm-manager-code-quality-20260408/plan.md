# Implementation Plan: Reduce Code Quality Violations in swarm-manager

## Greenfield Declaration

This is a **refactoring-only** item. No backward-compatibility shims, feature flags, or migration bridges are needed. All changes are internal structural improvements — extract helpers, split files, reduce complexity. No external API contracts change.

## 1. Purpose

Reduce the swarm-manager API code quality violations by refactoring the top ~30 highest-priority files. Eliminate all tech debt markers. CLI files are excluded (separate item).

## 2. Required Reading

```bash
prompt-manager skill read refactor code-cleanup implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

## 3. Problem Statement

The swarm-manager scenario has a code quality score of 0/100 with 379 violations:
- **107 complex functions** — functions with high cyclomatic complexity
- **48 long files** — files exceeding length thresholds
- **3 tech debt markers** — TODO/FIXME/HACK comments (1 confirmed in API: `stats/engine.go:289`)
- **1 lint issue**

**Scale (round 2 analysis):** 54 files with 200+ lines, 67 files with complexity_max >= 6. The full set is too large for one item. This plan targets the top ~30 API files by refactor priority.

## 4. Scope

### In Scope
- Top ~30 Go source files under `scenarios/swarm-manager/api/` ranked by refactor priority
- Refactoring to reduce complexity, length, and duplication
- Removing tech debt markers
- Fixing lint issues

### Out of Scope
- CLI files (`cli/`) — deferred to separate backlog item (round 2 d3)
- Changing external API contracts or behavior
- Modifying proto definitions
- Changes outside `scenarios/swarm-manager/`
- Adding new features

## 5. Current Technical Context

### Settled Decisions

**Round 1:**
- **d1:** Extract shared file-serving logic from `backlog/files.go` and `initiatives/files.go` into new `internal/fileserve` package
- **d2:** Keep `eventlog/emitter.go` explicit typed methods — accept duplication score
- **d3:** Extract `errorWriter` and shared test helpers into existing `internal/testutil`
- **d4:** Create `doUpdate()` helper in `handler_update_test.go` to reduce repetitive PATCH setup

**Round 2:**
- **d1:** Target top ~30 files, accept partial improvement
- **d2:** Extract sub-handlers and helpers within same package for production files
- **d3:** Exclude CLI from this item
- **d4:** Split large test files by category + extract shared helpers

**Round 3:**
- **d1:** Decompose `batch_handler.go` (complexity 59) by extracting each batch operation into named functions within the same file (not splitting across files)
- **d2:** Keep phase-based execution ordering (Phase 1 shared utilities first, then phases 2-4)
- **d3:** Agent must NOT create commits — user commits manually
- **d4:** No numeric score target; success = all 30 files addressed + tech debt markers removed

### Prioritized File List (Top 30 API Files)

**Tier 1: Highest-complexity production files (extract sub-handlers within package)**

| # | File | Lines | Cmplx Max | Dup % | Refactor Action |
|---|------|-------|-----------|-------|-----------------|
| 1 | `backlog/batch_handler.go` | 401 | **59** | 18% | Extract each batch operation into named functions in the same file (settled d1-r3) |
| 2 | `backlog/files.go` | 292 | **35** | 53% | Extract to `internal/fileserve` (settled d1-r1) |
| 3 | `initiatives/files.go` | 312 | **35** | 57% | Extract to `internal/fileserve` (settled d1-r1) |
| 4 | `scenarios/files.go` | 118 | **9** | 53% | Also extract to `internal/fileserve` |
| 5 | `backlog/workshop_save.go` | 360 | **29** | 24% | Extract validation/parsing helpers |
| 6 | `backlog/handler.go` | 455 | **28** | 18% | Split into handler_create.go, handler_list.go, etc. |
| 7 | `backlog/import_parser.go` | 403 | **24** | 22% | Extract parsing sub-functions |
| 8 | `backlog/archive_handlers.go` | 341 | **20** | 27% | Extract archive operation helpers |
| 9 | `ideas/handler.go` | 712 | **17** | 35% | Split by CRUD operation into separate files |
| 10 | `backlog/backlog_summary.go` | 178 | **14** | 43% | Extract summary computation helpers |
| 11 | `initiatives/handler.go` | 338 | **12** | 57% | Extract sub-handlers; benefit from fileserve extraction |
| 12 | `prompts/handler.go` | 370 | **10** | 29% | Split by endpoint |

**Tier 2: Large test files (split by category + extract helpers)**

| # | File | Lines | Cmplx Max | Dup % | Refactor Action |
|---|------|-------|-----------|-------|-----------------|
| 13 | `execution/service_test.go` | 1279 | 18 | 20% | Split by service method under test |
| 14 | `ideas/handler_test.go` | 1065 | 10 | 47% | Split by CRUD + extract request helpers |
| 15 | `backlog/handler_test.go` | 1012 | 8 | 21% | Split by handler endpoint |
| 16 | `scenarios/handler_test.go` | 757 | 11 | 32% | Split by endpoint |
| 17 | `backlog/store_test.go` | 730 | 29 | 37% | Split by store method |
| 18 | `backlog/import_test.go` | 727 | 18 | 28% | Split by import scenario |
| 19 | `backlog/batch_queue_handler_test.go` | 627 | 10 | 52% | Extract queue test helpers |
| 20 | `backlog/workshop_save_test.go` | 611 | 11 | 31% | Split by save scenario |
| 21 | `backlog/batch_handler_test.go` | 543 | 8 | 23% | Extract batch test helpers |
| 22 | `workshop/workshop_test.go` | 519 | 13 | 36% | Split by workshop phase |
| 23 | `agentactivity/service_test.go` | 484 | 17 | 17% | Extract service test helpers |
| 24 | `backlog/handler_update_test.go` | 472 | 8 | 67% | doUpdate() helper (settled d4-r1) |
| 25 | `backlog/initialize_test.go` | 473 | 5 | 41% | Extract init test helpers |

**Tier 3: High-duplication small files (extract to testutil)**

| # | File | Lines | Dup % | Refactor Action |
|---|------|-------|-------|-----------------|
| 26 | `queue/encode_error_test.go` | 39 | 100% | Extract errorWriter to testutil (settled d3-r1) |
| 27 | `backlog/encode_error_test.go` | 39 | 100% | Extract errorWriter to testutil (settled d3-r1) |
| 28 | `queue/store_default_test.go` | 13 | 100% | Extract to testutil (settled d3-r1) |
| 29 | `settings/helpers_test.go` | 13 | 100% | Extract to testutil (settled d3-r1) |
| 30 | `settings/normalize_test.go` | 136 | 89% | Convert to table-driven tests |

## 6. Target End State

- All 30 listed files refactored per their specified refactor actions
- tech_debt_markers = 0 (remove TODO at `stats/engine.go:289`)
- All existing tests continue to pass
- No behavioral changes to APIs
- Code formatted with `gofumpt` and clean under `golangci-lint run`

## 7. Implementation Strategy

### Phase 1: Extract shared utilities (highest leverage)
1. Create `internal/fileserve` package with generic file upload/download/list/delete helpers
2. Refactor `backlog/files.go`, `initiatives/files.go`, and `scenarios/files.go` to use `fileserve`
3. Extract `errorWriter` and shared test patterns into existing `internal/testutil`
4. Run tests to confirm no regressions

### Phase 2: Decompose highest-complexity production files
Target files with complexity_max >= 10 (Tier 1 items 1, 5-12):
1. `backlog/batch_handler.go` (complexity 59) — extract each batch operation into named functions within the same file; do NOT split across multiple files
2. `backlog/workshop_save.go` (complexity 29) — extract validation and parsing into helper functions
3. `backlog/handler.go` (complexity 28) — split into handler_create.go, handler_list.go, handler_search.go
4. `backlog/import_parser.go` (complexity 24) — extract parsing sub-functions for each import format
5. `backlog/archive_handlers.go` (complexity 20) — extract archive operation helpers
6. `ideas/handler.go` (712 lines, complexity 17) — split by CRUD operation
7. `backlog/backlog_summary.go` (complexity 14) — extract summary computation helpers
8. `initiatives/handler.go` (complexity 12) — extract sub-handlers
9. `prompts/handler.go` (complexity 10) — split by endpoint
10. Run tests after each file

### Phase 3: Split large test files
Target Tier 2 files (13 test files, 470-1300 lines each):
1. For each large test file:
   a. Identify natural split boundaries (by handler method, service operation, or test category)
   b. Extract shared test setup into per-package helpers or testutil
   c. Split into focused test files (e.g., `handler_create_test.go`, `handler_update_test.go`)
   d. Verify test count preserved: `go test -v ./internal/<pkg>/... | grep -c "=== RUN"`
2. Apply `doUpdate()` helper pattern to `handler_update_test.go` (settled d4-r1)
3. Run full test suite after each package

### Phase 4: Clean up remaining violations
1. Convert `settings/normalize_test.go` to table-driven tests (reduce 89% duplication)
2. Address remaining moderate-complexity files in the top 30
3. Remove tech debt marker at `stats/engine.go:289`
4. Run `gofumpt -w .` for consistent formatting
5. Run `golangci-lint run` to catch remaining lint issues

### Phase 5: Verify and final cleanup
1. Run full test suite: `go test ./... -timeout 300s`
2. Run `tidiness-manager recommend-refactors swarm-manager --limit 10` to confirm top violations addressed
3. Run `vrooli scenario restart swarm-manager` to verify the scenario starts cleanly after all changes
4. Document results

## 8. Contract Decisions

- No API contract changes
- No proto changes
- All refactoring is internal structural improvement only
- Agent must NOT create git commits — user handles commits manually

## 9. Testing Plan

- All existing tests must continue to pass after each phase
- Run `go test ./... -timeout 300s` after each phase
- For test file splits: verify test count preserved with `go test -v ./internal/<pkg>/... 2>&1 | grep -c "=== RUN"` before and after
- No new test coverage required (refactoring, not new behavior)
- Final validation via tidiness-manager recommendations
- Work incrementally: test after each file or small group of related files

## 10. Rollout/Validation Checklist

- [ ] Phase 1: fileserve + testutil extraction done, tests pass
- [ ] Phase 2: High-complexity production files decomposed, tests pass
- [ ] Phase 3: Large test files split, test count preserved, tests pass
- [ ] Phase 4: Remaining violations cleaned, tech debt = 0, tests pass
- [ ] Phase 5: Full test suite passes, tidiness scan confirms improvements, `vrooli scenario restart swarm-manager` succeeds

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| batch_handler.go (complexity 59) is hard to decompose | Medium | Medium | Read the function carefully; likely a single large switch or nested conditionals that can be extracted into named helpers |
| Test file splits break test isolation | Low | High | Go test files in same package share state; splitting within package is safe. Verify test counts. |
| fileserve extraction creates import cycles | Low | Medium | fileserve is a new leaf package with no internal dependencies |
| Refactoring breaks existing tests | Medium | High | Work incrementally, run tests after each file; user commits at safe points |
| 30 files not enough for meaningful improvement | Low | Medium | The 30 files include the worst offenders (complexity 59, 35, 29, 28, 24, 20) which disproportionately affect scores |

## 12. Non-goals / Prohibited Patterns

- Do NOT change any API behavior or contracts
- Do NOT add new features during refactoring
- Do NOT modify proto definitions
- Do NOT refactor files outside `scenarios/swarm-manager/api/`
- Do NOT touch CLI files (deferred to separate item)
- Do NOT add unnecessary abstractions — only extract when genuinely reducing duplication or complexity
- Do NOT consolidate `eventlog/emitter.go` — keep explicit typed methods (settled d2-r1)
- Do NOT refactor files outside the top-30 list unless they're trivial wins discovered during work
- Do NOT create git commits — user commits manually (settled d3-r3)

## 13. Definition of Done

- [ ] All 30 listed files have been refactored per their specified actions
- [ ] tech_debt_markers = 0 (specifically `stats/engine.go:289` removed)
- [ ] All existing tests pass (`go test ./... -timeout 300s`)
- [ ] Code formatted with `gofumpt`
- [ ] No lint issues from `golangci-lint run`
- [ ] `vrooli scenario restart swarm-manager` succeeds cleanly
