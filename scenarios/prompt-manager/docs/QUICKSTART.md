# Quick Start Guide

Get the prompt-manager scenario running in 5 minutes.

## Prerequisites

- Go 1.21+
- Node.js 18+
- (Optional) Ollama for skill testing

## Start the Scenario

```bash
# From Vrooli root
cd scenarios/prompt-manager
make start
```

This starts:
- **API Server** on a dynamically allocated port
- **UI** on a dynamically allocated port (check logs for actual ports)

## Verify It's Running

```bash
# Check scenario health
make status

# Or use CLI
./cli/prompt-manager status
```

## Access the Application

### Web UI

Open your browser to the UI port shown in the logs. The interface provides:
- Skill browser with folder organization (core/local/drafts)
- Skill editor with live preview
- 3D world visualization for agents
- Usage tracking and metrics

### CLI

```bash
# Build the CLI
cd cli && go build -o prompt-manager .

# List all skills
./prompt-manager skill list

# Show a specific skill
./prompt-manager skill show <skill-id>

# Use a skill (records usage + copies to clipboard)
./prompt-manager skill use <skill-id>

# Search skills
./prompt-manager search "debugging"

# See all commands
./prompt-manager --help
```

### API

The REST API is available at the API port shown in logs.

```bash
# Health check
curl http://localhost:PORT/health

# List skills
curl http://localhost:PORT/api/v1/skills

# Get a skill
curl http://localhost:PORT/api/v1/skills/{id}

# Search through the generated Connect-backed CLI
prompt-manager search debugging --text --json
```

## Common Operations

### Create a Skill

```bash
# Via CLI (interactive)
./prompt-manager skill add "My New Skill" --folder=local --tags=debugging,testing

# Via API
curl -X POST http://localhost:PORT/api/v1/skills \
  -H "Content-Type: application/json" \
  -d '{"name": "My Skill", "content": "...", "folder": "local"}'
```

### Test a Skill with Ollama

Requires Ollama running with a model loaded.

```bash
# Via CLI
./prompt-manager test run <skill-id> --model=llama3.2

# Via API
curl -X POST http://localhost:PORT/api/v1/skills/{id}/test \
  -H "Content-Type: application/json" \
  -d '{"model": "llama3.2"}'
```

### View Version History

```bash
# Via CLI
./prompt-manager skill versions <skill-id>

# Revert to a previous version
./prompt-manager skill revert <skill-id> 2
```

## Stop the Scenario

```bash
make stop
```

## Next Steps

- [Architecture Overview](concepts/ARCHITECTURE.md) - System design and data flow
- [API Reference](reference/api-endpoints.md) - Complete endpoint documentation
- [CLI Reference](reference/cli-commands.md) - All CLI commands and options
- [Configuration](reference/configuration.md) - Environment variables and settings

## Troubleshooting

### API not responding

```bash
# Check if scenario is running
make status

# View logs
make logs
```

### SQLite database failed

Prompt-manager creates its SQLite database automatically under the Vrooli storage root. If you need a known location for debugging, set an explicit file path before starting:

```bash
# Redirect the whole storage tree, not one database file. The root is
# scenario-agnostic, so every scenario beneath it still resolves to its own
# separate path.
export VROOLI_STORAGE_ROOT=/tmp/vrooli-storage
make start
```

### CLI can't connect to API

The CLI auto-detects the API URL from the scenario lifecycle. If running manually:
```bash
./prompt-manager --api-base=http://localhost:YOUR_PORT skill list
```
