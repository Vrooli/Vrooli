# Implementation Plan: Improve scenario-to-desktop Code Quality

> **Greenfield declaration:** This is a pure refactoring task with no backward compatibility constraints. All changes are internal structural improvements — no API contracts, CLI interfaces, or external behavior changes.

## Purpose

Reduce 500 code quality violations (105 long_files, 100 complex_functions, 24 lint_issues) to achieve a passing GCT code quality score for the scenario-to-desktop scenario.

## Required Reading

```bash
prompt-manager skill read refactor seam-discovery-and-enforcement test
prompt-manager skill read implementation-plan-authoring
```

## Problem Statement

The scenario-to-desktop codebase (~106K lines Go, ~51K lines TS/TSX across source files) has a GCT code quality score of 0 with 500 violations:
- **long_files (105):** 46 Go files >500 lines, 24 TS/TSX files >500 lines, plus test files
- **complex_functions (100):** Functions with high cyclomatic complexity in orchestrator, service, and component files
- **lint_issues (24):** Go linting completely unconfigured (no `.golangci.yml`); frontend ESLint already configured

Key offenders by line count:
| File | Lines | Issue |
|------|-------|-------|
| `api/smoketest/service_test.go` | 2,433 | Largest test file |
| `api/shared/errors/errors.go` | 1,874 | Data-heavy error definitions |
| `cli/pipeline/commands.go` | 1,581 | CLI command definitions |
| `runtime/supervisor_test.go` | 1,544 | Large test file |
| `api/preflight/service_test.go` | 1,535 | Large test file |
| `api/pipeline/orchestrator_test.go` | 1,506 | Large test file |
| `api/smoketest/service.go` | 1,163 | 33 functions |
| `api/smoketest/mocks/mocks.go` | 1,035 | Auto-generated mocks |
| `ui/src/components/generator/GeneratorForm.tsx` | 979 | Monolithic form component |
| `api/pipeline/orchestrator.go` | 898 | 28 functions, core state machine |
| `ui/src/store/pipelineStore.ts` | 878 | Store with polling/rate limiting |
| `api/signing/validation/prerequisites.go` | 789 | Validation logic |
| `api/preflight/service.go` | 782 | Service logic |
| `api/pipeline/handler.go` | 752 | Handler with many routes |
| `api/pipeline/types.go` | 736 | Type definitions |

## Scope

**In scope:**
- `scenarios/scenario-to-desktop/**` (api, cli, runtime, ui)
- File splitting, function extraction, complexity reduction
- Go lint configuration setup
- Preserving all existing behavior and passing tests

**Out of scope:**
- Feature changes or new functionality
- Framework migrations or architectural rewrites
- Changes outside `scenarios/scenario-to-desktop/`
- Test coverage expansion (unless needed to maintain coverage during splits)

## Current Technical Context

- **Backend:** Go 1.24, sub-packages under `api/` (pipeline, smoketest, preflight, signing, tasks, etc.), 282 .go files (~106K lines)
- **CLI:** Go CLI layer in `cli/`, pipeline command definitions
- **Runtime:** Go runtime layer in `runtime/`, supervisor and service launcher, 60 .go files (~16K lines)
- **Frontend:** React + TypeScript, Zustand stores, custom hooks, 262 TS/TSX source files (~51K lines)
- **Build:** Makefile orchestration, vitest for TS, `go test` with race detection
- **Lint:** ESLint configured for frontend; **no golangci-lint config for Go** (major gap)
- **Tests:** 11 passing tests, 0 failures as of last run

## Target End State

- GCT code quality score ≥ 1 (passing)
- Violation count significantly reduced from 500
- All existing tests still pass
- No behavioral changes

## Implementation Strategy

**Approach:** Batch by violation type. Process all violations of one type before moving to the next. Run tests after each phase.

### Phase 1: Establish Go Linting
- Create `.golangci.yml` with moderate strictness: go vet, errcheck, unused, gocritic, gocyclo, funlen, revive
  - gocyclo and funlen directly target the GCT violation categories (complex_functions, long_files)
  - Set pragmatic thresholds: funlen max ~500 lines, gocyclo max ~15
- Run initial lint pass to identify and categorize auto-fixable issues
- Fix auto-fixable lint issues first, then address remaining manually
- **Validation:** `golangci-lint run ./...` passes with zero errors

### Phase 2: Address Long Files (105 violations — highest count)

**Splitting threshold:** Mandatory splitting for files over **700 lines**. Files in the 500–700 line range are left for opportunistic cleanup — many will naturally shrink when complex functions are extracted in Phase 3. This focuses effort on the worst offenders and avoids unnecessary churn on borderline files.

**Go files (30+ over 700 lines) — split by cohesive function groups:**
- `api/shared/errors/errors.go` (1,874 lines) → split into domain-specific error files (pipeline_errors.go, smoketest_errors.go, etc.)
- `cli/pipeline/commands.go` (1,581 lines) → split by command group (commands_run.go, commands_status.go, commands_config.go, etc.). Each CLI command is self-contained, making this a clean structural split.
- `api/smoketest/service.go` (1,163 lines) → split by operation (create, run, report, etc.)
- `api/pipeline/orchestrator.go` (898 lines) → split state machine stages into separate files
- `api/signing/validation/prerequisites.go` (789 lines) → split by validation category
- `api/preflight/service.go` (782 lines) → split by preflight check type
- `api/pipeline/handler.go` (752 lines) → split by route group
- `api/pipeline/types.go` (736 lines) → split by domain (request types, response types, internal types)
- Remaining Go files over 700 lines: prioritize by line count descending

**Test files — split + extract helpers:**
- `api/smoketest/service_test.go` (2,433 lines) → split by function under test, extract shared setup into `testutil_test.go`
- `runtime/supervisor_test.go` (1,544 lines) → split by supervisor operation
- `api/preflight/service_test.go` (1,535 lines) → split by preflight scenario
- `api/pipeline/orchestrator_test.go` (1,506 lines) → split by pipeline stage
- Remaining test files over 700 lines: split by logical grouping + extract common helpers into `*_testutil_test.go`

**Generated files:** Skip `api/smoketest/mocks/mocks.go` (auto-generated, 1,035 lines). Exclude from quality metrics.

**React/TypeScript files (10+ over 700 lines) — extract sub-components and custom hooks:**
- `ui/src/components/generator/GeneratorForm.tsx` (979 lines) → decompose into section sub-components (GeneratorFormHeader, GeneratorFormConfig, GeneratorFormActions, etc.) and extract form logic into custom hooks
- `ui/src/store/pipelineStore.ts` (878 lines) → extract polling/rate-limiting logic into separate utility hooks (e.g., `usePolling.ts`, `useRateLimiter.ts`), keep core store slim
- `ui/src/components/signing/SigningPage.tsx` (687 lines) → extract section sub-components (at 687 lines, borderline — split if naturally decomposable)
- `ui/src/components/sections/preflight/PreflightSection.tsx` (624 lines) → borderline, opportunistic split
- `ui/src/lib/api/protoAdapters.ts` (600 lines) → borderline, split by domain if natural groupings exist
- Test files (useScenarioState.test.ts at 1,388 lines, etc.) → split by test scenario

**Validation:** File count over 700-line threshold reduced to zero; `make test` passes after each batch of splits.

### Phase 3: Reduce Function Complexity (100 violations)

**Primary technique: Early returns + guard clauses** to flatten nesting. This is the lowest-risk refactor — no new functions to wire up — and is often sufficient alone to drop cyclomatic complexity below threshold.

**Secondary technique: Extract named helper functions** for cohesive blocks that remain complex after flattening. Use clear, descriptive names that convey intent.

**For state-machine functions** in orchestrator: consider table-driven dispatch only where it genuinely improves clarity (not as a blanket pattern).

- Focus areas: `api/pipeline/orchestrator.go`, `api/smoketest/service.go`, `api/preflight/service.go`, handler files
- **Validation:** `golangci-lint run` with gocyclo threshold ≤15 passes; `make test` passes after each batch

### Phase 4: Validate
- Re-run GCT code quality review
- Confirm all tests pass (`make test`)
- Verify no behavioral regressions
- Run `vrooli scenario restart scenario-to-desktop` to confirm clean startup

## Contract Decisions

No API or CLI contract changes. All refactoring is internal structural improvement. Public function signatures and package exports remain unchanged.

## Testing Plan

- **Phase 1:** `golangci-lint run ./...` passes after config setup and fixes
- **Phase 2:** `make test` after each batch of file splits (every 5-10 files). Verify no import breakage.
- **Phase 3:** `make test` after each batch of complexity reductions. `golangci-lint run` confirms gocyclo compliance.
- **Phase 4:** Full `make test`, GCT re-review, scenario restart verification

## Rollout/Validation Checklist

- [ ] `.golangci.yml` created and `golangci-lint run` passing
- [ ] Long files split below 700-line threshold
- [ ] 500-700 line files reviewed; split opportunistically where Phase 3 extractions didn't reduce them enough
- [ ] Complex functions simplified below gocyclo threshold (≤15)
- [ ] All existing tests pass (`make test`)
- [ ] GCT code quality re-review shows score ≥ 1
- [ ] `vrooli scenario restart scenario-to-desktop` succeeds cleanly

## Risks + Mitigations

| Risk | Mitigation |
|------|-----------|
| File splits break imports | Go packages stay the same (same directory = same package); TS exports maintained via index re-exports if needed |
| Test file splits cause flaky tests | Run full test suite after each split batch |
| Refactoring introduces bugs | No logic changes — only structural moves and extractions |
| Generated files (mocks.go) resist refactoring | Exclude generated files from quality metrics |
| Large number of files to modify (potentially 100+) | Batch by type, validate per batch, prioritize highest-impact files first |
| CLI commands.go split breaks subcommand registration | Verify all CLI commands remain registered after split; Go same-package splitting preserves init() registration |
| React sub-component extraction breaks prop drilling | Keep extracted components in same directory, co-locate with parent; verify UI renders correctly |

## Non-goals / Prohibited Patterns

- Do not change any external behavior or API contracts
- Do not introduce new frameworks or libraries
- Do not add unnecessary abstractions — prefer direct, simple extractions
- Do not "improve" code beyond what reduces violations
- Do not modify generated mock files
- Do not add backward compatibility shims — this is greenfield refactoring

## Definition of Done

- GCT code quality score > 0
- All existing tests pass (`make test` in scenario root)
- No changes outside `scenarios/scenario-to-desktop/**`
- `vrooli scenario restart scenario-to-desktop` succeeds
