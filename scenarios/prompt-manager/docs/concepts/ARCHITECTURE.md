# Architecture Overview

This document describes the overall architecture of the prompt-manager scenario, including system components, data flow, and design decisions.

## System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         Clients                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────────────────────────────┐  │
│  │   CLI   │  │   UI    │  │        External APIs            │  │
│  │  (Go)   │  │ (React) │  │    (any HTTP client)            │  │
│  └────┬────┘  └────┬────┘  └───────────────┬─────────────────┘  │
└───────┼────────────┼───────────────────────┼────────────────────┘
        │            │                       │
        └────────────┴───────────┬───────────┘
                                 │ HTTP/REST
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                        API Server (Go)                          │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────────┐  │
│  │  skills/ │  tags/   │ members/ │ testing/ │   search/    │  │
│  │  domain  │  domain  │  domain  │  domain  │   domain     │  │
│  └────┬─────┴────┬─────┴────┬─────┴────┬─────┴──────┬───────┘  │
│       │          │          │          │            │          │
│  ┌────┴──────────┴──────────┴──────────┴────────────┴───────┐  │
│  │                    Interfaces Layer                       │  │
│  │   SkillStore │ TagRepository │ MemberStore │ LLMClient   │  │
│  └────┬──────────────────────────────────────────────┬──────┘  │
└───────┼──────────────────────────────────────────────┼─────────┘
        │                                              │
        ▼                                              ▼
┌───────────────────┐                    ┌─────────────────────┐
│   File System     │                    │     PostgreSQL      │
│   (skills/*.md)   │                    │  (tags, metrics,    │
│                   │                    │   test results)     │
└───────────────────┘                    └─────────────────────┘
                                                   │
                                         ┌─────────┴─────────┐
                                         ▼                   ▼
                              ┌──────────────┐    ┌──────────────┐
                              │    Ollama    │    │    Qdrant    │
                              │   (testing)  │    │  (optional)  │
                              └──────────────┘    └──────────────┘
```

## Domain-Driven Design

The API follows screaming architecture principles where folder structure reflects the business domain:

```
api/
├── skills/          # Core domain - skill CRUD, versions, metrics
│   ├── handlers.go  # HTTP handlers
│   ├── models.go    # Domain types
│   ├── interfaces.go # Contracts for testing seams
│   ├── store.go     # File-based storage
│   └── query.go     # Filtering/search logic
│
├── tags/            # Tag categorization
├── members/         # Team member management (3D world)
├── testing/         # LLM-based skill testing
├── search/          # Full-text search
├── metrics/         # Usage tracking
└── ogmeta/          # URL metadata fetching
```

### Interface-Based Design

All domains use interfaces for dependency injection, enabling:
- Unit testing without external dependencies
- Swappable implementations
- Clear contracts between layers

**Example:** [CODE: api/skills/interfaces.go]
```go
type SkillStore interface {
    GetAll() ([]Metadata, error)
    FindByID(id string) (*Metadata, string, error)
    // ...
}
```

Handlers depend on interfaces, not concrete types:
```go
type Handlers struct {
    store   SkillStore      // interface
    metrics MetricsService  // interface
}
```

## Data Storage Strategy

| Data Type | Storage | Rationale |
|-----------|---------|-----------|
| Skills (content) | File system (`.md` files) | Human-readable, version control friendly |
| Skill metadata | File system (`metadata.json`) | Colocated with content |
| Tags | PostgreSQL | Relational queries, shared taxonomy |
| Usage metrics | PostgreSQL | Aggregations, time-series queries |
| Test results | PostgreSQL | History, analysis |
| Members | File system (`data/members.json`) | Simple structure, rarely changes |

### File Storage Layout

```
skills/
├── core/              # System skills (read-only)
│   ├── metadata.json
│   ├── debugging.md
│   └── testing.md
├── local/             # User-created skills
│   ├── metadata.json
│   └── my-skill.md
└── drafts/            # Work-in-progress
    └── metadata.json
```

## CLI Architecture

The CLI is a thin wrapper over the API, following noun-verb command patterns:

```
cli/
├── app.go           # Command registration
├── skills/          # skill list|show|add|update|delete|...
├── tags/            # tag list|create
├── members/         # member list|show|create|update|delete
├── testing/         # test run|history
├── search/          # search <query>
├── metadata/        # metadata fetch
└── internal/
    ├── appctx/      # API client context
    ├── clipboard/   # Cross-platform clipboard
    ├── output/      # Formatting helpers
    └── types/       # Shared types
```

**Design Principle:** Every CLI command maps 1:1 to an API endpoint. No business logic in CLI.

## UI Architecture

React-based SPA with:
- **React Query** for data fetching/caching
- **Zustand** stores for UI state
- **React Three Fiber** for 3D world visualization

See [3D World Architecture](3D-WORLD-ARCHITECTURE.md) for visualization details.

### Skill Content Editing

The skill editor supports two coordinated viewing states over the same markdown source:

- **Code**: Monaco editor for raw markdown input.
- **Rich Text**: TipTap WYSIWYG editor with markdown ↔ HTML conversion via `ui/src/services/content`.
- **Preview**: Markdown renderer (`ui/src/components/markdown`) built with `react-markdown` + `remark-gfm`, used for preview-only and split-view rendering.

The editor exposes **Editor / Preview** view modes so users can inspect markdown output even when rich-text conversion is unsafe. On desktop, preview renders as a split view (editor + renderer); on mobile, preview renders as a full-width renderer. View mode selection lives in `SkillContentEditor`, while rendering and conversion are isolated behind clear seams.

### State Boundaries

| State Type | Owner | Storage |
|------------|-------|---------|
| Skills, Tags, Metrics | API | PostgreSQL + Files |
| Favorites | UI | localStorage |
| View preferences | UI | Component state |
| Theme | UI | localStorage |

## Version Control

Skills support version history for content changes:

1. Each update creates a new version
2. Versions stored alongside current content
3. Revert restores content and creates new version

**API:**
- `GET /api/v1/skills/{id}/versions` - List versions
- `POST /api/v1/skills/{id}/revert/{ver}` - Revert to version

## Testing Strategy

The interface-based design enables testing at multiple levels:

| Level | What to Test | How |
|-------|--------------|-----|
| Unit | Handler logic | Mock interfaces |
| Integration | Storage operations | temp directories, sqlmock |
| E2E | Full flows | Test containers |

See [SEAMS.md](../internal/SEAMS.md) for detailed testing seam documentation.

## Related Documentation

- [QUICKSTART.md](../QUICKSTART.md) - Getting started guide
- [API Reference](../reference/api-endpoints.md) - Endpoint documentation
- [CLI Reference](../reference/cli-commands.md) - Command documentation
- [Testing Seams](../internal/SEAMS.md) - Testing architecture details
