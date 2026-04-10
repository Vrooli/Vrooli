# Implementation Plan: Reduce Code Quality Violations in git-control-tower

## Greenfield Constraint

This is a **refactoring-only** task. No backward compatibility shims, feature flags, or migration bridges are needed. All changes must preserve existing behavior — the API/CLI contracts do not change. Old patterns are replaced in-place, not wrapped.

## Purpose

Refactor the git-control-tower scenario codebase to achieve a code quality score ≥70 by reducing complex functions (<10), long files (<10), and removing all tech debt markers.

## Required Reading

```bash
prompt-manager skill read refactor code-cleanup cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

## Problem Statement

The git-control-tower scenario has a code quality score of 0/100 with 88 violations:
- **45 complex functions** — repeated boilerplate in HTTP clients and test files
- **43 long files** — test files with repeated setup, client files with duplicated HTTP helpers, plus many large service/handler files
- **2 tech debt markers** — TODO comments in UI source files

Root causes identified from codebase analysis:
1. **No shared HTTP client base** — 5 client files (`tidiness_manager_client.go`, `test_genie_client.go`, `auditor_client.go`, `browser_automation_client.go`, `agent_manager_client.go`) each duplicate `doJSON()`, `doGet()`, `resolveBaseURL()`, and error parsing methods. Total: ~1,117 lines across 5 files.
2. **No test helpers** — Test files repeat setup, assertion, and validation boilerplate
3. **No language scanner abstraction** — `golang/scanner.go` and `python/scanner.go` have 90%+ identical `Scan()` method bodies
4. **No table-driven tests** — Individual test functions duplicate nearly identical setups
5. **Large service/handler files** — main.go (1,363 lines), review_handler_test.go (1,355 lines), git_runner.go (1,149 lines), and many others exceed thresholds independent of the duplication issues above

## Scope

**In scope:**
- All Go source files under `scenarios/git-control-tower/api/`
- UI TODO markers in `scenarios/git-control-tower/ui/src/`
- Files touched by HTTP client dedup, test refactoring, and scanner abstraction phases
- Top 5 largest non-client/non-test files for structural splitting (R3-d2): `main.go`, `git_runner.go`, `diff_service.go`, `visual_capture_storage.go`, `agent_manager_model.go`

**Out of scope:**
- Functional behavior changes — all refactors must preserve existing behavior
- Adding new features or endpoints
- Changes to `packages/` or other scenarios
- UI refactoring beyond removing TODO markers
- Large files beyond the explicit top-5 split list (other files not touched by any phase remain as-is)

## Current Technical Context

### Full File Inventory (top 20 by size)

| File | Lines | Primary Issue | In Scope? |
|------|-------|---------------|-----------|
| `api/main.go` | 1,363 | Massive entry point | Yes — Phase 5 split |
| `api/review_handler_test.go` | 1,355 | Huge test file | Yes — Phase 2 test refactor |
| `api/git_runner.go` | 1,149 | Large service file | Yes — Phase 5 split |
| `api/git_runner_fake_test.go` | 997 | Large test fake | Yes — Phase 2 test refactor |
| `api/diff_service_test.go` | 794 | Test duplication | Yes — Phase 2 test refactor |
| `api/agent_manager_client_test.go` | 779 | Test duplication | Yes — Phase 2 test refactor |
| `api/diff_service.go` | 763 | Large service | Yes — Phase 5 split |
| `api/visual_capture_storage.go` | 649 | Large storage layer | Yes — Phase 5 split |
| `api/visual_capture_service_test.go` | 634 | Test duplication | Yes — Phase 2 test refactor |
| `api/agent_manager_model.go` | 607 | Large model file | Yes — Phase 5 split |
| `api/review_handler.go` | 530 | Large handler | No — not touched by any phase |
| `api/visual_capture_handler.go` | 501 | Large handler | No |
| `api/visual_capture_storage_test.go` | 486 | Test duplication | Yes — Phase 2 test refactor |
| `api/agent_manager_handler.go` | 472 | Large handler | No |
| `api/credentials_service.go` | 451 | Large service | No |
| `api/gitignore_health_service_test.go` | 441 | Test duplication, complexity 8 | Yes — Phase 2 test refactor |
| `api/file_service.go` | 412 | Large service | No |
| `api/visual_capture_service.go` | 411 | Large service | No |
| `api/sync_status_service_test.go` | 407 | Test duplication | Yes — Phase 2 test refactor |
| `api/agent_manager_client.go` | 397 | HTTP client with caching | Yes — Phase 1 client dedup |

### HTTP Client Duplication Analysis

**Common methods duplicated across all 5 clients:**
- `resolveBaseURL(ctx) (string, error)` — identical except service name string
- `doGet(ctx, path, result) error` — identical pattern
- `doJSON(ctx, path, body, result) error` — identical pattern
- `parseXxxError(resp) error` — identical except error prefix string

**Client-specific variants (R2-d1 — variants stay on child):**
- `AgentManagerClient.resolveBaseURL()` — overrides base to add mutex-based URL caching
- `AuditorClient.doJSONAccept(ctx, path, body, result, acceptStatuses ...int)` — thin wrapper on child
- `BrowserAutomationClient.doRaw(ctx, path) ([]byte, string, error)` — fetches binary data, stays on child

### Scanner Duplication

Both `golang/scanner.go` (60 lines) and `python/scanner.go` (58 lines) share identical:
- Struct definition and constructor
- `Scan()` method structure (context cancellation, result initialization, import loop)
- Only differ in: extensions list, match function names, and which import field to extract

### UI TODO Markers

1. **App.tsx:1320** — `TODO: Use lineNumber to scroll to the specific line in the source viewer`
2. **ScenarioReviewPanel.tsx:665** — `TODO: Remove this client-side fallback once the unified review endpoint is stable`

## All Settled Decisions

| Round | ID | Decision | Choice | Detail |
|-------|-----|----------|--------|--------|
| R1 | d1 | HTTP client extraction approach | Embedded struct composition | BaseClient struct with shared methods, service clients embed it |
| R1 | d2 | Test refactoring scope | Refactor all test files with >50% duplication | Comprehensive approach |
| R1 | d3 | Scanner duplication resolution | Extract shared ScanImports helper | Shared function, not complex interface |
| R1 | d4 | Tech debt marker handling | Implement TODO functionality and remove | With endpoint stability check for ScenarioReviewPanel |
| R2 | d1 | BaseClient variant handling | Common methods only; variants on child | doGet/doJSON/resolveBaseURL/parseError on base; doJSONAccept/doRaw/cached URL on children |
| R2 | d2 | Scope of long_files beyond client/test | **Overridden by R3-d2** | Originally limited scope; now expanded |
| R2 | d3 | ScenarioReviewPanel TODO | Verify endpoint stability first | Remove fallback only if confirmed; otherwise remove TODO comment and file separate item |
| R2 | d4 | Long file threshold | Run tidiness-manager in Phase 0 | Get exact threshold and full violation list before refactoring |
| R3 | d1 | Test helper organization | Hybrid approach | Shared `test_helpers_test.go` for cross-cutting concerns (mock HTTP server, assertion helpers) + per-domain helper files co-located with tests |
| R3 | d2 | Long_files target handling | Expand scope to split top 5 largest files | Add main.go, git_runner.go, diff_service.go, visual_capture_storage.go, agent_manager_model.go to refactoring scope |
| R3 | d3 | Phase execution order | Sequential with metrics checkpoints | Metrics check after Phase 1 and after Phase 2 to validate approach early |

## Target End State

- Code quality score ≥70
- complex_functions < 10
- long_files < 10
- tech_debt_markers = 0
- All existing tests pass with no behavioral changes
- Shared HTTP client base eliminates duplication across 5 client files
- Test helpers reduce boilerplate in test files
- Scanner abstraction eliminates duplication between language scanners
- Top 5 largest non-test/non-client files split into cohesive smaller modules

## Implementation Strategy

### Phase 0: Gather Metrics Baseline
Run `tidiness-manager recommend-refactors git-control-tower --limit 50 --format json` to get:
- Exact line count threshold for long_files violations
- Complete list of all violating files with their scores
- Complexity scores per function

This data informs which files in subsequent phases need the most attention and validates whether the approach can hit targets.

### Phase 1: Shared HTTP Client Base
**Approach:** Embedded struct composition (R1-d1) with common-only base (R2-d1).

Create `api/httpclient.go` with:
```go
type BaseClient struct {
    httpClient  *http.Client
    serviceName string
    resolver    discovery.Resolver
}
```

**BaseClient provides:** `doGet`, `doJSON`, `resolveBaseURL`, `parseError` (with configurable service name prefix).

**Variant handling (R2-d1):**
- `AuditorClient`: `doJSONAccept` stays on child as thin wrapper calling `BaseClient.doJSON` internally with status check
- `BrowserAutomationClient`: `doRaw` stays on child
- `AgentManagerClient`: overrides `resolveBaseURL` to add mutex caching

**Files affected:**
- New: `api/httpclient.go`
- Modified: `tidiness_manager_client.go`, `test_genie_client.go`, `auditor_client.go`, `browser_automation_client.go`, `agent_manager_client.go`

**Expected impact:** Eliminates ~15-20 complex_functions (3-4 duplicated methods × 5 clients).

**Metrics checkpoint:** After Phase 1, run `tidiness-manager recommend-refactors git-control-tower --limit 50 --format json` and `go test ./... -timeout 300s` to validate the approach is reducing violations and tests still pass (R3-d3).

### Phase 2: Test Helpers & Table-Driven Tests
**Approach:** Refactor all test files with >50% duplication (R1-d2).

**Test helper organization (R3-d1 — hybrid approach):**
- **Shared `test_helpers_test.go`** — Cross-cutting utilities: mock HTTP server setup, common assertion helpers, shared test fixtures
- **Per-domain helper files** — Co-located with their test files: e.g., `gitignore_health_test_helpers_test.go` next to `gitignore_health_service_test.go` for domain-specific setup, fixtures, and fake builders

**Priority order by file size:**
1. `api/review_handler_test.go` (1,355 lines) — largest test file
2. `api/git_runner_fake_test.go` (997 lines)
3. `api/diff_service_test.go` (794 lines)
4. `api/agent_manager_client_test.go` (779 lines)
5. `api/visual_capture_service_test.go` (634 lines)
6. `api/visual_capture_storage_test.go` (486 lines)
7. `api/gitignore_health_service_test.go` (441 lines, complexity 8)
8. `api/sync_status_service_test.go` (407 lines)
9. `api/ssh/service_test.go` (152 lines) — subtests, extract `newTestDeps()`
10. All other test files exceeding the threshold (per Phase 0 data)

**Techniques:**
- Convert repeated test functions to table-driven subtests
- Extract shared setup into helper functions
- Use `t.Helper()` for clear error attribution

**Metrics checkpoint:** After Phase 2, run `tidiness-manager recommend-refactors git-control-tower --limit 50 --format json` and `go test ./... -timeout 300s` to confirm progress toward targets (R3-d3).

### Phase 3: Scanner Abstraction
**Approach:** Extract shared ScanImports helper (R1-d3).

Create a shared function in `api/filerelations/languages/`:
```go
func ScanImports(ctx context.Context, content string, matchers []MatchFunc) (*scanner.ScanResult, error)
```

Both `golang/scanner.go` and `python/scanner.go` call this with their language-specific matchers.

**Files affected:**
- New: `api/filerelations/languages/scan_imports.go` (or similar)
- Modified: `golang/scanner.go`, `python/scanner.go`

### Phase 4: Tech Debt Markers
**App.tsx:1320** — Implement scroll-to-line feature using the existing lineNumber prop.

**ScenarioReviewPanel.tsx:665** — Per R2-d3: verify that the unified review endpoint (`review_readiness.go`) covers all cases the fallback handles. If confirmed stable, remove the fallback code. If not stable or unclear, remove only the TODO comment and file a separate backlog item for the fallback removal.

### Phase 5: Split Top 5 Largest Non-Client/Non-Test Files (R3-d2)
**Approach:** Structural splitting of the 5 largest files not addressed by Phases 1-2 to bring long_files count under 10.

**Target files and splitting strategy:**
1. **`api/main.go` (1,363 lines)** — Extract route registration, middleware setup, and server configuration into separate files (e.g., `routes.go`, `middleware.go`, `server.go`). Keep `main()` as a thin entry point.
2. **`api/git_runner.go` (1,149 lines)** — Split by responsibility: separate git operations into logical groups (e.g., `git_runner_clone.go`, `git_runner_commit.go`, `git_runner_diff.go` or by domain boundaries discovered during refactoring).
3. **`api/diff_service.go` (763 lines)** — Extract diff computation helpers, formatting logic, or sub-services into companion files.
4. **`api/visual_capture_storage.go` (649 lines)** — Separate storage backends or helper functions into their own files.
5. **`api/agent_manager_model.go` (607 lines)** — Split by model domain or extract validation/serialization logic.

**Constraint:** All splits must stay within the same Go package. No new packages, no import cycle risk. Each resulting file should be under the long_files threshold (determined in Phase 0).

**Verification:** After each file split, run `go build ./...` and `go test ./... -timeout 300s` to confirm no regressions.

### Post-Refactor: Final Verification
1. Run `go test ./... -timeout 300s` to verify all tests pass
2. Run `golangci-lint run` to check for new lint issues
3. Run `gofumpt -w .` to ensure consistent formatting
4. Run `tidiness-manager recommend-refactors git-control-tower --limit 50 --format json` to verify metric improvements
5. Run `git-control-tower review-run git-control-tower --json` to confirm score ≥70
6. Run `vrooli scenario restart git-control-tower` to validate the scenario starts cleanly

## Contract Decisions

- No API or CLI contract changes — this is purely internal refactoring
- All existing tests must continue to pass
- No new dependencies

## Testing Plan

1. **Per-file verification:** Run `go test ./...` after refactoring each file to catch regressions immediately
2. **Per-phase verification:** Run `golangci-lint run` after each phase to catch new lint issues
3. **Metrics checkpoints (R3-d3):** Run `tidiness-manager recommend-refactors git-control-tower --limit 50 --format json` after Phase 1 and after Phase 2 to validate approach early
4. **Format check:** Run `gofumpt -w .` after all Go changes
5. **Final metrics:** Run `tidiness-manager recommend-refactors git-control-tower --limit 50 --format json` after all phases
6. **Final score:** Run `git-control-tower review-run git-control-tower --json` to confirm score ≥70
7. **Scenario health:** Run `vrooli scenario restart git-control-tower` to validate clean startup

## Rollout/Validation Checklist

- [ ] Phase 0 complete — metrics baseline gathered, full violation list known, threshold confirmed
- [ ] Phase 1 complete — shared HTTP client base created, all 5 clients refactored, tests pass
- [ ] Phase 1 metrics checkpoint — violation counts trending down
- [ ] Phase 2 complete — test helpers created, table-driven tests converted, tests pass
- [ ] Phase 2 metrics checkpoint — complex_functions and long_files significantly reduced
- [ ] Phase 3 complete — scanner abstraction in place, tests pass
- [ ] Phase 4 complete — tech debt markers resolved (implemented or tracked separately)
- [ ] Phase 5 complete — top 5 large files split, each under threshold, tests pass
- [ ] All tests pass (`go test ./... -timeout 300s`)
- [ ] Code quality score ≥70 confirmed via GCT review
- [ ] `vrooli scenario restart git-control-tower` succeeds cleanly

## Risks & Mitigations

| ID | Risk | Likelihood | Impact | Mitigation |
|----|------|-----------|--------|------------|
| R1 | long_files still >10 after all phases | Medium | Medium | Phase 5 targets top 5 non-client/non-test files. If still >10, file a follow-up item for remaining violations. Phase 0 metrics data will inform whether this is likely. |
| R2 | Refactoring breaks subtle test behavior | Medium | Medium | Run tests after each file change, not just per phase |
| R3 | HTTP client base introduces new coupling | Low | Medium | Use composition (embedding), not inheritance; variants stay on child |
| R4 | Scanner abstraction over-engineered for 2 languages | Medium | Low | Keep it simple — shared function, not complex interface hierarchy |
| R5 | ScenarioReviewPanel fallback removal breaks UI | Medium | High | R2-d3: verify endpoint stability before removing; file separate item if not stable |
| R6 | Phase 5 file splits introduce import cycles | Low | High | All splits stay within the same Go package — no new packages |
| R7 | Phase 5 file splits create hard-to-navigate code | Medium | Low | Split along clear domain/responsibility boundaries, not arbitrary line counts |

## Non-goals / Prohibited Patterns

- Do NOT add new features or change API behavior
- Do NOT introduce new external dependencies
- Do NOT create deeply nested abstraction hierarchies
- Do NOT change test assertions to make refactored code pass — fix the code instead
- Do NOT split files into packages that create import cycles
- Do NOT add backward compatibility shims, feature flags, or migration bridges
- Do NOT create new Go packages for Phase 5 splits — all files stay in their current package

## Definition of Done

- [ ] Code quality score ≥70 (verified via `git-control-tower review-run git-control-tower --json`)
- [ ] complex_functions < 10
- [ ] long_files < 10
- [ ] tech_debt_markers = 0
- [ ] All existing tests pass without modification to assertions
- [ ] `vrooli scenario restart git-control-tower` succeeds
- [ ] Greenfield constraint honored — no compatibility shims or migration code
