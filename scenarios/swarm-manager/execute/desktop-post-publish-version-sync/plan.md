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
  - Currently does NOT have a "publish to LPBS" step — this plan adds it as Step 9
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

## 5. Decisions Made

### Workshop Round 1

| Decision | Selected | Rationale |
|----------|----------|-----------|
| Sync mechanism | **Pull** (orchestrator extracts from pipeline response) | Simplest, no new inter-service contract, deployment-manager owns its own data |
| Version storage | **Full history** (append-only, query latest via window function) | Enables "what was live at time T?" queries and rollback audit trail |
| Error handling | **Non-fatal** (log error, deploy still succeeds) | Publish is the real action; version record is convenience cache |
| Version source | **BuildProvenance.Version** from pipeline Status | Already available in pipeline status response |

### Workshop Round 2

| Decision | Selected | Rationale |
|----------|----------|-----------|
| Orchestrator integration point | **New orchestrator step** ("Publish to LPBS") that triggers scenario-to-desktop pipeline, polls completion, extracts version | Makes orchestrator the single entry point for the entire deploy-through-publish flow |
| Repository placement | **In `deployments` package** alongside `ApprovalsRepository` | Published versions are a deployment concern; orchestrator is primary caller; avoids new package |
| DeployResult deserialization | **JSON round-trip** (marshal `Details` interface{} to JSON, unmarshal to local typed struct) | Standard Go pattern; no coupling to scenario-to-desktop's Go types |
| Pipeline trigger config | **Deploy stage only**, passing pre-built artifact paths as input | Orchestrator already did build/package; only LPBS upload needed; minimizes redundant work |

## 6. Implementation Strategy: Orchestrator-Side Pull

The orchestrator adds a new "Publish to LPBS" step (Step 9) after visual validation. This step triggers scenario-to-desktop's pipeline with deploy-stage-only config, polls for completion, then extracts and persists version data.

### Phase 1: Schema + Repository (in `deployments` package)
1. Add `published_versions` table to deployment-manager schema (append-only, no UNIQUE constraint)
2. Add `PublishedVersion` type to `deployments` package
3. Add `PublishedVersionsRepository` interface + SQL implementation in `deployments` package, alongside existing `ApprovalsRepository`
4. Methods: `RecordPublish(ctx, record)`, `GetLatestByProfile(ctx, profileID)`, `GetHistory(ctx, profileID, platform, limit)`

### Phase 2: Local DeployResult Type + Deserialization
1. Define a local `DeployResult` struct in deployment-manager mirroring the fields we need: `Artifacts []DeployArtifactResult` and `UpdateURL`
2. Define local `DeployArtifactResult` struct: `ArtifactID int64`, `Platform string`
3. Define local `BuildProvenance` struct: `Version string`, `GitCommitHash string`
4. Extraction helper: marshal `Status.Stages["deploy"].Details` (interface{}) to JSON, unmarshal into local `DeployResult`; marshal `Status.Provenance` to JSON, unmarshal into local `BuildProvenance`

### Phase 3: Orchestrator Integration (Step 9: Publish to LPBS)
1. After visual validation (current final step), add Step 9 "Publish to LPBS":
   - Trigger scenario-to-desktop pipeline via `POST /api/v1/pipeline/run` with deploy-stage-only config, passing pre-built installer artifact paths
   - Poll `GET /api/v1/pipeline/{id}` until completion
   - On success: extract `DeployResult` from `Status.Stages["deploy"].Details` via JSON round-trip
   - Extract `Version` and `GitCommitHash` from `Status.Provenance` via JSON round-trip
   - For each `DeployArtifactResult`, call `RecordPublish` with profile ID, platform, version, git hash, artifact ID, deployment ID
2. Non-fatal error handling: wrap persist in a recovery block — log errors but do not fail the deployment
3. Update `DeployDesktopResponse` to optionally include published version info

### Phase 4: Query Endpoint
1. Add `GET /api/v1/profiles/{id}/published-versions` endpoint
2. Handler in `deployments` package (or new handler file), registered in `server/routes.go`
3. Returns latest published version per platform (using `DISTINCT ON` window pattern)
4. Optional `?platform=` filter and `?history=true` for full ordered history with `?limit=` support

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

### Local DeployResult Types (in deployment-manager)

```go
// Local mirror of scenario-to-desktop's DeployResult — decoupled via JSON round-trip
type PipelineDeployResult struct {
    Artifacts []PipelineDeployArtifact `json:"artifacts"`
    UpdateURL string                   `json:"update_url"`
}

type PipelineDeployArtifact struct {
    ArtifactID int64  `json:"artifact_id"`
    Platform   string `json:"platform"`
}

type PipelineBuildProvenance struct {
    Version       string `json:"version"`
    GitCommitHash string `json:"git_commit_hash"`
}
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

**Response (history mode, `?history=true&platform=windows&limit=10`):**
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
    },
    {
      "platform": "windows",
      "version": "1.2.2",
      "git_commit_hash": "def456...",
      "artifact_id": 38,
      "deployment_id": "deploy-abc",
      "published_at": "2026-03-28T10:00:00Z"
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

### Pipeline Trigger Config

When the orchestrator triggers scenario-to-desktop's pipeline for the deploy stage only:
- Set `stages: ["deploy"]` (or equivalent config field) to skip build/package stages
- Pass pre-built installer paths as input artifacts in the pipeline run request
- The exact request shape depends on scenario-to-desktop's `POST /api/v1/pipeline/run` contract

## 8. Testing Plan

### Unit Tests
1. **Repository layer** (testcontainers): `RecordPublish` inserts correctly; `GetLatestByProfile` returns only latest per platform; `GetHistory` returns ordered history with limit; multiple publishes for same platform return correct latest
2. **DeployResult deserialization**: Test JSON round-trip extraction from mock `interface{}` pipeline status details — verify correct parsing of artifacts array and provenance fields
3. **Orchestrator extraction logic**: Test that version data is correctly extracted from a mock pipeline status response and passed to repository; test that each platform artifact produces a separate `RecordPublish` call
4. **Non-fatal error handling**: Test that repository errors during persist are logged but don't propagate as deployment failures; orchestrator step reports success even when persist fails
5. **Handler layer**: Test query endpoint with mocked repository — latest mode, history mode, platform filter, empty results

### Integration Tests
1. **End-to-end**: Deploy flow triggers pipeline → extracts version → persists → query endpoint returns it
2. **Multiple publishes**: Second publish for same platform appears as latest; first still in history
3. **Multi-platform**: Single deploy with Windows + Mac + Linux artifacts creates three records; latest endpoint returns all three
4. **Failed deploy**: Pipeline failure means no version record is created

### What to mock
- Pipeline status response (for orchestrator unit tests) — provide realistic `interface{}` Details
- Repository interface (for handler unit tests)
- Pipeline client (for orchestrator unit tests — mock the trigger + poll)
- Real DB via testcontainers (for repository + integration tests)

## 9. Rollout/Validation Checklist

- [ ] Schema migration runs cleanly on fresh and existing databases
- [ ] Orchestrator Step 9 triggers deploy-stage-only pipeline with correct config
- [ ] Orchestrator extracts DeployResult and BuildProvenance via JSON round-trip
- [ ] Orchestrator persists one record per platform per deploy
- [ ] GET endpoint returns correct published versions (latest per platform)
- [ ] History mode returns full publish history with limit
- [ ] Platform filter works correctly
- [ ] Failed deploy does not create version records
- [ ] Non-fatal: persist failure doesn't break deploy response
- [ ] Existing deploy flow (steps 1-8) is not broken by new Step 9
- [ ] `go build ./...` and `go test ./...` succeed for deployment-manager

## 10. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Schema migration fails on existing data | Low | Medium | Migration is additive (new table), no existing data affected |
| Orchestrator error during version persist breaks deploy | Medium | High | Persist in a non-fatal path — log error but don't fail the deployment |
| Version drift if deploy succeeds but persist fails | Low | Low | Can always re-derive from LPBS; this is a convenience cache |
| Pipeline status `Details` field needs JSON round-trip | Low | Low | Well-understood Go pattern: marshal to JSON, unmarshal to typed struct; tested explicitly |
| Append-only table grows unbounded | Low | Low | Typical publish frequency is very low; add retention policy later if needed |
| Deploy-stage-only pipeline config may not be supported yet | Medium | Medium | Verify scenario-to-desktop supports selective stage execution before implementing; if not, add stage filtering as a prerequisite |
| Pre-built artifact paths may not be in the format deploy stage expects | Low | Medium | Test with actual artifact paths from orchestrator build step; add path normalization if needed |

## 11. Non-goals / Prohibited Patterns

- Do NOT modify LPBS (separate scenario, separate backlog items)
- Do NOT add UI for viewing published versions (can be a follow-up)
- Do NOT implement full webhook system (separate backlog item)
- Do NOT add backward-compatibility shims for the new table
- Do NOT add UNIQUE constraints — this is intentionally append-only
- Do NOT import scenario-to-desktop Go types directly — use local mirror structs with JSON round-trip

## 12. Definition of Done

- [ ] `published_versions` table exists in deployment-manager schema
- [ ] `PublishedVersionsRepository` interface + SQL implementation in `deployments` package
- [ ] Local `PipelineDeployResult` and `PipelineBuildProvenance` types for deserialization
- [ ] Orchestrator Step 9 triggers deploy-stage-only pipeline and extracts version data
- [ ] Version persisted per platform per deploy via `RecordPublish`
- [ ] Non-fatal error handling: persist failure logged but deploy succeeds
- [ ] `GET /api/v1/profiles/{id}/published-versions` returns current versions
- [ ] History query support with `?history=true&platform=X&limit=N`
- [ ] All tests pass (unit + integration)
- [ ] `go build ./...` and `go test ./...` succeed for deployment-manager

## 13. File Change Manifest

| File | Action | Description |
|------|--------|-------------|
| `scenarios/deployment-manager/initialization/postgres/schema.sql` | Edit | Add `published_versions` table + index |
| `scenarios/deployment-manager/api/deployments/published_versions.go` | Create | `PublishedVersion` type, `PublishedVersionsRepository` interface, SQL implementation |
| `scenarios/deployment-manager/api/deployments/published_versions_test.go` | Create | Repository unit tests using testcontainers |
| `scenarios/deployment-manager/api/deployments/pipeline_types.go` | Create | Local `PipelineDeployResult`, `PipelineDeployArtifact`, `PipelineBuildProvenance` mirror types + extraction helper |
| `scenarios/deployment-manager/api/deployments/pipeline_types_test.go` | Create | JSON round-trip deserialization tests |
| `scenarios/deployment-manager/api/deployments/orchestrator.go` | Edit | Add Step 9: trigger deploy-stage-only pipeline, extract version, persist via repository |
| `scenarios/deployment-manager/api/deployments/orchestrator_test.go` | Edit/Create | Test version extraction + non-fatal error handling in Step 9 |
| `scenarios/deployment-manager/api/server/routes.go` | Edit | Register `GET /api/v1/profiles/{id}/published-versions` |
| `scenarios/deployment-manager/api/deployments/handlers.go` or new handler file | Edit/Create | Handler for published versions query endpoint |
| `scenarios/deployment-manager/api/deployments/handlers_test.go` | Edit/Create | Handler unit tests with mocked repository |
