# SEAMS - deployment-manager

Architectural seams and variation points for the deployment-manager scenario.

## Visual Validation Seams (Mar 2026)

### ValidationRepository Seam
- **Interface**: `validation.Repository` (`api/validation/repository.go`)
- **Default Implementation**: `SQLRepository` — PostgreSQL persistence for validation records
- **Variation Point**: Could be replaced with in-memory store for tests or file-backed store for single-node deployments
- **Test Double**: `inMemoryRepo` in `api/validation/handler_test.go`

### VideoTransport Seam
- **Interface**: `DesktopPackagerClient.DownloadVideo()` (`api/deployments/desktop_client.go`)
- **Default Implementation**: HTTP streaming from scenario-to-desktop `/api/v1/smoketest/{id}/video`
- **Variation Point**: Could be replaced with MinIO/S3 download for multi-server deployments
- **Test Strategy**: Mock HTTP server in client tests

### DesktopPackagerClient Seam (extended)
- **Interface**: Methods on `DesktopPackagerClient` (`api/deployments/desktop_client.go`)
- **New Methods**: `RunSmokeTest`, `GetSmokeTestStatus`, `WaitForSmokeTest`, `DownloadVideo`
- **Default Implementation**: HTTP calls to scenario-to-desktop API
- **Variation Point**: Already an established seam; smoke test methods follow the same HTTP polling pattern as build methods

## Deployment Approvals Seams (Mar 2026)

### ApprovalsRepository Seam
- **Interface**: `deployments.ApprovalsRepository` (`api/deployments/approvals_repository.go`)
- **Default Implementation**: `SQLApprovalsRepository` — PostgreSQL persistence for per-platform, per-commit approval records
- **Purpose**: Tracks approval status (pending/approved/rejected/stale) tied to specific git commits. When a new commit is built, previous approvals for the same profile+platform are automatically marked stale.
- **Key Methods**: `Create`, `Get`, `ListByCommit`, `ListByProfile`, `UpdateDecision`, `MarkStale`, `CheckReleaseGate`, `GetRequiredPlatforms`, `SetRequiredPlatforms`
- **Test Double**: In-memory mock in handler tests (planned)

### Release Gate Seam
- **Integration Point**: `Orchestrator.DeployDesktop()` (`api/deployments/orchestrator.go`)
- **Purpose**: Before a deployment proceeds, if `GitCommitHash` is provided, the orchestrator checks `ApprovalsRepository.CheckReleaseGate()` to verify all required platforms are approved for that exact commit
- **Bypass**: Omit `git_commit_hash` from `DeployDesktopRequest` to skip gate check (allows legacy/ungated deployments)
- **Status Values**: pending, approved, rejected, stale, missing

### Required Platforms Seam
- **Storage**: `profile_required_platforms` table (profile_id, platform)
- **Purpose**: Configurable per profile — defines which platforms must be approved before the release gate opens
- **Endpoints**: `PUT/GET /api/v1/profiles/{id}/required-platforms`
