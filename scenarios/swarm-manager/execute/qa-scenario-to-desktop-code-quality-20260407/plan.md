# Implementation Plan: Improve scenario-to-desktop Code Quality

## Purpose

Reduce 500 code quality violations (105 long_files, 100 complex_functions, 24 lint_issues) to achieve a passing GCT code quality score for the scenario-to-desktop scenario.

## Required Reading

```bash
prompt-manager skill read refactor seam-discovery-and-enforcement test
prompt-manager skill read implementation-plan-authoring
```

## Problem Statement

The scenario-to-desktop codebase (~152K lines across 604 source files) has a GCT code quality score of 0 with 500 violations:
- **long_files (105):** 46 Go files >500 lines, 24 TS/TSX files >500 lines, plus test files
- **complex_functions (100):** Functions with high cyclomatic complexity in orchestrator, service, and component files
- **lint_issues (24):** Go linting completely unconfigured (no `.golangci.yml`); frontend ESLint already configured

Key offenders:
| File | Lines | Issue |
|------|-------|-------|
| `api/shared/errors/errors.go` | 1,874 | Data-heavy error definitions |
| `api/smoketest/service_test.go` | 2,433 | Largest test file |
| `api/smoketest/service.go` | 1,163 | 33 functions |
| `api/pipeline/orchestrator.go` | 898 | 28 functions, core state machine |
| `ui/src/components/generator/GeneratorForm.tsx` | 979 | Monolithic form component |
| `ui/src/store/pipelineStore.ts` | 878 | Store with polling/rate limiting |

## Scope

**In scope:**
- `scenarios/scenario-to-desktop/**` (api, runtime, ui)
- File splitting, function extraction, complexity reduction
- Go lint configuration setup
- Preserving all existing behavior and passing tests

**Out of scope:**
- Feature changes or new functionality
- Framework migrations or architectural rewrites
- Changes outside `scenarios/scenario-to-desktop/`
- Test coverage expansion (unless needed to maintain coverage during splits)

## Current Technical Context

- **Backend:** Go 1.24, flat package in `api/`, service-per-file pattern, 282 .go files (84K lines)
- **Runtime:** Go runtime layer, 60 .go files (16K lines)
- **Frontend:** React + TypeScript, Zustand stores, custom hooks, 262 TS/TSX files
- **Build:** Makefile orchestration, vitest for TS, `go test` with race detection
- **Lint:** ESLint configured for frontend; **no golangci-lint config for Go** (major gap)

## Target End State

- GCT code quality score ≥ 1 (passing)
- Violation count significantly reduced from 500
- All existing tests still pass
- No behavioral changes

## Implementation Strategy

<!-- TBD — pending decisions on approach (incremental vs. batch), tooling, and prioritization -->

### Phase 1: Establish Go Linting
- Create `.golangci.yml` with pragmatic rules
- Run initial lint pass to identify and categorize issues
- Fix auto-fixable lint issues

### Phase 2: Address Long Files (highest count)
- Use `tidiness-manager recommend-refactors` to prioritize files
- Split large Go files by extracting cohesive function groups into new files within same package
- Split large React components into sub-components
- Split large test files into per-function or per-scenario test files

### Phase 3: Reduce Function Complexity
- Extract helper functions from complex orchestration and service methods
- Use early returns and guard clauses to flatten nesting
- Decompose monolithic React components (GeneratorForm, SigningPage)

### Phase 4: Validate
- Re-run GCT code quality review
- Confirm all tests pass
- Verify no behavioral regressions

## Contract Decisions

<!-- TBD — pending decision on whether to touch API contracts or keep strictly internal -->

No API or CLI contract changes expected. All refactoring is internal structural improvement.

## Testing Plan

- Run `make test` after each phase to confirm no regressions
- Run GCT code quality review after completion to verify score improvement
- Spot-check that split files maintain the same public API surface

## Rollout/Validation Checklist

- [ ] `.golangci.yml` created and lint passing
- [ ] Long files split below threshold
- [ ] Complex functions simplified
- [ ] All existing tests pass
- [ ] GCT code quality re-review shows improvement

## Risks + Mitigations

| Risk | Mitigation |
|------|-----------|
| File splits break imports | Go package stays the same; TS exports maintained via barrel files if needed |
| Test file splits cause flaky tests | Run full test suite after each split |
| Refactoring introduces bugs | No logic changes — only structural moves and extractions |
| Generated files (mocks.go) resist refactoring | Exclude generated files from quality metrics |

## Non-goals / Prohibited Patterns

- Do not change any external behavior or API contracts
- Do not introduce new frameworks or libraries
- Do not add unnecessary abstractions — prefer direct, simple extractions
- Do not "improve" code beyond what reduces violations

## Definition of Done

- GCT code quality score > 0
- All existing tests pass (`make test` in scenario root)
- No changes outside `scenarios/scenario-to-desktop/**`
