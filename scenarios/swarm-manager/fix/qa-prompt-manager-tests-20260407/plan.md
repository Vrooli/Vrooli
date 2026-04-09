# Fix prompt-manager test failures

## Purpose

Resolve all 5 failing test phases in the prompt-manager scenario's GCT test suite, bringing it from 6/11 to 11/11 passing phases. This ensures prompt-manager meets quality gates for deployment readiness.

## Required Reading
- `prompt-manager skill read scientific-debugging` — hypothesis-driven root cause analysis
- `prompt-manager skill read documentation-health` — docs validation patterns and fixes
- `prompt-manager skill read test` — test infrastructure conventions
- `prompt-manager skill read cli-steer` — CLI architecture patterns and unknown-command handling

## Problem Statement

GCT review of the prompt-manager scenario (2026-04-07) shows 5 of 11 test phases failing:

| # | Phase | Failure | Root Cause (Hypothesis) |
|---|-------|---------|------------------------|
| 1 | **standards** | "standards violations exceed fail_on=high (highest=critical)" — 8 blocking, 154 warnings | Missing `cli/prompt-manager` compiled binary (critical); P0 targets in `requirements/index.json` missing linked requirements (4× critical) |
| 2 | **docs** | "Docs validation failed" | Broken links or missing files referenced in `docs/manifest.json` |
| 3 | **unit** | "node unit tests failed in ui: exit status 1" | UI unit test failures — need to run and inspect output |
| 4 | **integration** | "CLI returned success (exit 0) for unknown command" | `cli-core` unknown-command handling returns 0 instead of non-zero |
| 5 | **playbooks** | "registry not found: .../bas/registry.json" | `bas/` directory has `actions/`, `cases/`, `flows/` but no `registry.json` |

### Workshop Round 2 Investigation Findings

Direct investigation on 2026-04-08 revealed:

1. **Standards - CLI binary**: The binary `cli/prompt-manager` now exists (8.8 MB, executable). This was likely built after the GCT run.
2. **Standards - P0 requirements**: All 8 P0 modules in `requirements/index.json` have properly defined requirements in their `module.json` files. This may have been fixed after the GCT run, or the auditor uses a different validation path.
3. **Docs**: All 20 files referenced in `docs/manifest.json` exist on disk. No broken links found.
4. **CLI unknown command**: `cli-core` at `packages/cli-core/cliapp/app.go:115` correctly returns `fmt.Errorf("Unknown command: %s", remaining[0])` and the prompt-manager CLI exits with code 1 for unknown commands. This may have been fixed, or the test runs the binary differently (e.g., via `install.sh` wrapper).
5. **Playbooks registry**: `bas/registry.json` is **confirmed missing**. Other scenarios (test-genie, swarm-manager) have this file auto-generated via `test-genie registry build`.

**Conclusion**: 4 of 5 failures may already be resolved. A fresh test run is needed to confirm current state before implementing fixes.

## Scope

### Acceptance Boundaries
- **acceptance_allow**: `scenarios/prompt-manager/**`
- **acceptance_deny**: not set

**Note**: Round 1 decision d2 selected expanding scope to include `packages/cli-core/**` for the unknown-command fix. However, investigation shows cli-core already handles this correctly. Scope expansion may not be needed — pending test re-run confirmation.

### In Scope
- Generate missing `bas/registry.json`
- Fix any remaining failures identified by fresh test run
- Standards: resolve critical violations if any persist
- Docs: fix validation errors if any persist
- Unit: fix UI test failures if any persist
- Integration: fix CLI unknown-command exit code if still failing

### Out of Scope
- Addressing the 154 warning-level standards violations (not blocking)
- New feature work
- Refactoring or code cleanup beyond what's needed to fix tests

## Current Technical Context

### Scenario Structure
```
scenarios/prompt-manager/
├── api/           — Go API server (prompt-manager-api binary)
├── bas/           — Browser automation (actions/, cases/, flows/ — NO registry.json)
├── cli/           — Go CLI (prompt-manager binary, app.go uses cli-core)
├── docs/          — manifest.json + 20 markdown files across concepts/, reference/, guides/, internal/
├── requirements/  — index.json with 20+ modules, P0-P3
├── ui/            — React/Vite frontend with vitest tests
└── Makefile       — build/test/start targets
```

### Key Files
- `cli/app.go` — CLI entry point, uses `packages/cli-core/cliapp/cliapp.ScenarioApp`
- `packages/cli-core/cliapp/app.go:115` — Unknown command handling: `fmt.Errorf("Unknown command: %s", remaining[0])`
- `requirements/index.json` — Module registry with P0-P3 priorities
- `docs/manifest.json` — Docs navigation index, references 20 files
- `bas/cases/` — Playbook test cases (e.g., `01-foundation/01-smoke/world-ui-loads.json`)

### Registry Generation Pattern
Other scenarios use `test-genie registry build` to generate `bas/registry.json`. The file contains:
- Scenario name
- Generated timestamp
- Array of playbook entries with file path, description, order, requirements, fixtures, reset mode
- Metadata including execution_mode

## Target End State

All 11 GCT test phases pass:
- `standards`: 0 critical/high violations
- `docs`: Docs validation passes
- `unit`: All UI unit tests pass
- `integration`: CLI returns non-zero for unknown commands
- `playbooks`: `bas/registry.json` exists and is valid
- All 6 previously-passing phases continue to pass

## Implementation Strategy

### Phase 0: Baseline (Prerequisite)
**Goal**: Determine which failures still exist before fixing anything.

1. Run full GCT test suite: `vrooli scenario test prompt-manager`
2. Record which phases still fail
3. If all 5 now pass, this item is done (mark as complete)
4. If some still fail, proceed to fix only the remaining failures

### Phase 1: Playbooks Registry (Confirmed Fix)
**Goal**: Generate the missing `bas/registry.json`.

1. Run `test-genie registry build` for prompt-manager (exact command TBD based on test-genie CLI usage)
2. Verify `bas/registry.json` was created with expected schema
3. Re-run playbooks phase to confirm it passes

### Phase 2: Remaining Failures (Conditional)
**Goal**: Fix any failures that persist after Phase 0 re-run.

For each remaining failure, apply scientific debugging:
1. Observe exact failure output
2. Hypothesize root cause
3. Test hypothesis with minimal reproduction
4. Fix root cause
5. Verify fix

### Phase 3: Full Verification
**Goal**: Confirm all 11 phases pass.

1. Run `vrooli scenario test prompt-manager`
2. Verify 11/11 phases pass
3. `vrooli scenario restart prompt-manager`

## Contract Decisions

| Decision | Resolution | Source |
|----------|-----------|--------|
| CLI binary: build before tests | Add pre-test build step (Makefile) | Round 1 d1 → A |
| CLI unknown-command scope | Fix in cli-core if needed, expand acceptance_allow | Round 1 d2 → A (may not be needed — cli-core already works) |
| Requirements: fix approach | Run auditor, fix whatever modules are flagged | Round 1 d3 → A |

## Testing Plan

### Primary Verification
- **Full GCT suite**: `vrooli scenario test prompt-manager` — must pass 11/11 phases
- **Regression**: 6 currently-passing phases must remain passing

### Per-Phase Verification
1. **Standards**: `scenario-auditor audit prompt-manager --standards-only --timeout 60` — 0 critical/high violations
2. **Docs**: Docs validation phase passes
3. **Unit**: `cd scenarios/prompt-manager/ui && npm test` — all tests pass
4. **Integration**: `prompt-manager __test_genie_nonexistent_command_12345__; echo $?` returns non-zero
5. **Playbooks**: `bas/registry.json` exists and is valid JSON matching expected schema

## Rollout/Validation Checklist

- [ ] Phase 0: Run baseline test, record current failures
- [ ] Phase 1: Generate `bas/registry.json`
- [ ] Phase 2: Fix any remaining failures (if any)
- [ ] Phase 3: Full test suite passes 11/11
- [ ] `vrooli scenario restart prompt-manager`

## Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Most failures already fixed, wasted investigation time | Low — quick confirmation | Phase 0 baseline check before doing any work |
| `test-genie registry build` command doesn't exist or works differently | Can't generate registry | Check `test-genie --help` and other scenario Makefiles for registry generation patterns |
| UI test failures are complex/numerous | Scope creep | Focus on root causes, not individual test fixes; batch similar failures |
| Stale test results — GCT report from different env | Misleading plan | Always use fresh test run (Phase 0) as ground truth |

## Non-goals / Prohibited Patterns

- Do NOT fix warning-level standards violations (only critical/high)
- Do NOT refactor or clean up code beyond what's needed to fix tests
- Do NOT add new tests — only fix existing failing ones
- Do NOT commit the CLI binary to git (it's a build artifact)
- Do NOT modify `archive/` files
- Do NOT manually craft `bas/registry.json` — use the generation tool

## Definition of Done

1. `vrooli scenario test prompt-manager` passes 11/11 phases
2. No new critical/high standards violations introduced
3. `bas/registry.json` exists and is auto-generated (not hand-crafted)
4. `vrooli scenario restart prompt-manager` completes successfully
