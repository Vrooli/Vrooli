# Test Genie CLI Reference

> **Status**: Placeholder - To be populated with complete CLI documentation

## Overview

The `test-genie` CLI provides command-line access to all test management operations. Commands are designed for both interactive use and CI/CD scripting.

## Installation

```bash
cd scenarios/test-genie/cli
./install.sh
```

Or use directly:
```bash
./cli/test-genie [command]
```

## Global Options

| Option | Description |
|--------|-------------|
| `--help`, `-h` | Show help for any command |
| `--verbose`, `-v` | Enable verbose output |
| `--json` | Output in JSON format (scriptable) |
| `--timeout` | Override default timeout |

## Commands

### Generate Tests

Generate test suites for a scenario.

```bash
test-genie generate <scenario-name> [options]
```

**Options:**
| Option | Default | Description |
|--------|---------|-------------|
| `--types` | `unit` | Test types: unit, integration, performance, vault |
| `--coverage` | `90` | Target coverage percentage |
| `--parallel` | `false` | Enable parallel generation |
| `--dry-run` | `false` | Preview without executing |

**Examples:**
```bash
# Generate unit tests
test-genie generate my-scenario --types unit

# Generate comprehensive suite
test-genie generate my-scenario --types unit,integration,performance --coverage 95

# Dry run to preview
test-genie generate my-scenario --types unit --dry-run
```

### Execute Tests

Execute a test suite.

```bash
test-genie execute <scenario-name|suite-id> [options]
```

**Options:**
| Option | Default | Description |
|--------|---------|-------------|
| `--preset` | `comprehensive` | Preset: quick, smoke, comprehensive |
| `--fail-fast` | `false` | Stop on first failure |
| `--watch` | `false` | Live progress monitoring |
| `--request-id` | - | Link to queued suite request |

**Examples:**
```bash
# Run comprehensive tests
test-genie execute my-scenario --preset comprehensive

# Quick smoke test
test-genie execute my-scenario --preset quick

# Fail fast for CI
test-genie execute my-scenario --preset smoke --fail-fast
```

### Coverage Analysis

Analyze test coverage for a scenario.

```bash
test-genie coverage <scenario-name> [options]
```

**Options:**
| Option | Default | Description |
|--------|---------|-------------|
| `--depth` | `standard` | Analysis depth: quick, standard, deep, comprehensive |
| `--threshold` | `80` | Minimum coverage threshold |
| `--report` | - | Output file for report |
| `--show-gaps` | `false` | Show only uncovered areas |

**Examples:**
```bash
# Standard coverage check
test-genie coverage my-scenario

# Deep analysis with report
test-genie coverage my-scenario --depth deep --report coverage.json

# Show only gaps
test-genie coverage my-scenario --show-gaps
```

### Vault Testing

Create and run vault tests.

```bash
test-genie vault <scenario-name> [options]
```

**Options:**
| Option | Default | Description |
|--------|---------|-------------|
| `--phases` | all | Phases to include |
| `--criteria` | - | Path to success criteria file |
| `--timeout` | `1800` | Per-phase timeout |

**Examples:**
```bash
# Run all vault phases
test-genie vault my-scenario

# Specific phases
test-genie vault my-scenario --phases setup,develop,test

# With custom criteria
test-genie vault my-scenario --criteria ./vault-criteria.yaml
```

### Status

Check test-genie operational status.

```bash
test-genie status [options]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--resources` | Include resource status |
| `--queue` | Show queue depth and pending |
| `--executions` | Show recent executions |

**Examples:**
```bash
# Full status
test-genie status

# Queue status only
test-genie status --queue
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Test failures |
| 2 | Configuration error |
| 3 | Timeout |
| 4 | Infrastructure error |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `TEST_GENIE_API_URL` | API endpoint |
| `TEST_GENIE_DEBUG` | Enable debug output |
| `TEST_GENIE_LOG_LEVEL` | Log verbosity |

## See Also

- [API Endpoints](api-endpoints.md) - REST API reference
- [Presets](presets.md) - Preset definitions
- [Sync Execution Cheatsheet](sync-execution-cheatsheet.md) - Quick reference
