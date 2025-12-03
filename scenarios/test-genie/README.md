# Test Genie

Go-native test orchestration platform for Vrooli scenarios and resources.

## Quick Start

```bash
# Start test-genie
cd scenarios/test-genie
make start

# Run tests for any scenario
test-genie execute my-scenario --preset comprehensive

# View results
test-genie status --executions
```

## What Test Genie Does

- **Executes tests** via 7-phase pipeline (structure → dependencies → unit → integration → e2e → business → performance)
- **Tracks requirements** by auto-syncing `[REQ:ID]` tags from test results
- **Provides APIs** for agent automation (REST + CLI)
- **Queues test generation** requests for downstream AI agents

## Architecture

```
test-genie/
├── api/           # Go REST API + orchestrator
├── cli/           # test-genie CLI binary
├── ui/            # React dashboard
└── docs/          # Comprehensive documentation
```

## Test Phases

| Phase | Timeout | Purpose |
|-------|---------|---------|
| Structure | 15s | Validate files, JSON configs |
| Dependencies | 30s | Check runtimes, tools, resources |
| Unit | 60s | Run Go/Node/Python unit tests |
| Integration | 120s | Test API endpoints, CLI commands |
| E2E | 120s | Execute BAS browser automation workflows |
| Business | 180s | Validate requirements coverage |
| Performance | 60s | Build time budgets, benchmarks (optional) |

## Presets

| Preset | Phases | Use Case |
|--------|--------|----------|
| `quick` | Structure, Dependencies | Fast sanity check |
| `smoke` | Structure, Dependencies, Unit | Pre-commit validation |
| `comprehensive` | All 7 phases | Full CI/CD validation |

```bash
test-genie execute my-scenario --preset smoke
```

## CLI Usage

```bash
# Execute tests
test-genie execute <scenario> [--preset quick|smoke|comprehensive] [--fail-fast]

# Check status
test-genie status [--executions] [--verbose]

# Queue test generation (delegates to AI agents)
test-genie generate <scenario> --types unit,integration
```

## REST API

```bash
# Get API port
API_PORT=$(vrooli scenario port test-genie API_PORT)

# Health check
curl http://localhost:${API_PORT}/health

# Execute tests (synchronous)
curl -X POST "http://localhost:${API_PORT}/api/v1/executions" \
  -H "Content-Type: application/json" \
  -d '{"scenarioName": "my-scenario", "preset": "comprehensive"}'

# List executions
curl "http://localhost:${API_PORT}/api/v1/executions?scenario=my-scenario&limit=10"

# Get phase catalog
curl "http://localhost:${API_PORT}/api/v1/phases"
```

See [docs/reference/api-endpoints.md](docs/reference/api-endpoints.md) for complete API reference.

## Requirements Tracking

Tag tests with `[REQ:ID]` to auto-sync requirement coverage:

```go
// Go
t.Run("creates project [REQ:PROJECT-CREATE]", func(t *testing.T) { ... })
```

```typescript
// Vitest
describe('projectStore [REQ:PROJECT-CRUD]', () => { ... })
```

After running comprehensive tests, requirements are synced to `requirements/*.json`.

See [docs/guides/requirements-sync.md](docs/guides/requirements-sync.md) for details.

## Configuration

### Per-Scenario (`.vrooli/testing.json`)

```json
{
  "phases": {
    "unit": { "timeout": 120 },
    "performance": { "enabled": false }
  },
  "requirements": { "sync": true },
  "presets": { "default": "smoke" }
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | API server port | `8200` |
| `POSTGRES_HOST` | Database host | `localhost` |
| `POSTGRES_DB` | Database name | `test_genie` |

## Development

```bash
# Start with logs
make start && make logs

# Run tests
make test

# Stop
make stop
```

## Documentation

Comprehensive docs are in `docs/`:

- [QUICKSTART.md](docs/QUICKSTART.md) - Get started in 5 minutes
- [Phased Testing Guide](docs/guides/phased-testing.md) - 7-phase architecture
- [Requirements Sync](docs/guides/requirements-sync.md) - Auto-tracking from tests
- [API Reference](docs/reference/api-endpoints.md) - REST API documentation
- [CLI Reference](docs/reference/cli-commands.md) - CLI command reference
- [Safety Guidelines](docs/safety/GUIDELINES.md) - Critical safety rules for test scripts

See [docs/manifest.json](docs/manifest.json) for complete documentation index.
