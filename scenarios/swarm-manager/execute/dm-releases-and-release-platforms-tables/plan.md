# Implementation Plan: Add Releases and Release Platforms Tables

> **Greenfield declaration:** This task creates new files only. No existing files are modified except `server.go` and `routes.go` (additive wiring).

## Required Reading

```bash
prompt-manager skill read seam-discovery-and-enforcement implementation-plan-authoring api-steer test
swarm-manager backlog file-get --kind research --name release-record-contract-and-history-model --path conclusion.md
```

## 1. Purpose

Add the `releases` and `release_platforms` tables to deployment-manager, along with Go types, repository layer (CRUD + promotion + supersede logic), and handler scaffolding for release history API endpoints. This is the foundational data layer for the desktop release governance initiative.

## 2. Problem Statement

Deployment-manager has no first-class "release event" record. Version history is implicitly scattered across `published_versions`, `deployment_approvals`, and LPBS artifacts. The research conclusion (research/release-record-contract-and-history-model) designed a canonical release record schema that unifies these concepts. This task implements that schema and the Go data access layer.

## 3. Scope

### In Scope
- SQL migration `005_add_releases.sql` (tables, indexes, constraints)
- Go types: `Release`, `ReleasePlatform`, channel/status constants (using `sql.Null*` types for nullable fields)
- Repository interface + SQL implementation (CRUD, promotion, supersede with `SELECT ... FOR UPDATE`)
- Full CRUD handler scaffolding (List, Get, Create) for release history API endpoints
- `released_by` column included as nullable TEXT (populated later by orchestrator integration)
- EnsureSchema for idempotent startup
- Server wiring (repo creation, handler registration, route setup)
- Repository integration tests using testcontainers in `deployments/` package
- Handler unit tests

### Out of Scope
- Orchestrator integration (wiring release creation into deploy flow — separate backlog item)
- LPBS `release_id` correlation column (separate item)
- S2D pipeline changes (separate item)
- Channel promotion endpoint logic beyond repository layer
- UI changes

## 4. Current Technical Context

### Key Files
- `api/migrations/` — existing migrations 002-004; next is **005**
- `api/deployments/` — approvals, published_versions patterns to follow
  - `approvals_types.go` — Go types with `sql.NullString`, `sql.NullTime` for nullable columns
  - `approvals_repository.go` — Interface + `SQLApprovalsRepository` with `conn shared.DBTX` / `db *sql.DB`, `WithTx`, `EnsureSchema`
  - `approvals_handler.go` — Struct with repo + logFn, factory constructor
  - `published_versions.go` — Simpler repository pattern, same DBTX approach
- `api/server/server.go` — Handler wiring, repo creation, EnsureSchema calls (lines 110-141)
- `api/server/routes.go` — Route registration with nil-guard pattern (lines 71-84)
- `api/shared/dbtx.go` — DBTX interface for tx support
- `api/shared/response.go` — JSON response helpers

### Established Patterns (from round 1 decisions)
- **Package placement**: Releases stay in `deployments/` package (d1→A), alongside approvals and published_versions
- **Repository**: Interface + `SQL*Repository` struct with `conn shared.DBTX` + `db *sql.DB`
- **WithTx**: Returns new repo instance backed by transaction
- **EnsureSchema**: `CREATE TABLE IF NOT EXISTS` called at startup (non-fatal warning on failure)
- **Handler**: Struct with repo + log func, factory constructor
- **Routes**: Registered in `setupRoutes()`, nil-checked for optional handlers
- **IDs**: TEXT PRIMARY KEY (UUID strings)
- **Timestamps**: TIMESTAMPTZ with `NOW()` default
- **Null handling**: `sql.NullString`, `sql.NullInt64`, `sql.NullTime` + helper functions (d1 round 2 pre-selected→A)

## 5. Target End State

A fully functional releases data layer in `deployments/` providing:
1. Migration creating both tables with proper FKs, indexes, and unique constraints
2. Go types using `sql.Null*` for nullable fields, matching existing package conventions
3. Repository with: Create, Get, List (with filters + limit/offset pagination), UpdateStatus (with atomic supersede via `SELECT ... FOR UPDATE`), Promote, platform CRUD
4. Full CRUD HTTP handlers: list releases by profile, get release detail, create release
5. All wired into server startup and routes

## 6. Implementation Strategy

### Phase 1: Migration + Types
1. Create `api/migrations/005_add_releases.sql` with schema from research Finding 7
2. Create `api/deployments/releases_types.go` with Release, ReleasePlatform structs and constants
   - Use `sql.NullString` for: `DeploymentID`, `ReleaseNotes`, `ReleasedBy`, `PromotedFromReleaseID`, `Error`, `ApprovalID`
   - Use `sql.NullInt64` for: `LPBSArtifactID`
   - Use `sql.NullTime` for: `PublishedAt`
   - Use `*time.Time` only for JSON response DTOs if needed

### Phase 2: Repository Layer
3. Create `api/deployments/releases_repository.go`:
   - `ReleasesRepository` interface
   - `SQLReleasesRepository` struct following DBTX + WithTx pattern
   - `EnsureSchema` method (CREATE TABLE IF NOT EXISTS for both tables + indexes)
   - CRUD: `Create`, `Get`, `GetByProfileAndVersion`, `ListByProfile` (channel/status filters, limit/offset pagination defaulting to limit=50, offset=0)
   - Status management: `UpdateStatus` (sets status + published_at when publishing)
   - **Supersede logic** (d2 round 1→A): In same transaction as publish — `SELECT ... FOR UPDATE` on previous `published` release for (profile_id, channel), then set to `superseded`
   - Platform operations: `AddPlatform`, `UpdatePlatformStatus`, `ListPlatforms`
   - Promotion: `Promote` (creates new release with `promoted_from_release_id`)

### Phase 3: Handler Scaffolding
4. Create `api/deployments/releases_handler.go` (d3 round 1→A: full CRUD):
   - `ReleasesHandler` struct with repo + log
   - `ListByProfile` — GET with query params for channel, status, limit, offset
   - `Get` — GET single release with platforms embedded
   - `Create` — POST (for orchestrator use, may be internal-only initially)

### Phase 4: Server Wiring
5. Update `api/server/server.go`:
   - Create `SQLReleasesRepository`, call `EnsureSchema`
   - Create `ReleasesHandler`
   - Add `ReleasesHandler` field to `Server` struct
6. Update `api/server/routes.go`:
   - Register release endpoints under profile and top-level paths (nil-guarded)

### Phase 5: Tests
7. Repository integration tests in `api/deployments/releases_repository_test.go` using testcontainers (postgres:15-alpine)
   - Self-contained test DB setup running migrations 002-005
8. Handler unit tests in `api/deployments/releases_handler_test.go` with mock repository

## 7. Contract Decisions

### API Endpoints
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/profiles/{id}/releases` | List releases for profile (query: channel, status, limit, offset) |
| GET | `/api/v1/releases/{id}` | Get release detail with platforms |
| POST | `/api/v1/profiles/{id}/releases` | Create release (internal/orchestrator use) |

### Status Transitions
- `pending` → `publishing` → `published` → `superseded`
- `pending` → `publishing` → `failed`
- Platform statuses: `pending` → `building` → `uploading` → `published` / `failed`

### Supersede Transaction (d2 round 1→A)
When a release reaches `published`:
1. BEGIN TX
2. SELECT id FROM releases WHERE profile_id=$2 AND channel=$3 AND status='published' FOR UPDATE
3. UPDATE releases SET status='published', published_at=NOW(), updated_at=NOW() WHERE id=$1
4. UPDATE releases SET status='superseded', updated_at=NOW() WHERE profile_id=$2 AND channel=$3 AND status='published' AND id != $1
5. COMMIT

### Pagination
- `ListByProfile` uses limit+offset (default: limit=50, offset=0)
- Results ordered by `created_at DESC`

## 8. Testing Plan

### Integration Tests (repository — in deployments/ package)
- Create release + verify Get returns it with correct fields
- Create two releases for same (profile, channel), publish second → verify first is superseded
- Concurrent supersede: verify FOR UPDATE prevents double-publish
- Unique constraint: attempt duplicate (profile_id, git_commit_hash, channel) → error
- Platform CRUD: add platforms, update status, verify listing
- Promotion: create nightly, promote to beta → verify promoted_from_release_id
- List with filters: by channel, by status, with limit/offset
- List pagination: verify offset skips, limit caps

### Unit Tests (handlers)
- ListByProfile: valid request, empty results, with filters, with pagination params
- Get: found, not found (404)
- Create: valid, missing fields (400)

### Test DB Setup
- Use testcontainers with `postgres:15-alpine`
- Run migrations 002-005 in order to establish dependent tables
- Create test profile in setup (releases FK to profiles)
- Cleanup between tests using `t.Cleanup`

## 9. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| FK to `profiles(id)` requires profiles table in test DB | Test setup complexity | Run full migration chain in testcontainers setup; create test profile |
| FK to `deployments(id)` is nullable — need to handle NULL | Data integrity | Use `sql.NullString` for deployment_id, allow NULL in Create |
| FK to `deployment_approvals(id)` on release_platforms | Cross-table dependency | `approval_id` is nullable, no strict FK enforcement initially |
| Supersede race condition if two publishes happen concurrently | Double-published state | `SELECT ... FOR UPDATE` serializes concurrent publishes per (profile, channel) |
| Migration 005 number collision if another task lands first | Migration ordering | Check migration files before implementation; coordinate via backlog |

## 10. Non-goals / Prohibited Patterns

- Do NOT modify orchestrator.go or deploy flow (separate backlog item)
- Do NOT add LPBS correlation columns (separate item)
- Do NOT create a separate `releases/` package — keep in `deployments/`
- Do NOT add compatibility shims for the old published_versions table
- Do NOT add channel promotion validation logic (separate item) — just the data operation
- Do NOT add authentication/authorization middleware for release endpoints

## 11. Definition of Done

- [ ] Migration `005_add_releases.sql` applies cleanly on fresh DB
- [ ] Go types compile with correct JSON tags and sql.Null* handling
- [ ] Repository CRUD operations pass integration tests
- [ ] Supersede atomicity verified: publish + supersede in single transaction with FOR UPDATE
- [ ] Unique constraint (profile_id, git_commit_hash, channel) enforced
- [ ] Handler endpoints return correct JSON responses
- [ ] Server starts successfully with new repo/handler wired in
- [ ] All existing tests continue to pass
- [ ] `go build ./...` and `go test ./...` pass

## 12. Rollout / Validation Checklist

- [ ] `go build ./...` compiles without errors
- [ ] `go test ./... -timeout 300s` — all tests pass (existing + new)
- [ ] `gofumpt -l .` — no formatting violations
- [ ] `golangci-lint run` — no new lint warnings
- [ ] Manual smoke: start server, verify EnsureSchema logs, hit GET `/api/v1/profiles/{id}/releases` returns empty list
- [ ] `vrooli scenario restart deployment-manager`
