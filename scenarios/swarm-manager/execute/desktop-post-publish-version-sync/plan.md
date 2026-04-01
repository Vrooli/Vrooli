# Implementation Plan: Post-Publish Version Sync

## 1. Purpose

Close the feedback loop between scenario-to-desktop and deployment-manager so that after a desktop release is published to LPBS, deployment-manager automatically records what version was published for which platform under which profile. Today this information is lost once the pipeline completes.

## Required Reading

```bash
prompt-manager skill read api-steer interoperability-steer deployment-coordinator
prompt-manager skill read cli-steer seam-discovery-and-enforcement
```

## 2. Problem Statement

When deployment-manager orchestrates a desktop release via `POST /api/v1/deploy-desktop`, it:
1. Calls scenario-to-desktop `POST /api/v1/pipeline/run`
2. Polls `GET /api/v1/pipeline/{id}` until completion
3. Checks `Status.Stages["deploy"].Details` for `DeployResult`

But after step 3, deployment-manager **does not persist** the published version, platform, or artifact IDs anywhere. The `deployments` table stores `status`, `artifacts` (JSONB), `message`, `logs`, and `error` — but `artifacts` is a generic JSONB array without structured version-per-platform semantics.

The `DeployResult` struct (in scenario-to-desktop) contains `[]DeployArtifactResult{ArtifactID, Platform}` plus `UpdateURL`. The pipeline `Status.Provenance` contains `BuildProvenance{Version, GitCommitHash, GitBranch, GitDirty, BuiltAt}`.

## 3. Scope

**In scope:**
- `scenarios/deployment-manager/api/**` — new table, repository, endpoint, orchestrator integration
- `scenarios/scenario-to-desktop/api/**` — only if changes needed to expose data (likely none with pull approach)

**Out of scope:**
- LPBS webhook (separate backlog item: `lpbs-publish-webhook`)
- Rollback API (separate backlog item: `desktop-rollback-api`)
- UI changes in either scenario
- CLI changes in either scenario

## 4. Current Technical Context

### deployment-manager
- **Schema**: `scenarios/deployment-manager/initialization/postgres/schema.sql`
  - `profiles` table with JSONB fields for tiers, swaps, secrets, settings
  - `deployments` table: `id, profile_id, status, started_at, completed_at, artifacts JSONB, message, logs, error`
  - `deployment_approvals` table: per-platform release gating with git_commit_hash
  - No dedicated table for published version tracking
- **Orchestrator**: `scenarios/deployment-manager/api/deployments/orchestrator.go`
  - Handles full desktop deploy workflow: validate → build → package → desktop gen → installers → visual validation
  - Uses `DesktopPackagerClient` to communicate with scenario-to-desktop
  - Does NOT currently have a "publish to LPBS" step — the deploy stage runs through scenario-to-desktop's pipeline separately
  - Returns `DeployDesktopResponse` with Steps, BuildResults, Installers, etc.
- **Desktop client**: `scenarios/deployment-manager/api/deployments/desktop_client.go`
  - Has `QuickGenerate`, `WaitForBuild`, `RunSmokeTest` methods
  - Can call `POST /api/v1/pipeline/run` and poll status
- **Routes**: gorilla/mux in `server/routes.go`
- **Repository pattern**: Interface + SQL implementation (e.g., `profiles.Repository` → `profiles.SQLRepository`)

### scenario-to-desktop
- **Deploy stage**: `pipeline/stage_deploy.go` — uploads artifacts to LPBS via `LPBSClient`, produces `DeployResult`
- **Types**: `DeployResult{Artifacts []DeployArtifactResult, UpdateURL string}`
- **DeployArtifactResult**: `{ArtifactID int64, Platform string}`
- **BuildProvenance**: `{GitCommitHash, GitBranch string, GitDirty bool, BuiltAt time.Time, Version string}`
- **Pipeline Status**: `Status.Stages["deploy"].Details` contains `DeployResult` (as `interface{}`, needs JSON round-trip to deserialize)
- **Status.Provenance**: Contains `BuildProvenance` at the pipeline level

## 5. Decisions Made (Workshop Round 1)

| Decision | Selected | Rationale |
|----------|----------|-----------|
| Sync mechanism | **Pull** (orchestrator extracts from pipeline response) | Simplest, no new inter-service contract, deployment-manager owns its own data |
| Version storage | **Full history** (append-only, query latest via window function) | Enables "what was live at time T?" queries and rollback audit trail |
| Error handling | **Non-fatal** (log error, deploy still succeeds) | Publish is the real action; version record is convenience cache |
| Version source | **BuildProvenance.Version** from pipeline Status | Already available in pipeline status response |

## 6. Implementation Strategy: Orchestrator-Side Pull

Since the orchestrator already (or will) poll pipeline status which includes `DeployResult` and `BuildProvenance`, the version persist happens entirely within deployment-manager:

### Phase 1: Schema + Repository
1. Add `published_versions` table to deployment-manager schema (append-only, no UNIQUE constraint)
2. Create `PublishedVersionsRepository` interface + SQL implementation
3. Methods: `RecordPublish(ctx, record)`, `GetLatestByProfile(ctx, profileID)`, `GetHistory(ctx, profileID, platform, limit)`

### Phase 2: Orchestrator Integration
1. After observing successful deploy stage completion in pipeline status, extract:
   - `Version` and `GitCommitHash` from `Status.Provenance`
   - `ArtifactID` and `Platform` from each entry in `DeployResult.Artifacts`
2. Call `RecordPublish` for each platform artifact
3. Log errors but do not fail the deployment

### Phase 3: Query Endpoint
1. Add `GET /api/v1/profiles/{id}/published-versions` endpoint
2. Returns latest published version per platform (using window function over history)
3. Optional `?platform=` filter and `?history=true` for full history

## 7. Contract Details

### New Table: `published_versions` (append-only history)

```sql
CREATE TABLE IF NOT EXISTS published_versions (
    id SERIAL PRIMARY KEY,
    profile_id VARCHAR(255) NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    version VARCHAR(100) NOT NULL,
    git_commit_hash VARCHAR(64),
    artifact_id BIGINT,
    deployment_id VARCHAR(255) REFERENCES deployments(id),
    published_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_published_versions_profile_platform
    ON published_versions(profile_id, platform, published_at DESC);
```

No UNIQUE constraint — each publish appends a new row. The index supports efficient "latest per platform" queries.

### Latest-per-platform query pattern

```sql
SELECT DISTINCT ON (platform) *
FROM published_versions
WHERE profile_id = $1
ORDER BY platform, published_at DESC;
```

### New Endpoint: `GET /api/v1/profiles/{id}/published-versions`

**Response (latest mode, default):**
```json
{
  "profile_id": "my-profile",
  "versions": [
    {
      "platform": "windows",
      "version": "1.2.3",
      "git_commit_hash": "abc123...",
      "artifact_id": 42,
      "deployment_id": "deploy-xyz",
      "published_at": "2026-03-30T12:00:00Z"
    }
  ]
}
```

### Repository Interface

```go
type PublishedVersionsRepository interface {
    RecordPublish(ctx context.Context, record *PublishedVersion) error
    GetLatestByProfile(ctx context.Context, profileID string) ([]PublishedVersion, error)
    GetHistory(ctx context.Context, profileID, platform string, limit int) ([]PublishedVersion, error)
}
```

## 8. Testing Plan

### Unit Tests
1. **Repository layer**: Test `RecordPublish` inserts correctly, `GetLatestByProfile` returns only latest per platform, `GetHistory` returns ordered history with limit
2. **Orchestrator extraction**: Test that version data is correctly extracted from a mock pipeline status response and passed to repository
3. **Non-fatal error handling**: Test that repository errors during persist don't propagate as deployment failures

### Integration Tests
1. **End-to-end**: Deploy flow records version → query endpoint returns it
2. **Multiple publishes**: Second publish for same platform appears as latest, first is still in history
3. **Failed deploy**: Verify no version record is created

### What to mock
- Pipeline status response (for orchestrator unit tests)
- Repository interface (for handler unit tests)
- Real DB via testcontainers (for repository + integration tests)

## 9. Rollout/Validation Checklist

- [ ] Schema migration runs cleanly on fresh and existing databases
- [ ] Orchestrator persists version after successful deploy
- [ ] GET endpoint returns correct published versions (latest per platform)
- [ ] History mode returns full publish history
- [ ] Failed deploy does not update published versions
- [ ] Non-fatal: persist failure doesn't break deploy response
- [ ] Existing deploy flow is not broken
- [ ] `go build ./...` and `go test ./...` succeed for deployment-manager

## 10. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Schema migration fails on existing data | Low | Medium | Migration is additive (new table), no existing data affected |
| Orchestrator error during version persist breaks deploy | Medium | High | Persist in a non-fatal path — log error but don't fail the deployment |
| Version drift if deploy succeeds but persist fails | Low | Low | Can always re-derive from LPBS; this is a convenience cache |
| Pipeline status `Details` field needs JSON round-trip | Low | Low | Well-understood Go pattern: marshal to JSON, unmarshal to typed struct |
| Append-only table grows unbounded | Low | Low | Typical publish frequency is very low; add retention policy later if needed |

## 11. Non-goals / Prohibited Patterns

- Do NOT modify LPBS (separate scenario, separate backlog items)
- Do NOT add UI for viewing published versions (can be a follow-up)
- Do NOT implement full webhook system (separate backlog item)
- Do NOT add backward-compatibility shims for the new table
- Do NOT add UNIQUE constraints — this is intentionally append-only

## 12. Definition of Done

- [ ] `published_versions` table exists in deployment-manager schema
- [ ] `PublishedVersionsRepository` interface + SQL implementation
- [ ] Orchestrator persists version per platform after successful deploy
- [ ] `GET /api/v1/profiles/{id}/published-versions` returns current versions
- [ ] History query support with `?history=true&platform=X`
- [ ] Non-fatal error handling verified
- [ ] All tests pass
- [ ] `go build ./...` and `go test ./...` succeed for deployment-manager

## 13. File Change Manifest

| File | Action | Description |
|------|--------|-------------|
| `scenarios/deployment-manager/initialization/postgres/schema.sql` | Edit | Add `published_versions` table + index |
| `scenarios/deployment-manager/api/deployments/published_versions.go` | Create | `PublishedVersion` type + `PublishedVersionsRepository` interface + SQL impl |
| `scenarios/deployment-manager/api/deployments/published_versions_test.go` | Create | Repository unit tests using testcontainers |
| `scenarios/deployment-manager/api/deployments/orchestrator.go` | Edit | Add version extraction + persist after deploy stage completion |
| `scenarios/deployment-manager/api/deployments/orchestrator_test.go` | Edit/Create | Test version extraction logic |
| `scenarios/deployment-manager/api/server/routes.go` | Edit | Register `GET /api/v1/profiles/{id}/published-versions` |
| `scenarios/deployment-manager/api/deployments/handlers.go` or new handler file | Edit/Create | Handler for published versions endpoint |
