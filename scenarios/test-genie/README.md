# Test Genie

Go-native test orchestration platform for Vrooli scenarios and resources.

## Quick Start

```bash
# Start test-genie
cd scenarios/test-genie
make start

# Run tests for any scenario
test-genie execute my-scenario --preset comprehensive

# Wait for the durable run printed by execute
test-genie runs wait --json --timeout=840 my-scenario <run-id>
```

## What Test Genie Does

- **Executes tests** via a descriptor-backed health-provider phase pipeline
- **Tracks requirements** by auto-syncing `[REQ:ID]` tags from test results
- **Turns completed execution evidence into ranked remediation jobs**
- **Delegates agent policy and protected-workspace execution to Agent Manager**

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
| Dependencies | 30s | Dependency health via scenario-dependency-analyzer |
| Quality | 120s | Static quality contracts via quality-health |
| Docs | 60s | Validate Markdown, mermaid, links, portability |
| Smoke | 90s | UI handshake via iframe-bridge |
| Unit | 60s | Run Go/Node/Python unit tests |
| Integration | 120s | Test API endpoints, CLI commands |
| Playbooks | 120s | Execute BAS browser automation workflows |
| Business | 180s | Validate requirements coverage |
| Performance | 60s | Build time budgets, benchmarks (optional) |

## Presets

| Preset | Phases | Use Case |
|--------|--------|----------|
| `quick` | Structure, Docs, Business, Unit, Proto | Fast sanity check |
| `smoke` | Structure, API, Quality, Docs, Business, Proto | Pre-commit validation |
| `comprehensive` | Every registered phase, including Quality | Full CI/CD validation |

```bash
test-genie execute my-scenario --preset smoke
```

Opt out per run with `--skip <phase>`.

## CLI Usage

```bash
# Execute tests
test-genie execute <scenario> [--preset quick|smoke|comprehensive] [--fail-fast]

# Check status
test-genie status [--executions] [--verbose]

# Launch one remediation job from completed execution evidence
test-genie remediate <scenario> --execution <uuid> --findings afid:example --role code.default

# Trigger the scenario-local runner
test-genie run-tests <scenario> [--type phased]

# Inspect the live descriptor-backed phase plan
test-genie phases --help
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

# Preview selected phases, estimate, and timeout budget
curl -X POST "http://localhost:${API_PORT}/api/v1/executions/plan" \
  -H "Content-Type: application/json" \
  -d '{"scenarioName": "my-scenario", "preset": "comprehensive"}'

# List executions
curl "http://localhost:${API_PORT}/api/v1/executions?scenario=my-scenario&limit=10"

# Get phase catalog
curl "http://localhost:${API_PORT}/api/v1/phases"

# Inspect a completed run's remediation plan, then create one job from stable IDs
curl "http://localhost:${API_PORT}/api/v1/scenarios/my-scenario/remediation/plans/<execution-uuid>"
curl -X POST "http://localhost:${API_PORT}/api/v1/scenarios/my-scenario/remediation/jobs" \
  -H "Content-Type: application/json" \
  -d '{"sourceExecutionId":"<execution-uuid>","findingIds":["afid:example"],"roleRef":"code.default"}'
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

See [docs/phases/business/requirements-sync.md](docs/phases/business/requirements-sync.md) for details.

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
| `TEST_GENIE_SQLITE_PATH` | Embedded SQLite database path | `${SCENARIO_DATA_DIR}/test-genie.db` |
| `SCENARIO_DATA_DIR` | Scenario-local persistent data root | Lifecycle-managed |

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
- [Phased Testing Guide](docs/guides/phased-testing.md) - 11-phase architecture
- [Requirements Sync](docs/phases/business/requirements-sync.md) - Auto-tracking from tests
- [API Reference](docs/reference/api-endpoints.md) - REST API documentation
- [CLI Reference](docs/reference/cli-commands.md) - CLI command reference
- [Execution Configuration](docs/reference/configuration.md) - Timeouts, planning, and estimate behavior
- [Safety Guidelines](docs/safety/GUIDELINES.md) - Critical safety rules for test scripts

See [docs/manifest.json](docs/manifest.json) for complete documentation index.
