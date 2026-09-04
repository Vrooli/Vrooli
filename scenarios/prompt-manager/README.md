# Prompt Manager

**Skills + Agents + Teams** management system for Vrooli, providing reusable AI skills, agent coordination, and markdown-based skill references with 3D world visualization. Prompt-manager also documents a proposed **Actions** layer for typed executable wrappers over Vrooli-controlled CLI operations.

## Features

- **Skills Management**: Full CRUD, versioning, AI search, testing, and pack organization (core/local/drafts)
- **Agents**: Entities with appearance, SOUL.md + agent files, capabilities, connectors, and heartbeat
- **Teams**: Organizational units with roles, members, org chart, and shared docs
- **3D World**: a diorama of the swarm where place is state (desks, team tables, commons), driven by the live run feed, with a HUD that makes the swarm actionable
- **Multiple Interfaces**: Web UI, REST API, and command-line tool
- **Text-Only Skills**: Agents and teams reference skills directly in markdown
- **Relations**: Team-member memberships
- **Actions (Proposed)**: Typed execution wrappers that let agents discover and run deterministic Vrooli-controlled CLI operations

## Quick Start

1. **Start the application:**
   ```bash
   bash deployment/startup.sh
   ```

2. **Access the web interface:**
   Open http://localhost:${UI_PORT} in your browser (port provided by lifecycle system)

3. **Use the CLI:**
   ```bash
   prompt-manager help
   prompt-manager add "My first skill" debugging
   prompt-manager list
   ```

4. **API access:**
   The REST API is available at http://localhost:${API_PORT} (port provided by lifecycle system)

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    prompt-manager                        │
├─────────────┬─────────────┬─────────────┬───────────────┤
│   Skills    │   Agents    │   Teams     │   Actions*    │
│ (judgment)  │  (souls)    │  (roles)    │ (execution)   │
└─────────────┴─────────────┴─────────────┴───────────────┘
                            │
            ┌───────────────┼───────────────┐
            ▼               ▼               ▼
      File Store        SQLite         Qdrant
      (entities)      (metrics)       (vectors)
```

`*` Actions are proposed and documented before implementation. See `docs/concepts/ACTIONS.md`.

### Components

- **Go API Server** (port allocated by lifecycle): RESTful backend with skills, agents, and teams management
- **React UI** (port allocated by lifecycle): Web interface with pack navigation, skill editor, and 3D world
- **Go CLI**: Command-line tool for quick operations
- **File-based Store**: Primary entity storage (store/skills/, store/agents/, store/teams/)
- **SQLite**: Embedded storage for tags, metrics, and test history
- **Qdrant** (optional): Vector database for semantic search
- **Ollama** (optional): Local LLM for skill testing

### Storage Structure

```
store/
├── skills/packs/
│   ├── core/           # System skills
│   ├── local/          # User-created skills
│   └── drafts/         # Work-in-progress
├── agents/             # Agent entities
├── teams/              # Team entities with roles
├── relations/          # Team-member mappings
└── indexes/            # Generated lookup indexes
```

The planned Actions store will follow the same per-entity, schema-validated posture as skills, agents, and teams once implemented.

## API Endpoints

Stable domains use generated Connect services under
`/vrooli.prompt_manager.v1.<domain>.<Service>/<Method>`:

- `SkillsService`, `ActionsService`, and `TagsService` own authored capability state.
- `SearchService`, `AISearchService`, and `DiscoveryService` own deterministic, semantic, and composed discovery.
- `AgentsService` and `TeamsService` own identity, membership, team structure, files, roles, and exchange.
- `TopicsService`, `TemplatesService`, `TestingService`, and `MetadataService` own supporting taxonomy, templates, skill tests, and link metadata.
- `WorldService` owns world preferences, per-scene layout overrides and the server-streamed swarm feed (`docs/concepts/WORLD-ARCHITECTURE.md`).

Use the generated Go/TypeScript clients or the CLI instead of constructing
service URLs by hand. REST remains only for domains still listed in
[`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md#baseline-rest-route-inventory).

### Search & Discovery
- `SearchService.SearchSkills` - Full-text search
- `AISearchService.SearchSkills` - Vector similarity search
- `DiscoveryService.Discover` - Budgeted capability discovery

## CLI Commands

```bash
# Status and health
prompt-manager status

# Skill operations
prompt-manager skill list [--folder=core|local|drafts] [--tag=TAG]
prompt-manager skill show <id>
prompt-manager skill add <name> [--folder=local] [--tags=...]
prompt-manager skill update <id> [--name=...] [--tags=...]
prompt-manager skill delete <id> [--force]
prompt-manager skill read <id>                  # Read and record usage
prompt-manager skill versions <id>               # View version history
prompt-manager skill revert <id> <version>       # Revert to version

# Agent operations
prompt-manager agent list
prompt-manager agent show <id>
prompt-manager agent create <name> [--body-color=...]
prompt-manager agent update <id> [--name=...]
prompt-manager agent delete <id> [--force]

# Search
prompt-manager search <query> [--folder=...] [--tag=...]

# Tag operations
prompt-manager tag list
prompt-manager tag create <name> [--color=...]

# Testing (requires Ollama)
prompt-manager test run <skill-id> [--model=...]
prompt-manager test history <skill-id>
```

## Configuration

### App Configuration
Located in `api/internal/<domain>/configuration/app-config.json`:
- Port settings
- SQLite override hooks
- Feature toggles
- UI preferences
- Resource limits

### Campaign Templates
Located in `api/internal/<domain>/configuration/campaign-templates.json`:
- Pre-configured campaign types
- Color and icon schemes
- Quick setup options

## Development

### Prerequisites
- Go 1.21+
- Node.js 16+
- (Optional) Qdrant, Ollama

### Setup
```bash
# API Server
cd api
go run main.go

# UI Development
cd ui  
npm install
npm start

# CLI installation is handled by the control plane from the declared Go module.
```

### Testing
```bash
# Validation
bash deployment/validate.sh

# Integration tests
# (Test specification in scenario-test.yaml)
```

## Resource Requirements

- **Memory**: ~200MB
- **Storage**: ~50MB initial + data growth
- **CPU**: Minimal (1 core sufficient)
- **Network**: Ports allocated dynamically by lifecycle system

## Optional Enhancements

### Semantic Search (requires Qdrant)
- Vector embeddings for skill content
- Similarity-based discovery
- Related skill suggestions

### Skill Testing (requires Ollama)
- Test skills with local LLMs
- Performance and quality metrics
- Effectiveness ratings

## Use Cases

- **Agent Swarms**: Coordinate teams of agents (Debug, Feature, QA, Refactor) that analyze codebases and produce plans
- **Staging Plans via Swarm Manager**: Teams deposit their plans as backlog items into the `swarm-manager` scenario, where operators review and refine them with the Idea Agent before execution
- **Skill Libraries**: Build reusable AI capabilities with versioning and search
- **Team Coordination**: Organize agents into teams with shared context and roles
- **Developers**: Debug patterns, code review templates, architecture decisions
- **Designers**: UX research methods, design system components, user journey analysis

## Data Flow

```
CLI/UI → Go API → File Store (skills, agents, teams)
                → SQLite (tags, metrics, test history)
                → Qdrant (embeddings)
                → Ollama (testing)
```

All interfaces interact through the central Go API server, ensuring consistent data handling and business logic. The 3D world reads a deterministic simulation fed by WorldService's stream; see docs/concepts/WORLD-SIM.md.
