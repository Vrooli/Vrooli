# Web Console Storage Architecture Audit

## Last Updated
2026-03-17

## Resource Configuration Status
- [x] SQLite declared in service.json
- [x] Database file resolved via `api-core/storage` (ClassData)
- [x] Domain-owned embedded schema and seed (`api/internal/sessions/schema.sql`, `seed.sql`)
- [x] Schema initialized on startup via `database.EnsureSchemas` in `main.go`

## Connection Pattern Status
- [x] DSN built with performance pragmas (WAL, foreign_keys, busy_timeout)
- [x] Connection retry handled by `api-core/database` package
- [x] Health check implemented via `api-core/health.DB()`
- [x] Connection pool configured for SQLite (MaxOpenConns=1, MaxIdleConns=1)

## Schema Status
- [x] schema.sql exists and is idempotent (`IF NOT EXISTS` patterns throughout)
- [x] Tables use proper constraints and indexes
- [x] Greenfield default applied (no migration compatibility layers)
- [x] CHECK constraints for `policy_mode` and `scope` replace PG enum types
- [x] seed.sql uses `ON CONFLICT DO NOTHING` for idempotent defaults
- [x] Boolean columns use INTEGER (0/1) per SQLite convention

## Abstraction Status
- [x] Repository interfaces defined (`ShortcutStore`, `AIConfigStore` in `repository.go`, `WorkspaceStore` in `workspace_store.go`)
- [x] Server struct uses interfaces, not concrete types — handlers are storage-agnostic
- [x] In-memory implementations available for unit tests (`ShortcutProfileStore`, `AIProviderConfigStore`, `MemWorkspaceStore`)
- [x] SQLite implementations for production (`SQLShortcutStore`, `SQLAIConfigStore`, `SQLWorkspaceStore`)
- [x] Multiple storage concerns are separated into dedicated files

## Filesystem Status
- [x] Voice config resolved via `api-core/storage` (ClassState)
- [x] SQLite DB file resolved via `api-core/storage` (ClassData)
- [x] Deploy directory treated as disposable — mutable state lives outside
- [x] Atomic writes used for voice config persistence

## Current Storage Architecture

| Store | File | Purpose | Persistence |
|-------|------|---------|-------------|
| `SQLShortcutStore` | `shortcut_profiles_sql.go` | Launch shortcut profiles | **SQLite** (`shortcut_profiles` table) |
| `SQLAIConfigStore` | `ai_provider_config_sql.go` | Provider config (SQLite) + health tracking (memory) | **Hybrid**: config in SQLite, health in-memory |
| `SQLWorkspaceStore` | `workspace_store_sql.go` | Workspace layout (panes, groups, ordering) | **SQLite** (`workspace_panes`, `tab_groups` tables) |
| `SessionManager` | `session.go` | Terminal sessions + PTY processes | In-memory (process-bound) |
| `EventLogger` | `events.go` | Structured lifecycle events (ring buffer) | In-memory (cap 1000) |
| `Metrics` | `metrics.go` | Operational counters | In-memory (atomic) |
| `idempotencyCache` | `session_handlers.go` | POST replay safety | In-memory (5-min TTL) |
| `VoiceStreamConfig` | `voice_config.go` | Voice streaming parameters | **File** (JSON via api-core/storage) |

### Interface Hierarchy

```
ShortcutStore (repository.go)
├── ShortcutProfileStore (shortcut_profiles.go)     — in-memory, used in tests
└── SQLShortcutStore (shortcut_profiles_sql.go)     — SQLite, used in production

AIConfigStore (repository.go)
├── AIProviderConfigStore (ai_provider_config.go)   — in-memory, used in tests
└── SQLAIConfigStore (ai_provider_config_sql.go)    — hybrid SQLite+memory, used in production

WorkspaceStore (workspace_store.go)
├── MemWorkspaceStore (workspace_store_mem.go)      — in-memory, used in tests
└── SQLWorkspaceStore (workspace_store_sql.go)      — SQLite, used in production
```

## Portability
- [x] Pure Go SQLite driver (`modernc.org/sqlite`) — CGO_ENABLED=0 compatible
- [x] No external server dependencies for storage
- [x] Cross-platform path resolution via `api-core/storage`
- [x] Static binary builds work on linux/darwin/windows

## Priority Improvements (for future phases)

1. **Session metadata persistence** — Store session metadata in `sessions` table for audit trail
2. **SQLite backup strategy** — Periodic snapshots for data safety
