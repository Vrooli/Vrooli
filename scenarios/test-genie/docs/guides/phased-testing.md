# Phased Testing Architecture

This guide explains Vrooli's comprehensive phased testing architecture and how to use Test Genie for scenario validation.

## Overview

Vrooli uses a **descriptor-backed testing architecture** that progressively validates scenarios from basic structure through provider-backed health phases. Test Genie orchestrates planning, applicability, provider readiness, lifecycle, run records, and reporting; each health provider owns its phase metadata and maturity contract in `scenarios/<provider>/.vrooli/test-genie.json`.

## The Phase Architecture

The effective phase registry is built from checked-in provider descriptors plus Test Genie-owned runner bindings. A run follows this order:

1. Load provider descriptors from `scenarios/*/.vrooli/test-genie.json`.
2. Evaluate target applicability from cheap declarative facts such as files, capabilities, UI/API presence, and testing config sections.
3. Select phases from the requested preset or explicit phase list against applicable phases.
4. Check or start selected providers according to descriptor policy.
5. Start target runtime surfaces only when selected phases need them.
6. Apply runnability gates for the current environment.
7. Execute provider validations and record findings, observations, metrics, and phase status.

Applicability answers "should this phase judge this target?" Runnability answers "can this already-applicable phase execute in this environment?" Provider readiness answers "is the selected provider usable for this execution?" These are separate states in JSON previews and run records.

See [Phases Overview](../phases/README.md) for the generated effective registry, policy dimensions, and phase definitions.

## Running Tests with Test Genie

### Using Test Presets

Test Genie provides preconfigured presets for common testing scenarios:

```bash
# Quick sanity check (~1 min)
test-genie execute my-scenario --preset quick

# Smoke test before pushing (~4 min)
test-genie execute my-scenario --preset smoke

# Full validation before release (~8 min)
test-genie execute my-scenario --preset comprehensive
```

| Preset | Phases | Use Case |
|--------|--------|----------|
| **Quick** | Structure, Docs, Business, Unit, Proto | Fast feedback during development |
| **Smoke** | Structure, Quality, Docs, Business, Integration, Proto | Pre-push validation |
| **Architecture Audit** | Structure, Contracts, UI Health, Docs, Architecture, Proto | Surface and architecture review |
| **Comprehensive** | All applicable registry phases | Full coverage before release |

See [Presets Reference](../reference/presets.md) for detailed preset definitions.

### Using the REST API

For CI/CD and agent automation, use the synchronous execution API:

```bash
# Get API port
API_PORT=$(vrooli scenario port test-genie API_PORT)

# Execute with comprehensive preset
curl -X POST "http://localhost:${API_PORT}/api/v1/test-suite/my-scenario/execute-sync" \
  -H "Content-Type: application/json" \
  -d '{
    "preset": "comprehensive",
    "failFast": true
  }'
```

See [Sync Execution Guide](sync-execution.md) for complete API usage.

### Using the Makefile

From within a scenario directory:

```bash
cd scenarios/my-scenario
make test     # Run all tests via test-genie
make logs     # View test logs
```

## What Each Phase Validates

### Phase 1: Structure Validation

**Purpose**: Ensure scenario has required files and valid configuration.

**Checks**:
- Required files exist (README.md, PRD.md, Makefile)
- `.vrooli/service.json` is valid JSON with required fields
- Test directory structure is correct
- File permissions are appropriate

**Example failures**:
- Missing `.vrooli/service.json`
- Schema validation failure in configuration
- Missing test directory

### Phase 2: Dependencies Check

**Purpose**: Verify all required tools and resources are available.

**Checks**:
- Language runtimes (Go, Node.js, Python) at required versions
- Package managers (pnpm, go mod) functional
- Required CLI tools present
- Resource dependencies declared in service.json

**Example failures**:
- Missing runtime or package manager reported by SDA readiness
- Required resource or scenario dependency unavailable
- Declared-vs-actual dependency graph drift
- Missing pnpm release-age policy or blocked dependency governance finding

### Phase 3: Quality

**Purpose**: Run static analysis and type checking to catch errors before runtime.

**Checks**:
- Static quality contracts are delegated to `quality-health audit run <scenario> --json`.
- TypeScript/JavaScript: `tsc --noEmit`, `eslint`
- Python: `ruff`/`flake8`, optional `mypy`

Warnings are informational; error findings fail. See [Quality Phase](../phases/quality/README.md).

### Phase 4: Docs Validation

**Purpose**: Ensure documentation stays portable and healthy.

**Checks**:
- Markdown structure (unclosed fences)
- Mermaid diagram headers + bracket balance
- Link integrity (local files must exist; external URLs HTTP-checked)
- Absolute filesystem paths are blocked (for example, `<absolute-path>` or `C:\\repo\\...`)

See [Docs Phase](../phases/docs/README.md) for configuration options.

### Phase 5: Unit Tests

**Purpose**: Run unit tests for all languages present in the scenario.

**Runners**:
| Language | Detection | Command |
|----------|-----------|---------|
| Go | `go.mod` in `api/` | `go test ./...` |
| Node.js | `package.json` in `ui/` | `pnpm test` |
| Python | `pytest.ini` or `test_*.py` | `pytest` |

**Coverage requirements**:
- Warning: < 80%
- Error: < 70%

See [Scenario Unit Testing](../phases/unit/scenario-unit-testing.md) for writing effective unit tests.

### Phase 6: Integration Tests

**Purpose**: Validate component interactions with running scenario.

**Requires**: Scenario must be running

**Checks**:
- API health endpoints responding
- UI accessible
- CLI commands functional
- Cross-component communication working

**Example tests**:
```bash
# API health check
curl -f http://localhost:${API_PORT}/health

# UI accessibility
curl -f http://localhost:${UI_PORT}

# CLI functionality
my-scenario-cli --version
```

### Phase 7: Workflow Tests

**Purpose**: Validate end-to-end UI workflows through workflow-health.

**Requires**: Scenario + BAS running

**Validates**:
- Declarative browser workflows in `bas/cases/` (via `bas/registry.json`)
- Isolation + seeds (temporary DB/Redis + `coverage/runtime/seed-state.json`)
- Contract correctness with BAS execution + timeline responses

See [Workflow Phase](../phases/workflow/README.md) for current behavior.

### Phase 8: Business Logic Tests

**Purpose**: Validate end-to-end workflows and business requirements.

**Requires**: Scenario must be running

**Validates**:
- Complete user journeys (multi-step workflows)
- Business rules and domain logic
- Data integrity across operations
- Error recovery workflows

**Example workflow test**:
```bash
# Create → Update → Verify workflow
project_id=$(curl -s -X POST "$API_URL/projects" -d '{"name":"Test"}' | jq -r '.id')
curl -s -X PUT "$API_URL/projects/$project_id" -d '{"name":"Updated"}'
result=$(curl -s "$API_URL/projects/$project_id" | jq -r '.name')
[ "$result" = "Updated" ] && echo "PASS" || echo "FAIL"
```

For UI workflows, use [BAS playbooks](ui-testability.md) instead of curl-based testing.

### Phase 9: Performance Tests

**Purpose**: Establish performance baselines and detect regressions.

**Requires**: Scenario must be running

**Measures**:
- API response times (p50, p95, p99)
- Build duration (Go, UI)
- Resource usage
- Throughput under load

**Example checks**:
```bash
# Build time budget
time go build -o /dev/null ./... # Should complete < 30s

# API response time
time curl -s "$API_URL/health" # Should respond < 100ms
```

## Test Directory Structure

Scenarios should have this test structure:

```
scenario/
├── .vrooli/
│   ├── service.json      # Scenario configuration (lifecycle.test.steps invokes test-genie)
│   └── testing.json      # Test-genie configuration (optional)
├── test/                  # Test artifacts directory
│   └── playbooks/        # BAS workflow tests (optional)
├── api/
│   └── *_test.go         # Go unit tests
└── ui/
    └── *.test.ts         # Vitest/Jest tests
```

> **Note**: Testing is orchestrated via `.vrooli/service.json` `lifecycle.test.steps` which invokes `test-genie execute`. The legacy `coverage/run-tests.sh` + `coverage/phases/*` pattern is deprecated.

## Configuration with `.vrooli/testing.json`

Customize test-genie behavior per scenario:

```json
{
  "phases": {
    "unit": {
      "timeout": 120,
      "coverageWarn": 85,
      "coverageError": 75
    },
    "performance": {
      "enabled": false
    }
  },
  "requirements": {
    "sync": true,
    "syncOnSuccess": true
  },
  "presets": {
    "default": "smoke"
  }
}
```

See [Phases Overview](../phases/README.md) for all configuration options.

## Requirements Tracking

Tag tests with `[REQ:ID]` to automatically track requirement coverage:

```go
func TestCreateProject(t *testing.T) {
    t.Run("creates project with valid data [REQ:MY-PROJECT-CREATE]", func(t *testing.T) {
        // Test implementation
    })
}
```

```typescript
describe('projectStore [REQ:MY-PROJECT-CRUD]', () => {
    it('creates project', () => { /* ... */ });
    it('updates project', () => { /* ... */ });
});
```

After running comprehensive tests, requirements are automatically synced:

```bash
test-genie execute my-scenario --preset comprehensive
# Requirements synced to requirements/index.json
```

See [Requirements Sync Guide](../phases/business/requirements-sync.md) for complete documentation.

## Dynamic Port Discovery

Never hardcode ports. Use dynamic discovery:

```bash
# Get ports from vrooli CLI
API_PORT=$(vrooli scenario port "$scenario_name" API_PORT)
UI_PORT=$(vrooli scenario port "$scenario_name" UI_PORT)

# Build URLs
API_URL="http://localhost:$API_PORT"
UI_URL="http://localhost:$UI_PORT"
```

## Coverage Standards

| Component | Minimum | Target |
|-----------|---------|--------|
| Unit Tests | 70% | 80%+ |
| Integration | All endpoints | All endpoints |
| Business Logic | Core workflows | All PRD requirements |
| Performance | Baselines set | No regressions |

## Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| "service.json not found" | Create `.vrooli/service.json` |
| "Invalid JSON" | Validate: `jq empty .vrooli/service.json` |
| "Port discovery fails" | Ensure scenario is running |
| "Phase skipped" | Check if scenario needs to be running |
| "Coverage too low" | Add more unit tests |
| "Timeout exceeded" | Increase in `.vrooli/testing.json` |

## Testing Checklist

Before considering a scenario test-ready:

- [ ] `.vrooli/service.json` properly configured with `lifecycle.test.steps` invoking `test-genie execute`
- [ ] Test directory exists for artifacts (playbooks, fixtures, logs)
- [ ] Unit tests with coverage > 70%
- [ ] `[REQ:ID]` tags on tests matching PRD requirements
- [ ] Integration tests for all API endpoints
- [ ] Business logic tests for core workflows
- [ ] Performance baselines established
- [ ] Dynamic port discovery (no hardcoded ports)
- [ ] `make test` works from scenario directory

## See Also

### Phase Documentation
- [Phases Overview](../phases/README.md) - Complete phase reference with mermaid diagrams
- [Structure Phase](../phases/structure/README.md) - File and CLI validation
- [Unit Phase](../phases/unit/README.md) - Test runners and coverage
- [Integration Phase](../phases/integration/README.md) - CLI and API testing
- [Workflow Phase](../phases/workflow/README.md) - BAS browser automation
- [Business Phase](../phases/business/README.md) - Requirements validation
- [Performance Phase](../phases/performance/README.md) - Build benchmarks and Lighthouse

### Related Guides
- [Requirements Sync](../phases/business/requirements-sync.md) - Automatic requirement tracking
- [Scenario Unit Testing](../phases/unit/scenario-unit-testing.md) - Writing effective unit tests
- [Performance Testing](../phases/performance/performance-testing.md) - Build benchmarks and Lighthouse audits
- [Custom Presets](custom-presets.md) - Create tailored presets for CI/CD
- [Dashboard Guide](dashboard-guide.md) - Using the web UI
- [UI Testability](ui-testability.md) - Design testable UIs
- [Sync Execution](sync-execution.md) - API usage for CI/CD
- [Troubleshooting](troubleshooting.md) - Debug common issues

### Reference
- [Presets](../reference/presets.md) - Preset configurations
- [API Endpoints](../reference/api-endpoints.md) - REST API reference
- [Test Runners](../phases/unit/test-runners.md) - Language-specific runners

### Concepts
- [Architecture](../concepts/architecture.md) - Go orchestrator design
