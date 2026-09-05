# SEAMS - deployment-manager

Architectural seams and variation points for the deployment-manager scenario.

## Seam Registry

| Seam | Declaration | Production implementation | Test double | Why it exists |
|---|---|---|---|---|
| `deployments.ApprovalsRepository` | `api/deployments/approvals_repository.go` | `SQLApprovalsRepository` | `sqlmock` expectations in `approvals_repository_test.go` | Isolates release-gate persistence and makes every gate state executable |
| `profiles.Repository` | `api/profiles/repository.go` | `SQLRepository` | `mockRepository` in handler tests and `sqlmock` repository tests | Keeps profile storage out of handlers and orchestration policy |
| `deployments.CloudHealthClient` | `api/deployments/cloud_client.go` | HTTP client | `fakeCloudClient` in orchestration tests | Makes optional cloud readiness deterministic |
| `deployments.LPBSReleaseClient` | `api/deployments/lpbs_release_client.go` | HTTP client | `fakeLPBSClient` in orchestration tests | Makes release verification deterministic |
| `codesigning.Repository` | `api/codesigning/interfaces.go` | scenario-to-desktop proxy | `mockRepository` in `handler_test.go` | Keeps signing credentials and execution owned by the ramp |
| `evidence.Repository` | `api/internal/evidence/repository.go` | `SQLRepository` | `fakeEvidenceRepository` and `FakeProducer` | Keeps verdict persistence deterministic and reference-only |
| `readiness.ReviewRepository` | `api/readiness/review.go` | `internal/readiness.SQLRepository` | `memoryReviewRepository` plus SQLite repository tests | Separates immutable review, evidence, human check, waiver, goal, approval, and promotion state |
| `readiness.EvidenceProducer` | `api/readiness/preparation.go` | Policy-keyed `ObservationProducer` instances | Fixed and memory producers in preparation tests | Reads only typed observations for the exact identity and declared producer binding |
| `readiness.PredecessorResolver` | `api/readiness/preparation.go` | `server.readinessPredecessorResolver` | `fixedPredecessor` and server adapter fakes | Selects the latest actually published release for the same profile, target set, and channel |
| `readiness.goalOpener` / goal reader | `api/readiness/goal.go` and `api/handlers/readiness/connect_handler.go` | Swarm Manager `GoalClient` | `memoryGoals` and goal-client HTTP fixtures | Keeps independent work projection and verified close-out outside Deployment Manager |
| Release readiness lookup/promoter | `api/releases/handlers.go` | Readiness repository adapters in `api/server/server.go` | handler callbacks | Enforces exact approval before release and records promoted lifecycle only after publication |

## Readiness ownership

Deployment Manager owns policy, normalized references, aggregation, identity,
review lifecycle, and release disposition. Evidence-producing scenarios retain
their raw artifacts and report only attributable status, version, observation
time, and external reference. Swarm Manager owns the independent goal. The
release target owns recovery execution; the governed recovery program refuses
to improvise when that owner operation is unavailable.

## Deployment Approvals Seams

### ApprovalsRepository Seam
- **Interface**: `deployments.ApprovalsRepository` (`api/deployments/approvals_repository.go`)
- **Default Implementation**: `SQLApprovalsRepository` — SQLite persistence for per-platform, per-commit approval records
- **Purpose**: Tracks approval status (pending/approved/rejected/stale) tied to specific git commits. When a new commit is built, previous approvals for the same profile+platform are automatically marked stale.
- **Key Methods**: `Create`, `Get`, `ListByCommit`, `ListByProfile`, `UpdateDecision`, `MarkStale`, `CheckReleaseGate`, `GetRequiredPlatforms`, `SetRequiredPlatforms`
- **Test Double**: `sqlmock` rows in `api/deployments/approvals_repository_test.go`

### Release Gate Seam
- **Integration Point**: `Orchestrator.DeployDesktop()` (`api/deployments/orchestrator.go`)
- **Purpose**: Before a deployment proceeds, `DeployDesktopRequest` must carry an exact commit and the orchestrator checks `ApprovalsRepository.CheckReleaseGate()` for that commit
- **Bypass**: None. Missing commit identifiers are rejected with HTTP 400.
- **Status Values**: pending, approved, rejected, stale, missing

### Required Platforms Seam
- **Storage**: `profile_required_platforms` table (profile_id, platform)
- **Purpose**: Configurable per profile — defines which platforms must be approved before the release gate opens
- **Endpoints**: `PUT/GET /api/v1/profiles/{id}/required-platforms`
