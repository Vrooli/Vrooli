# Plan: Add Release ID Correlation Column To LPBS Download Artifacts

## Required Reading

- `prompt-manager skill read api-steer` — API design conventions for LPBS handler changes
- `prompt-manager skill read test` — Testing conventions for Go test additions
- `prompt-manager skill read seam-discovery-and-enforcement` — Testability boundaries documentation

## Problem Statement

The LPBS `download_artifacts` table has no way to correlate artifacts back to deployment-manager release records. The research item `release-record-contract-and-history-model` (Finding 8) established that a `release_id` TEXT column on `download_artifacts` serves as a lightweight foreign correlation key (not an FK constraint, since DM and LPBS use different databases).

Currently, S2D uploads pass `git_commit_hash` and `release_version` to the LPBS commit endpoint but not `release_id`. Once the DM orchestrator mints release UUIDs (a sibling initiative item), the S2D pipeline will pass `release_id` alongside these existing fields.

## Scope

### In Scope
- Add `release_id TEXT` column to `download_artifacts` table
- Add index on `release_id` for efficient lookups
- Add `ReleaseID` field to `CommitArtifactRequest` struct
- Add `ReleaseID` field to `DownloadArtifact` struct
- Add `releaseID` to `artifactScanTargets` struct with `scanDest()`/`hydrate()` support
- Update `CommitArtifact()` SQL to INSERT/UPDATE `release_id`
- Update all SELECT column lists for `download_artifacts` to include `release_id`
- Add test coverage for `release_id` persistence via the commit handler

### Out of Scope
- DM `releases`/`release_platforms` table creation (separate item)
- S2D pipeline changes to carry `release_id` (separate item)
- LPBS API endpoints to query by `release_id` (future work if needed)
- `download_assets` table changes (not needed for this correlation)

## Approach

Follow the exact pattern used when `git_commit_hash` was added (the most recent column addition to this table). This is a single-phase change:

### Phase 1: Schema + Code + Tests (single commit)

**1. Schema migration** (`main.go` migration list, after line ~806):
```sql
ALTER TABLE download_artifacts ADD COLUMN IF NOT EXISTS release_id TEXT;
CREATE INDEX IF NOT EXISTS idx_download_artifacts_release_id ON download_artifacts(release_id);
```

**2. Go model** (`download_hosting.go`):
- Add `ReleaseID string` field to `DownloadArtifact` struct (after `GitCommitHash`, line ~93)
- Add `ReleaseID string` field to `CommitArtifactRequest` struct (after `GitCommitHash`, line ~747)
- Add `releaseIDOut sql.NullString` field to `artifactScanTargets` struct (after `commitHash`, line ~118)

**3. Scan helper** (`download_hosting.go`):
- Add `&t.releaseIDOut` to `scanDest()` return slice (after `&t.commitHash`, line ~141)
- Add `t.artifact.ReleaseID = t.releaseIDOut.String` to `hydrate()` (after `GitCommitHash` line, line ~164)
- Update the `scanDest()` comment to list `release_id` in the column order

**4. CommitArtifact SQL** (`download_hosting.go:787-824`):
- Add `release_id` to INSERT column list and VALUES ($15)
- Add `release_id = EXCLUDED.release_id` to ON CONFLICT UPDATE
- Add `release_id` to RETURNING clause
- Add `normalizeOptionalString(&req.ReleaseID)` to QueryRowContext args
- Shift parameter numbers ($10→$15 etc.) to accommodate the new column

**5. All other SELECT queries** in `download_hosting.go` that select from `download_artifacts`:
- `GetArtifact()` — add `release_id` to SELECT list
- `ListArtifacts()` — add `release_id` to SELECT list
- `DeleteArtifact()` — check if it SELECTs (likely just DELETE, no change)
- Any other queries returning artifact rows — add `release_id` in the correct column position

**6. Tests** (`download_hosting_test.go`):
- Mirror the `TestDownloadHostingService_CommitArtifact_GitCommitHash` test pattern (line ~530)
- Create `TestDownloadHostingService_CommitArtifact_ReleaseID` that:
  - Commits an artifact with `release_id` set
  - Asserts the returned artifact has `release_id` populated
  - Re-commits (upsert) with a different `release_id` and asserts it updates
  - Commits without `release_id` and asserts existing value is preserved (COALESCE behavior) or cleared (depending on decision d1)

## Test Plan

1. **Unit test: release_id round-trip** — Commit artifact with `release_id`, verify it persists and returns correctly
2. **Unit test: release_id upsert** — Re-commit same artifact with new `release_id`, verify it updates
3. **Unit test: release_id optional** — Commit without `release_id`, verify no error and field is empty
4. **Existing tests pass** — Run full `go test ./...` to ensure no regressions from column additions
5. **Build check** — `go build ./...` succeeds for both api and cli packages

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Column position mismatch in scan helpers | Medium | High (runtime panic) | Follow exact `git_commit_hash` pattern; run all tests |
| Missing a SELECT query that returns artifact columns | Low | Medium (scan error) | Grep for all `FROM download_artifacts` queries |
| Parameter numbering off in CommitArtifact SQL | Medium | High (wrong values) | Carefully re-count all $N parameters |
| Migration fails on existing data | Very Low | Low | `ADD COLUMN IF NOT EXISTS` + nullable TEXT is safe |

## Files to Modify

| File | Change |
|------|--------|
| `scenarios/landing-page-business-suite/api/main.go` | Add ALTER TABLE + CREATE INDEX migrations |
| `scenarios/landing-page-business-suite/api/download_hosting.go` | Add field to structs, update scan helpers, update all SQL queries |
| `scenarios/landing-page-business-suite/api/download_hosting_test.go` | Add release_id test cases |

## Verification

```bash
cd scenarios/landing-page-business-suite/api && go build ./... && go test ./... -timeout 300s
```
