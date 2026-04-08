# Implementation Plan: Improve deployment-manager Code Quality

## Purpose

Reduce 151 code quality violations (score 0) in the deployment-manager scenario to achieve a GCT score >= 70. Primary violation categories: complex_functions (32), long_files (21), lint_issues (17).

## Greenfield Constraint

This is a pure refactoring task — no backward-compatibility shims, no feature flags, no renamed `_unused` variables. All changes are internal restructuring. If a function signature needs to change for cleanliness, change all call sites directly.

## Required Reading

```bash
prompt-manager skill read refactor seam-discovery-and-enforcement
prompt-manager skill read cli-steer api-steer react-coherence
```

## Problem Statement

The deployment-manager scenario has a GCT code quality score of 0 with 151 violations across three main categories:

1. **Complex functions (32)**: Several functions exceed reasonable complexity thresholds. The worst offender is `DeployDesktop` in `orchestrator.go` at ~515 lines — a monolithic handler managing the entire deployment workflow.
2. **Long files (21)**: Multiple files exceed 600 lines, with `orchestrator.go` (1,364 lines), `prerequisites.go` (1,003 lines), and `profiles/commands.go` (831 lines) being the worst.
3. **Lint issues (17)**: Missing golangci-lint configuration for Go code; TypeScript has CRITICAL comments missing in ESLint config.

## Scope

### In Scope
- Refactoring complex Go functions in API and CLI into smaller, well-named helpers
- Splitting long files into focused modules
- Adding `.golangci.yml` configuration for Go linting (funlen, cyclop, gocognit — minimal config targeting GCT violation categories)
- Adding CRITICAL comments to ESLint config
- Maintaining all existing behavior and passing tests
- Targeting the highest-impact violations to reach score >= 70 (not necessarily all 151)

### Out of Scope
- Adding new features or changing API contracts
- Migrating frameworks or build tools
- Addressing test failures (docs, playbooks, UI build) — those are separate items
- Rewriting the UI architecture
- Fixing all 151 violations in one pass — focus on biggest offenders first
- **TypeScript `as` type cast fixes** — these are mostly in generated shadcn/ui components and are lower-weight in GCT scoring; Go refactoring alone should reach the target

## Current Technical Context

### Worst Offenders by Category

**Complex Functions (API):**
| Function | File | Lines |
|----------|------|-------|
| `DeployDesktop` | `api/deployments/orchestrator.go` | ~515 |
| `ExportBundle` | `api/bundles/handler.go` | ~143 |
| `stageBundleArtifacts` | `api/bundles/handler.go` | ~84 |

**Complex Functions (CLI):**
| Function | File | Lines |
|----------|------|-------|
| `DeployDesktop` | `cli/deployments/commands.go` | ~141 |
| `Build` | `cli/deployments/commands.go` | ~116 |

**Complex Functions (UI):**
| Component | File | Lines |
|-----------|------|-------|
| `GuidedFlow` | `ui/src/components/GuidedFlow.tsx` | ~513 |
| `BundleTelemetry` | `ui/src/pages/BundleTelemetry.tsx` | ~442 |
| `Analyze` | `ui/src/pages/Analyze.tsx` | ~414 |
| `ProfileDetail` | `ui/src/pages/ProfileDetail.tsx` | ~412 |

**Long Files (prioritized — fix these first, then tackle complex functions by size descending):**
| File | Lines |
|------|-------|
| `api/deployments/orchestrator.go` | 1,364 |
| `api/codesigning/validation/prerequisites.go` | 1,003 |
| `cli/profiles/commands.go` | 831 |
| `api/bundles/handler.go` | 748 |
| `cli/deployments/commands.go` | 725 |
| `api/deployments/desktop_client.go` | 651 |
| `cli/signing/commands.go` | 621 |
| `ui/src/components/GuidedFlow.tsx` | 513 |

### Lint Configuration Status
- **Go**: No `.golangci.yml` — only basic `go vet` runs
- **TypeScript**: ESLint config exists but missing CRITICAL comments; `as` type casts in UI components (12 files) — **decided to skip `as` cast fixes** (generated shadcn code, lower GCT weight)

### Codebase Structure
- **API**: 15 packages, 33 test files, well-structured with clear package boundaries
- **CLI**: 9 command packages, 10 test files, each with commands.go + test
- **UI**: React/TypeScript with shadcn components, 10 component files, 13 pages

## Settled Decisions

### Round 1
1. **Orchestrator refactoring**: Extract `DeployDesktop` into sequential phase functions within the same file (not separate files). Phases: validate, sign, assemble, build, publish.
2. **Prerequisites splitting**: Split by platform into separate files without build tags (`prerequisites_windows.go`, `prerequisites_macos.go`, `prerequisites_linux.go`). All compile everywhere for cross-platform validation.
3. **Score target**: Fix top violations to reach score >= 70, then stop. Remaining minor violations addressed incrementally.
4. **UI GuidedFlow**: Extract each step into its own file in a `GuidedFlow/` directory. Parent component becomes thin state coordinator.

### Round 2
5. **Violation prioritization**: Fix all long_files first (by line count descending), then complex_functions by size descending. Long files are often the root cause of complex functions — splitting files naturally breaks up functions. Start with orchestrator.go (1,364 lines) which has both the longest file AND most complex function.
6. **TypeScript `as` casts**: Skip `as` cast fixes entirely. Focus lint effort on `.golangci.yml` and CRITICAL comments only. These are mostly in generated shadcn components and are lower-weight in GCT scoring. If Go refactoring alone gets to >= 70, no UI lint changes needed beyond CRITICAL comments.

## Target End State

- GCT code quality score >= 70
- Highest-impact long_files violations resolved (files over 600 lines split)
- Highest-impact complex_functions violations resolved (top ~15-20 functions)
- `.golangci.yml` added with funlen + cyclop + gocognit linters
- CRITICAL comments added to ESLint config
- All existing passing tests still pass
- No behavioral changes to API, CLI, or UI

## Implementation Strategy

**Prioritization order**: Fix long_files first (by line count descending), since splitting long files naturally reduces function complexity. Then address remaining complex_functions by size descending.

### Phase 1: Go Lint Infrastructure
- Add `.golangci.yml` with minimal complexity-focused linters: funlen, cyclop, gocognit
- Run `golangci-lint run` to get a baseline violation list
- Fix `gofumpt` formatting issues
- **Checkpoint**: Run GCT to establish post-lint-config baseline

### Phase 2: API Refactoring (Highest Impact)
Tackle long files first, then remaining complex functions:
- **orchestrator.go (1,364 lines)**: Extract `DeployDesktop` into phased sub-functions within the same file:
  - `deployValidate()` — input validation and prerequisite checks
  - `deploySign()` — code signing orchestration
  - `deployAssemble()` — manifest assembly
  - `deployBuild()` — build execution per platform
  - `deployPublish()` — publishing and notification
- **prerequisites.go (1,003 lines)**: Split into `prerequisites_windows.go`, `prerequisites_macos.go`, `prerequisites_linux.go` without build tags
- **bundles/handler.go (748 lines)**: Extract `ExportBundle` and `stageBundleArtifacts` into smaller helpers
- **desktop_client.go (651 lines)**: Separate polling logic from API wrappers
- **Checkpoint**: Run GCT — if score >= 70, consider stopping early

### Phase 3: CLI Refactoring
- **profiles/commands.go (831 lines)**: Group related commands into separate files (CRUD, analysis, versioning)
- **deployments/commands.go (725 lines)**: Extract payload building and output formatting helpers
- **signing/commands.go (621 lines)**: Extract common signing patterns
- **Checkpoint**: Run GCT — if score >= 70, stop

### Phase 4: UI Refactoring (only if score < 70 after Phase 3)
- **GuidedFlow.tsx (513 lines)**: Extract step components into `GuidedFlow/` directory (`StepProfileSelect.tsx`, `StepReadiness.tsx`, `StepExport.tsx`, `StepIssues.tsx`), plus `GuidedFlow/index.tsx` as thin coordinator
- **Large page components**: Extract reusable sub-components from BundleTelemetry, Analyze, ProfileDetail (only if needed for score)
- Add CRITICAL comments to ESLint config
- **No `as` cast fixes** — skip per settled decision

### Phase 5: Validation
- Re-run GCT code quality review
- Verify all tests pass
- Confirm no behavioral regressions

## Contract Decisions

- No API contract changes — all refactoring is internal
- No CLI flag or output format changes
- No UI workflow changes — only component extraction

## Testing Plan

- Run `go build ./...` in both `api/` and `cli/` after Go changes to catch compilation errors
- Run `go test ./... -timeout 300s` after each phase to ensure no regressions
- Run `golangci-lint run` after Go changes to verify lint compliance
- Run `pnpm lint` in `ui/` after TypeScript changes (if Phase 4 is reached)
- Re-run GCT review after each phase to check if score >= 70 (stop early if target met)

## Rollout/Validation Checklist

- [ ] Phase 1 complete: `.golangci.yml` added, `golangci-lint run` shows baseline; GCT checkpoint
- [ ] Phase 2 complete: `go build ./...` and `go test ./...` pass in api/; GCT checkpoint
- [ ] Phase 3 complete: `go build ./...` and `go test ./...` pass in cli/; GCT checkpoint — stop if >= 70
- [ ] Phase 4 complete (if needed): `pnpm lint` passes in ui/; GCT checkpoint
- [ ] Phase 5 complete: GCT score >= 70 confirmed
- [ ] `vrooli scenario restart deployment-manager` — verify scenario starts cleanly

## Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Refactoring introduces subtle behavioral changes | Medium | High | Run full test suite after each file change |
| Go build tags needed for platform-specific files | Low | Medium | Decided: no build tags, plain file split |
| Large number of files changed makes review difficult | Medium | Medium | Phase changes by component area |
| Score >= 70 not reached after Go refactoring alone | Low | Medium | GCT checkpoints after each phase; extend to UI if needed |
| Skipping `as` cast fixes leaves lint_issues unresolved | Low | Low | Lower GCT weight; CRITICAL comments still addressed |

## Non-goals / Prohibited Patterns

- Do not add new abstractions that aren't justified by current complexity
- Do not change function signatures on exported functions unless necessary for extraction
- Do not introduce new dependencies
- Do not "fix" working test assertions to match refactored code
- Do not address the 3 failing test phases (docs, playbooks, performance) — those are separate items
- Do not add compatibility shims or feature flags
- Do not fix TypeScript `as` type casts in shadcn/ui components

## Definition of Done

- [ ] GCT code quality score >= 70
- [ ] Highest-impact long_files violations resolved (files over 600 lines split)
- [ ] Highest-impact complex_functions violations resolved
- [ ] `.golangci.yml` added with funlen + cyclop + gocognit
- [ ] All existing passing tests still pass
- [ ] `go build ./...` succeeds for API and CLI
- [ ] `golangci-lint run` passes with new config
- [ ] No behavioral changes to API, CLI, or UI
- [ ] `vrooli scenario restart deployment-manager` succeeds
