# Implementation Plan: Add Releases and Release Platforms Tables

## Required Reading

```bash
prompt-manager skill read seam-discovery-and-enforcement implementation-plan-authoring
swarm-manager backlog file-get --kind research --name release-record-contract-and-history-model --path conclusion.md
```

## 1. Purpose

Add the `releases` and `release_platforms` tables to deployment-manager, along with Go types, repository layer (CRUD + promotion + supersede logic), and handler scaffolding for release history API endpoints. This is the foundational data layer for the desktop release governance initiative.

## 2. Problem Statement

Deployment-manager has no first-class "release event" record. Version history is implicitly scattered across `published_versions`, `deployment_approvals`, and LPBS artifacts. The research conclusion (research/release-record-contract-and-history-model) designed a canonical release record schema that unifies these concepts. This task implements that schema and the Go data access layer.

## 3. Scope

### In Scope
- SQL migration `005_add_releases.sql` (tables, indexes, constraints)
- Go types: `Release`, `ReleasePlatform`, channel/status constants
- Repository interface + SQL implementation (CRUD, promotion, supersede)
- Handler scaffolding for release history API endpoints
- EnsureSchema for idempotent startup
- Server wiring (repo creation, handler registration, route setup)
- Unit/integration tests for repository and handlers

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
- `api/server/server.go` — handler wiring, repo creation, EnsureSchema calls
- `api/server/routes.go` — route registration
- `api/shared/dbtx.go` — DBTX interface for tx support
- `api/shared/response.go` — JSON response helpers
- `initialization/postgres/schema.sql` — base schema (profiles, deployments tables)

### Established Patterns
- **Repository**: Interface + `SQL*Repository` struct with `conn shared.DBTX` + `db *sql.DB`
- **WithTx**: Returns new repo instance backed by transaction
- **EnsureSchema**: `CREATE TABLE IF NOT EXISTS` called at startup (non-fatal)
- **Handler**: Struct with repo + log func, factory constructor
- **Routes**: Registered in `setupRoutes()`, nil-checked for optional handlers
- **IDs**: TEXT PRIMARY KEY (UUID strings)
- **Timestamps**: TIMESTAMPTZ with `NOW()` default
- **Null handling**: `sql.NullString`, `sql.NullInt64`, `sql.NullTime` + helper functions

## 5. Target End State

A fully functional `releases` package (or additions to `deployments/`) providing:
1. Migration creating both tables with proper FKs, indexes, and unique constraints
2. Go types matching the research conclusion schema (Finding 10)
3. Repository with: Create, Get, List (with filters), UpdateStatus, Promote, Supersede, platform CRUD
4. HTTP handlers for: list releases by profile, get release detail, list release platforms
5. All wired into server startup and routes

## 6. Implementation Strategy

### Phase 1: Migration + Types
1. Create `api/migrations/005_add_releases.sql` with schema from research Finding 7
   - **Note**: Research assumed migration 004 but actual next number is **005**
2. Create `api/deployments/releases_types.go` with Release, ReleasePlatform structs and constants

### Phase 2: Repository Layer
3. Create `api/deployments/releases_repository.go`:
   - `ReleasesRepository` interface
   - `SQLReleasesRepository` struct following DBTX + WithTx pattern
   - `EnsureSchema` method
   - CRUD: `Create`, `Get`, `GetByProfileAndVersion`, `ListByProfile` (with channel/status filters)
   - Status management: `UpdateStatus` (sets status + published_at when publishing)
   - **Supersede logic**: In same transaction as publish status update — query previous `published` release for (profile_id, channel), set to `superseded`
   - Platform operations: `AddPlatform`, `UpdatePlatformStatus`, `ListPlatforms`
   - Promotion: `Promote` (creates new release with `promoted_from_release_id`)

### Phase 3: Handler Scaffolding
4. Create `api/deployments/releases_handler.go`:
   - `ReleasesHandler` struct with repo + log
   - `ListByProfile` — GET with query params for channel, status, limit
   - `Get` — GET single release with platforms embedded
   - `Create` — POST (for orchestrator use, may be internal-only initially)

### Phase 4: Server Wiring
5. Update `api/server/server.go`:
   - Create `SQLReleasesRepository`, call `EnsureSchema`
   - Create `ReleasesHandler`
   - Add `ReleasesHandler` field to `Server` struct
6. Update `api/server/routes.go`:
   - Register release endpoints under `/api/v1/profiles/{id}/releases`

### Phase 5: Tests
7. Repository integration tests using testcontainers (postgres:15-alpine)
8. Handler unit tests with mock repository

## 7. Contract Decisions

### API Endpoints
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/profiles/{id}/releases` | List releases for profile (query: channel, status, limit) |
| GET | `/api/v1/releases/{id}` | Get release detail with platforms |
| POST | `/api/v1/profiles/{id}/releases` | Create release (internal/orchestrator use) |

### Status Transitions
- `pending` → `publishing` → `published` → `superseded`
- `pending` → `publishing` → `failed`
- Platform statuses: `pending` → `building` → `uploading` → `published` / `failed`

### Supersede Transaction
When a release reaches `published`:
1. BEGIN TX
2. UPDATE releases SET status='published', published_at=NOW() WHERE id=$1
3. UPDATE releases SET status='superseded', updated_at=NOW() WHERE profile_id=$2 AND channel=$3 AND status='published' AND id != $1
4. COMMIT

## 8. Testing Plan

### Integration Tests (repository)
- Create release + verify Get returns it
- Create two releases for same (profile, channel), publish second → verify first is superseded
- Unique constraint: attempt duplicate (profile_id, git_commit_hash, channel) → error
- Platform CRUD: add platforms, update status, verify listing
- Promotion: create nightly, promote to beta → verify promoted_from_release_id
- List with filters: by channel, by status, with limit

### Unit Tests (handlers)
- ListByProfile: valid request, empty results, with filters
- Get: found, not found (404)
- Create: valid, missing fields (400)

## 9. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| FK to `profiles(id)` requires profiles table in test DB | Test setup complexity | Use same testcontainers pattern as approvals; create test profile in setup |
| FK to `deployments(id)` is nullable — need to handle NULL | Data integrity | Use sql.NullString for deployment_id, allow NULL in Create |
| FK to `deployment_approvals(id)` on release_platforms | Cross-table dependency | approval_id is nullable, no strict FK enforcement needed initially |
| Supersede race condition if two publishes happen concurrently | Double-published state | Transaction + row-level locking (SELECT ... FOR UPDATE) on the supersede query |

## 10. Non-goals / Prohibited Patterns

- Do NOT modify orchestrator.go or deploy flow (separate backlog item)
- Do NOT add LPBS correlation columns (separate item)
- Do NOT create a separate `releases/` package — keep in `deployments/` alongside approvals and published_versions
- Do NOT add compatibility shims for the old published_versions table
- Do NOT add channel promotion validation logic (separate item) — just the data operation

## 11. Definition of Done

- [ ] Migration `005_add_releases.sql` applies cleanly on fresh DB
- [ ] Go types compile with correct JSON tags and null handling
- [ ] Repository CRUD operations pass integration tests
- [ ] Supersede atomicity verified: publish + supersede in single transaction
- [ ] Unique constraint (profile_id, git_commit_hash, channel) enforced
- [ ] Handler endpoints return correct JSON responses
- [ ] Server starts successfully with new repo/handler wired in
- [ ] All existing tests continue to pass
- [ ] `go build ./...` and `go test ./...` pass
