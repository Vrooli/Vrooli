# Testing Shell Libraries Reference

This document provides a complete reference for the shell testing libraries available in `/scripts/scenarios/testing/shell/`.

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

### resources.sh
Resource integration testing utilities.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/resources.sh"
```

#### Functions

##### `testing::resources::test_postgres()`
Tests PostgreSQL connectivity and basic operations.

```bash
testing::resources::test_postgres "my-scenario"
```

##### `testing::resources::test_redis()`
Tests Redis connectivity and basic operations.

```bash
testing::resources::test_redis "my-scenario"
```

##### `testing::resources::test_ollama()`
Tests Ollama availability and model loading.

```bash
testing::resources::test_ollama "my-scenario"
```

##### `testing::resources::test_n8n()`
Tests N8n workflow automation integration.

```bash
testing::resources::test_n8n "my-scenario"
```

##### `testing::resources::test_qdrant()`
Tests Qdrant vector database integration.

```bash
testing::resources::test_qdrant "my-scenario"
```

##### `testing::resources::test_all()`
Tests all configured resources for a scenario.

```bash
testing::resources::test_all "my-scenario"
```

### cli.sh
CLI testing utilities.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/cli.sh"
```

#### Functions

##### `testing::cli::test_integration()`
Tests CLI binary installation and basic functionality.

```bash
testing::cli::test_integration "my-cli"
```

##### `testing::cli::test_command()`
Tests a specific CLI command.

```bash
testing::cli::test_command "my-cli" "list --json"
```

##### `testing::cli::test_with_input()`
Tests CLI with input/output validation.

```bash
testing::cli::test_with_input "my-cli" "process" "input.txt" "expected output"
```

### phase-helpers.sh
Phase lifecycle helpers used across all test phases.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/phase-helpers.sh"
```

#### Functions

##### `testing::phase::init()`
Initialises phase context, loads the scenario requirements registry (`docs/requirements.json` or `requirements/`), and prepares cleanup hooks.

##### `testing::phase::add_requirement()`
Associates requirement IDs with pass/fail/skipped outcomes and supporting evidence.

##### `testing::phase::run_workflow_yaml()`
Runs YAML-defined automations (with optional timeouts and success patterns) and updates requirement status automatically.

##### `testing::phase::end_with_summary()`
Writes JSON summaries to `coverage/phase-results/<phase>.json` for downstream reporting.

### orchestration.sh
Comprehensive test suite execution.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/orchestration.sh"
```

#### Functions

##### `testing::orchestration::run_unit_tests()`
Runs all unit tests for detected languages.

```bash
testing::orchestration::run_unit_tests "my-scenario" 80 70
# Args: scenario_name, coverage_warn%, coverage_error%
```

##### `testing::orchestration::run_integration_tests()`
Runs integration tests only.

```bash
testing::orchestration::run_integration_tests "my-scenario"
```

##### `testing::orchestration::run_comprehensive_tests()`
Runs the full test suite (all phases).

```bash
testing::orchestration::run_comprehensive_tests "my-scenario"
```

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

### Complete Test Script

```bash
#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)}"
source "$APP_ROOT/scripts/scenarios/testing/shell/orchestration.sh"

# Run everything
testing::orchestration::run_comprehensive_tests "$1"
```

### Selective Testing

```bash
#!/bin/bash
# Test only specific resources

source "$APP_ROOT/scripts/scenarios/testing/shell/resources.sh"

# Test only databases
testing::resources::test_postgres "my-scenario"
testing::resources::test_redis "my-scenario"
```

### Custom Coverage Thresholds

```bash
#!/bin/bash
# Strict coverage requirements

source "$APP_ROOT/scripts/scenarios/testing/shell/orchestration.sh"

# 80% warning, 70% error thresholds (standard)
testing::orchestration::run_unit_tests "my-scenario" 80 70
```

## Environment Variables

The testing libraries respect these environment variables:

- `APP_ROOT` - Root directory of Vrooli project
- `VROOLI_ROOT` - Alternative root directory variable
- `TEST_VERBOSE` - Enable verbose output (1=enabled)
- `TEST_TIMEOUT` - Override default timeouts (seconds)
- `SKIP_RESOURCE_TESTS` - Skip resource integration tests
- `COVERAGE_WARN` - Default coverage warning threshold
- `COVERAGE_ERROR` - Default coverage error threshold

## Error Handling

All functions follow consistent error handling:

```bash
# Functions return 0 on success, non-zero on failure
if ! testing::connectivity::test_api; then
    echo "API test failed"
    exit 1
fi

# Most functions output errors to stderr
testing::resources::test_postgres 2>&1 | tee test.log
```

## Debugging

Enable verbose output for debugging:

```bash
# Set verbose mode
export TEST_VERBOSE=1

# Or use bash debugging
bash -x test-script.sh
```

## See Also

- [Testing Architecture](../architecture/PHASED_TESTING.md)
- [Shell Library Implementation](/scripts/scenarios/testing/shell/)
- [Unit Test Runners](test-runners.md)
