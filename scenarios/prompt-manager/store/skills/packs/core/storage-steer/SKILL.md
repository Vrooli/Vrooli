## Steer focus: Storage Architecture

Prioritize **hardening persistent storage mechanisms** in `scenarios/{{TARGET}}/` to ensure reliable, isolated, and professionally structured data persistence. This skill steers toward storage that is correctly configured, properly abstracted, and follows Vrooli resource conventions.

Your goal is to ensure the target scenario's storage layer is **production-ready**: using environment-driven configuration, scenario-isolated databases, proper schema management, and clean abstraction boundaries that allow storage mechanisms to change without affecting business logic.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve the scenario's storage architecture.

Required reading:
- `prompt-manager skills read visited-tracker-tools`

---

### 0. Why This Skill Exists

Storage problems are invisible until they cause outages, data corruption, or cross-scenario pollution:

- **Shared database collisions:** Scenarios using default "vrooli" database overwrite each other's tables
- **Hard-coded connection strings:** Credentials in code, impossible to configure per environment
- **Missing schema initialization:** Manual database setup required, deployment failures
- **Tight coupling:** Business logic directly depends on PostgreSQL, cannot swap to SQLite for testing
- **No retry logic:** Transient database failures cause cascading application crashes
- **Redis key collisions:** Multiple scenarios write to same keys, unpredictable behavior
- **Filesystem sprawl:** Data written to arbitrary paths, security and cleanup issues
- **Redeploy stale file drift:** Old files survive deployments when mutable state is stored in app targets
- **Data-loss risk on cleanup:** Clearing deploy targets can delete real runtime data when storage lives under scenario folders

**The Vrooli resource system solves this** by providing:
- Shared resources (postgres, redis, qdrant) with environment variable injection
- Scenario-specific schema isolation via service.json configuration
- Standard initialization patterns for schema and seed data

But "using resources" alone isn't enough. This skill ensures storage is:
- **Properly declared** in service.json with correct schema naming
- **Environment-driven** using injected variables, never hard-coded
- **Well-abstracted** behind repository/service interfaces
- **Correctly initialized** with idempotent schema files
- **Properly isolated** to prevent cross-scenario data pollution
- **Filesystem-safe by default** via `github.com/vrooli/api-core/storage`

---

### 1. Scope Boundaries

**In scope**
- service.json resource dependency declaration and schema naming
- Environment variable usage for database connections
- Connection patterns: retries, pooling, health checks
- Schema initialization files and idempotency patterns
- Storage abstraction: repository/service layer design
- Redis key namespacing and Qdrant collection naming
- Filesystem storage standardized on `api-core/storage`
- Greenfield-first storage posture and brownfield migration exceptions

**Out of scope**
- Database query optimization or indexing strategy -> see performance skills
- Data modeling and domain design -> see domain architecture skills
- Backup and disaster recovery -> operational concern
- Database administration (users, roles, permissions) -> infrastructure concern
- Specific ORM/driver selection -> implementation choice

---

### 2. The Vrooli Storage Hierarchy

Understanding how Vrooli manages storage prevents configuration mistakes:

```
                    VROOLI STORAGE HIERARCHY
┌─────────────────────────────────────────────────────────┐
│  resources/*/config/defaults.sh                         │
│  ─────────────────────────────                          │
│  RESOURCE CONFIGURATION                                 │
│  • Defines environment variables (POSTGRES_HOST, etc.)  │
│  • Sets defaults (ports, users, container names)        │
│  • Vrooli lifecycle injects these into scenario env     │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼ lifecycle injection
┌─────────────────────────────────────────────────────────┐
│  scenarios/{{TARGET}}/.vrooli/service.json              │
│  ─────────────────────────────────────────              │
│  RESOURCE DEPENDENCIES                                  │
│  • Declares which resources scenario needs              │
│  • Specifies schema name for database isolation         │
│  • Points to initialization files                       │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼ on first startup
┌─────────────────────────────────────────────────────────┐
│  initialization/storage/postgres/schema.sql             │
│  ─────────────────────────────────────────              │
│  SCHEMA INITIALIZATION                                  │
│  • Creates tables, indexes, triggers                    │
│  • Must be idempotent (IF NOT EXISTS patterns)          │
│  • Executed by API on first startup                     │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼ at runtime
┌─────────────────────────────────────────────────────────┐
│  Scenario Code (API, CLI, Services)                     │
│  ─────────────────────────────────                      │
│  STORAGE CONSUMERS                                      │
│  • Connect using environment variables                  │
│  • Access through repository/service abstraction        │
│  • Use api-core/storage for filesystem runtime state    │
│  • Never hard-code connection details                   │
└─────────────────────────────────────────────────────────┘
```

**Steer:** Configuration flows downward. Scenarios consume what Vrooli provides. Never bypass the resource system.

---

### 3. service.json Resource Declaration

#### 3.1 Database Resource Pattern

Every scenario using persistent storage MUST declare it in service.json:

```json
{
  "dependencies": {
    "resources": {
      "postgres": {
        "type": "postgres",
        "enabled": true,
        "required": true,
        "description": "Primary storage for [describe data]",
        "schema": "{{TARGET}}"
      }
    }
  }
}
```

**Schema naming convention:**
- Use the scenario slug (the folder name) as the schema name
- Prefer hyphens for consistency: `agent-manager`, `browser-automation-studio`
- This creates an isolated database/schema that won't conflict with other scenarios

#### 3.2 Initialization Declaration

For scenarios with complex schemas, declare initialization files:

```json
{
  "dependencies": {
    "resources": {
      "postgres": {
        "type": "postgres",
        "enabled": true,
        "required": true,
        "description": "...",
        "schema": "{{TARGET}}",
        "initialization": [
          {
            "file": "initialization/storage/postgres/schema.sql",
            "type": "schema"
          },
          {
            "file": "initialization/storage/postgres/seed.sql",
            "type": "seed"
          }
        ]
      }
    }
  }
}
```

#### 3.3 Multiple Storage Resources

Scenarios may use multiple storage types:

```json
{
  "dependencies": {
    "resources": {
      "postgres": {
        "type": "postgres",
        "enabled": true,
        "required": true,
        "description": "Structured data and indexes",
        "schema": "{{TARGET}}"
      },
      "redis": {
        "type": "redis",
        "enabled": true,
        "required": false,
        "description": "Caching and session storage"
      },
      "qdrant": {
        "type": "qdrant",
        "enabled": true,
        "required": false,
        "description": "Vector embeddings for semantic search"
      },
      "minio": {
        "type": "minio",
        "enabled": true,
        "required": false,
        "description": "Object storage for large files"
      }
    }
  }
}
```

**Convergence Pattern: Resource Selection**

```
What type of data are you storing?
│
├─ Structured, relational, queryable?
│   └─ PostgreSQL (primary choice for most data)
│
├─ Key-value, ephemeral, cached?
│   └─ Redis (sessions, rate limits, real-time state)
│
├─ Vector embeddings for similarity search?
│   └─ Qdrant (semantic search, recommendations)
│
├─ Large binary files (images, videos, documents)?
│   └─ MinIO/S3 or filesystem (with DB index)
│
└─ Small, simple, single-user storage?
    └─ SQLite (desktop deployments, testing)
```

---

### 4. Environment Variable Usage

#### 4.1 PostgreSQL Environment Variables

The Vrooli resource system exports these variables (from `resources/postgres/config/defaults.sh`):

| Variable | Purpose | Default |
|----------|---------|---------|
| `POSTGRES_HOST` | Database hostname | `localhost` |
| `POSTGRES_PORT` | Database port | `5433` (Vrooli default, not 5432) |
| `POSTGRES_USER` | Database user | `vrooli` |
| `POSTGRES_PASSWORD` | Database password | (generated) |
| `POSTGRES_DB` | Database name | `vrooli` (override with schema) |
| `DATABASE_URL` | Full connection string | (constructed) |

#### 4.2 Correct Connection Pattern (Go)

```go
// ✅ CORRECT: Use environment variables, override DB name with scenario schema
func connectPostgres() (*sql.DB, error) {
    host := os.Getenv("POSTGRES_HOST")
    port := os.Getenv("POSTGRES_PORT")
    user := os.Getenv("POSTGRES_USER")
    password := os.Getenv("POSTGRES_PASSWORD")
    dbName := os.Getenv("POSTGRES_DB")

    // Override with scenario-specific database if using default
    if dbName == "" || dbName == "vrooli" {
        dbName = "{{TARGET}}"  // Use scenario slug
    }

    connStr := fmt.Sprintf(
        "postgres://%s:%s@%s:%s/%s?sslmode=disable",
        user, password, host, port, dbName,
    )

    return sql.Open("postgres", connStr)
}
```

```go
// ❌ WRONG: Hard-coded values
func connectPostgres() (*sql.DB, error) {
    return sql.Open("postgres",
        "postgres://vrooli:password123@localhost:5432/vrooli?sslmode=disable")
}
```

#### 4.3 Correct Connection Pattern (TypeScript)

```typescript
// ✅ CORRECT: Environment-driven configuration
const dbConfig = {
    host: process.env.POSTGRES_HOST || 'localhost',
    port: parseInt(process.env.POSTGRES_PORT || '5433', 10),
    user: process.env.POSTGRES_USER || 'vrooli',
    password: process.env.POSTGRES_PASSWORD,
    database: process.env.POSTGRES_DB || '{{TARGET}}',
};

// Or using DATABASE_URL
const connectionString = process.env.DATABASE_URL;
```

```typescript
// ❌ WRONG: Hard-coded credentials
const pool = new Pool({
    connectionString: 'postgres://vrooli:secret@localhost:5432/mydb'
});
```

#### 4.4 Connection Retry with Exponential Backoff

Database connections should use exponential backoff with jitter:

```go
// ✅ CORRECT: Exponential backoff with jitter
const (
    maxRetries        = 10
    baseDelay         = 1 * time.Second
    maxDelay          = 30 * time.Second
    jitterFactor      = 0.25
)

func connectWithRetry() (*sql.DB, error) {
    var db *sql.DB
    var err error

    for attempt := 0; attempt < maxRetries; attempt++ {
        db, err = connectPostgres()
        if err == nil {
            if err = db.Ping(); err == nil {
                return db, nil
            }
        }

        // Calculate delay with exponential backoff and jitter
        delay := baseDelay * time.Duration(1<<attempt)
        if delay > maxDelay {
            delay = maxDelay
        }
        jitter := time.Duration(float64(delay) * jitterFactor * rand.Float64())

        log.Printf("Connection attempt %d failed, retrying in %v: %v",
            attempt+1, delay+jitter, err)
        time.Sleep(delay + jitter)
    }

    return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
}
```

#### 4.5 Connection Pool Configuration

```go
// ✅ CORRECT: Configurable connection pool
db.SetMaxOpenConns(getEnvInt("DB_MAX_OPEN_CONNS", 25))
db.SetMaxIdleConns(getEnvInt("DB_MAX_IDLE_CONNS", 5))
db.SetConnMaxLifetime(time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_MS", 300000)) * time.Millisecond)

// SQLite special case: single connection only
if dialect == DialectSQLite {
    db.SetMaxOpenConns(1)
}
```

---

### 5. Schema Initialization Patterns

#### 5.1 Standard File Location

```
scenarios/{{TARGET}}/
├── initialization/
│   └── storage/
│       └── postgres/
│           ├── schema.sql      # Table definitions
│           ├── seed.sql        # Optional: default data
│           └── migrations/     # Optional: versioned migrations
│               ├── 001_initial.sql
│               └── 002_add_indexes.sql
```

#### 5.2 Idempotent Schema Pattern

Schema files MUST be idempotent (safe to run multiple times):

```sql
-- ✅ CORRECT: Idempotent schema
-- Enable extensions (idempotent)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create types with existence check
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'status_type') THEN
        CREATE TYPE status_type AS ENUM ('pending', 'active', 'completed');
    END IF;
END$$;

-- Create tables (idempotent)
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    status status_type DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes (idempotent)
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_created ON tasks(created_at DESC);

-- Upsert seed data (idempotent)
INSERT INTO tasks (id, name, status)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Task', 'pending')
ON CONFLICT (id) DO NOTHING;
```

```sql
-- ❌ WRONG: Non-idempotent schema (fails on second run)
CREATE TYPE status_type AS ENUM ('pending', 'active', 'completed');
CREATE TABLE tasks (...);
INSERT INTO tasks VALUES (...);
```

#### 5.3 Migration Patterns for Existing Schemas

Use migration logic only when the task is explicitly brownfield (or when existing persisted data must be preserved). Otherwise assume greenfield and skip migration shims:

```go
// Check and add columns if missing (migration pattern)
func ensureColumnExists(db *sql.DB, table, column, colType string) error {
    var exists bool
    err := db.QueryRow(`
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = $1 AND column_name = $2
        )`, table, column).Scan(&exists)

    if err != nil {
        return err
    }

    if !exists {
        _, err = db.Exec(fmt.Sprintf(
            "ALTER TABLE %s ADD COLUMN %s %s", table, column, colType))
    }
    return err
}
```

---

### 6. Storage Abstraction Patterns

#### 6.1 The Repository Pattern

Business logic should NOT directly depend on storage implementation:

```go
// ✅ CORRECT: Repository interface abstracts storage
type TaskRepository interface {
    Create(ctx context.Context, task *Task) error
    FindByID(ctx context.Context, id string) (*Task, error)
    FindByStatus(ctx context.Context, status Status) ([]*Task, error)
    Update(ctx context.Context, task *Task) error
    Delete(ctx context.Context, id string) error
}

// PostgreSQL implementation
type PostgresTaskRepository struct {
    db *sql.DB
}

func (r *PostgresTaskRepository) FindByID(ctx context.Context, id string) (*Task, error) {
    // PostgreSQL-specific query
}

// SQLite implementation (for testing/desktop)
type SQLiteTaskRepository struct {
    db *sql.DB
}

// In-memory implementation (for unit tests)
type InMemoryTaskRepository struct {
    tasks map[string]*Task
}

// Service uses the interface, not concrete implementation
type TaskService struct {
    repo TaskRepository  // Can be any implementation
}
```

```go
// ❌ WRONG: Business logic directly uses database
type TaskService struct {
    db *sql.DB  // Tight coupling to PostgreSQL
}

func (s *TaskService) GetTask(id string) (*Task, error) {
    row := s.db.QueryRow("SELECT * FROM tasks WHERE id = $1", id)
    // Direct SQL in business logic
}
```

#### 6.2 Abstraction Decision Tree

```
Where should this storage logic live?
│
├─ Is it a database query or storage operation?
│   └─ Repository layer (e.g., TaskRepository)
│
├─ Is it business logic using stored data?
│   └─ Service layer (e.g., TaskService using TaskRepository)
│
├─ Is it data validation or transformation?
│   └─ Domain layer (entity methods or validation functions)
│
└─ Is it API request/response handling?
    └─ Handler layer (uses Service, never Repository directly)
```

#### 6.3 Multi-Storage Abstraction

For scenarios using multiple storage systems:

```go
// Storage service that coordinates multiple backends
type StorageService struct {
    postgres *PostgresRepository  // Structured data
    redis    *RedisCache          // Caching
    minio    *MinioClient         // Large files
}

// Hybrid pattern: DB for metadata, filesystem/S3 for content
type DocumentService struct {
    metadataRepo DocumentMetadataRepository  // PostgreSQL
    contentStore DocumentContentStore        // MinIO/Filesystem
}

func (s *DocumentService) Save(doc *Document) error {
    // Save content to object storage
    contentPath, err := s.contentStore.Upload(doc.Content)
    if err != nil {
        return err
    }

    // Save metadata with reference to content
    doc.ContentPath = contentPath
    return s.metadataRepo.Save(doc.Metadata)
}
```

---

### 7. Redis Key Namespacing

Prevent key collisions between scenarios:

```go
// ✅ CORRECT: Namespaced keys
const redisKeyPrefix = "{{TARGET}}:"

func cacheKey(key string) string {
    return redisKeyPrefix + key
}

// Usage
client.Set(ctx, cacheKey("user:123"), userData, ttl)
// Actual key: "{{TARGET}}:user:123"
```

```go
// ❌ WRONG: Global keys (collision risk)
client.Set(ctx, "user:123", userData, ttl)  // Might conflict with other scenarios
```

**Standard key patterns:**

| Pattern | Example | Purpose |
|---------|---------|---------|
| `{scenario}:session:{id}` | `agent-manager:session:abc123` | User sessions |
| `{scenario}:cache:{entity}:{id}` | `agent-manager:cache:task:456` | Entity caching |
| `{scenario}:rate:{resource}:{id}` | `agent-manager:rate:api:user-789` | Rate limiting |
| `{scenario}:lock:{resource}` | `agent-manager:lock:migration` | Distributed locks |

---

### 8. Qdrant Collection Naming

```go
// ✅ CORRECT: Scenario-prefixed collection names
const (
    collectionPrefix = "{{TARGET}}_"
    WorkflowEmbeddings = collectionPrefix + "workflow_embeddings"
    DocumentEmbeddings = collectionPrefix + "document_embeddings"
)
```

---

### 9. Filesystem Storage Standard (`api-core/storage`)

#### 9.1 Core Rule

Scenario deploy directories are disposable. Mutable runtime state must live outside app targets and be resolved through `github.com/vrooli/api-core/storage`.

This prevents both:
- stale-file drift during redeploys/auto-updates
- accidental data loss when app directories are replaced

#### 9.2 Required Adoption Pattern (Go)

```go
import (
    "path/filepath"

    "github.com/vrooli/api-core/storage"
)

resolver, err := storage.NewResolver(storage.ResolverConfig{
    AppID:   "vrooli",
    Profile: storage.ProfileAuto,
})
if err != nil {
    return err
}

paths, err := storage.EnsureAllDirs(resolver, storage.Options{
    ScenarioID: "{{TARGET}}",
}, 0)
if err != nil {
    return err
}

runtimePath, err := resolver.Path(
    storage.Options{ScenarioID: "{{TARGET}}"},
    storage.ClassState,
    "runtime.json",
)
if err != nil {
    return err
}

if err := storage.WriteFileAtomic(runtimePath, payload, storage.DefaultFilePerm); err != nil {
    return err
}
```

#### 9.3 Filesystem Anti-Patterns

```go
// ❌ WRONG: scenario-local mutable writes in app tree
path := filepath.Join(".", "data", "runtime.json")
_ = os.WriteFile(path, payload, 0o644)
```

```go
// ❌ WRONG: ad hoc custom path policy instead of api-core/storage
base := os.Getenv("DATA_DIR")
if base == "" {
    base = filepath.Join(".", "data")
}
```

#### 9.4 Hybrid Database + Filesystem Pattern

For large payloads (media/documents), store metadata in DB and content on disk, but resolve disk paths via `api-core/storage` classes (`data`/`cache`/`state`) rather than scenario-local folders.

```sql
-- Database stores metadata and index (fast queries)
CREATE TABLE documents (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    content_path VARCHAR(1000) NOT NULL,  -- Relative path under resolved storage class
    content_hash VARCHAR(64),
    size_bytes BIGINT,
    mime_type VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

#### 9.5 Tracked Scenario Assets vs Runtime State

Not every UI-edited file belongs in runtime storage.

Before moving a file out of the source tree, ask:

Is this file meant to be shared through git as part of the scenario's authored behavior?

- Yes:
  - keep it in the repo in an explicit source directory such as `config/`, or `policy/`
  - do not treat it as runtime state just because a UI edits it
- No:
  - move it to `api-core/storage`

Use this 3-way model:

- repo metadata
  - structural files such as `.vrooli/service.json`
- tracked scenario-authored assets
  - versioned defaults, policy, plans, and other shared source artifacts
- runtime mutable state
  - queues, runs, locks, telemetry, databases, caches, and user-local mutable config

Do not use `.vrooli/` as a generic bucket for checked-in authoring content. Reserve it for repo-owned metadata unless the repo contract explicitly says otherwise.

---

### 10. Greenfield Default, Brownfield Exception

Assume greenfield unless explicitly stated otherwise.

Default behavior:
- Do not add migration compatibility layers by default.
- Build clean storage paths and schema state directly.
- Prefer consolidation over carrying forward legacy scaffolding.

Brownfield behavior (only when explicitly requested or existing persisted data must be preserved):
- Add migration and compatibility logic needed to preserve data continuity.
- Make migrations idempotent and removable after cutover.
- Document migration constraints in `docs/internal/STORAGE_AUDIT.md`.

#### 10.1 Greenfield Consolidation Pattern

```
BEFORE (accumulated migrations):
initialization/storage/postgres/migrations/
├── 001_initial.sql
├── 002_add_status_column.sql
├── 003_rename_status.sql
├── 004_add_index.sql
├── 005_fix_constraint.sql
└── 006_add_another_column.sql

AFTER (consolidated):
initialization/storage/postgres/
├── schema.sql                    # Clean, complete schema
└── seed.sql                      # Essential seed data only
```

#### 10.2 Brownfield-Only Migration Pattern

Use this only for explicitly brownfield work:

```go
// Check and add columns if missing (migration pattern)
func ensureColumnExists(db *sql.DB, table, column, colType string) error {
    var exists bool
    err := db.QueryRow(`
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = $1 AND column_name = $2
        )`, table, column).Scan(&exists)

    if err != nil {
        return err
    }

    if !exists {
        _, err = db.Exec(fmt.Sprintf(
            "ALTER TABLE %s ADD COLUMN %s %s", table, column, colType))
    }
    return err
}
```

#### 10.3 Compatibility Code Removal

```go
// REMOVE: Compatibility shims for old schema
// if oldColumnExists { migrateOldData() }

// REMOVE: Feature flags for storage backends
// if useNewStorage { ... } else { ... }

// REMOVE: Deprecated table references
// Legacy: SELECT * FROM old_tasks
```

---

### 11. Storage Architecture Audit

Before making changes, assess `{{TARGET}}`'s current storage posture.

#### 11.1 Audit Commands

```bash
# Check service.json resource declarations
cat scenarios/{{TARGET}}/.vrooli/service.json | jq '.dependencies.resources'

# Find database connection code
rg "sql\.Open|NewConnection|DATABASE_URL|POSTGRES_" scenarios/{{TARGET}}/api --type go

# Find hard-coded credentials (anti-pattern)
rg "postgres://[^$]" scenarios/{{TARGET}}/ --type go
rg "password.*=.*['\"]" scenarios/{{TARGET}}/ --type go

# Check for proper repository pattern
rg "interface.*Repository" scenarios/{{TARGET}}/api --type go

# Find direct SQL in handlers (anti-pattern)
rg "db\.Query|db\.Exec" scenarios/{{TARGET}}/api/handlers --type go

# Check schema initialization
ls -la scenarios/{{TARGET}}/initialization/storage/postgres/

# Check filesystem storage standard adoption
rg "storage\\.NewResolver|storage\\.EnsureAllDirs|storage\\.EnsureClassDir|storage\\.WriteFileAtomic|\\.Path\\(" scenarios/{{TARGET}}/api --type go

# Find direct filesystem writes (anti-pattern)
rg "os\\.WriteFile|ioutil\\.WriteFile|os\\.Create\\(|os\\.OpenFile\\(" scenarios/{{TARGET}}/api --type go

# Find scenario-local data path conventions (anti-pattern)
rg "filepath\\.Join\\(\\s*\"\\.\"\\s*,\\s*\"data\"|DATA_DIR|/data/" scenarios/{{TARGET}}/ --type go

# Find Redis key usage (check for namespacing)
rg "redis\.(Set|Get|Del)" scenarios/{{TARGET}}/ --type go -A 1

# Check for environment variable usage
rg "os\.Getenv.*POSTGRES" scenarios/{{TARGET}}/ --type go
```

#### 11.2 Red Flags Checklist

- [ ] No `postgres` resource in service.json `dependencies.resources`
- [ ] Missing `schema` field in postgres resource declaration
- [ ] Hard-coded database credentials in code
- [ ] Using `POSTGRES_DB=vrooli` without override (shared database)
- [ ] No connection retry logic (single connection attempt)
- [ ] No connection pool configuration
- [ ] Missing `initialization/storage/postgres/schema.sql`
- [ ] Non-idempotent schema files (no `IF NOT EXISTS`)
- [ ] Direct SQL in handler/controller code (no repository abstraction)
- [ ] Redis keys without scenario prefix
- [ ] Filesystem runtime writes bypass `api-core/storage`
- [ ] Mutable files stored under scenario deploy directories
- [ ] Multiple dialect support without abstraction

#### 11.3 Document Findings

Record audit results in `scenarios/{{TARGET}}/docs/internal/STORAGE_AUDIT.md`:

```markdown
# {{TARGET}} Storage Architecture Audit

## Last Updated
[Date]

## Resource Configuration Status
- [ ] postgres declared in service.json
- [ ] schema field uses scenario slug
- [ ] initialization files referenced
- [ ] redis/qdrant properly configured (if used)

## Connection Pattern Status
- [ ] Environment variables used (no hard-coded values)
- [ ] Connection retry with exponential backoff
- [ ] Connection pool configured
- [ ] Health check implemented

## Schema Status
- [ ] schema.sql exists and is idempotent
- [ ] Tables use proper constraints and indexes
- [ ] Greenfield default applied unless brownfield was explicitly requested
- [ ] Brownfield migrations documented only when required

## Abstraction Status
- [ ] Repository interfaces defined
- [ ] Business logic uses interfaces, not direct DB
- [ ] Multiple storage backends abstracted (if applicable)

## Filesystem Status
- [ ] Runtime filesystem writes go through `api-core/storage`
- [ ] Deploy directory treated as disposable
- [ ] Atomic writes used for persisted files

## Issues Found
1. [File:line] - Issue description
2. ...

## Priority Fixes
1. [Highest impact] - Why
2. ...
```

---

### 12. Memory Management with Visited Tracker

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `storage-architecture`.

---

### 13. Documentation and Memory Loop

#### 13.1 At Session Start

Read existing storage documentation:
- `scenarios/{{TARGET}}/.vrooli/service.json` - Resource declarations
- `scenarios/{{TARGET}}/docs/internal/STORAGE_AUDIT.md` - Prior audit findings (if exists)
- `resources/postgres/config/defaults.sh` - Available environment variables
- `packages/api-core/docs/storage.md` - Canonical filesystem storage contract

#### 13.2 At Session End

Update `scenarios/{{TARGET}}/docs/internal/STORAGE_AUDIT.md`:
- The code is the source of truth. Verify existing claims against actual code.
- Correct any inaccuracies discovered.
- Add new anti-pattern instances found.
- Update priority fixes based on work completed.
- Note areas not yet audited.
- Create the `docs/internal/` directory if needed.

---

### 14. Output Expectations

You may update in `scenarios/{{TARGET}}/`:
- Add or modify service.json resource declarations
- Add initialization/storage/postgres/schema.sql (or other storage types)
- Refactor connection code to use environment variables
- Add connection retry logic with exponential backoff
- Add repository interfaces and implementations
- Add storage service abstractions
- Namespace Redis keys and Qdrant collections
- Adopt `api-core/storage` for runtime filesystem paths

You must:
- Keep `{{TARGET}}` fully functional and non-regressed
- Use environment variables for all database configuration
- Override `POSTGRES_DB` with scenario slug (not default "vrooli")
- Make schema files idempotent (IF NOT EXISTS patterns)
- Abstract storage behind interfaces when business logic uses it
- Prefix Redis keys and Qdrant collections with scenario slug
- Route mutable filesystem storage through `api-core/storage`
- Assume greenfield by default; only add migrations when brownfield is explicitly required

You must NOT:
- Hard-code database credentials or connection strings
- Use the default "vrooli" database without scenario-specific override
- Write SQL directly in handler or controller code
- Store mutable runtime files under scenario deploy directories
- Replace `api-core/storage` path policy with custom `DATA_DIR` schemes
- Remove retry/backoff logic from database connections
- Create non-idempotent schema migrations

**Avoid superficial changes that rename variables or restructure code without materially improving storage architecture.**
