# Plan: Fix deployment-manager test failures

> **Greenfield declaration:** This is a targeted fix — no backward-compatibility shims or migration bridges needed.

## Purpose

Restore deployment-manager GCT tests to a fully passing state by fixing 4 broken doc links and generating the missing playbooks registry. These failures block the scenario from reaching green readiness.

## Required Reading

```bash
prompt-manager skill read scientific-debugging
prompt-manager skill read documentation-health
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

## Problem Statement

GCT tests for deployment-manager fail in 2 of 10 phases (8/11 passed):

1. **phase-docs**: 4 broken local links in markdown files referencing `scenario-to-desktop/docs/` paths that were reorganized into subdirectories.
2. **phase-playbooks**: Missing `bas/registry.json` — the `bas/cases/` directory is empty and no registry has been generated.

A third failure (phase-performance / UI build) was transient and now passes.

## Scope

**In scope:**
- Fix 4 broken markdown links in deployment-manager docs
- Update link display text to match new file names (per workshop decision d2)
- Generate `bas/registry.json` via `test-genie registry build` (per workshop decision d1)
- Verify all 10 GCT phases pass

**Out of scope:**
- Adding new E2E test cases to `bas/cases/` (separate backlog item)
- Fixing code quality violations (151 violations tracked separately)
- Fixing standards warnings (eslint config comments, TypeScript patterns)
- UI build issues (transient, already resolved)

## Current Technical Context

### Key Files

| File | Role |
|------|------|
| `scenarios/deployment-manager/docs/guides/code-signing.md` | Contains 2 broken links (lines 24, 101) to old `scenario-to-desktop/docs/SIGNING.md` |
| `scenarios/deployment-manager/docs/tutorials/hello-desktop-walkthrough.md` | Contains 1 broken link (line 423) to old `scenario-to-desktop/docs/CROSS_PLATFORM_BUILDS.md` |
| `scenarios/deployment-manager/docs/workflows/desktop-deployment.md` | Contains 1 broken link (line 58) to old `scenario-to-desktop/docs/workflows/logging-bundled-desktop.md` |
| `scenarios/deployment-manager/bas/registry.json` | Missing — must be generated |
| `scenarios/deployment-manager/bas/cases/` | Empty directory — registry will be empty but valid |

### Root Cause

The `scenario-to-desktop` scenario reorganized its docs from flat files into `docs/guides/` subdirectories. The deployment-manager docs still reference the old paths.

**Link mapping:**

| Old Path | New Path |
|----------|----------|
| `scenario-to-desktop/docs/SIGNING.md` | `scenario-to-desktop/docs/guides/code-signing.md` |
| `scenario-to-desktop/docs/CROSS_PLATFORM_BUILDS.md` | `scenario-to-desktop/docs/guides/cross-platform-builds.md` |
| `scenario-to-desktop/docs/workflows/logging-bundled-desktop.md` | `scenario-to-desktop/docs/guides/logging-bundled-desktop.md` |

## Target End State

- All 10 GCT phases pass (0 failures)
- No broken doc links in deployment-manager docs
- `bas/registry.json` exists and is valid (empty registry is acceptable)
- Link display text updated to match new target file names

## Implementation Strategy

### Phase 1: Fix broken doc links (4 edits)

1. Edit `scenarios/deployment-manager/docs/guides/code-signing.md`:
   - Line 24: Update href from `../../../scenario-to-desktop/docs/SIGNING.md` → `../../../scenario-to-desktop/docs/guides/code-signing.md` and update display text to reference the new name
   - Line 101: Same change
2. Edit `scenarios/deployment-manager/docs/tutorials/hello-desktop-walkthrough.md`:
   - Line 423: Update href from `../../../scenario-to-desktop/docs/CROSS_PLATFORM_BUILDS.md` → `../../../scenario-to-desktop/docs/guides/cross-platform-builds.md` and update display text
3. Edit `scenarios/deployment-manager/docs/workflows/desktop-deployment.md`:
   - Line 58: Update href from `../../../scenario-to-desktop/docs/workflows/logging-bundled-desktop.md` → `../../../scenario-to-desktop/docs/guides/logging-bundled-desktop.md` and update display text

### Phase 2: Generate playbooks registry

4. Run `test-genie registry build -scenario scenarios/deployment-manager` to generate `bas/registry.json`
   - Expected: empty but valid registry (since `bas/cases/` is empty)
   - The playbooks phase should pass with 0 tests executed

### Phase 3: Verify and clean up

5. Run `vrooli scenario test deployment-manager` and confirm all 10 phases pass
6. Run `vrooli scenario restart deployment-manager` for final cleanup

## Contract Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Empty `bas/cases/` handling | Generate empty registry (workshop d1, option A) | Simplest fix; empty registry is valid and the phase passes with 0 tests. Adding test cases is separate scope. |
| Link text updates | Update both href and display text (workshop d2, option A) | Keeps docs consistent; readers see accurate file references matching actual target names. |

## Testing Plan

1. **Automated verification**: Run `vrooli scenario test deployment-manager` — all 10 phases must pass
2. **Specific phase checks**:
   - `phase-docs`: Validates all markdown links resolve — confirms the 4 link fixes
   - `phase-playbooks`: Validates `bas/registry.json` exists and is well-formed — confirms registry generation
3. **Regression**: No other phases should regress since changes are docs-only + registry generation

## Rollout/Validation Checklist

- [ ] Fix 4 broken doc links (Phase 1)
- [ ] Generate `bas/registry.json` (Phase 2)
- [ ] Run `vrooli scenario test deployment-manager` — all 10 phases pass
- [ ] Run `vrooli scenario restart deployment-manager` — final cleanup

## Risks + Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Empty registry still fails playbooks phase | Low | `test-genie` is expected to handle empty registries; if not, create a minimal valid `{"cases":[]}` manually |
| Other broken links not caught | Low | The docs phase scanner is comprehensive; fixing these 4 should suffice |
| Link display text no longer matches target content | Low | Verified that target docs at new paths cover the same topics |

## Non-goals / Prohibited Patterns

- Do NOT add new E2E test cases to `bas/cases/` — that is separate scope
- Do NOT fix code quality violations (151 violations) — tracked separately
- Do NOT add compatibility shims for old doc paths
- Do NOT modify any files outside `scenarios/deployment-manager/**`

## Definition of Done

1. All 10 GCT phases pass with 0 failures
2. The 4 broken doc links are fixed with updated hrefs and display text
3. `bas/registry.json` exists and is valid
4. No files modified outside `scenarios/deployment-manager/**`
5. `vrooli scenario restart deployment-manager` runs cleanly as final step
