# Implementation Plan: Reduce Code Quality Violations in swarm-manager

## 1. Purpose

Reduce the swarm-manager code quality score from 0/100 to >=70 by systematically refactoring files with high complexity, excessive length, and code duplication. Eliminate all tech debt markers.

## 2. Required Reading

```bash
prompt-manager skill read refactor code-cleanup implementation-plan-authoring
prompt-manager skill read cli-steer api-steer
```

## 3. Problem Statement

The swarm-manager scenario has a code quality score of 0/100 with 379 violations:
- **107 complex functions** — functions with high cyclomatic complexity
- **48 long files** — files exceeding length thresholds
- **3 tech debt markers** — TODO/FIXME/HACK comments
- **1 lint issue**

The top 20 files by refactor priority show heavy code duplication (up to 100%) and structural repetition, particularly in test files and the eventlog emitter.

## 4. Scope

### In Scope
- All Go source files under `scenarios/swarm-manager/api/`
- Refactoring to reduce complexity, length, and duplication
- Removing tech debt markers (TODOs, FIXMEs, HACKs)
- Fixing lint issues

### Out of Scope
- Changing external API contracts or behavior
- Modifying proto definitions
- Changes outside `scenarios/swarm-manager/`
- Adding new features

## 5. Current Technical Context

### Violation Categories by Priority

**Tier 1 — Highest leverage (complexity + duplication + length):**
| File | Lines | Complexity Max | Duplication % |
|---|---|---|---|
| `api/internal/initiatives/files.go` | 312 | 35 | 57% |
| `api/internal/backlog/files.go` | 292 | 35 | 53% |
| `api/internal/initiatives/handler.go` | 338 | 12 | 57% |
| `api/internal/backlog/batch_queue_handler_test.go` | 627 | 10 | 52% |
| `api/internal/backlog/handler_update_test.go` | 472 | 8 | 67% |

**Tier 2 — High duplication (near-identical code across packages):**
| File | Lines | Duplication % | Pattern |
|---|---|---|---|
| `api/internal/backlog/encode_error_test.go` | 39 | 100% | Identical errorWriter + test |
| `api/internal/queue/encode_error_test.go` | 39 | 100% | Identical errorWriter + test |
| `api/internal/settings/helpers_test.go` | 13 | 100% | Identical helper pattern |
| `api/internal/queue/store_default_test.go` | 13 | 100% | Identical helper pattern |
| `api/internal/eventlog/emitter.go` | 244 | 95% | Repetitive thin wrappers |

**Tier 3 — Moderate duplication in tests:**
| File | Lines | Duplication % |
|---|---|---|
| `api/internal/settings/normalize_test.go` | 136 | 89% |
| `api/internal/backlog/validation_test.go` | 100 | 77% |
| `api/internal/backlog/blocking_test.go` | 265 | 52% |

**Tech debt marker (1 instance):**
- `api/internal/stats/engine.go:289` — `// TODO: implement follow-up tracking once execution history is available.`

### Key Observations
1. **`files.go` duplication**: `initiatives/files.go` and `backlog/files.go` likely share the same file-serving logic (complexity_max=35 each). This is the single highest-leverage refactoring target.
2. **`errorWriter` test helper**: Identical across `backlog` and `queue` packages — should be extracted to a shared `testutil` package.
3. **`emitter.go` repetition**: 42 functions at 244 lines, 95% duplication. Each emit method follows the same 1-3 line pattern. Could potentially use code generation or a more generic approach, but may be intentional for type safety.
4. **Test table duplication**: `normalize_test.go` has 6 nearly identical table-driven test functions that could share a generic clamp-test helper.
5. **`handler_update_test.go`**: 472 lines with repetitive test setup (create item, build request, send PATCH, assert). Setup could be extracted into a test helper.

## 6. Target End State

- Code quality score >= 70
- complex_functions < 20
- long_files < 10
- tech_debt_markers = 0
- All existing tests continue to pass
- No behavioral changes to APIs

## 7. Implementation Strategy

### Phase 1: Extract shared utilities (highest leverage)
1. **Extract shared file-serving logic** from `backlog/files.go` and `initiatives/files.go` into a common package (likely `internal/fileserve` or extending `httputil`). This addresses 2 files with complexity_max=35 each.
2. **Extract shared test helpers** (`errorWriter`, common assertion patterns) into `internal/testutil` or existing test helper files.

### Phase 2: Reduce complexity in handlers
1. **Simplify `initiatives/handler.go`** (338 lines, complexity 12) — extract sub-handlers or helper functions.
2. **Simplify `scenarios/files.go`** (118 lines, complexity 9) — similar file-serving pattern.
3. **Address `execution/handler_lifecycle.go`** (149 lines, complexity 6).

### Phase 3: Reduce test file length and duplication
1. **Refactor `batch_queue_handler_test.go`** (627 lines) — extract shared test setup, use table-driven tests.
2. **Refactor `handler_update_test.go`** (472 lines) — extract `updateTestCase` helper pattern.
3. **Consolidate `normalize_test.go`** — generic clamp-test helper for repeated table patterns.
4. **Refactor `blocking_test.go`** (265 lines) — reduce duplication.

### Phase 4: Cleanup
1. **Remove tech debt marker** in `stats/engine.go:289`.
2. **Address the eventlog `emitter.go`** duplication — decide on approach (keep for type safety vs. consolidate).
3. **Run `gofumpt -w .`** to ensure consistent formatting.
4. **Run `golangci-lint run`** to catch remaining lint issues.

### Phase 5: Verify
1. Run `go test ./...` to confirm all tests pass.
2. Run `git-control-tower review-run swarm-manager --json` to verify score improvement.
3. Run `tidiness-manager recommend-refactors swarm-manager --limit 10` to confirm top violations are addressed.

## 8. Contract Decisions

- No API contract changes
- No proto changes
- All refactoring is internal structural improvement only

## 9. Testing Plan

- All existing tests must continue to pass after each phase
- Run `go test ./... -timeout 300s` after each phase
- No new test coverage required (this is refactoring, not new behavior)
- Final validation via GCT review-run

## 10. Rollout/Validation Checklist

- [ ] Phase 1: Shared utilities extracted, tests pass
- [ ] Phase 2: Handler complexity reduced, tests pass
- [ ] Phase 3: Test files shortened, tests pass
- [ ] Phase 4: Tech debt markers removed, lint clean
- [ ] Phase 5: GCT score >= 70, all targets met

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Refactoring breaks existing tests | Medium | High | Run tests after every change, commit frequently |
| Shared file-serving extraction introduces bugs | Medium | High | Keep existing tests as regression suite |
| Emitter consolidation loses type safety | Low | Medium | Decision needed: keep explicit methods vs. generic approach |
| Score doesn't reach 70 after top files addressed | Low | Medium | Iterate on remaining files if needed |

## 12. Non-goals / Prohibited Patterns

- Do NOT change any API behavior or contracts
- Do NOT add new features during refactoring
- Do NOT modify proto definitions
- Do NOT refactor files outside `scenarios/swarm-manager/`
- Do NOT add unnecessary abstractions — only extract when genuinely reducing duplication

## 13. Definition of Done

- [ ] Code quality score >= 70 (via `git-control-tower review-run swarm-manager`)
- [ ] complex_functions < 20
- [ ] long_files < 10
- [ ] tech_debt_markers = 0
- [ ] All existing tests pass (`go test ./... -timeout 300s`)
- [ ] Code formatted with `gofumpt`
- [ ] No lint issues from `golangci-lint run`
