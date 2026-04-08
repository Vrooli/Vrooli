# Plan: Fix deployment-manager test failures

## Required Reading
- `prompt-manager skill read scientific-debugging` — hypothesis-driven root cause analysis
- `prompt-manager skill read documentation-health` — docs validation patterns

## Problem Statement

GCT tests for deployment-manager are failing in 2 of 10 phases (down from 3 — UI build now passes):

1. **phase-docs**: 4 broken local links in markdown files referencing `scenario-to-desktop/docs/` paths that were reorganized
2. **phase-playbooks**: Missing `bas/registry.json` — the `bas/cases/` directory is empty, so registry generation may produce an empty registry or the phase may need to be skipped

The UI build failure was transient and now passes.

## Root Cause Analysis

### Docs Failures
The `scenario-to-desktop` scenario reorganized its docs from flat files (`docs/SIGNING.md`, `docs/CROSS_PLATFORM_BUILDS.md`) into subdirectories (`docs/guides/code-signing.md`, `docs/guides/cross-platform-builds.md`, `docs/guides/logging-bundled-desktop.md`). The deployment-manager docs still reference the old paths.

**Broken links (4 total):**
| File | Line | Old Target | New Target |
|------|------|-----------|------------|
| `docs/guides/code-signing.md` | 24 | `../../../scenario-to-desktop/docs/SIGNING.md` | `../../../scenario-to-desktop/docs/guides/code-signing.md` |
| `docs/guides/code-signing.md` | 101 | `../../../scenario-to-desktop/docs/SIGNING.md` | `../../../scenario-to-desktop/docs/guides/code-signing.md` |
| `docs/tutorials/hello-desktop-walkthrough.md` | 423 | `../../../scenario-to-desktop/docs/CROSS_PLATFORM_BUILDS.md` | `../../../scenario-to-desktop/docs/guides/cross-platform-builds.md` |
| `docs/workflows/desktop-deployment.md` | 58 | `../../../scenario-to-desktop/docs/workflows/logging-bundled-desktop.md` | `../../../scenario-to-desktop/docs/guides/logging-bundled-desktop.md` |

### Playbooks Failure
The `bas/` directory exists with `actions/`, `flows/`, `seeds/`, and `README.md` but `cases/` is empty and `registry.json` doesn't exist. The playbooks phase requires `registry.json` to run. The fix is to generate it via `test-genie registry build`, which will scan `bas/cases/` and produce a (possibly empty) registry. An empty registry should cause the phase to pass with no tests executed.

## Implementation Steps

### Phase 1: Fix broken doc links (4 edits)
1. Edit `scenarios/deployment-manager/docs/guides/code-signing.md`:
   - Line 24: Change `../../../scenario-to-desktop/docs/SIGNING.md` → `../../../scenario-to-desktop/docs/guides/code-signing.md`
   - Line 101: Same change
2. Edit `scenarios/deployment-manager/docs/tutorials/hello-desktop-walkthrough.md`:
   - Line 423: Change `../../../scenario-to-desktop/docs/CROSS_PLATFORM_BUILDS.md` → `../../../scenario-to-desktop/docs/guides/cross-platform-builds.md`
3. Edit `scenarios/deployment-manager/docs/workflows/desktop-deployment.md`:
   - Line 58: Change `../../../scenario-to-desktop/docs/workflows/logging-bundled-desktop.md` → `../../../scenario-to-desktop/docs/guides/logging-bundled-desktop.md`

### Phase 2: Generate playbooks registry
4. Run `test-genie registry build -scenario scenarios/deployment-manager` to generate `bas/registry.json`

### Phase 3: Verify
5. Run `vrooli scenario test deployment-manager` and confirm all phases pass

## Risks & Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Empty registry still fails playbooks phase | Low | Check if test-genie handles empty registries; if not, create a minimal valid registry |
| Other broken links not caught | Low | The docs phase scanner is comprehensive; fixing these 4 should suffice |
| Link text no longer matches target content | Low | Verify target docs still cover the same topics |

## Acceptance Criteria
- All 10 GCT phases pass (0 failures)
- No broken doc links in deployment-manager docs
- `bas/registry.json` exists and is valid
