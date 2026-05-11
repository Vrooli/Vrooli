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
│  │  skills/ │  tags/   │  agents/ │ actions* │   search/    │  │
│  │  domain  │  domain  │  domain  │ domain   │   domain     │  │
│  └────┬─────┴────┬─────┴────┬─────┴────┬─────┴──────┬───────┘  │
│       │          │          │          │            │          │
│  ┌────┴──────────┴──────────┴──────────┴────────────┴───────┐  │
│  │                    Interfaces Layer                       │  │
│  │   SkillStore │ TagRepository │ AgentStore │ LLMClient    │  │
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
├── agents/          # Agent management with appearance, SOUL.md + other .md files, capabilities
├── actions/         # Proposed: typed wrappers over Vrooli-controlled CLI commands
├── teams/           # Team management with roles, members, org charts
├── tags/            # Tag categorization
├── store/           # Storage layer (per-entity files)
├── testing/         # LLM-based skill testing
├── search/          # Full-text and semantic search
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
| Skill metadata | File system (`skill.json` per skill) | Colocated with content |
| Tags | PostgreSQL | Relational queries, shared taxonomy |
| Usage metrics | PostgreSQL | Aggregations, time-series queries |
| Test results | PostgreSQL | History, analysis |
| Agents | File system (`agent.json` per agent) | Normalized entity structure |
| Teams | File system (`team.json` per team) | Normalized entity structure |
| Relations | File system (`relations/` directory) | Agent-skill and team-member mappings |
| Actions (proposed) | File system (`action.json` per action) | Typed execution contracts over Vrooli-controlled CLIs |

### New Storage Layout (v2.0)

The storage system uses a per-entity file structure under the `store/` directory:

```
store/
├── skills/
│   ├── _pack-order.json        # Active pack precedence
│   └── packs/
│       ├── core/               # System skills
│       │   ├── debugging/
│       │   │   ├── skill.json  # Metadata
│       │   │   ├── SKILL.md    # Content
│       │   │   └── history.jsonl
│       │   └── testing/
│       │       └── ...
│       ├── local/              # User-created skills
│       └── drafts/             # Work-in-progress
├── agents/
│   └── agent-1/
│       └── agent.json
├── teams/
│   └── engineering/
│       ├── team.json
│       ├── roles.json
│       └── org-chart.json
├── actions/                    # Proposed
│   ├── _pack-order.json
│   └── packs/
│       ├── core/
│       │   └── scenario-ui-screenshot/
│       │       ├── action.json
│       │       └── history.jsonl
│       ├── local/
│       └── drafts/
├── relations/
│   └── team-member/
│       └── team-id--agent-1.json
├── indexes/                    # Generated (never hand-edit)
│   ├── skills.json
│   ├── agents.json
│   └── teams.json
└── schemas/                    # JSON Schemas for validation
    ├── skill.schema.json
    ├── agent.schema.json
    ├── action.schema.json       # Executable Action contracts
    └── team.schema.json
```

**Key Design Changes:**
- Per-entity files instead of monolithic `metadata.json`
- Pack-based skill organization with precedence ordering
- Normalized relations in separate directory
- Generated indexes for fast lookups
- Schemas for runtime validation
- Actions use the same per-entity pattern while keeping execution logic in Vrooli-controlled CLIs

See [STORE-MIGRATION.md](STORE-MIGRATION.md) for migration details.

## Entity Ontology

Prompt-manager's implemented entities are skills, agents, teams, relations, topics, variants, experiments, and Actions. Actions add an execution layer without changing the responsibility of skills:

```text
Truth lives in the Plan of Record.
Judgment lives in Skills.
Execution lives in Actions.
Implementation lives in CLIs.
Unbuilt work lives in the Backlog.
Raw learning starts in typed knowledge topics.
```

Actions are not arbitrary code. They are typed contracts that call exactly one Vrooli-controlled CLI command. Branching, retries, resource access, and implementation details remain in the owning CLI. See [Actions](ACTIONS.md) and [Memory Promotion](MEMORY-PROMOTION.md).

## CLI Architecture

The CLI is an API-first client with a small amount of contract-aware flag resolution, following noun-verb command patterns:

```
cli/
├── app.go           # Command registration
├── skills/          # skill list|show|add|update|delete|...
├── actions/         # action list|show|create|update|delete|validate|run
├── tags/            # tag list|create
├── agents/          # agent list|show|create|update|delete
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

The Action CLI follows the same principle. `prompt-manager action run <id>` resolves an Action contract through the API and executes one controlled command through the governed runtime; it does not contain business logic or shell recipes.

## UI Architecture

React-based SPA with:
- **React Query** for data fetching/caching
- **Zustand** stores for UI state
- **React Three Fiber** for 3D world visualization

See [3D World Architecture](3D-WORLD-ARCHITECTURE.md) for visualization details.

The 3D world’s ground rendering now has a dedicated seam: `GroundSurface` composes grid vs textured planes, while `ui/src/lib/groundTextures.ts` and `ui/src/lib/groundShader.ts` own procedural texture generation and projection logic. This keeps shader/material concerns out of `WorldScene` and makes ground visuals configurable without touching scene orchestration.

### Skill Content Editing

The skill editor supports two coordinated viewing states over the same markdown source:

- **Code**: Monaco editor for raw markdown input.
- **Rich Text**: TipTap WYSIWYG editor with markdown ↔ HTML conversion via `ui/src/services/content`.
- **Preview**: Markdown renderer (`ui/src/components/markdown`) built with `react-markdown` + `remark-gfm`, used for preview-only and split-view rendering.

The editor exposes **Editor / Preview** view modes so users can inspect markdown output even when rich-text conversion is unsafe. On desktop, preview renders as a split view (editor + renderer); on mobile, preview renders as a full-width renderer. View mode selection lives in `SkillContentEditor`, while rendering and conversion are isolated behind clear seams.

### State Boundaries

| State Type | Owner | Storage |
|------------|-------|---------|
| Skills, Agents, Teams | API | File system (store/) |
| Actions (proposed) | API | File system (store/actions/) |
| Tags, Metrics, Test Results | API | PostgreSQL |
| Relations (team-member) | API | File system (store/relations/) |
| Favorites | UI | localStorage |
| View preferences | UI | Component state |
| Theme | UI | localStorage |
| 3D World state | UI | Zustand stores |

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

### Getting Started
- [QUICKSTART.md](../QUICKSTART.md) - Getting started guide
- [API Reference](../reference/api-endpoints.md) - Endpoint documentation
- [CLI Reference](../reference/cli-commands.md) - Command documentation

### Core Concepts
- [Swarm Model](SWARM-MODEL.md) - The Skills + Agents + Teams architecture and Action execution layer
- [Relations](RELATIONS.md) - Team-member relations
- [SOUL System](PERSONA-SYSTEM.md) - Agent personality via SOUL.md (plus optional agent .md files)
- [Capability Matching](CAPABILITY-MATCHING.md) - Skill-to-agent matching
- [3D World Architecture](3D-WORLD-ARCHITECTURE.md) - Visualization system

### Internal
- [Testing Seams](../internal/SEAMS.md) - Testing architecture details
- [Store Migration](STORE-MIGRATION.md) - Migration from legacy storage
