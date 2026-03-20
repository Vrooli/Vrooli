# Quick Start

Get the Stream of Consciousness Analyzer running locally in under 5 minutes.

## Prerequisites

- Vrooli CLI installed (`vrooli help` to verify)
- PostgreSQL resource enabled (check `.vrooli/service.json`)

## Start the Scenario

```bash
cd scenarios/stream-of-consciousness-analyzer
make start
```

This starts the Go API server and React UI dev server. The output displays the allocated ports.

## Verify Health

```bash
# Check scenario status
vrooli scenario status stream-of-consciousness-analyzer

# Get current API port
vrooli scenario port stream-of-consciousness-analyzer API_PORT
```

## First Use

1. Open the UI URL shown in `make start` output (typically `http://localhost:<UI_PORT>`)
2. Click **New Scheme** to create a capture workspace
3. Use the text input to add information items to the canvas
4. Switch to **Graph View** to see thought relationships
5. Use the **Suggestions** panel to generate LLM-powered connections

## Run Tests

```bash
make test                                           # Full test suite
vrooli scenario test stream-of-consciousness-analyzer unit  # Unit tests only
```

## Configuration

Key environment variables (set automatically by the lifecycle system):

| Variable | Purpose | Default |
|----------|---------|---------|
| `API_PORT` | API server port | Allocated dynamically |
| `UI_PORT` | UI dev server port | Allocated dynamically |
| `POSTGRES_*` | Database connection | From postgres resource |
| `OLLAMA_URL` | LLM for suggestions | `http://localhost:11434` |

## Next Steps

- [Architecture](concepts/ARCHITECTURE.md) — understand the system design
- [API Reference](reference/api-endpoints.md) — explore the REST API
- [PRD](../PRD.md) — operational targets and product vision
