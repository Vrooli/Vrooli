# Implementation Plan: Reduce Code Quality Violations in git-control-tower

## Purpose

Refactor the git-control-tower scenario codebase to achieve a code quality score ≥70 by reducing complex functions (<10), long files (<10), and removing all tech debt markers.

## Required Reading

```bash
prompt-manager skill read refactor code-cleanup cli-steer api-steer
```

## Problem Statement

The git-control-tower scenario has a code quality score of 0/100 with 88 violations:
- **45 complex functions** — repeated boilerplate in HTTP clients and test files
- **43 long files** — test files with repeated setup, client files with duplicated HTTP helpers
- **2 tech debt markers** — TODO comments in UI source files

Root causes identified from codebase analysis:
1. **No shared HTTP client base** — 5 client files (`tidiness_manager_client.go`, `test_genie_client.go`, `auditor_client.go`, `browser_automation_client.go`, `agent_manager_client.go`) each duplicate `doJSON()`, `doGet()`, `resolveBaseURL()`, and error parsing methods
2. **No test helpers** — Test files repeat setup, assertion, and validation boilerplate
3. **No language scanner abstraction** — `golang/scanner.go` and `python/scanner.go` have 90%+ identical `Scan()` method bodies
4. **No table-driven tests** — Individual test functions duplicate nearly identical setups

## Scope

**In scope:**
- All Go source files under `scenarios/git-control-tower/api/`
- UI TODO markers in `scenarios/git-control-tower/ui/src/`

**Out of scope:**
- Functional behavior changes — all refactors must preserve existing behavior
- Adding new features or endpoints
- Changes to `packages/` or other scenarios
- UI refactoring beyond removing TODO markers

## Current Technical Context

### Key Files (by refactor priority)

| File | Lines | Complexity | Primary Issue |
|------|-------|-----------|---------------|
| `api/gitignore_health_service_test.go` | 392→441 | 8 | Repeated test setup, no table-driven tests |
| `api/tidiness_manager_client.go` | 156 | 7 | Duplicate HTTP helpers shared across 5 clients |
| `api/test_genie_client.go` | 123 | 7 | Same HTTP client duplication |
| `api/ssh/service_test.go` | 137 | 5 | Test boilerplate, no subtests |
| `api/filerelations/languages/golang/scanner.go` | 49→61 | 4 | Duplicate context checks, shared logic with python |
| `api/filerelations/languages/python/scanner.go` | 48→59 | 4 | Nearly identical to golang scanner |

### Duplication Analysis

**HTTP Client Pattern** (repeated in 5 files):
```go
func (c *XxxClient) doJSON(ctx, path, body, result) error { ... }
func (c *XxxClient) doGet(ctx, path, result) error { ... }
func parseXxxError(resp) error { ... }
```

**Test Setup Pattern** (repeated in test files):
```go
platform := &FakePlatform{SSHDirPath: "...", HomeDirPath: "..."}
deps := XxxDeps{Platform: platform}
```

**Scanner Pattern** (repeated in golang/ and python/):
```go
func (s *Scanner) Scan(ctx, content) ([]string, error) {
    select { case <-ctx.Done(): return nil, ctx.Err(); default: }
    // aggregate imports from matchers...
}
```

## Target End State

- Code quality score ≥70
- complex_functions < 10
- long_files < 10
- tech_debt_markers = 0
- All existing tests pass with no behavioral changes
- Shared HTTP client base eliminates duplication across 5 client files
- Test helpers reduce boilerplate in test files
- Scanner abstraction eliminates duplication between language scanners

## Implementation Strategy

### Phase 1: Shared HTTP Client Base
<!-- Pending decision d1 on approach -->

Extract common HTTP client logic (`doJSON`, `doGet`, `resolveBaseURL`, error parsing) into a shared base type. Refactor all 5 client files to compose this base.

**Files affected:**
- New: `api/httpclient.go` (or similar)
- Modified: `api/tidiness_manager_client.go`, `api/test_genie_client.go`, `api/auditor_client.go`, `api/browser_automation_client.go`, `api/agent_manager_client.go`

### Phase 2: Test Helpers & Table-Driven Tests
<!-- Pending decision d2 on scope -->

Create test helper functions and convert repeated test patterns to table-driven tests.

**Files affected:**
- `api/gitignore_health_service_test.go` — table-driven test conversion, extract helper
- `api/ssh/service_test.go` — extract `newTestDeps()`, use subtests
- `api/tidiness_manager_handler_test.go`, `api/test_genie_handler_test.go`, `api/auditor_handler_test.go`, `api/repo_status_service_test.go`

### Phase 3: Scanner Abstraction
<!-- Pending decision d3 on approach -->

Extract shared scanning logic between golang and python scanners.

**Files affected:**
- `api/filerelations/languages/golang/scanner.go`
- `api/filerelations/languages/python/scanner.go`
- Possibly new shared scanner base

### Phase 4: Tech Debt Markers
Remove or resolve the 2 TODO comments in UI source files.

**Files affected:**
- `ui/src/App.tsx` (line ~1320)
- `ui/src/components/ScenarioReviewPanel.tsx` (line ~665)

## Contract Decisions

- No API or CLI contract changes — this is purely internal refactoring
- All existing tests must continue to pass
- No new dependencies

## Testing Plan

1. Run `go test ./...` after each phase to verify no regressions
2. Run `golangci-lint run` to check for new issues
3. Re-run `tidiness-manager recommend-refactors git-control-tower` after all phases to verify metric improvements
4. Final verification: `git-control-tower review-run git-control-tower --json` to confirm score ≥70

## Rollout/Validation Checklist

- [ ] Phase 1 complete — shared HTTP client, all 5 clients refactored
- [ ] Phase 2 complete — test helpers created, table-driven tests converted
- [ ] Phase 3 complete — scanner abstraction in place
- [ ] Phase 4 complete — tech debt markers resolved
- [ ] All tests pass (`go test ./... -timeout 300s`)
- [ ] Code quality score ≥70 confirmed via GCT review

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Refactoring breaks subtle test behavior | Medium | Medium | Run tests after each file change, not just per phase |
| HTTP client base introduces new coupling | Low | Medium | Use composition (embedding), not inheritance |
| Scanner abstraction over-engineered for 2 languages | Medium | Low | Keep it simple — shared function, not complex interface hierarchy |

## Non-goals / Prohibited Patterns

- Do NOT add new features or change API behavior
- Do NOT introduce new external dependencies
- Do NOT create deeply nested abstraction hierarchies
- Do NOT change test assertions to make refactored code pass — fix the code instead

## Definition of Done

- [ ] Code quality score ≥70 (verified via `git-control-tower review-run git-control-tower --json`)
- [ ] complex_functions < 10
- [ ] long_files < 10
- [ ] tech_debt_markers = 0
- [ ] All existing tests pass without modification to assertions
- [ ] No new files exceed 150 lines
