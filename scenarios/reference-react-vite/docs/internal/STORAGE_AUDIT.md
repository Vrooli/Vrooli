# reference-react-vite Storage Architecture Audit

## Last Updated
2026-03-11

## Resource Configuration Status
- [x] postgres declared in service.json
- [x] schema field uses scenario slug (`reference-react-vite`)
- [x] initialization files in `initialization/postgres/schema.sql`
- [ ] initialization referenced in service.json (manual execution required)

## Connection Pattern Status
- [x] Environment variables used via `api-core/database.Connect()`
- [x] Connection retry with exponential backoff (via api-core)
- [x] Connection pool configured (via api-core defaults)
- [x] Health check implemented (via api-core/health)

## Schema Status
- [x] schema.sql exists and is idempotent
- [x] Tables use proper constraints and indexes
- [x] Greenfield default applied (fresh schema, no migrations)
- [x] UUID extension enabled
- [x] Auto-update triggers for updated_at columns

### Tables Defined
| Table | Purpose | Key Constraints |
|-------|---------|-----------------|
| projects | Container for tasks | status check, color regex |
| tasks | Core work items | status check, priority range, FK to projects |
| notes | Task annotations | FK cascade delete to tasks |

### Indexes
- `idx_projects_status`, `idx_projects_created`
- `idx_tasks_project`, `idx_tasks_status`, `idx_tasks_priority`, `idx_tasks_due_date`, `idx_tasks_created`
- `idx_notes_task`, `idx_notes_created`

## Abstraction Status
- [x] Repository interfaces defined (`repository/repository.go`)
- [x] Business logic uses interfaces, not direct DB
- [x] Handlers use repository interfaces (injected)
- [x] Repositories bundled for dependency injection

### Repository Implementations
| Interface | Implementation | File |
|-----------|----------------|------|
| TaskRepository | PostgresTaskRepository | `tasks_postgres.go` |
| ProjectRepository | PostgresProjectRepository | `projects_postgres.go` |
| NoteRepository | PostgresNoteRepository | `notes_postgres.go` |

## Filesystem Status
- [ ] Runtime filesystem writes go through `api-core/storage` (not yet needed)
- [x] Deploy directory treated as disposable
- [ ] Atomic writes used for persisted files (not yet needed)

*Note: This scenario currently only uses PostgreSQL for storage. Filesystem storage patterns would apply if file uploads or local caching were added.*

## Issues Found
None currently - greenfield implementation following best practices.

## Priority Fixes
1. Add initialization file reference to service.json dependencies
2. Add testcontainers setup for repository integration tests
3. Consider adding SQLite implementation for testing

## Patterns Verified

### Environment Variable Usage
```go
// Uses api-core which reads POSTGRES_* from environment
db, err := database.Connect(context.Background(), database.Config{
    Driver: database.DriverPostgres,
})
```

### Repository Pattern
```go
// Interface-based injection in handlers
type TaskHandler struct {
    repo repository.TaskRepository  // Interface, not concrete
}

// Concrete implementations created at composition root
repos := repository.NewRepositories(db)
taskHandler := handlers.NewTaskHandler(repos.Tasks)
```

### Idempotent Schema
```sql
-- All CREATE statements use IF NOT EXISTS
CREATE TABLE IF NOT EXISTS tasks (...)
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)
DROP TRIGGER IF EXISTS tasks_updated_at ON tasks;
CREATE TRIGGER tasks_updated_at ...
```

## Anti-Patterns Avoided
- [x] No hard-coded database credentials
- [x] No direct SQL in handler code
- [x] No scenario-local mutable file writes
- [x] No ad-hoc DATA_DIR conventions
- [x] No shared "vrooli" database without scenario schema
