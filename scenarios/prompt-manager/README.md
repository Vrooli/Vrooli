# Prompt Manager

**Skills + Agents + Teams** management system for Vrooli, providing reusable AI skills, agent coordination, and team-based skill grants with 3D world visualization.

## Features

- **Skills Management**: Full CRUD, versioning, AI search, testing, and pack organization (core/local/drafts)
- **Agents**: Entities with appearance, SOUL.md + agent files, capabilities, connectors, heartbeat, and skill pins
- **Teams**: Organizational units with roles, members, org chart, and role-based skill grants
- **3D World Visualization**: Interactive React Three Fiber visualization for agent coordination
- **Multiple Interfaces**: Web UI, REST API, and command-line tool
- **Effective Skills**: Computed from agent pins + team role grants
- **Relations**: Agent-skill assignments and team-member memberships

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
│   Skills    │   Agents    │   Teams     │   Relations   │
│  (packs)    │  (souls)    │  (roles)    │ (assignments) │
└─────────────┴─────────────┴─────────────┴───────────────┘
                            │
            ┌───────────────┼───────────────┐
            ▼               ▼               ▼
      File Store      PostgreSQL       Qdrant
      (entities)      (metrics)       (vectors)
```

### Components

- **Go API Server** (port allocated by lifecycle): RESTful backend with skills, agents, and teams management
- **React UI** (port allocated by lifecycle): Web interface with pack navigation, skill editor, and 3D world
- **Go CLI**: Command-line tool for quick operations
- **File-based Store**: Primary entity storage (store/skills/, store/agents/, store/teams/)
- **PostgreSQL** (optional): Analytics and metrics storage
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
├── relations/          # Agent-skill, team-member mappings
└── indexes/            # Generated lookup indexes
```

## API Endpoints

### Skills
- `GET /api/v1/skills` - List skills with filters (folder, tag, mode)
- `POST /api/v1/skills` - Create new skill
- `GET /api/v1/skills/{id}` - Get skill details
- `PUT /api/v1/skills/{id}` - Update skill
- `DELETE /api/v1/skills/{id}` - Delete skill
- `POST /api/v1/skills/{id}/use` - Record usage
- `GET /api/v1/skills/{id}/versions` - Version history
- `POST /api/v1/skills/{id}/revert/{ver}` - Revert to version

### Agents
- `GET /api/v1/agents` - List all agents
- `POST /api/v1/agents` - Create agent
- `GET /api/v1/agents/{id}` - Get agent details
- `PUT /api/v1/agents/{id}` - Update agent
- `DELETE /api/v1/agents/{id}` - Delete agent
- `GET /api/v1/agents/{id}/effective-skills` - Get computed skill set

### Teams
- `GET /api/v1/teams` - List all teams
- `POST /api/v1/teams` - Create team
- `GET /api/v1/teams/{id}` - Get team with roles and members
- `PUT /api/v1/teams/{id}` - Update team
- `DELETE /api/v1/teams/{id}` - Delete team
- `POST /api/v1/teams/{id}/members` - Add member
- `PUT /api/v1/teams/{id}/members/{agentId}` - Update member
- `DELETE /api/v1/teams/{id}/members/{agentId}` - Remove member

### Search & Discovery
- `GET /api/v1/search/skills?q={query}` - Full-text search
- `POST /api/v1/search/ai` - Vector similarity search

### Tags
- `GET /api/v1/tags` - List all tags
- `POST /api/v1/tags` - Create tag

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
prompt-manager skill use <id>                    # Copy and record usage
prompt-manager skill versions <id>               # View version history
prompt-manager skill revert <id> <version>       # Revert to version

# Agent operations
prompt-manager agent list
prompt-manager agent show <id>
prompt-manager agent create <name> [--skills=...] [--body-color=...]
prompt-manager agent update <id> [--name=...] [--skills=...]
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
Located in `initialization/configuration/app-config.json`:
- Port settings
- Database configuration  
- Feature toggles
- UI preferences
- Resource limits

### Campaign Templates
Located in `initialization/configuration/campaign-templates.json`:
- Pre-configured campaign types
- Color and icon schemes
- Quick setup options

## Development

### Prerequisites
- Go 1.21+
- Node.js 16+
- PostgreSQL
- (Optional) Qdrant, Ollama

### Setup
```bash
# Database
createdb prompt_manager
psql -d prompt_manager < initialization/storage/postgres/schema.sql
psql -d prompt_manager < initialization/storage/postgres/seed.sql

# API Server
cd api
go mod tidy
go run main.go

# UI Development
cd ui  
npm install
npm start

# CLI Installation
bash cli/install.sh
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

- **Agent Swarms**: Coordinate teams of agents for autonomous work (outreach, bug fixing, etc.)
- **Skill Libraries**: Build reusable AI capabilities with versioning and search
- **Team Coordination**: Organize agents into teams with role-based skill grants
- **Developers**: Debug patterns, code review templates, architecture decisions
- **Designers**: UX research methods, design system components, user journey analysis

## Data Flow

```
CLI/UI → Go API → File Store (skills, agents, teams)
                → PostgreSQL (metrics, analytics)
                → Qdrant (embeddings)
                → Ollama (testing)
```

All interfaces interact through the central Go API server, ensuring consistent data handling and business logic. The 3D world visualization connects via React Query for real-time agent state.
