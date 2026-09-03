# Architecture Overview

This document describes prompt-manager's governing target architecture. The
domain inventory, dependency graph, proto package map, binding policy, and
migration order live in [DOMAINS.md](DOMAINS.md). During the staged migration,
code that has not reached its assigned slice may still use the legacy REST
shape documented below; that is transitional state, not a second supported
architecture.

## Governing target

Prompt Manager is a **proto-first Connect API**. Each callable business domain
owns one versioned proto package under
`packages/proto/schemas/prompt-manager/v1/<domain>`. Generated Go and
TypeScript artifacts are the transport contract. Connect handlers are thin
adapters: decode, map to domain values, delegate once, map the result, and map
typed errors. Business validation, persistence, ranking, scheduling, and
policy do not live in handlers.

The CLI is a **thin generated-client wrapper**. It owns flags, attribution,
human/JSON rendering, and explicitly local conveniences only. It does not
reimplement domain behavior or make hand-authored REST calls. Every
runtime-visible command is either bound to one proto RPC with accurate
governance or appears in `omitted[]` with a specific reason and owner.

```text
CLI / UI / Program Runtime
          |
          v
generated Connect clients
          |
          v
api/handlers/<domain>       transport-only mapping
          |
          v
api/internal/<domain>       domain service + owned ports
          |
          v
api/internal/store          filesystem / SQLite adapters
          |
          +--> Qdrant / Ollama / governed scenario clients
```

The composition root mounts services and wires ports. Domains do not import
the composition root, handlers, CLI, or generated clients. Cross-domain reads
use narrow consumer-owned interfaces. Heartbeat and memberflow retain distinct
ownership but migrate together because their scheduling, declarations, and
prompt construction form one temporal contract.

Transport is portable by construction: clients resolve the lifecycle-managed
API address, paths use Go's platform-aware APIs, and no contract depends on a
shell, fixed home directory, executable suffix, or path separator.

## Migration invariants

- A migrated operation has one supported transport. Its REST registration is
  removed only after repository consumer search and live CLI parity.
- Generated messages are wire types, not the domain model.
- Stateful domains expose real, data-bearing measures or a named waiver.
- Reads are program-eligible by default; writes require bounded targets,
  attribution, and truthful effect declarations. Destructive/operator-only
  commands are omitted until those properties are proved.
- `skill read`, `skill list`, and `discover` are mandatory bindings because
  they are the exercised supply chain for agent recall and discovery.

## Transitional current system

The following snapshot describes the pre-migration implementation that the
phased work is replacing. It is retained to make seams and storage decisions
auditable, not to endorse hand-written REST as a parallel target.

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
                                 │ REST today; Connect is the target
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
│   File System     │                    │       SQLite        │
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

The current tree groups much behavior by business name, but it does not yet
meet the scenario layout contract: handlers and domain implementation are
mixed in `api/<domain>`, while the target `api/handlers/*` and
`api/internal/*` trees are mostly empty. Phase 11 normalizes that layout
without changing behavior; [DOMAINS.md](DOMAINS.md) is the authoritative map.

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
| Tags | SQLite | Relational queries, shared taxonomy |
| Usage metrics | SQLite | Aggregations and usage history |
| Test results | SQLite | Local prompt-test history |
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
└── indexes/                    # Generated (never hand-edit)
    ├── skills.json
    ├── agents.json
    └── teams.json
```

Store shapes are validated in Go rather than by JSON Schema: `team.json` in
`api/teamcontract/contract.go`, member `topics.json` and the operating-model
contract in `api/memberflow/`.

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

The target CLI is a Connect-first client with a small amount of
contract-aware flag resolution, following noun-verb command patterns:

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

**Design Principle:** Every remote CLI command maps 1:1 to a generated RPC.
CLI-local commands are explicitly omitted from program bindings. No business
logic lives in the CLI.

The Action CLI follows the same principle. `prompt-manager action run <id>` resolves an Action contract through the API and executes one controlled command through the governed runtime; it does not contain business logic or shell recipes.

## UI Architecture

React-based SPA with:
- **React Query** for data fetching/caching
- **Zustand** stores for UI state
- **React Three Fiber** for 3D world visualization

See [World Architecture](WORLD-ARCHITECTURE.md) for visualization details.

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
| Tags, Metrics, Test Results | API | SQLite |
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
- [World Architecture](WORLD-ARCHITECTURE.md) - Visualization system

### Internal
- [Testing Seams](../internal/SEAMS.md) - Testing architecture details
- [Store Migration](STORE-MIGRATION.md) - Migration from legacy storage
