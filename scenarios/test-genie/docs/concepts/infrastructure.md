# Testing Infrastructure

**Status**: Active (Go-Native)
**Last Updated**: 2025-12-02

---

## Overview

This document describes the testing infrastructure components used by Test Genie to execute and manage tests across Vrooli scenarios.

## Architecture

Test Genie's orchestrator is implemented entirely in Go:

```
api/
├── orchestrator/           # Test orchestration engine
│   ├── phases/            # Phase runners (structure, unit, etc.)
│   ├── requirements/      # Requirements sync engine
│   └── workspace.go       # Scenario workspace management
├── execution/             # Execution history and tracking
├── queue/                 # Suite request queue
└── scenarios/             # Scenario catalog and discovery
```

### Key Benefits

- **No external dependencies**: Pure Go implementation, no bash/shell dependencies
- **Typed interfaces**: All phases implement the `PhaseRunner` interface
- **Structured logging**: Consistent error handling and tracing
- **Direct integration**: CLI and API share the same orchestration code
- **Portable**: Works consistently across environments

## Test Directory Structure

### Scenario Test Layout

```
scenario/
├── test/
│   └── playbooks/           # BAS automation workflows
│       ├── capabilities/
│       │   ├── projects/
│       │   │   └── create-project.json
│       │   └── workflows/
│       │       └── execute-workflow.json
│       └── registry.json    # Playbook manifest
├── api/
│   ├── *_test.go            # Go unit tests
│   └── test_helpers.go      # Test utilities
├── ui/src/
│   └── **/*.test.tsx        # Vitest unit tests
└── cli/
    └── *.bats               # CLI BATS tests (if applicable)
```

### Coverage Output

```
coverage/
├── phase-results/           # Phase execution results
│   ├── structure.json
│   ├── dependencies.json
│   ├── unit.json
│   ├── integration.json
│   ├── business.json
│   └── performance.json
├── sync/                    # Requirements sync metadata
│   └── *.json
├── go-coverage.out          # Go coverage data
├── vitest-requirements.json # Vitest requirement evidence
└── lighthouse/              # Lighthouse reports
    └── *.html
```

## Environment Variables

### Standard Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ROOT` | Vrooli project root | Auto-detected |
| `VROOLI_ROOT` | Alternative root | Same as APP_ROOT |
| `TEST_VERBOSE` | Enable verbose output | `0` |
| `TEST_TIMEOUT` | Override phase timeout | Phase default |

### CI/CD Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `CI` | Running in CI environment | `true` |
| `SKIP_RESOURCE_TESTS` | Skip resource integration | `1` |
| `COVERAGE_WARN` | Coverage warning threshold | `70` |
| `COVERAGE_ERROR` | Coverage error threshold | `60` |

## Exit Codes

| Code | Meaning | Action |
|------|---------|--------|
| 0 | Success | Phase passed |
| 1 | Test failure | Phase failed |
| 2 | Setup error | Phase skipped |
| 124 | Timeout | Phase timed out |

## Debugging Tests

### Via CLI

```bash
# Run with verbose output
test-genie execute my-scenario --preset comprehensive --verbose

# Run specific phase
test-genie execute my-scenario --phases unit

# Check phase results
cat coverage/phase-results/unit.json | jq .

# View requirement evidence
cat coverage/phase-results/unit.json | jq '.requirements[] | select(.id == "MY-REQ")'
```

### Via API

```bash
# Start execution
curl -X POST "http://localhost:${API_PORT}/api/v1/executions" \
  -H "Content-Type: application/json" \
  -d '{"scenarioName": "my-scenario", "preset": "comprehensive"}'

# Check execution status
curl "http://localhost:${API_PORT}/api/v1/executions/{id}"
```

## See Also

- [Architecture](architecture.md) - Go orchestrator design details
- [Phase Catalog](../reference/phase-catalog.md) - Phase specifications
- [CLI Commands](../reference/cli-commands.md) - Full CLI reference
- [API Endpoints](../reference/api-endpoints.md) - REST API reference
- [Safety Guidelines](../safety/GUIDELINES.md) - Safe testing patterns
