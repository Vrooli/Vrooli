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

### dependencies.sh
Unified dependency validation helper. Validates runtimes, package managers, resources, and connectivity.

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

All powered by `vrooli scenario status --json` with fallbacks to file detection.

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

### phase-helpers.sh
Phase lifecycle helpers used across all test phases.

```bash
source "$APP_ROOT/scripts/scenarios/testing/shell/phase-helpers.sh"
```

#### Functions

##### `testing::phase::init()`
Initialises phase context, loads the scenario requirements registry from `requirements/`, and prepares cleanup hooks.

##### `testing::phase::add_requirement()`
Associates requirement IDs with pass/fail/skipped outcomes and supporting evidence.

##### `testing::phase::run_workflow_yaml()`
Runs YAML-defined automations (with optional timeouts and success patterns) and updates requirement status automatically.

##### `testing::phase::end_with_summary()`
Writes JSON summaries to `coverage/phase-results/<phase>.json` for downstream reporting.

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
testing::dependencies::validate_all 2>&1 | tee test.log
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
