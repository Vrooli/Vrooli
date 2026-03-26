# Architecture

Brand Manager follows a domain-driven design with clear layer separation.

## System Overview

```
┌─────────────────────────────────────────────────────┐
│                    Surfaces                          │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │  CLI      │  │  REST API │  │  React UI (future)│  │
│  │ cli/app.go│  │ handlers/ │  │  ui/src/          │  │
│  └─────┬─────┘  └─────┬─────┘  └────────┬──────────┘  │
│        │              │                  │             │
│        └──────┐       │       ┌──────────┘             │
│               ▼       ▼       ▼                        │
│         ┌──────────────────────────┐                   │
│         │     HTTP API Layer       │                   │
│         │   [CODE: api/main.go]    │                   │
│         └────────────┬─────────────┘                   │
│                      │                                 │
│         ┌────────────▼─────────────┐                   │
│         │    Handler Layer         │                   │
│         │ [CODE: api/handlers/     │                   │
│         │  brands.go]              │                   │
│         └────────────┬─────────────┘                   │
│                      │ uses interfaces                 │
│         ┌────────────▼─────────────┐                   │
│         │   Repository Interfaces  │                   │
│         │ [CODE: api/repository/   │                   │
│         │  interfaces.go]          │                   │
│         └────────────┬─────────────┘                   │
│                      │ implemented by                  │
│         ┌────────────▼─────────────┐                   │
│         │   SQLite Repositories    │                   │
│         │ [CODE: api/repository/   │                   │
│         │  sqlite_brands.go]       │                   │
│         └────────────┬─────────────┘                   │
│                      │                                 │
│         ┌────────────▼─────────────┐                   │
│         │   Database Connection    │                   │
│         │ [CODE: api/database/     │                   │
│         │  connection.go]          │                   │
│         └────────────┬─────────────┘                   │
│                      │                                 │
│         ┌────────────▼─────────────┐                   │
│         │   SQLite (WAL mode)      │                   │
│         │ [CODE: api/database/     │                   │
│         │  schema.sql]             │                   │
│         └──────────────────────────┘                   │
└─────────────────────────────────────────────────────┘
```

## Domain Model

The core domain is defined in [CODE: api/domain/types.go]:

- **Brand** — Root aggregate containing identity, colors, typography, voice facets. Each facet is a value object stored as JSON in SQLite.
- **BrandVersion** — Immutable snapshot of a Brand at a point in time. Created on every create/update.
- **Assignment** — Links a Brand (at a specific version) to a scenario. One assignment per scenario (upsert semantics).
- **BrandFilter** — Query filter for listing brands with optional name search, pagination.

## Data Flow

### Create Brand
1. Client sends `POST /api/v1/brands` with brand JSON
2. [CODE: api/handlers/brands.go#CreateBrand] validates input, generates UUID
3. [CODE: api/repository/sqlite_brands.go#Create] inserts brand with JSON-marshalled facets
4. [CODE: api/repository/sqlite_versions.go#Create] creates initial version snapshot
5. Response returns created brand with `version: 1`

### Assign Brand to Scenario
1. Client sends `POST /api/v1/assignments` with `brand_id` and `scenario_name`
2. [CODE: api/handlers/brands.go#CreateAssignment] verifies brand exists, captures current version
3. [CODE: api/repository/sqlite_assignments.go#Create] upserts assignment (one per scenario)
4. Response returns assignment with captured brand version

### Check Scenario Status
1. Client sends `GET /api/v1/scenarios/{name}/status`
2. [CODE: api/handlers/brands.go#GetScenarioStatus] queries assignment for scenario
3. Returns `has_brand: true/false` with brand details if assigned

## Storage Architecture

See [DOC: docs/internal/STORAGE_AUDIT.md] for detailed storage audit.

- **Database**: SQLite with WAL mode at `~/.vrooli/brand-manager/brand-manager.db`
- **Schema**: 4 tables — `brands`, `brand_versions`, `assignments`, `assets` (see [CODE: api/database/schema.sql])
- **Connection**: Via `api-core/database.Connect` with performance pragmas (see [CODE: api/database/connection.go#Connect])
- **Assets**: Planned at `~/.vrooli/brand-manager/assets/{brand_id}/` (not yet implemented)

## Key Patterns

| Pattern | Location | Purpose |
|---------|----------|---------|
| Repository interfaces | [CODE: api/repository/interfaces.go] | Decouple handlers from storage |
| JSON facets in SQLite | [CODE: api/repository/sqlite_brands.go] | Store complex objects without joins |
| Embedded schema | [CODE: api/database/connection.go] | Idempotent table creation on startup |
| Version snapshots | [CODE: api/repository/sqlite_versions.go] | Immutable brand history |
| Upsert assignments | [CODE: api/repository/sqlite_assignments.go] | One brand per scenario constraint |
| Handler factory | [CODE: api/handlers/brands.go#New] | Dependency injection for testability |

## Dependency Direction

```
main.go → handlers → repository (interfaces) ← sqlite implementations
                   → domain (types only)
```

Handlers depend on repository interfaces, never concrete implementations. The `main.go` entry point wires concrete SQLite repositories into handlers.

## Future Architecture (Planned)

- **AI Generation**: AIProviderChain pattern (Ollama-first, OpenRouter fallback) for text and image generation
- **Programmatic Application**: CSS custom property injection with `/* brand-manager:<element> */` markers
- **Scenario Auditor Integration**: HTTP provider for branding compliance rules
- **Discovery Scanner**: Auto-populate draft brands from existing scenario state
