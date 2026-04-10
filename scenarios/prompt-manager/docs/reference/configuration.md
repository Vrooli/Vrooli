# Configuration Reference

Environment variables and configuration options for prompt-manager.

## Environment Variables

### API Server

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_URL` | (disabled) | Ollama API URL for skill testing (e.g., `http://localhost:11434`) |
| `STORE_DIR` | `../store` | Path to the store directory containing skills, agents, teams, and relations |
| `DATABASE_URL` | (from lifecycle) | PostgreSQL connection string |
| `QDRANT_URL` | `http://localhost:6333` | Qdrant vector database URL for AI search |
| `QDRANT_API_KEY` | (none) | API key for Qdrant authentication |
| `AI_SEARCH_COLLECTION` | `prompt-manager-skills` | Qdrant collection name for skill embeddings |
| `AI_SEARCH_THRESHOLD` | `0.5` | Minimum similarity score for AI search results |

### CLI

| Variable | Default | Description |
|----------|---------|-------------|
| `PM_API_BASE` | (auto-detected) | Override API base URL |
| `NO_COLOR` | | Disable colored output when set |

## Port Allocation

Ports are dynamically allocated by the Vrooli lifecycle system. To find active ports:

```bash
# Check scenario status
cd scenarios/prompt-manager && make status

# Or check logs
make logs | grep "listening on"
```

## Database Configuration

PostgreSQL database with the following schema:

**Required Tables:**
- `tags` - Tag definitions
- `skill_metrics` - Usage tracking
- `test_results` - LLM test history

**Setup:**
```bash
createdb prompt_manager
psql -d prompt_manager < initialization/storage/postgres/schema.sql
```

## Store Directory Structure

The storage system uses a per-entity file structure under the `store/` directory:

```
store/
├── skills/
│   ├── _pack-order.json        # Active pack precedence
│   └── packs/
│       ├── core/               # System skills
│       │   └── debugging/
│       │       ├── skill.json  # Metadata
│       │       ├── SKILL.md    # Content
│       │       └── history.jsonl
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
├── relations/
│   └── team-member/
│       └── team-id__agent-1.json
├── indexes/                    # Generated (never hand-edit)
│   ├── skills.index.json
│   ├── agents.index.json
│   └── teams.index.json
└── schemas/                    # JSON Schemas for validation
    ├── skill.schema.json
    ├── agent.schema.json
    └── team.schema.json
```

### skill.json Format

```json
{
  "id": "debugging",
  "name": "Debugging",
  "description": "Systematic debugging approach",
  "modes": ["agent"],
  "tags": ["debugging"],
  "icon": "bug",
  "draft": false,
  "createdAt": "2024-01-15T10:00:00Z",
  "updatedAt": "2024-01-20T14:30:00Z"
}
```

### agent.json Format

```json
{
  "id": "agent-1",
  "displayName": "Alice",
  "status": "active",
  "appearance": {
    "body": "#3B82F6",
    "head": "#F59E0B",
    "accent": "#10B981"
  },
  "createdAt": "2024-01-15T10:00:00Z",
  "updatedAt": "2024-01-20T14:30:00Z"
}
```

### team.json Format

```json
{
  "id": "engineering",
  "displayName": "Engineering Team",
  "mission": "Build great software",
  "createdAt": "2024-01-15T10:00:00Z",
  "updatedAt": "2024-01-20T14:30:00Z"
}
```

## Optional Resources

### Ollama (Skill Testing)

Enable LLM-based skill testing:

```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull a model
ollama pull llama3.2

# Set environment variable
export OLLAMA_URL=http://localhost:11434
```

Test via CLI:
```bash
prompt-manager test run debugging --model=llama3.2
```

### Qdrant (Semantic Search)

Enable vector-based semantic search:

```bash
# Start Qdrant
docker run -p 6333:6333 qdrant/qdrant

# Set environment variable
export QDRANT_URL=http://localhost:6333
```

## App Configuration

Located at `initialization/configuration/app-config.json`:

```json
{
  "features": {
    "semanticSearch": false,
    "skillTesting": true,
    "versionHistory": true
  },
  "ui": {
    "defaultView": "grid",
    "theme": "system"
  },
  "limits": {
    "maxSkillSize": 100000,
    "maxVersionHistory": 50
  }
}
```

## Graph Health Configuration

Graph health scoring controls are persisted in:

`store/config/graph-health.json`

This file is git-tracked and can be tuned directly or through the Graph View `Health` settings tab.

Schema:

```json
{
  "team": {
    "outgoingEdges": 1,
    "incomingEdges": 1,
    "codeUsage": 0.5,
    "recentActivity": 0.5,
    "skillContentLength": 0,
    "agentContextLoad": 0,
    "teamMemberCountBalance": 0.75,
    "teamRoleCoverage": 0.75
  },
  "agent": {
    "outgoingEdges": 1,
    "incomingEdges": 1,
    "codeUsage": 0.5,
    "recentActivity": 0.5,
    "skillContentLength": 0,
    "agentContextLoad": 0.75,
    "teamMemberCountBalance": 0,
    "teamRoleCoverage": 0
  },
  "skill": {
    "outgoingEdges": 1,
    "incomingEdges": 1,
    "codeUsage": 0.5,
    "recentActivity": 0.5,
    "skillContentLength": 0.75,
    "agentContextLoad": 0,
    "teamMemberCountBalance": 0,
    "teamRoleCoverage": 0
  },
  "cli": {
    "neutralCommands": ["vrooli"],
    "externalToolScore": 0,
    "scenarioFallbackScore": 0
  }
}
```

After changing values, regenerate the graph to recompute health:

```bash
prompt-manager graph regenerate
```

## Campaign Templates

Located at `initialization/configuration/campaign-templates.json`:

Predefined campaign types with colors and icons for organizing skills.

## Docker Configuration

The scenario can run in Docker via the lifecycle system:

```bash
# Build and start
cd scenarios/prompt-manager && make docker-start

# View logs
make docker-logs

# Stop
make docker-stop
```

## Health Checks

The API exposes health endpoints:

```bash
# Basic health
curl http://localhost:PORT/health

# Detailed health (includes database status)
curl http://localhost:PORT/api/v1/health
```

Response:
```json
{
  "status": "healthy",
  "version": "2.0.0",
  "checks": {
    "database": "healthy"
  }
}
```

## Logging

Logs are written to stdout. Control verbosity with:

| Variable | Values | Description |
|----------|--------|-------------|
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | Minimum log level |
| `LOG_FORMAT` | `text`, `json` | Output format |
