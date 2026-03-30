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

Meanwhile, scenario-to-desktop has a `WebhookNotifier` interface defined in `pipeline/interfaces.go` but it is **not wired** into the orchestrator. The `DeployResult` struct contains exactly the data needed: `[]DeployArtifactResult{ArtifactID, Platform}` plus `UpdateURL`, and `BuildProvenance` has `Version` and `GitCommitHash`.

## 3. Scope

**In scope:**
- `scenarios/deployment-manager/api/**` — new endpoint + schema changes to record published versions
- `scenarios/scenario-to-desktop/api/**` — wire callback after deploy stage completion

**Out of scope:**
- LPBS webhook (separate backlog item: `lpbs-publish-webhook`)
- Rollback API (separate backlog item: `desktop-rollback-api`)
- UI changes in either scenario
- CLI changes in either scenario

## 4. Current Technical Context

### deployment-manager
- **Schema**: `scenarios/deployment-manager/initialization/postgres/schema.sql`
  - `deployments` table: `id, profile_id, status, started_at, completed_at, artifacts JSONB, message, logs, error`
  - No dedicated table for "published version per platform per profile"
- **Orchestrator**: `scenarios/deployment-manager/api/deployments/orchestrator.go` — drives the deploy flow, already extracts `DeployResult` from pipeline status
- **Desktop client**: `scenarios/deployment-manager/api/deployments/desktop_client.go` — HTTP client for scenario-to-desktop

### scenario-to-desktop
- **Deploy stage**: `scenarios/scenario-to-desktop/api/pipeline/stage_deploy.go` — uploads artifacts to LPBS, produces `DeployResult`
- **Types**: `scenarios/scenario-to-desktop/api/pipeline/types.go` — `DeployResult`, `DeployArtifactResult`, `BuildProvenance`
- **Interfaces**: `scenarios/scenario-to-desktop/api/pipeline/interfaces.go` — `WebhookNotifier` interface (defined, not wired)
- **LPBS client**: `scenarios/scenario-to-desktop/api/deploy/lpbs_client.go` — presign → upload → commit → apply flow

## 5. Target End State

After a successful deploy stage:
1. deployment-manager has a `published_versions` record per `(profile_id, platform)` with: `version`, `git_commit_hash`, `artifact_id`, `published_at`
2. This record is queryable via a new GET endpoint for "what's live?" per profile
3. The orchestrator automatically persists this data when it observes deploy stage completion

## 6. Implementation Strategy

<!-- Approach TBD pending workshop decisions -->

### Option A: Orchestrator-side extraction (deployment-manager pulls)

deployment-manager already polls `GET /pipeline/{id}` and has access to the full `Status` including `DeployResult` and `BuildProvenance`. After the pipeline succeeds:
1. Extract version + platform + artifact ID from `DeployResult`
2. Persist to a new `published_versions` table
3. No changes needed in scenario-to-desktop

**Pros:** Simpler, no new inter-service contract, deployment-manager owns its own data persistence timing.
**Cons:** Only works when deployment-manager initiates the pipeline (not standalone scenario-to-desktop runs).

### Option B: Webhook callback (scenario-to-desktop pushes)

Wire the existing `WebhookNotifier` interface in scenario-to-desktop's orchestrator to POST deploy results to a configurable callback URL. deployment-manager provides the callback URL when initiating the pipeline run.

**Pros:** Works for any pipeline initiator, follows event-driven pattern.
**Cons:** More moving parts, requires new endpoint in deployment-manager AND wiring in scenario-to-desktop.

### Option C: Hybrid (pull + optional push)

Implement Option A first (reliable, minimal scope). Leave webhook wiring as a follow-up when the LPBS publish webhook item is implemented.

## 7. Contract Decisions

<!-- TBD pending workshop decisions -->

### New Table: `published_versions`

```sql
CREATE TABLE IF NOT EXISTS published_versions (
    id SERIAL PRIMARY KEY,
    profile_id VARCHAR(255) NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    version VARCHAR(100) NOT NULL,
    git_commit_hash VARCHAR(64),
    artifact_id BIGINT,
    deployment_id VARCHAR(255) REFERENCES deployments(id),
    published_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(profile_id, platform)
);
```

The UNIQUE constraint on `(profile_id, platform)` means only the latest published version per platform is stored (upsert on conflict).

### New Endpoint: `GET /api/v1/profiles/{id}/published-versions`

Returns the current published version per platform for a profile.

### Orchestrator Update

After successful deploy, extract `DeployResult` and upsert into `published_versions`.

## 8. Testing Plan

<!-- TBD pending workshop decisions on approach -->

1. **Unit test**: Repository layer — upsert and query `published_versions`
2. **Unit test**: Orchestrator — verify published version is persisted after successful deploy
3. **Integration test**: End-to-end deploy flow records version correctly
4. **Regression test**: Failed deploys do NOT update published versions

## 9. Rollout/Validation Checklist

- [ ] Schema migration runs cleanly on fresh and existing databases
- [ ] Orchestrator persists version after successful deploy
- [ ] GET endpoint returns correct published versions
- [ ] Failed deploy does not update published versions
- [ ] Existing deploy flow is not broken

## 10. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Schema migration fails on existing data | Low | Medium | Migration is additive (new table), no existing data affected |
| Orchestrator error during version persist breaks deploy | Medium | High | Persist in a non-fatal path — log error but don't fail the deployment |
| Version drift if deploy succeeds but persist fails | Low | Low | Can always re-derive from LPBS; this is a convenience cache |

## 11. Non-goals / Prohibited Patterns

- Do NOT modify LPBS (separate scenario, separate backlog items)
- Do NOT add UI for viewing published versions (can be a follow-up)
- Do NOT implement full webhook system (separate backlog item)
- Do NOT add backward-compatibility shims for the new table

## 12. Definition of Done

- [ ] `published_versions` table exists in deployment-manager schema
- [ ] Orchestrator persists version per platform after successful deploy
- [ ] `GET /api/v1/profiles/{id}/published-versions` returns current versions
- [ ] All tests pass
- [ ] `go build ./...` and `go test ./...` succeed for deployment-manager
