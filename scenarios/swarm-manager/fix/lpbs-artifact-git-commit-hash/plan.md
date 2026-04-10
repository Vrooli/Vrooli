# Implementation Plan: Add git_commit_hash to LPBS download_artifacts

## 1. Purpose

Add a `git_commit_hash` column to the LPBS `download_artifacts` table so published artifacts are permanently traceable to their source commit without relying on scenario-to-desktop's transient pipeline state.

## 2. Required Reading

```bash
prompt-manager skill read api-steer test implementation-plan-authoring
```

## 3. Problem Statement

The `download_artifacts` table stores provenance metadata (platform, version, SHA hashes) but lacks the git commit hash that produced the build. This information exists transiently in scenario-to-desktop's `BuildProvenance` struct and is stored in build-stage metadata, but is never persisted to the artifact record in LPBS. After a pipeline run completes, the only way to trace an artifact to its source commit is through transient pipeline state — which may be garbage-collected or lost.

**Source:** research/desktop-release-control-plane-audit — Finding 2, Finding 9 gap #6.

## 4. Scope

### In Scope
- Add `git_commit_hash TEXT` column to `download_artifacts` table (LPBS)
- Update `DownloadArtifact` Go struct to include `GitCommitHash`
- Update `CommitArtifactRequest` to accept `git_commit_hash`
- Update `CommitArtifact` service method INSERT/UPDATE query
- Update scan targets (`artifactScanTargets`) to include the new column
- Update scenario-to-desktop's `proxyCommit` to pass `GitCommitHash` from `BuildProvenance`
- Add/update tests for the commit endpoint with the new field
- Update SEAMS.md if a new testability seam is introduced

### Out of Scope
- Backfilling existing artifacts with commit hashes
- Adding other BuildProvenance fields (git_branch, git_dirty, built_at) as dedicated columns — these remain in the JSONB `metadata` field
- UI changes to display git_commit_hash
- Changing the `download_artifacts` unique constraint

## 5. Current Technical Context

### Key Files — LPBS (`scenarios/landing-page-business-suite/api/`)
| File | Role |
|------|------|
| `main.go:742-769` | Table DDL (CREATE TABLE + ALTER TABLE statements) |
| `download_hosting.go:82-103` | `DownloadArtifact` struct |
| `download_hosting.go:733-745` | `CommitArtifactRequest` struct |
| `download_hosting.go:747-826` | `CommitArtifact` service method (INSERT ... ON CONFLICT DO UPDATE) |
| `download_hosting.go` | `artifactScanTargets` helper for row scanning |
| `download_hosting_handlers.go:142-163` | `handleAdminCommitDownloadArtifact` handler |
| `routes.go:123` | Route registration for commit endpoint |

### Key Files — scenario-to-desktop (`scenarios/scenario-to-desktop/api/`)
| File | Role |
|------|------|
| `pipeline/provenance.go:16-31` | `BuildProvenance` struct (has `GitCommitHash`) |
| `deploy/lpbs_client.go:306-329` | `proxyCommit` — builds commit request payload |
| `pipeline/stage_build.go:224-230` | Where provenance is stored in build metadata |
| `pipeline/stage_deploy.go` | Deploy stage entry point |

### Patterns
- LPBS uses `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` for migrations (append to existing DDL block in `main.go`)
- `artifactScanTargets` consolidates scan destinations; new columns must be added to `scanDest()` slice
- `normalizeOptionalString()` is used for optional TEXT fields in the INSERT query
- scenario-to-desktop passes provenance through `StageInput.Provenance`

## 6. Target End State

- `download_artifacts` table has a `git_commit_hash TEXT` column
- Committing an artifact with a git commit hash persists it to the dedicated column
- scenario-to-desktop's deploy stage passes `BuildProvenance.GitCommitHash` to the LPBS commit endpoint
- Existing artifacts without a commit hash continue to work (column is nullable)
- The field is queryable for traceability (e.g., "which artifact was built from commit X?")
- `git_commit_hash` is included in all artifact JSON responses (lightweight, serves traceability purpose)

## 7. Implementation Strategy

### Phase 1: LPBS Schema + API (scenarios/landing-page-business-suite/api/)

1. **Migration**: Add `ALTER TABLE download_artifacts ADD COLUMN IF NOT EXISTS git_commit_hash TEXT;` to `main.go` DDL block
2. **Struct**: Add `GitCommitHash string` field to `DownloadArtifact` with `json:"git_commit_hash"` and `db:"git_commit_hash"` tags
3. **Request**: Add `GitCommitHash string` field to `CommitArtifactRequest` with `json:"git_commit_hash"`
4. **Scan targets**: Add `&a.GitCommitHash` to `artifactScanTargets` `scanDest()` slice, add `"git_commit_hash"` to column list — **must match SQL SELECT column order exactly** (highest-risk step)
5. **Service**: Update `CommitArtifact` INSERT query to include `git_commit_hash` column, using `normalizeOptionalString()` for the value; include in ON CONFLICT UPDATE SET as well
6. **Tests**: Add test case for committing an artifact with `git_commit_hash` set, verify it round-trips through GET

### Phase 2: scenario-to-desktop Caller (scenarios/scenario-to-desktop/api/)

1. **UploadRequest**: Add `GitCommitHash string` field to the `UploadRequest` struct (confirmed: this is the natural carrier for artifact metadata)
2. **Deploy stage**: Populate `UploadRequest.GitCommitHash` from `StageInput.Provenance.GitCommitHash` (confirmed: StageInput already carries Provenance, populated by the orchestrator for every stage)
3. **proxyCommit**: Add `"git_commit_hash": req.GitCommitHash` to the commit request map in `lpbs_client.go`
4. **Tests**: Update any existing deploy/commit tests to verify the field is passed

## 8. Contract Decisions

Confirmed via workshop round-001:

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Column type | `TEXT` (not `CHAR(40)`) | Accommodates both SHA-1 (40 chars) and future SHA-256 (64 chars) git hashes |
| Nullability | Nullable (no NOT NULL) | Existing artifacts and non-git-sourced artifacts won't have this field |
| API field name | `git_commit_hash` | Matches BuildProvenance JSON tag and existing metadata key convention |
| No index initially | Deferred | Can be added later if query-by-hash becomes a hot path |
| Provenance threading | Via `StageInput.Provenance` | Already populated by orchestrator for every stage; minimal change (d1) |
| Response inclusion | All artifact responses | Lightweight field; whole purpose is traceability (d2) |
| Carrier struct | `UploadRequest.GitCommitHash` | Natural carrier for artifact metadata flowing into upload pipeline (d3) |

## 9. Testing Plan

- **Unit test (LPBS)**: Commit an artifact with `git_commit_hash` set → verify the field is returned in the response and persists on re-read
- **Unit test (LPBS)**: Commit an artifact WITHOUT `git_commit_hash` → verify it still works (backward compat)
- **Unit test (scenario-to-desktop)**: Verify `proxyCommit` includes `git_commit_hash` in request body when provenance is available
- **Integration**: Full pipeline run should result in artifact with populated `git_commit_hash`

## 10. Rollout/Validation Checklist

- [ ] `go build ./...` passes in both scenarios
- [ ] `go test ./... -timeout 300s` passes in LPBS
- [ ] `go test ./... -timeout 300s` passes in scenario-to-desktop
- [ ] `gofumpt -w .` applied to changed files
- [ ] Manual verification: commit an artifact via API with `git_commit_hash`, confirm it appears in response

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Column addition fails on existing DB | Low | Medium | `IF NOT EXISTS` guards the ALTER |
| Scan target ordering mismatch | Medium | High | Must match column order exactly in SQL SELECT and scanDest slice; test covers round-trip |
| Provenance nil in deploy stage | Low | Low | Provenance is captured at orchestrator start; nil-check before accessing |

## 12. Non-goals / Prohibited Patterns

- Do not add an index on `git_commit_hash` unless benchmarks show it's needed
- Do not backfill existing rows
- Do not promote other BuildProvenance fields to dedicated columns in this fix
- Do not modify the unique constraint on `download_artifacts`
- Do not add UI components

## 13. Definition of Done

- `download_artifacts` table has `git_commit_hash TEXT` column
- LPBS commit endpoint accepts and persists `git_commit_hash`
- `git_commit_hash` appears in all artifact JSON responses
- scenario-to-desktop passes `BuildProvenance.GitCommitHash` via `UploadRequest` to `proxyCommit`
- All existing tests pass; new test covers the round-trip
- Code formatted with `gofumpt`
