# Web Console Storage Architecture Audit

## Last Updated
2026-02-19

## Resource Configuration Status
- [x] postgres declared in service.json
- [x] schema field uses scenario slug (`web-console`)
- [x] initialization files present (`initialization/postgres/schema.sql`, `seed.sql`)
- [x] Schema initialized on startup via `initSchema()` in `main.go`

## Connection Pattern Status
- [x] Environment variables used via `api-core/database.Connect()` (no hard-coded values)
- [x] Connection retry handled by `api-core/database` package
- [x] Health check implemented via `api-core/health.DB()`
- [ ] Connection pool explicitly configured (uses `api-core` defaults)

## Schema Status
- [x] schema.sql exists and is idempotent (`IF NOT EXISTS` patterns throughout)
- [x] Tables use proper constraints and indexes
- [x] Greenfield default applied (no migration compatibility layers)
- [x] Enum types for `policy_mode` and `shortcut_scope` match Go domain types
- [x] seed.sql uses `ON CONFLICT DO NOTHING` for idempotent defaults

## Abstraction Status
- [x] Repository interfaces defined (`ShortcutStore`, `AIConfigStore` in `repository.go`)
- [x] Server struct uses interfaces, not concrete types — handlers are storage-agnostic
- [x] In-memory implementations available for tests (`ShortcutProfileStore`, `AIProviderConfigStore`)
- [x] PostgreSQL implementations for production (`PGShortcutStore`, `PGAIConfigStore`)
- [x] Multiple storage concerns are separated into dedicated files

## Filesystem Status
- [x] No mutable runtime files written (all state is in DB or in-memory)
- [x] Deploy directory treated as disposable
- [x] No filesystem storage anti-patterns detected

## Current Storage Architecture

| Store | File | Purpose | Persistence |
|-------|------|---------|-------------|
| `PGShortcutStore` | `shortcut_profiles_pg.go` | Launch shortcut profiles | **PostgreSQL** (`shortcut_profiles` table) |
| `PGAIConfigStore` | `ai_provider_config_pg.go` | Provider config (PG) + health tracking (memory) | **Hybrid**: config in PostgreSQL, health in-memory |
| `SessionManager` | `session.go` | Terminal sessions + PTY processes | In-memory (process-bound) |
| `EventLogger` | `events.go` | Structured lifecycle events (ring buffer) | In-memory (cap 1000) |
| `Metrics` | `metrics.go` | Operational counters | In-memory (atomic) |
| `idempotencyCache` | `session_handlers.go` | POST replay safety | In-memory (5-min TTL) |

### Interface Hierarchy

```
ShortcutStore (repository.go)
├── ShortcutProfileStore (shortcut_profiles.go)     — in-memory, used in tests
└── PGShortcutStore (shortcut_profiles_pg.go)       — PostgreSQL, used in production

AIConfigStore (repository.go)
├── AIProviderConfigStore (ai_provider_config.go)   — in-memory, used in tests
└── PGAIConfigStore (ai_provider_config_pg.go)      — hybrid PG+memory, used in production
```

## Completed Improvements

1. ~~**Repository interfaces**~~ — ✅ `ShortcutStore` and `AIConfigStore` interfaces extracted; Server uses interfaces
2. ~~**Shortcut persistence**~~ — ✅ `PGShortcutStore` reads/writes `shortcut_profiles` table; survives restarts
3. ~~**AI config persistence**~~ — ✅ `PGAIConfigStore` reads/writes `ai_provider_configs` table; health stays in-memory (ephemeral, high-frequency)

## Priority Improvements (for future phases)

1. **Session metadata persistence** — Store session metadata in `sessions` table for audit trail and cross-restart session listing (PTY state itself is inherently process-bound and cannot persist)
2. **Connection pool tuning** — Configure pool size based on expected concurrency
3. **PG integration tests** — Add testcontainers-based tests for `PGShortcutStore` and `PGAIConfigStore`

## Areas Not Yet Audited
- Redis usage (not currently used)
- Qdrant usage (not currently used)
- Connection pool tuning requirements under load
