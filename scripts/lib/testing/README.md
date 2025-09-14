# Centralized Testing Library

This directory contains shared testing utilities for Vrooli scenarios and resources.

## Directory Structure

```
testing/
├── unit/           # Language-specific unit test runners
│   ├── go.sh       # Go test runner
│   ├── node.sh     # Node.js test runner
│   ├── python.sh   # Python test runner
│   └── run-all.sh  # Universal runner that runs all languages
├── phases/         # Shared phase templates (future)
├── common/         # Common test utilities (future)
└── legacy/         # Support for legacy test formats
    └── run-scenario-tests.sh  # Universal scenario test runner
```

## Unit Test Runners

The unit test runners provide a standardized way to run tests across different languages.

### Using in Your Scenario

#### Method 1: Direct Sourcing (Recommended)

In your scenario's test phase script:

```bash
#!/bin/bash
set -euo pipefail

# Get paths
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"
TESTING_LIB="$APP_ROOT/scripts/lib/testing/unit"

# Source the testing library
source "$TESTING_LIB/run-all.sh"

# Run tests
cd "$SCENARIO_DIR"
testing::unit::run_all_tests --go-dir "api" --node-dir "ui" --skip-python
```

#### Method 2: Individual Language Runners

```bash
# Source individual runners
source "$TESTING_LIB/go.sh"
source "$TESTING_LIB/node.sh"

# Run specific language tests
testing::unit::run_go_tests --dir "api" --verbose
testing::unit::run_node_tests --dir "ui" --timeout 60000
```

### Test Coverage Thresholds

The centralized testing library includes configurable coverage thresholds to ensure code quality:

- **Warning Threshold (default: 80%)**: Shows a warning when coverage is below this level, but tests still pass
- **Error Threshold (default: 50%)**: Fails tests when coverage is below this level, indicating insufficient coverage

### Coverage Threshold Behavior

- **Above Warning Threshold**: ✅ Tests pass with good coverage message
- **Below Warning, Above Error**: ⚠️  Tests pass but show warning about low coverage  
- **Below Error Threshold**: ❌ Tests fail with error message about insufficient coverage

### Example Output

```
📊 Go Test Coverage Summary:
total:                                  (statements)            65.2%

⚠️  WARNING: Go test coverage (65.2%) is below warning threshold (80%)
   Consider adding more tests to improve code coverage.
```

## Available Functions

#### `testing::unit::run_all_tests`

Runs unit tests for all detected languages.

**Options:**
- `--go-dir PATH` - Directory for Go tests (default: api)
- `--node-dir PATH` - Directory for Node.js tests (default: ui)
- `--python-dir PATH` - Directory for Python tests (default: .)
- `--skip-go` - Skip Go tests
- `--skip-node` - Skip Node.js tests
- `--skip-python` - Skip Python tests
- `--verbose` - Verbose output for all tests
- `--fail-fast` - Stop on first test failure
- `--coverage-warn PERCENT` - Coverage warning threshold (default: 80)
- `--coverage-error PERCENT` - Coverage error threshold (default: 50)

#### `testing::unit::run_go_tests`

Runs Go unit tests.

**Options:**
- `--dir PATH` - Directory containing Go code (default: api)
- `--timeout SEC` - Test timeout in seconds (default: 30)
- `--no-coverage` - Skip coverage report
- `--verbose` - Verbose test output
- `--coverage-warn PERCENT` - Coverage warning threshold (default: 80)
- `--coverage-error PERCENT` - Coverage error threshold (default: 50)

#### `testing::unit::run_node_tests`

Runs Node.js unit tests.

**Options:**
- `--dir PATH` - Directory containing Node.js code (default: ui)
- `--timeout MS` - Test timeout in milliseconds (default: 30000)
- `--test-cmd CMD` - Custom test command (default: reads from package.json)
- `--verbose` - Verbose test output
- `--coverage-warn PERCENT` - Coverage warning threshold (default: 80)
- `--coverage-error PERCENT` - Coverage error threshold (default: 50)

#### `testing::unit::run_python_tests`

Runs Python unit tests.

**Options:**
- `--dir PATH` - Directory containing Python code (default: .)
- `--timeout SEC` - Test timeout in seconds (default: 30)
- `--framework FW` - Test framework: pytest, unittest, nose (default: auto-detect)
- `--verbose` - Verbose test output

## Legacy Support

The `legacy/run-scenario-tests.sh` script provides backward compatibility for scenarios using the old test format.

It automatically detects and runs:
1. New phased testing format (test/run-tests.sh)
2. V2 lifecycle testing (service.json lifecycle.test)
3. Legacy format (scenario-test.yaml)

This ensures all scenarios work during the migration period.

## Migration Guide

### From Individual Test Scripts

If your scenario has individual test scripts in `test/unit/`:

1. Remove the individual language scripts (go.sh, node.sh, python.sh)
2. Update your test phase script to source the centralized library
3. Use `testing::unit::run_all_tests` or individual language functions

### From Legacy scenario-test.yaml

1. Create a `test/` directory structure
2. Add phase scripts in `test/phases/`
3. Update `.vrooli/service.json` to include a test lifecycle event
4. Remove the old `scenario-test.yaml`

See `docs/scenarios/PHASED_TESTING_ARCHITECTURE.md` for detailed migration instructions.