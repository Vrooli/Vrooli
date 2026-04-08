# Plan: Exclude captures from git in swarm-manager

## Required Reading

- `prompt-manager skill read implementation-plan-authoring` — canonical plan structure

## Problem Statement

Capture data files (test output logs, screenshots, review artifacts) under `scenarios/swarm-manager/` are committed to git. These are short-lived, often large (especially images), and not valuable enough to persist in version control. They bloat the repo and slow clones.

### Key Observations

- **Data captures** live at two path patterns:
  - `scenarios/swarm-manager/captures/` — top-level capture dir (currently empty)
  - `scenarios/swarm-manager/{execute,fix,chore,idea}/**/review/captures/` — per-item review captures (10+ items, ~17 tracked files, ~67KB currently)
- **Source code** at `scenarios/swarm-manager/api/internal/captures/` contains Go implementation files (`handler.go`, `classify.go`, etc.) — these must remain tracked
- The existing `.gitignore` at `scenarios/swarm-manager/.gitignore` only excludes CLI binaries

## Scope

### In Scope
- Add gitignore rules to exclude capture data directories
- Remove already-tracked capture data files from git index (without deleting from disk)
- Preserve `api/internal/captures/` Go source files (not data)

### Out of Scope
- Changing capture storage mechanism or retention policy
- Modifying capture creation logic in the API
- Other scenarios' capture handling
- Modifying the root `.gitignore` (scoped to swarm-manager only per d2=A)

## Resolved Decisions

| ID | Decision | Selected | Rationale |
|----|----------|----------|-----------|
| d1 | Gitignore pattern strategy | A — Specific path patterns | `api/internal/captures/` does not match `/captures/` or `**/review/captures/`, so no negation is needed. Simpler and more reliable than broad ignore + negation. |
| d2 | Scope of gitignore changes | A — Only `scenarios/swarm-manager/.gitignore` | Minimal scope; other scenarios add their own rules when needed. |
| d3 | Tracked file handling | A — `git rm --cached` (keep on disk) | Non-destructive; files remain locally for short-term reference but stop being tracked. |

## Approach

### Phase 1: Update .gitignore

Append rules to `scenarios/swarm-manager/.gitignore`:

```gitignore
# Capture data (short-lived review artifacts, images, logs)
/captures/
**/review/captures/
```

**Why these patterns work without negation (d1=A):** The top-level `/captures/` pattern is anchored to the `.gitignore` location, matching only `scenarios/swarm-manager/captures/`. The `**/review/captures/` pattern matches nested review capture dirs. Neither pattern matches `api/internal/captures/` because that path contains neither a root-level `captures/` nor a `review/captures/` segment.

### Phase 2: Remove tracked captures from git index

```bash
git rm -r --cached scenarios/swarm-manager/captures/
git rm -r --cached scenarios/swarm-manager/execute/*/review/captures/
git rm -r --cached scenarios/swarm-manager/fix/*/review/captures/
git rm -r --cached scenarios/swarm-manager/chore/*/review/captures/
git rm -r --cached scenarios/swarm-manager/idea/*/review/captures/
```

Per d3=A, `--cached` removes from index only — files remain on disk.

Some of these globs may match nothing (e.g., if no chore items have captures yet); non-matching globs can be skipped or use `--ignore-unmatch`.

### Phase 3: Commit

Single commit with the `.gitignore` update and index removal.

## Testing / Verification

1. After `.gitignore` update: `git status` should show capture data files as deleted (from index) and `.gitignore` as modified
2. `git ls-files -- 'scenarios/swarm-manager/**/captures/**'` should only return `api/internal/captures/*.go` files
3. New captures created after the change should not appear in `git status`
4. `api/internal/captures/*.go` files must still be tracked: `git ls-files -- scenarios/swarm-manager/api/internal/captures/` should list all Go source files

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Some `git rm --cached` globs match nothing | Medium | None | Use `--ignore-unmatch` flag |
| Other developers lose local capture files | None | N/A | `--cached` only affects index, not working tree |

## Acceptance Criteria

- [ ] No capture data files tracked in git (only `api/internal/captures/` Go source)
- [ ] `.gitignore` rules prevent future capture data from being tracked
- [ ] All `api/internal/captures/*.go` files remain tracked
- [ ] Single clean commit
