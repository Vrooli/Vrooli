# Vrooli Resource Test Framework

## 🏗️ Architecture Overview

The Vrooli Resource Test Framework is a comprehensive testing ecosystem designed to validate resource integrations without mocking dependencies. It provides real-world testing against actual services while maintaining isolation and reproducibility.

```
┌─────────────────────────────────────────────────────────────┐
│                    Test Runner (run.sh)                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │  Discovery   │  │   Runner     │  │  Reporter    │    │
│  │              │  │              │  │              │    │
│  │ • Resources  │  │ • Execute    │  │ • Text       │    │
│  │ • Scenarios  │  │ • Isolate    │  │ • JSON       │    │
│  │ • Health     │  │ • Timeout    │  │ • Insights   │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
│                                                             │
│  ┌──────────────────────────────────────────────────┐     │
│  │            Helper Utilities                       │     │
│  │                                                   │     │
│  │  assertions  cleanup  logging  health-checker    │     │
│  │  metadata    config   http-logger  validation    │     │
│  └──────────────────────────────────────────────────┘     │
│                                                             │
│  ┌──────────────────────────────────────────────────┐     │
│  │         Interface Compliance System               │     │
│  │                                                   │     │
│  │  Validates standard resource interfaces           │     │
│  └──────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

## 📁 Directory Structure

```
framework/
├── discovery.sh              # Resource discovery and health validation
├── runner.sh                 # Test execution engine
├── reporter.sh               # Report generation (text/JSON)
├── interface-compliance.sh   # Interface validation system
├── config/
│   ├── health-checks.conf    # Health check timeouts and retries
│   └── timeouts.conf         # Test execution timeouts
└── helpers/
    ├── assertions.sh         # Test assertion library
    ├── cleanup.sh            # Test cleanup utilities
    ├── config-manager.sh     # Configuration management
    ├── debug-enhanced.sh     # Advanced debugging features
    ├── fixtures.sh           # Test data generation
    ├── health-checker.sh     # Standardized health checking
    ├── http-logger.sh        # HTTP request/response logging
    ├── logging.sh            # Unified logging functions
    ├── metadata.sh           # Test metadata extraction
    ├── secure-config.sh      # Secure configuration handling
    └── test-validator.sh     # Test file validation
```

## 🔄 Test Execution Flow

### Phase 1: Discovery
```bash
discover_all_resources() → filter_enabled_resources() → validate_resource_health()
```
- Discovers available resources via `index.sh --action discover`
- Filters by configuration in `~/.vrooli/resources.local.json`
- Validates health status of each resource

### Phase 2: Validation (Optional)
```bash
run_test_validation() → test-validator.sh
```
- Validates test files for compliance
- Checks required patterns and imports

### Phase 2b: Interface Compliance (Mandatory)
```bash
run_interface_compliance_validation() → test_resource_interface_compliance()
```
- Validates all resource manage.sh scripts
- Ensures standard interface implementation
- Can be skipped with `--skip-interface`

### Phase 3: Test Execution
```bash
run_single_resource_tests() → execute_test_file() → isolated environment
```
- Executes individual resource tests
- Creates isolated test environments
- Streams output in real-time

### Phase 4: Business Scenarios
```bash
run_scenario_tests() → execute_scenario_test()
```
- Runs multi-resource integration tests
- Validates business use cases

### Phase 5: Reporting
```bash
generate_final_report() → text/JSON output
```
- Generates comprehensive reports
- Provides actionable recommendations

## 🛠️ Core Components

### Discovery System (`discovery.sh`)
Handles resource discovery, configuration validation, and health checking.

**Key Functions:**
- `discover_all_resources()` - Finds all available resources
- `filter_enabled_resources()` - Checks configuration
- `validate_resource_health()` - Tests resource health
- `check_resource_health()` - Resource-specific health checks
- `discover_scenarios()` - Finds business scenario tests

### Test Runner (`runner.sh`)
Executes tests with isolation, timeout management, and real-time output.

**Key Functions:**
- `run_single_resource_tests()` - Individual resource tests
- `execute_test_file()` - Test execution with isolation
- `run_scenario_tests()` - Business scenario execution
- `setup_test_environment()` - Creates isolated directories

### Reporter (`reporter.sh`)
Generates comprehensive reports in multiple formats.

**Key Functions:**
- `generate_text_report()` - Human-readable output
- `generate_json_report()` - Machine-readable JSON
- `generate_recommendations()` - Actionable insights

### Interface Compliance (`interface-compliance.sh`)
Validates resource scripts follow the standard interface.

**Required Actions:**
- `install`, `start`, `stop`, `status`, `logs`

**Required Arguments:**
- `--help`, `-h`, `--version`
- `--action <action>`
- `--yes` (recommended)

## 🎯 Writing Tests

### Single Resource Test Template

```bash
#!/bin/bash
set -euo pipefail

# Test metadata
TEST_RESOURCE="resource-name"
TEST_TIMEOUT="${TEST_TIMEOUT:-60}"

# Source framework helpers
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"

# Test setup
setup_test() {
    echo "🔧 Setting up test..."
    require_resource "$TEST_RESOURCE"
    require_tools "curl" "jq"
}

# Test implementation
test_resource_functionality() {
    echo "🧪 Testing functionality..."
    
    local response
    response=$(curl -s "http://localhost:$PORT/endpoint")
    
    assert_http_success "$response" "API responds"
    assert_json_valid "$response" "Response is valid JSON"
}

# Main execution
main() {
    setup_test
    test_resource_functionality
    print_assertion_summary
}

main "$@"
```

### Using Assertions

```bash
# Basic assertions
assert_equals "$actual" "$expected" "Values match"
assert_contains "$output" "success" "Output contains success"
assert_not_empty "$result" "Result is not empty"

# HTTP assertions
curl_and_assert_success "http://localhost:$PORT/health" "Health check"
curl_and_assert_status "http://localhost:$PORT/api" 200 "API returns 200"

# JSON assertions
assert_json_valid "$response" "Valid JSON response"
assert_json_field "$response" ".status" "Has status field"

# Resource requirements
require_resource "ollama" "Ollama is required"
require_tools "docker" "jq" "curl"
```

## 🔧 Common Patterns

### Health Check Pattern
```bash
check_resource_health() {
    local port="$1"
    local response
    
    if response=$(curl -s --max-time 10 "http://localhost:${port}/health"); then
        echo "healthy"
    else
        echo "unreachable"
    fi
}
```

### Timeout Pattern
```bash
wait_for_condition() {
    local condition="$1"
    local timeout="${2:-30}"
    
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if eval "$condition"; then
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done
    return 1
}
```

### Skip Test Pattern
```bash
if [[ -z "$required_dependency" ]]; then
    skip_test "Missing required dependency"
fi
```

## 🐛 Debugging

### Enable Verbose Output
```bash
./run.sh --verbose --resource ollama
```

### Enable Debug Mode
```bash
./run.sh --debug --resource ollama
```

### Check Specific Resource Health
```bash
source framework/discovery.sh
check_resource_health "ollama"
```

### Validate Test File
```bash
./framework/helpers/test-validator.sh --path single/ai/ollama.test.sh
```

### Run Interface Compliance Only
```bash
./run.sh --interface-only --resource ollama
```

## 📋 Exit Codes

- `0` - Test passed successfully
- `1` - Test failed with errors
- `77` - Test skipped (missing dependencies)
- `124` - Test timed out
- `2` - Configuration error

## 🔍 Environment Variables

- `TEST_TIMEOUT` - Override default timeout (seconds)
- `TEST_VERBOSE` - Enable verbose output
- `TEST_DEBUG` - Enable debug mode
- `HEALTHY_RESOURCES_STR` - Pre-discovered resources
- `HTTP_LOG_ENABLED` - Enable HTTP logging

## 🚀 Extending the Framework

### Adding a New Resource Test

1. Create test file: `single/<category>/<resource>.test.sh`
2. Include required metadata and sourcing
3. Implement setup and test functions
4. Add assertions for expected behavior

### Adding a New Helper

1. Create helper in `framework/helpers/`
2. Follow naming convention: `<function>-<purpose>.sh`
3. Include proper error handling
4. Document functions with comments

### Adding Resource-Specific Health Check

In `discovery.sh`, add to `check_resource_health()`:
```bash
"your-resource")
    check_your_resource_health "$port"
    ;;
```

Then implement:
```bash
check_your_resource_health() {
    local port="$1"
    # Your health check logic
}
```

## 💡 Best Practices

1. **No Mocking** - Test against real services
2. **Isolation** - Each test gets its own environment
3. **Idempotency** - Tests should be repeatable
4. **Clear Messages** - Use descriptive assertion messages
5. **Resource Checks** - Always use `require_resource()`
6. **Cleanup** - Register cleanup handlers
7. **Timeouts** - Set appropriate timeouts
8. **Exit Codes** - Use standard exit codes

## 🔗 Related Documentation

- [Interface Compliance Guide](../INTERFACE_COMPLIANCE_README.md)
- [Resource Management](../../README.md)
- [Test Writing Guide](../README.md)

---

The Vrooli Resource Test Framework ensures reliable integration testing while maintaining flexibility for diverse resource types. For questions or contributions, see the main project documentation.