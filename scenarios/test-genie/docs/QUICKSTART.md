# Test Genie Quick Start

Welcome to Test Genie - the comprehensive testing platform for Vrooli scenarios and resources.

## What is Test Genie?

Test Genie is a Go-native testing orchestration platform that:

- **Runs tests** - Execute multi-phase test suites with configurable presets
- **Tracks coverage** - Monitor test health across all scenarios
- **Syncs requirements** - Automatically track `[REQ:ID]` tags from tests
- **Provides APIs** - REST and CLI interfaces for agent automation

## Quick Start

### 1. Start Test Genie

```bash
cd scenarios/test-genie
make start
```

Or via CLI:
```bash
vrooli scenario start test-genie
```

### 2. Run Tests for a Scenario

**Via CLI:**
```bash
test-genie execute my-scenario --preset comprehensive
```

**Via API:**
```bash
API_PORT=$(vrooli scenario port test-genie API_PORT)
curl -X POST "http://localhost:${API_PORT}/api/v1/test-suite/my-scenario/execute-sync" \
  -H "Content-Type: application/json" \
  -d '{"preset": "comprehensive"}'
```

**Via Dashboard:**
1. Open `http://localhost:${UI_PORT}`
2. Select a scenario from the Catalog
3. Click "Run Tests"

### 3. View Results

Results are available via:
- **Dashboard** - Visual results with phase breakdowns
- **API** - `GET /api/v1/executions/{id}`
- **CLI** - `test-genie status --executions`

## Test Presets

| Preset | Phases | Time | Use Case |
|--------|--------|------|----------|
| **Quick** | Structure, Unit | ~1 min | Fast sanity check |
| **Smoke** | Structure, Dependencies, Unit, Integration | ~4 min | Pre-push validation |
| **Comprehensive** | All 6 phases | ~8 min | Full coverage |

See [Presets Reference](reference/presets.md) for details.

## Test Phases

Test Genie uses a 6-phase testing architecture:

```
Structure → Dependencies → Unit → Integration → Business → Performance
```

| Phase | Purpose | Timeout |
|-------|---------|---------|
| **Structure** | Validate files and config | 15s |
| **Dependencies** | Check tools and resources | 30s |
| **Unit** | Run unit tests (Go, Node, Python) | 60s |
| **Integration** | Test API/UI connectivity | 120s |
| **Business** | Validate workflows and rules | 180s |
| **Performance** | Run benchmarks | 60s |

See [Phased Testing Guide](guides/phased-testing.md) for the complete architecture.

## Documentation Navigation

### Guides (How-To)
- [Phased Testing](guides/phased-testing.md) - Understanding the 6-phase architecture
- [Test Generation](guides/test-generation.md) - AI-powered test creation
- [Requirements Sync](guides/requirements-sync.md) - Automatic requirement tracking
- [Scenario Unit Testing](guides/scenario-unit-testing.md) - Go, Node, Python unit tests
- [CLI Testing](guides/cli-testing.md) - BATS testing for CLIs
- [UI Testability](guides/ui-testability.md) - Design testable UIs
- [Vault Testing](guides/vault-testing.md) - Multi-phase lifecycle validation
- [Sync Execution](guides/sync-execution.md) - Blocking execution for agents

### Reference (Technical Details)
- [API Endpoints](reference/api-endpoints.md) - REST API reference
- [CLI Commands](reference/cli-commands.md) - test-genie CLI reference
- [Presets](reference/presets.md) - Quick/Smoke/Comprehensive definitions
- [Phase Catalog](reference/phase-catalog.md) - Detailed phase specs
- [Test Runners](reference/test-runners.md) - Language-specific runners

### Concepts (Architecture)
- [Architecture](concepts/architecture.md) - Go orchestrator design

## Common Tasks

### Generate Tests for a New Scenario

```bash
test-genie generate my-scenario --types unit,integration
```

See [Test Generation Guide](guides/test-generation.md).

### Track Requirements from Tests

Add `[REQ:ID]` tags to your tests:

```go
func TestCreateProject(t *testing.T) {
    t.Run("creates project [REQ:MY-PROJECT-CREATE]", func(t *testing.T) {
        // test code
    })
}
```

Run comprehensive tests to auto-sync:
```bash
test-genie execute my-scenario --preset comprehensive
```

See [Requirements Sync Guide](guides/requirements-sync.md).

### Use Sync Execution for CI/Agents

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/test-suite/my-scenario/execute-sync" \
  -H "Content-Type: application/json" \
  -d '{"preset": "smoke", "failFast": true}'
```

Returns complete results in a single blocking request.

See [Sync Execution Guide](guides/sync-execution.md) and [Cheatsheet](reference/sync-execution-cheatsheet.md).

## Troubleshooting

### Tests Fail with "scenario not found"

Ensure the scenario exists:
```bash
vrooli scenario list | grep my-scenario
```

### Phase Times Out

Increase timeout in `.vrooli/testing.json`:
```json
{
  "phases": {
    "unit": { "timeout": 120 }
  }
}
```

### Requirements Not Syncing

Ensure tests have `[REQ:ID]` tags and run with comprehensive preset.

See [PROBLEMS.md](PROBLEMS.md) for known issues.

## Next Steps

1. **New to testing?** Start with [Phased Testing Guide](guides/phased-testing.md)
2. **Writing unit tests?** See [Scenario Unit Testing](guides/scenario-unit-testing.md)
3. **Building CI pipelines?** Check [Sync Execution Guide](guides/sync-execution.md)
4. **Designing testable UIs?** Read [UI Testability](guides/ui-testability.md)
