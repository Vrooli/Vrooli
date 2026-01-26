# Configuration Reference

Environment variables and configuration options for prompt-manager.

## Environment Variables

### API Server

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_URL` | (disabled) | Ollama API URL for skill testing (e.g., `http://localhost:11434`) |
| `SKILLS_DIR` | `../skills` | Path to skills directory |
| `DATABASE_URL` | (from lifecycle) | PostgreSQL connection string |

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

## Skills Directory Structure

```
skills/
├── core/              # System skills (read-only in production)
│   ├── metadata.json  # Skill metadata array
│   └── *.md          # Skill content files
├── local/             # User-created skills
│   └── metadata.json
└── drafts/            # Work-in-progress
    └── metadata.json
```

### metadata.json Format

```json
[
  {
    "id": "skill-id",
    "name": "Skill Name",
    "description": "Brief description",
    "filename": "skill-id.md",
    "modes": ["agent", "human"],
    "tags": ["debugging"],
    "icon": "bug",
    "draft": false,
    "createdAt": "2024-01-15T10:00:00Z",
    "updatedAt": "2024-01-20T14:30:00Z"
  }
]
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
pm test run debugging --model=llama3.2
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
