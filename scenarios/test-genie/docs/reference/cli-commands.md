# Test Genie CLI Reference

Complete reference for the `test-genie` command-line interface. Commands are designed for both interactive use and CI/CD scripting.

## Installation

```bash
# Install globally (recommended)
cd scenarios/test-genie/cli
./install.sh

# Or use directly without installation
./cli/test-genie [command]

# Verify installation
test-genie --version
```

## Global Options

These options apply to all commands:

| Option | Short | Description |
|--------|-------|-------------|
| `--help` | `-h` | Show help for any command |
| `--verbose` | `-v` | Enable verbose output |
| `--json` | | Output in JSON format (scriptable) |
| `--timeout <seconds>` | `-t` | Override default timeout |
| `--quiet` | `-q` | Suppress non-essential output |

## Commands Overview

| Command | Description |
|---------|-------------|
| `generate` | Generate test suites for a scenario |
| `execute` | Execute a test suite |
| `coverage` | Analyze test coverage |
| `vault` | Create and run vault (lifecycle) tests |
| `status` | Check test-genie operational status |
| `scenarios` | List and inspect scenarios |
| `phases` | List available test phases |
| `history` | View execution history |

---

## generate

Generate test suites for a scenario. Tests are created based on code analysis, PRD, and requirements.

```bash
test-genie generate <scenario-name> [options]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--types <types>` | `unit` | Comma-separated test types |
| `--coverage <percent>` | `90` | Target coverage percentage (1-100) |
| `--parallel` | `false` | Enable parallel generation |
| `--dry-run` | `false` | Preview without executing |
| `--priority <level>` | `normal` | Queue priority |
| `--output <dir>` | auto | Output directory for generated tests |
| `--fallback` | `false` | Use local templates instead of delegation |

### Test Types

| Type | Description |
|------|-------------|
| `unit` | Unit tests for functions/methods |
| `integration` | Integration tests for component interaction |
| `performance` | Performance benchmarks |
| `vault` | Lifecycle validation tests |
| `regression` | Regression test suites |

### Priority Levels

| Priority | Description |
|----------|-------------|
| `low` | Background processing |
| `normal` | Standard queue position |
| `high` | Prioritized processing |
| `urgent` | Immediate processing |

### Examples

```bash
# Generate unit tests (default)
test-genie generate my-scenario

# Generate unit and integration tests
test-genie generate my-scenario --types unit,integration

# Comprehensive suite with high coverage target
test-genie generate my-scenario --types unit,integration,performance --coverage 95

# Dry run to preview
test-genie generate my-scenario --types unit --dry-run

# High priority generation
test-genie generate my-scenario --types unit --priority high

# Use local templates (fallback mode)
test-genie generate my-scenario --types unit --fallback
```

### Output

```
Analyzing scenario: my-scenario
  Found 12 requirements in requirements/*.json
  Identified coverage gaps: 8 untested functions
  Delegated generation to App Issue Tracker
    Issue ID: #12345
    Estimated completion: 15-30 minutes

Track progress:
  vrooli issue status 12345

View generated tests:
  git status  # After issue completion
```

---

## execute

Execute a test suite for a scenario. Supports presets for common configurations.

```bash
test-genie execute <scenario-name> [options]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--preset <name>` | `comprehensive` | Preset configuration |
| `--phases <phases>` | all | Comma-separated phases to run |
| `--skip <phases>` | none | Phases to skip |
| `--fail-fast` | `false` | Stop on first failure |
| `--watch` | `false` | Live progress monitoring |
| `--request-id <uuid>` | | Link to queued suite request |
| `--timeout <seconds>` | preset | Override phase timeout |
| `--sync` | `false` | Trigger requirements sync after execution |

### Presets

| Preset | Phases | Timeout | Use Case |
|--------|--------|---------|----------|
| `quick` | structure, dependencies | ~45s | Fast iteration |
| `smoke` | structure, dependencies, unit | ~2min | Pre-commit check |
| `comprehensive` | All 7 phases | ~10min | CI/CD, releases |

### Phase Names

| Phase | Description |
|-------|-------------|
| `structure` | Validates scenario layout and manifests |
| `dependencies` | Confirms required commands/runtimes |
| `unit` | Executes Go unit tests |
| `integration` | Runs CLI/BATS tests |
| `e2e` | Executes BAS browser automation workflows |
| `business` | Audits requirements modules |
| `performance` | Builds API and checks duration budgets (optional) |

### Examples

```bash
# Run comprehensive tests (default)
test-genie execute my-scenario

# Quick smoke test
test-genie execute my-scenario --preset quick

# Specific preset
test-genie execute my-scenario --preset smoke

# Fail fast for CI
test-genie execute my-scenario --preset smoke --fail-fast

# Run specific phases
test-genie execute my-scenario --phases structure,dependencies,unit

# Skip optional phase
test-genie execute my-scenario --skip performance

# Live monitoring
test-genie execute my-scenario --preset comprehensive --watch

# Link to suite request
test-genie execute my-scenario --request-id 550e8400-e29b-41d4-a716-446655440000

# Trigger requirements sync
test-genie execute my-scenario --preset comprehensive --sync
```

### Output

```
Executing test suite for: my-scenario
  Preset: comprehensive
  Phases: structure, dependencies, unit, integration, e2e, business, performance

[1/7] structure     PASSED  (5s)
[2/7] dependencies  PASSED  (12s)
[3/7] unit          PASSED  (45s)  Coverage: 87%
[4/7] integration   PASSED  (30s)
[5/7] e2e           SKIPPED (no e2e workflows found)
[6/7] business      PASSED  (8s)
[7/7] performance   PASSED  (20s)

Summary:
  Status: PASSED
  Duration: 2m 0s
  Phases: 6 passed, 1 skipped
  Execution ID: 660e8400-e29b-41d4-a716-446655440001

Log: /tmp/test-genie/logs/my-scenario-comprehensive.log
```

---

## coverage

Analyze test coverage for a scenario. Identifies gaps and suggests improvements.

```bash
test-genie coverage <scenario-name> [options]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--depth <level>` | `standard` | Analysis depth |
| `--threshold <percent>` | `80` | Minimum coverage threshold |
| `--report <file>` | | Output file for report |
| `--format <type>` | `text` | Output format |
| `--show-gaps` | `false` | Show only uncovered areas |
| `--by-requirement` | `false` | Group by requirement |

### Depth Levels

| Depth | Description | Time |
|-------|-------------|------|
| `quick` | Basic line coverage | ~10s |
| `standard` | Line + branch coverage | ~30s |
| `deep` | Full analysis + gap identification | ~60s |
| `comprehensive` | Deep + requirement mapping | ~120s |

### Output Formats

| Format | Description |
|--------|-------------|
| `text` | Human-readable summary |
| `json` | Machine-readable JSON |
| `markdown` | Markdown report |
| `html` | Interactive HTML report |

### Examples

```bash
# Standard coverage check
test-genie coverage my-scenario

# Deep analysis with report
test-genie coverage my-scenario --depth deep --report coverage.json

# Show only gaps
test-genie coverage my-scenario --show-gaps

# Coverage by requirement
test-genie coverage my-scenario --by-requirement

# Fail if below threshold
test-genie coverage my-scenario --threshold 90

# Markdown report
test-genie coverage my-scenario --format markdown --report coverage.md
```

### Output

```
Coverage Report: my-scenario
==============================

Overall Coverage: 87.3%
  Go API:     89.2% (target: 80%)  PASSED
  Node UI:    85.1% (target: 80%)  PASSED
  CLI:        82.4% (target: 70%)  PASSED

Requirement Coverage: 20/25 (80%)
  P0:  8/8  (100%)  PASSED
  P1: 10/12 ( 83%)  PASSED
  P2:  2/5  ( 40%)  OK

Uncovered Areas:
  api/handlers/projects.go:45-62      ProjectDelete handler
  api/handlers/workflows.go:120-145   WorkflowExecute error paths
  ui/src/stores/authStore.ts:89-102   Token refresh logic

Recommendations:
  1. Add tests for ProjectDelete handler
  2. Cover WorkflowExecute error scenarios
  3. Test token refresh edge cases
```

---

## vault

Create and run vault tests that validate scenarios through their complete lifecycle.

```bash
test-genie vault <scenario-name> [options]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--phases <phases>` | all | Phases to include |
| `--criteria <file>` | | Path to success criteria YAML |
| `--timeout <seconds>` | `1800` | Per-phase timeout |
| `--continue-on-failure` | `false` | Don't abort on phase failure |
| `--report <file>` | | Output report file |

### Vault Phases

| Phase | Description |
|-------|-------------|
| `setup` | Initialize environment and resources |
| `develop` | Build and configure application |
| `test` | Run test suites |
| `deploy` | Deploy to target environment |
| `monitor` | Validate operational health |

### Examples

```bash
# Run all vault phases
test-genie vault my-scenario

# Specific phases
test-genie vault my-scenario --phases setup,develop,test

# With custom criteria
test-genie vault my-scenario --criteria ./vault-criteria.yaml

# Extended timeout
test-genie vault my-scenario --timeout 3600

# Generate report
test-genie vault my-scenario --report vault-report.json
```

### Success Criteria Format

```yaml
# vault-criteria.yaml
name: my-scenario-vault
phases:
  setup:
    timeout: 300
    requirements:
      - resource_availability: true
      - database_schema: loaded

  develop:
    timeout: 600
    requirements:
      - api_endpoints: functional
      - business_logic: implemented

  test:
    timeout: 900
    requirements:
      - unit_tests: passing
      - integration_tests: passing
      - coverage_threshold: 80
```

### Output

```
Vault Execution: my-scenario
=============================

[1/5] setup       PASSED  (45s)
  Resources initialized
  Database schema loaded

[2/5] develop     PASSED  (120s)
  API built successfully
  Endpoints verified

[3/5] test        PASSED  (180s)
  Unit tests: 45 passed
  Integration: 12 passed
  Coverage: 87%

[4/5] deploy      PASSED  (90s)
  Deployed to staging
  Health check passed

[5/5] monitor     PASSED  (60s)
  Latency: 45ms (target: <100ms)
  Error rate: 0% (target: <1%)

Vault Result: PASSED
Total Duration: 8m 15s
```

---

## status

Check test-genie operational status and service health.

```bash
test-genie status [options]
```

### Options

| Option | Description |
|--------|-------------|
| `--resources` | Include resource status |
| `--queue` | Show queue depth and pending |
| `--executions` | Show recent executions |
| `--all` | Show all status information |

### Examples

```bash
# Basic status
test-genie status

# Full status
test-genie status --all

# Queue status only
test-genie status --queue

# Recent executions
test-genie status --executions
```

### Output

```
Test Genie Status
=================

Service: healthy
Version: 1.0.0
Uptime: 4h 32m

Database: connected
API: http://localhost:8080

Queue:
  Total: 15
  Pending: 3
  Running: 1
  Completed: 10
  Failed: 1

Recent Executions:
  my-scenario        PASSED  2m ago
  another-scenario   PASSED  15m ago
  test-scenario      FAILED  1h ago
```

---

## scenarios

List and inspect available scenarios.

```bash
test-genie scenarios [subcommand] [options]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `list` | List all scenarios (default) |
| `show <name>` | Show scenario details |
| `test <name>` | Run tests for scenario |

### Examples

```bash
# List all scenarios
test-genie scenarios
test-genie scenarios list

# Show scenario details
test-genie scenarios show my-scenario

# Run scenario tests
test-genie scenarios test my-scenario --type unit
```

### Output (list)

```
Scenarios (25 total)
====================

Name                    Tests   Last Run     Status
----                    -----   --------     ------
my-scenario             Yes     2h ago       passed
another-scenario        Yes     1d ago       passed
new-scenario            No      never        -
browser-automation      Yes     30m ago      failed
```

---

## phases

List available test phases and their configuration.

```bash
test-genie phases [options]
```

### Options

| Option | Description |
|--------|-------------|
| `--detailed` | Show full descriptions |
| `--json` | Output as JSON |

### Examples

```bash
# List phases
test-genie phases

# Detailed view
test-genie phases --detailed
```

### Output

```
Test Phases
===========

Name          Optional  Timeout  Description
----          --------  -------  -----------
structure     No        15m      Validates scenario layout
dependencies  No        15m      Confirms dependencies
unit          No        15m      Go unit tests
integration   No        15m      CLI/BATS tests
e2e           No        15m      BAS browser automation workflows
business      No        15m      Requirements audit
performance   Yes       15m      Duration budgets
```

---

## history

View execution history.

```bash
test-genie history [options]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--scenario <name>` | | Filter by scenario |
| `--limit <n>` | `10` | Number of results |
| `--status <status>` | | Filter by status |
| `--since <duration>` | | Filter by time (e.g., "24h", "7d") |

### Examples

```bash
# Recent history
test-genie history

# Scenario-specific
test-genie history --scenario my-scenario

# Failed executions
test-genie history --status failed

# Last 24 hours
test-genie history --since 24h

# Combined filters
test-genie history --scenario my-scenario --status passed --limit 5
```

### Output

```
Execution History
=================

ID        Scenario          Preset         Status   Duration  Time
--        --------          ------         ------   --------  ----
a1b2c3    my-scenario       comprehensive  passed   2m 15s    2h ago
d4e5f6    my-scenario       quick          passed   45s       4h ago
g7h8i9    another-scenario  smoke          failed   1m 30s    6h ago
j0k1l2    my-scenario       comprehensive  passed   2m 30s    1d ago
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Test failures |
| 2 | Configuration error |
| 3 | Timeout |
| 4 | Infrastructure error |
| 5 | Validation error |

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TEST_GENIE_API_URL` | API endpoint | `http://localhost:8080` |
| `TEST_GENIE_DEBUG` | Enable debug output | `false` |
| `TEST_GENIE_LOG_LEVEL` | Log verbosity | `info` |
| `TEST_GENIE_TIMEOUT` | Default timeout (seconds) | `900` |
| `TEST_GENIE_CONFIG` | Config file path | `~/.test-genie/config.yaml` |

---

## Configuration File

Create `~/.test-genie/config.yaml` for persistent settings:

```yaml
# Default settings
api_url: http://localhost:8080
timeout: 900
log_level: info

# Default options for commands
defaults:
  generate:
    types: ["unit", "integration"]
    coverage: 90
  execute:
    preset: comprehensive
    fail_fast: false
  coverage:
    threshold: 80
    depth: standard
```

---

## Scripting Examples

### CI/CD Pipeline

```bash
#!/bin/bash
set -e

# Quick validation
test-genie execute my-scenario --preset quick --fail-fast

# Full test suite
test-genie execute my-scenario --preset comprehensive --json > results.json

# Check exit code
if [ $? -eq 0 ]; then
    echo "Tests passed"
else
    echo "Tests failed"
    exit 1
fi
```

### Coverage Gate

```bash
#!/bin/bash

# Check coverage meets threshold
test-genie coverage my-scenario --threshold 80 --json > coverage.json

if [ $? -ne 0 ]; then
    echo "Coverage below threshold"
    test-genie coverage my-scenario --show-gaps
    exit 1
fi
```

### Parallel Execution

```bash
#!/bin/bash

# Run tests for multiple scenarios in parallel
scenarios=("scenario-a" "scenario-b" "scenario-c")

for scenario in "${scenarios[@]}"; do
    test-genie execute "$scenario" --preset smoke &
done

wait

echo "All scenarios tested"
```

---

## See Also

- [API Endpoints](api-endpoints.md) - REST API reference
- [Presets](presets.md) - Preset definitions
- [Sync Execution Cheatsheet](sync-execution-cheatsheet.md) - Quick reference
- [Troubleshooting](../guides/troubleshooting.md) - Debug common issues
