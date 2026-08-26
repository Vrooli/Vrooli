# Web Console Storage Architecture Audit

## Last Updated
2026-08-26

## Resource Configuration Status
- [x] SQLite declared in service.json
- [x] Database file resolved via `api-core/storage` (ClassData)
- [x] Domain-owned embedded schema and seed (`api/internal/sessions/schema.sql`, `seed.sql`)
- [x] Schema initialized on startup via `database.EnsureSchemas` in `main.go`
- [x] Transcript cursors use one source-scoped `agent_transcript_checkpoints` table
- [x] Conversation search uses external-content FTS5 over `conversation_events`
- [x] Conversation retention is configurable and runs from the owner cleanup seam

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
- [x] Agent homes are lazy: plain shells do not create Codex/Grok homes
- [x] Recovery copies only session-owned rollout files, not regenerable runtime state

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
| `agent_transcript_checkpoints` | `agent_transcript_checkpoint_store.go` | Claude/Codex/Grok/OpenCode ingestion cursors | **SQLite** (one source/key table) |
| `conversation_events_fts` | `schema_migrations.go` | Archive conversation search index | **SQLite FTS5 external-content index** |

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

1. **Storage-manager backlog cleanup** — Existing orphaned homes remain an operator/storage-manager concern; web-console now reports them with byte counts.
2. **SQLite backup strategy** — Periodic snapshots for data safety.

## Current Findings and Boundaries

- Session-owned rollout transcripts are durable state under the routed state root.
- Agent runtime entries are symlinked to the shared user home where supported and are not recreated for shells that never launch an agent.
- `api/internal/sessions/schema.sql` remains the domain-owned declarative schema; the boot migration code only repairs existing local databases and removes the retired Codex-only checkpoint table.
- No host cleanup implementation was added. The `/api/v1/cleanup/orphans` endpoint reports filesystem/database drift for storage-manager or operator approval.
