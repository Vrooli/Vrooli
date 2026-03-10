# Lifestyle Dashboard Architecture

## Overview

Lifestyle Dashboard is a unified personal lifestyle intelligence platform that provides a shared data model, correlation engine, and analytics layer for domain-specific health scenarios.

## Mental Model

```
                          LIFESTYLE DASHBOARD
┌─────────────────────────────────────────────────────────────┐
│                        UI (React/Vite)                       │
│  pages/ → Layout → components/dashboard/ → lib/api.ts       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼ HTTP/JSON
┌─────────────────────────────────────────────────────────────┐
│                        API (Go/Gorilla)                      │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                    HANDLERS LAYER                        │ │
│  │  handlers/events.go   - Event CRUD, queries              │ │
│  │  handlers/domains.go  - Domain registration, health      │ │
│  │  handlers/stats.go    - Timeline, summary aggregation    │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                               │
│                              ▼ interfaces                    │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                   REPOSITORY LAYER                       │ │
│  │  repository/interfaces.go    - EventRepository, etc.     │ │
│  │  repository/sqlite_events.go - SQLite implementation     │ │
│  │  repository/sqlite_domains.go                            │ │
│  │  repository/sqlite_stats.go                              │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                               │
│                              ▼                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                    DOMAIN LAYER                          │ │
│  │  domain/types.go   - Event, Domain, Request/Response     │ │
│  │  domain/schema.go  - SQLite table initialization         │ │
│  └─────────────────────────────────────────────────────────┘ │
│                              │                               │
│                              ▼                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                    STORAGE (SQLite)                      │ │
│  │  lifestyle.db - WAL mode, single-writer                  │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Key Domain Concepts

### Events
Cross-domain data points that flow into the dashboard from health scenarios:
- **domain**: Source scenario (e.g., "nutrition", "sleep", "fitness")
- **event_type**: Category within domain (e.g., "meal", "sleep_session", "workout")
- **payload**: JSON data specific to the event type
- **is_intervention**: Whether this represents an action taken
- **hypothesis_id**: Link to correlation hypotheses

### Domains
Registered health/wellness scenarios that integrate with the dashboard:
- **name**: Unique identifier (slug format)
- **capabilities**: What the domain can track/provide
- **health_url**: Endpoint for health checks
- **status**: active | inactive | unhealthy

### Statistics
Aggregated views across domains:
- **Timeline**: Event counts by day and domain
- **Summary**: Total events, active domains, breakdown by source

## Architectural Boundaries

### Entry Layer (UI + main.go)
- React pages handle routing and user interaction
- Go main.go handles configuration and wiring
- CORS middleware enables UI-API communication

### Presentation Layer (handlers/)
- HTTP request/response handling
- Input validation and error formatting
- Delegates to repository interfaces

### Abstraction Layer (repository/)
- Defines storage contracts via interfaces
- SQLite implementations handle queries
- Custom `ErrNotFound` for not-found handling

### Domain Layer (domain/)
- Pure type definitions
- No business logic, no dependencies
- Schema initialization

### Infrastructure Layer (SQLite)
- Single-file embedded database
- WAL mode for concurrent reads
- `api-core/database.Connect()` with retry

## Data Flow

```
1. UI page fetches data → lib/api.ts
2. api.ts calls API endpoint → fetch()
3. Handler parses request, validates
4. Handler calls Repository method
5. Repository executes SQL, maps to domain types
6. Handler formats response, returns JSON
7. UI receives and renders data
```

## Design Decisions

### Why SQLite (not PostgreSQL)?
- Single-user personal dashboard
- Mobile/desktop deployment targets
- Zero operational complexity
- File-based backup/restore
- See: docs/internal/STORAGE_AUDIT.md (ADR-001)

### Why Repository Pattern?
- Handlers testable with mocks
- Storage backend swappable
- Clear separation of SQL from HTTP
- See: docs/internal/STORAGE_AUDIT.md (ADR-002)

### Why api-core/database?
- Consistent with Vrooli patterns
- Automatic retry with backoff
- Resolves auditor compliance
- See: docs/internal/STORAGE_AUDIT.md (ADR-003)

## Test Coverage

| Package | Tests | Focus |
|---------|-------|-------|
| main | 22 | Integration (full API tests) |
| domain | 4 | Type serialization |
| handlers | 10 | Handler logic with mocks |
| repository | 15 | Storage operations |

## File Structure

```
api/
├── main.go           # Server wiring, config, startup
├── main_test.go      # Integration tests
├── domain/
│   ├── types.go      # Core entities
│   ├── types_test.go
│   └── schema.go     # Schema initialization
├── handlers/
│   ├── handlers.go   # Shared Handler struct
│   ├── events.go
│   ├── domains.go
│   ├── stats.go
│   └── handlers_test.go
└── repository/
    ├── interfaces.go     # Repository contracts
    ├── sqlite_events.go
    ├── sqlite_domains.go
    ├── sqlite_stats.go
    └── repository_test.go

ui/src/
├── App.tsx              # Router setup
├── main.tsx             # Entry point
├── components/
│   ├── Layout.tsx       # Navigation shell
│   ├── dashboard/       # Domain-specific components
│   └── ui/              # shadcn/ui primitives
├── pages/
│   ├── DashboardPage.tsx
│   ├── DomainsPage.tsx
│   ├── EventsPage.tsx
│   └── SettingsPage.tsx
└── lib/
    ├── api.ts           # API client
    └── format.ts        # Formatting utilities
```

## Related Documentation

- [STORAGE_AUDIT.md](../internal/STORAGE_AUDIT.md) - Storage architecture decisions
- [PRD.md](../../PRD.md) - Product requirements
- [PROGRESS.md](../PROGRESS.md) - Development history
