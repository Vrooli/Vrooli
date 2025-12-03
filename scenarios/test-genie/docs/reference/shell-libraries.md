# Testing Shell Libraries Reference

**Status**: Active
**Last Updated**: 2025-12-02

---

This document provides a complete reference for the shell testing libraries available in `/scripts/scenarios/testing/`.

## Directory Structure

```
scripts/scenarios/testing/
├── shell/                    # Sourceable shell libraries
│   ├── core.sh              # Scenario detection, configuration
│   ├── runner.sh            # Low-level phase execution engine
│   ├── suite.sh             # High-level test orchestrator
│   ├── phase-helpers.sh     # Phase lifecycle management
│   ├── config-helpers.sh    # JSON configuration parsing utilities
│   ├── requirements-sync.sh # Requirements registry synchronization
│   ├── connectivity.sh      # API/UI endpoint testing
│   ├── dependencies.sh      # Dependency validation
│   ├── resources.sh         # Resource integration testing
│   ├── cli.sh               # CLI testing utilities
│   └── orchestration.sh     # Comprehensive test suite execution
├── unit/                     # Language-specific unit test runners
│   ├── run-all.sh           # Universal test runner
│   ├── go.sh                # Go test runner
│   ├── node.sh              # Node.js test runner
│   └── python.sh            # Python test runner
├── playbooks/                # BAS workflow execution
│   └── workflow-runner.sh   # Generic workflow runner
├── templates/                # Copy-and-customize templates
│   ├── README.md            # Template usage guide
│   └── go/                  # Go testing templates
│       ├── test_helpers.go.template
│       └── error_patterns.go.template
└── lint-tests.sh            # Safety linter for test scripts
```

## Quick Start

### Using Shell Libraries

```bash
#!/bin/bash
# Source the modules you need
source "${APP_ROOT}/scripts/scenarios/testing/shell/suite.sh"

# Run comprehensive tests with automatic phase discovery
testing::suite::run
```

### Using Unit Test Runners

```bash
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Run all language tests with coverage thresholds
testing::unit::run_all_tests --coverage-warn 80 --coverage-error 70
```

### Using Templates

```bash
# Copy Go templates to your scenario
cp scripts/scenarios/testing/templates/go/test_helpers.go.template \
   scenarios/my-scenario/api/test_helpers.go

# Customize for your needs (change package, add helpers, etc.)
```

## Available Modules

### core.sh

Core utilities for scenario detection and configuration.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/core.sh"
```

#### Functions

##### `testing::core::detect_scenario()`

Automatically detects the current scenario from directory structure.

```bash
scenario_name=$(testing::core::detect_scenario)
echo "Current scenario: $scenario_name"
```

##### `testing::core::get_scenario_config()`

Reads scenario configuration from `.vrooli/service.json`.

```bash
config=$(testing::core::get_scenario_config "my-scenario")
api_port=$(echo "$config" | jq -r '.api.port')
```

##### `testing::core::detect_languages()`

Detects programming languages used in the scenario.

```bash
languages=$(testing::core::detect_languages)
# Returns: "go node python" (space-separated)
```

##### `testing::core::is_scenario_running()`

Checks if a scenario is currently running.

```bash
if testing::core::is_scenario_running "my-scenario"; then
    echo "Scenario is running"
fi
```

##### `testing::core::wait_for_scenario()`

Waits for scenario to become ready (with timeout).

```bash
testing::core::wait_for_scenario "my-scenario" 30  # 30 second timeout
```

---

### connectivity.sh

API and UI endpoint testing utilities.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/connectivity.sh"
```

#### Functions

##### `testing::connectivity::get_api_url()`

Gets the API URL for a scenario (with dynamic port discovery).

```bash
api_url=$(testing::connectivity::get_api_url "my-scenario")
# Returns: http://localhost:8080 (or actual port)
```

##### `testing::connectivity::get_ui_url()`

Gets the UI URL for a scenario.

```bash
ui_url=$(testing::connectivity::get_ui_url "my-scenario")
# Returns: http://localhost:3000 (or actual port)
```

##### `testing::connectivity::test_api()`

Tests API connectivity and health.

```bash
if testing::connectivity::test_api "my-scenario"; then
    echo "API is healthy"
else
    echo "API is not responding"
fi
```

##### `testing::connectivity::test_ui()`

Tests UI availability.

```bash
testing::connectivity::test_ui "my-scenario"
```

##### `testing::connectivity::test_all()`

Tests all configured endpoints.

```bash
testing::connectivity::test_all "my-scenario"
```

---

### dependencies.sh

Unified dependency validation helper.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/dependencies.sh"
```

#### Functions

##### `testing::dependencies::validate_all()`

Validates all dependencies for a scenario automatically:
- Detects tech stack from service.json and file structure
- Validates language runtimes (Go, Node.js, Python)
- Checks package managers and dependency resolution
- Tests resource health (postgres, redis, ollama, etc.)
- Validates runtime connectivity (if scenario is running)

```bash
testing::dependencies::validate_all \
  --scenario "my-scenario" \
  --enforce-resources \
  --enforce-runtimes
```

**Parameters:**
- `--scenario`: Scenario name (auto-detected if omitted)
- `--enforce-resources`: Fail on unhealthy resources (default: warn only)
- `--enforce-runtimes`: Fail on missing runtimes (default: warn only)

**Example usage in test-dependencies.sh:**

```bash
#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/dependencies.sh"

testing::phase::init --target-time "60s"

testing::dependencies::validate_all \
  --scenario "$TESTING_PHASE_SCENARIO_NAME"

testing::phase::end_with_summary "Dependency validation completed"
```

---

### phase-helpers.sh

Phase lifecycle helpers used across all test phases.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/phase-helpers.sh"
```

#### Functions

##### `testing::phase::init()`

Initializes phase context, loads the scenario requirements registry, and prepares cleanup hooks.

```bash
testing::phase::init --target-time "60s"
testing::phase::init --target-time "120s" --require-runtime
```

**Parameters:**
- `--target-time`: Expected phase duration (for timeout calculation)
- `--require-runtime`: Fail if scenario is not running

##### `testing::phase::add_requirement()`

Associates requirement IDs with pass/fail/skipped outcomes and supporting evidence.

```bash
testing::phase::add_requirement "MY-REQ-ID" "passed" "Test completed successfully"
testing::phase::add_requirement "MY-REQ-ID" "failed" "Assertion failed on line 42"
testing::phase::add_requirement "MY-REQ-ID" "skipped" "Precondition not met"
```

##### `testing::phase::add_error()`

Adds an error to the phase (doesn't fail immediately).

```bash
testing::phase::add_error "Test file not found: foo_test.go"
```

##### `testing::phase::add_warning()`

Adds a warning to the phase.

```bash
testing::phase::add_warning "Coverage below threshold: 68%"
```

##### `testing::phase::run_workflow_yaml()`

Runs YAML-defined automations (with optional timeouts and success patterns).

```bash
testing::phase::run_workflow_yaml "test/workflows/login.yaml" --timeout 30
```

##### `testing::phase::end_with_summary()`

Writes JSON summaries to `coverage/phase-results/<phase>.json` for downstream reporting.

```bash
testing::phase::end_with_summary "Phase completed successfully"
```

---

### runner.sh

Low-level phase execution engine.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/runner.sh"
```

#### Functions

##### `testing::runner::register_phase()`

Register a test phase for execution.

```bash
testing::runner::register_phase "my-custom-phase" "test/my-phase.sh"
```

##### `testing::runner::execute()`

Execute all registered phases in order.

```bash
testing::runner::register_phase "structure" "test/phases/test-structure.sh"
testing::runner::register_phase "unit" "test/phases/test-unit.sh"
testing::runner::execute
```

---

### suite.sh

High-level test suite orchestrator with automatic phase discovery.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/suite.sh"
```

#### Functions

##### `testing::suite::run()`

Main entry point for running complete test suites. Automatically discovers phases and runs with requirements sync.

```bash
# Simplest usage - auto-discovers phases
testing::suite::run

# In a test/run-tests.sh script
#!/bin/bash
source "${APP_ROOT}/scripts/scenarios/testing/shell/suite.sh"
testing::suite::run
```

---

### config-helpers.sh

JSON configuration parsing utilities.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/config-helpers.sh"
```

#### Functions

##### `config::get_boolean()`

Parse boolean values with defaults.

```bash
is_enabled=$(config::get_boolean "$json" ".features.enabled" "false")
```

##### `config::get_string()`

Parse string values with defaults.

```bash
api_url=$(config::get_string "$json" ".api.url" "http://localhost:8080")
```

##### `config::get_number()`

Parse numeric values with defaults.

```bash
timeout=$(config::get_number "$json" ".timeout" "30")
```

##### `config::get_array()`

Extract arrays from JSON.

```bash
phases=$(config::get_array "$json" ".phases")
```

---

### requirements-sync.sh

Requirements registry synchronization.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/requirements-sync.sh"
```

#### Functions

##### `testing::requirements::sync()`

Sync test results with requirements registry.

```bash
testing::requirements::sync --scenario "my-scenario"
```

---

### resources.sh

Resource integration testing.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/resources.sh"
```

#### Functions

##### `testing::resources::test_postgres()`

Test PostgreSQL integration.

```bash
if testing::resources::test_postgres "my-scenario"; then
    echo "PostgreSQL is healthy"
fi
```

##### `testing::resources::test_redis()`

Test Redis integration.

```bash
testing::resources::test_redis "my-scenario"
```

##### `testing::resources::test_ollama()`

Test Ollama integration.

```bash
testing::resources::test_ollama "my-scenario"
```

##### `testing::resources::test_qdrant()`

Test Qdrant integration.

```bash
testing::resources::test_qdrant "my-scenario"
```

##### `testing::resources::test_all()`

Test all configured resources for a scenario.

```bash
testing::resources::test_all "my-scenario"
```

---

### cli.sh

CLI testing utilities.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/cli.sh"
```

#### Functions

##### `testing::cli::test_integration()`

Test CLI binary functionality.

```bash
testing::cli::test_integration "my-scenario"
```

##### `testing::cli::test_command()`

Test specific CLI command.

```bash
testing::cli::test_command "my-scenario" "help"
testing::cli::test_command "my-scenario" "version"
```

##### `testing::cli::test_with_input()`

Test CLI with input/output.

```bash
testing::cli::test_with_input "my-scenario" "create --name test"
```

---

### orchestration.sh

Comprehensive test suite execution.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/orchestration.sh"
```

#### Functions

##### `testing::orchestration::run_unit_tests()`

Run all unit tests.

```bash
testing::orchestration::run_unit_tests
```

##### `testing::orchestration::run_integration_tests()`

Run integration tests only.

```bash
testing::orchestration::run_integration_tests
```

##### `testing::orchestration::run_comprehensive_tests()`

Run full test suite.

```bash
testing::orchestration::run_comprehensive_tests
```

---

## Unit Test Runners

Located in `/scripts/scenarios/testing/unit/`, these runners provide language-specific test execution.

### Universal Runner

```bash
source "$APP_ROOT/scripts/scenarios/testing/unit/run-all.sh"
testing::unit::run_all_tests [options]
```

**Options:**
- `--go-dir PATH` - Go code directory (default: api)
- `--node-dir PATH` - Node.js code directory (default: ui)
- `--python-dir PATH` - Python code directory (default: .)
- `--skip-go` - Skip Go tests
- `--skip-node` - Skip Node.js tests
- `--skip-python` - Skip Python tests
- `--verbose` - Verbose output
- `--fail-fast` - Stop on first failure
- `--coverage-warn PERCENT` - Warning threshold (default: 80)
- `--coverage-error PERCENT` - Error threshold (default: 70)

### Language-Specific Runners

```bash
# Go tests
source "$APP_ROOT/scripts/scenarios/testing/unit/go.sh"
testing::unit::run_go_tests --dir api --coverage-warn 80

# Node.js tests
source "$APP_ROOT/scripts/scenarios/testing/unit/node.sh"
testing::unit::run_node_tests --dir ui --timeout 60000

# Python tests
source "$APP_ROOT/scripts/scenarios/testing/unit/python.sh"
testing::unit::run_python_tests --dir . --framework pytest
```

---

## Templates

Templates are **starting points** for your test helpers. They're meant to be copied and customized, not imported.

### Available Templates

- **go/test_helpers.go.template** - HTTP testing utilities for Go
- **go/error_patterns.go.template** - Sophisticated error testing patterns

### Template Usage

1. **Copy** the template to your scenario
2. **Customize** package names and imports
3. **Adapt** functions to your needs
4. **Delete** unused code
5. **Add** scenario-specific helpers

```bash
# Example: Copy and customize
cp scripts/scenarios/testing/templates/go/test_helpers.go.template \
   scenarios/my-scenario/api/test_helpers.go
# Then edit the file to match your package name and needs
```

---

## Safety

**CRITICAL:** Before using any testing templates or writing test scripts, review the [Safety Guidelines](../safety/GUIDELINES.md) to prevent data loss.

### Quick Safety Checklist

- [ ] BATS `teardown()` functions validate variables before `rm` commands
- [ ] Test files are isolated under `/tmp` directories
- [ ] Wildcard patterns (`*`) are never used with empty variables
- [ ] Use our safe BATS template: `templates/bats/cli-test.bats.template`

### Safety Linter

Run the safety linter on your test scripts:

```bash
# Lint all test scripts in current directory
scripts/scenarios/testing/lint-tests.sh

# Lint specific scenario
scripts/scenarios/testing/lint-tests.sh scenarios/my-scenario/
```

---

## Usage Patterns

### Basic Usage

```bash
#!/bin/bash
set -euo pipefail

# Source what you need
source "$APP_ROOT/scripts/scenarios/testing/shell/connectivity.sh"

# Use the functions
api_url=$(testing::connectivity::get_api_url)
if testing::connectivity::test_api; then
    echo "API is ready at $api_url"
fi
```

### Complete Dependency Validation

```bash
#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)}"
source "$APP_ROOT/scripts/scenarios/testing/shell/dependencies.sh"

# Run comprehensive dependency validation
testing::dependencies::validate_all \
  --scenario "$1" \
  --enforce-resources \
  --enforce-runtimes
```

### Unit Testing with Custom Thresholds

```bash
#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)}"
source "$APP_ROOT/scripts/scenarios/testing/shell/core.sh"
source "$APP_ROOT/scripts/scenarios/testing/unit/run-all.sh"

# Detect languages and build skip flags
languages=$(testing::core::detect_languages)
runner_args=("--coverage-warn" "80" "--coverage-error" "70")

for lang in go node python; do
    if ! echo "$languages" | grep -q "$lang"; then
        runner_args+=("--skip-$lang")
    fi
done

# Run unit tests
testing::unit::run_all_tests "${runner_args[@]}"
```

---

## Environment Variables

The testing libraries respect these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ROOT` | Root directory of Vrooli project | Auto-detected |
| `VROOLI_ROOT` | Alternative root directory variable | Same as APP_ROOT |
| `TEST_VERBOSE` | Enable verbose output (1=enabled) | 0 |
| `TEST_TIMEOUT` | Override default timeouts (seconds) | Phase default |
| `SKIP_RESOURCE_TESTS` | Skip resource integration tests | 0 |
| `COVERAGE_WARN` | Default coverage warning threshold | 70 |
| `COVERAGE_ERROR` | Default coverage error threshold | 60 |

---

## Error Handling

All functions follow consistent error handling:

```bash
# Functions return 0 on success, non-zero on failure
if ! testing::connectivity::test_api; then
    echo "API test failed"
    exit 1
fi

# Most functions output errors to stderr
testing::dependencies::validate_all 2>&1 | tee test.log
```

---

## Debugging

Enable verbose output for debugging:

```bash
# Set verbose mode
export TEST_VERBOSE=1

# Or use bash debugging
bash -x test-script.sh
```

---

## See Also

- [Testing Infrastructure](../concepts/infrastructure.md) - Architecture overview
- [CLI Testing Guide](../guides/cli-testing.md) - BATS testing patterns
- [Phase Catalog](phase-catalog.md) - Phase specifications
- [Test Runners](test-runners.md) - Language-specific runners
