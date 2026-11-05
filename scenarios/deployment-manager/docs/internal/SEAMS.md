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
| `readiness.EvidenceProducer` | `api/readiness/preparation.go` | Deployment Manager-owned policy, asset, provenance, ramp, recovery, cloud observability, and candidate operational-ownership producers; explicitly unavailable adapters for unconnected DM bindings; external owner bindings are read from the exact repository observation | Fixed producers plus repository observation fakes in preparation tests | Keeps producer registration separate from executed owner evidence, refuses request-supplied passed signals for producer-bound or human-review criteria, prevents stale observations from satisfying unavailable DM-owned bindings, and reads external observations only for the exact identity and declared binding |
| `releases.RecoveryExecutor` | `api/releases/types.go`, `api/deployments/orchestrator.go` | Exact release identity routed to the owning cloud client | `recoveryExecutorFake` and HTTP owner fixture | Keeps recovery authorization and durable DM receipts separate from owner-side VPS mutation |
| `readiness.PredecessorResolver` | `api/readiness/preparation.go` | `server.readinessPredecessorResolver` | `fixedPredecessor` and server adapter fakes | Selects the latest actually published release for the same profile, target set, and channel |
| `deployments.commandRunner` | `api/deployments/orchestrator.go`, `orchestrator_helpers.go` | `processCommandRunner` | `recordingCommandRunner` in `orchestrator_helpers_test.go` | Keeps installer process execution injectable so packaging tests do not require a package manager or a real child process |
| `capabilities.scenarioStatusRunner` | `api/internal/capabilities/registry.go` | `controlPlaneStatusRunner` | `scenarioStatusRunnerFunc` in `registry_test.go` | Keeps dependency-health checks on the control-plane status contract without requiring a live `vrooli` process in unit tests |
| `readiness.goalOpener` / goal reader | `api/readiness/goal.go` and `api/handlers/readiness/connect_handler.go` | Swarm Manager `GoalClient` | `memoryGoals` and goal-client HTTP fixtures | Keeps independent work projection and verified close-out outside Deployment Manager |
| Release readiness lookup/promoter | `api/releases/handlers.go`, `api/readiness/review.go` | Readiness repository adapters in `api/server/server.go` | handler callbacks | Enforces exact approval, current identity/freshness/disposition/goal standing before release, and records promoted lifecycle only after publication |
| Canonical release contracts | `api/releases/identity.go`, `identity_proto.go` | Generated `deployment-manager/v1/releases/contracts.proto` bindings | `identity_test.go` golden and tamper cases | Keeps candidate, capability declaration, destination, review, receipt, operation, and target dimensions stable across Go and typed transports |
| Release dossier projection | `api/releases/handlers.go`, `api/releases/health.go` | REST compatibility read plus typed `ReleasesService.Dossier` adapter | `handlers_test.go`, `internal/transport/connect_handler_test.go` | Keeps reviewer retrieval read-only and makes missing identity/effect proof explicit |
| Owner observation and reconciliation | `api/releases/handlers.go` | Cloud, LPBS, and desktop owner clients | owner HTTP fixtures and recovery tests | Preserves ambiguous or failed standing until the owner supplies attributable current evidence |

## Readiness ownership

Deployment Manager owns policy, normalized references, aggregation, identity,
review lifecycle, and release disposition. Evidence-producing scenarios retain
their raw artifacts and report only attributable status, version, observation
time, and external reference. Swarm Manager owns the independent goal. The
release target owns recovery execution; deployment-manager checks the exact
release/review/candidate/destination/deployment binding before routing recovery.
The governed recovery program remains read-only until an explicit grant and
dry-run evidence are supplied, and it records only an owner-produced effect
receipt.

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
