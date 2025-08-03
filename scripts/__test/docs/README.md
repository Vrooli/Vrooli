# 🧪 Vrooli Test Framework

A modern, developer-friendly testing infrastructure for the Vrooli platform with comprehensive coverage for unit tests, integration tests, and end-to-end scenarios.

## 📚 Quick Navigation

- [**🚀 Quick Start**](quick-start.md) - Get testing in 5 minutes
- [**🎯 Test Patterns**](patterns/) - Common testing patterns and examples
- [**🔧 Troubleshooting**](troubleshooting/) - Solutions to common issues
- [**📖 Reference**](reference/) - Complete API reference
- [**🔄 Migration Guide**](migration/) - Upgrading from old test structure

## 🏗️ Architecture Overview

The Vrooli test framework is built around three core principles:

1. **🎯 Simplicity First** - Easy to read, write, and debug tests
2. **🔧 Developer Experience** - Comprehensive tooling and helpful error messages  
3. **⚡ Performance** - Fast execution with smart parallel processing

### Directory Structure

```
scripts/__test/
├── 📁 config/          # Centralized configuration (YAML)
├── 📁 fixtures/        # Unit testing with BATS + mocks
├── 📁 integration/     # Real service integration testing
├── 📁 runners/         # Unified test execution scripts
├── 📁 shared/          # Common utilities and helpers
└── 📁 docs/           # Documentation (you are here!)
```

## 🎮 Test Runners

The framework provides four unified test runners:

### 🌟 **run-all.sh** - Complete Test Suite
```bash
./runners/run-all.sh                    # Run everything
./runners/run-all.sh --parallel --jobs 8  # Parallel execution
./runners/run-all.sh --filter "ollama"    # Filter by pattern
```

### 📦 **run-unit.sh** - Unit Tests Only
```bash
./runners/run-unit.sh                   # All unit tests
./runners/run-unit.sh --parallel        # Parallel execution
./runners/run-unit.sh test_file.bats    # Specific test
```

### 🔌 **run-integration.sh** - Integration Tests
```bash
./runners/run-integration.sh            # All integration tests
./runners/run-integration.sh ollama     # Test specific service
./runners/run-integration.sh --scenarios # Scenario tests only
```

### 🎯 **run-changed.sh** - Smart Change Detection
```bash
./runners/run-changed.sh                # Test changes in last commit
./runners/run-changed.sh --range origin/main..HEAD  # From main branch
./runners/run-changed.sh --dry-run      # Show what would run
```

## 🚀 Writing Your First Test

### Unit Test Example
```bash
#!/usr/bin/env bats
# File: test_example.bats

# Load the unified test setup (this is all you need!)
source "${VROOLI_TEST_ROOT}/fixtures/setup.bash"

setup() {
    # Automatic test isolation and cleanup registration
    vrooli_setup_unit_test
}

@test "example service responds correctly" {
    # Use rich assertions
    assert_command_success curl -sf http://localhost:8080/health
    assert_output_contains "healthy"
    assert_json_has_key "$.status"
}
```

### Integration Test Example
```bash
#!/usr/bin/env bash
# File: integration/services/ollama.sh

# Load shared utilities
source "${VROOLI_TEST_ROOT}/shared/utils.bash"
source "${VROOLI_TEST_ROOT}/shared/logging.bash"

test_ollama_health() {
    local response
    response=$(curl -sf http://localhost:11434/api/version)
    
    if echo "$response" | jq -e '.version' >/dev/null; then
        vrooli_log_success "Ollama health check passed"
        return 0
    else
        vrooli_log_error "Ollama health check failed"
        return 1
    fi
}

# Run all test functions
test_ollama_health
```

## 🔧 Configuration

Tests are configured through YAML files in the `config/` directory:

```yaml
# config/test-config.yaml
global:
  max_parallel_tests: 4
  default_timeout: 30
  
services:
  ollama:
    port: 11434
    timeout: 60
    health_check: "http://localhost:11434/api/version"
```

Environment-specific overrides:
- `config/environments/local.yaml`
- `config/environments/ci.yaml`
- `config/environments/docker.yaml`

## 🎯 Key Features

### ✅ **Rich Assertions**
- `assert_command_success` / `assert_command_failure`
- `assert_output_contains` / `assert_output_not_contains`
- `assert_json_has_key` / `assert_json_equals`
- `assert_file_exists` / `assert_directory_exists`
- `assert_service_healthy` / `assert_port_open`

### 🧹 **Automatic Cleanup**
- Test isolation with unique namespaces
- Automatic resource cleanup on exit
- Cleanup registration for custom resources
- Process and container lifecycle management

### ⚡ **Parallel Execution**
- Safe parallel test execution
- Port allocation and management
- Resource conflict prevention
- Smart dependency resolution

### 📊 **Smart Reporting**
- Detailed test results with timing
- Progress indicators for long test runs
- Failure analysis with context
- Coverage reporting (planned)

## 🎨 Test Patterns

### Testing Services
```bash
@test "service starts and responds" {
    # Start service in test isolation
    local port=$(vrooli_port_allocate "my-service")
    start_service_on_port "$port"
    
    # Wait for readiness
    assert_service_healthy "localhost:$port"
    
    # Test functionality
    assert_command_success curl "http://localhost:$port/api/status"
    assert_json_equals "$.status" "running"
}
```

### Testing Configuration Changes
```bash
@test "configuration updates take effect" {
    # Create isolated config
    local config_file=$(vrooli_isolation_create_file "test-config")
    echo "debug: true" > "$config_file"
    
    # Test with config
    assert_command_success my_app --config "$config_file"
    assert_output_contains "DEBUG"
}
```

### Testing Resource Interactions
```bash
@test "multi-service workflow" {
    # Register services for coordinated cleanup
    vrooli_resource_register "postgres" "docker" "postgres:13"
    vrooli_resource_register "redis" "docker" "redis:alpine"
    
    # Start services with dependency resolution
    vrooli_resource_start "postgres"
    vrooli_resource_start "redis"
    
    # Test workflow
    run_workflow_test
    
    # Cleanup happens automatically
}
```

## 🔍 Debugging Tests

### Verbose Output
```bash
./runners/run-unit.sh --verbose           # Show all output
./runners/run-integration.sh --verbose    # Integration details
```

### Test Isolation Debugging
```bash
# Check current test namespace
vrooli_isolation_get_namespace

# Show cleanup statistics  
vrooli_isolation_stats

# Manual cleanup if needed
./cleanup-all.sh --aggressive
```

### Failed Test Analysis
```bash
# Show only failed tests
./runners/run-all.sh --fail-fast

# Run specific failing test with debug
bats --verbose test_file.bats

# Check test logs
find /tmp -name "*vrooli_test*" -type f -name "*.log"
```

## 🚀 Performance Tips

### Parallel Execution
```bash
# Enable parallel mode with optimal job count
./runners/run-all.sh --parallel --jobs $(nproc)

# Unit tests are safe for parallel execution
./runners/run-unit.sh --parallel

# Integration tests run sequentially for safety
```

### Test Selection
```bash
# Only test changed files (fastest)
./runners/run-changed.sh

# Filter by pattern
./runners/run-all.sh --filter "core|config"

# Skip slow integration tests during development
./runners/run-unit.sh
```

### Resource Management
```bash
# Cleanup between test runs
./cleanup-all.sh --clean

# Full cleanup including Docker
./cleanup-all.sh --full

# Show resource usage
./cleanup-all.sh --status
```

## 🤝 Contributing

### Adding New Tests

1. **Unit Tests**: Add to `fixtures/tests/` directory
2. **Integration Tests**: Add to `integration/services/` or `integration/scenarios/`
3. **Test Templates**: Use templates in `fixtures/templates/`

### Testing Guidelines

- **Descriptive Names**: Use clear, specific test names
- **Isolation**: Each test should be independent
- **Cleanup**: Use automatic cleanup registration
- **Assertions**: Use rich assertion functions
- **Documentation**: Document complex test scenarios

### Common Patterns to Follow

```bash
# ✅ Good: Specific, isolated, well-structured
@test "ollama service generates text response with valid model" {
    vrooli_setup_service_test "ollama"
    
    local response
    response=$(curl -sf http://localhost:11434/api/generate -d '{"model":"llama2","prompt":"hello"}')
    
    assert_json_has_key "$.response"
    assert_json_not_equals "$.response" ""
}

# ❌ Avoid: Vague, coupled, hard to debug
@test "test ollama" {
    curl localhost:11434 | grep something
    [[ $? -eq 0 ]]
}
```

## 📖 More Documentation

- **[Quick Start Guide](quick-start.md)** - 5-minute setup
- **[Migration Guide](migration/)** - Upgrading existing tests
- **[Pattern Examples](patterns/)** - Common testing scenarios
- **[Troubleshooting](troubleshooting/)** - Problem-solving guide
- **[API Reference](reference/)** - Complete function reference

## 🆘 Getting Help

1. **Check [Troubleshooting](troubleshooting/)** for common issues
2. **Review [Pattern Examples](patterns/)** for implementation guidance
3. **Use `--verbose` flags** for detailed debugging output
4. **Check test logs** in `/tmp/vrooli_test_*` directories

## 🎉 Success Stories

> "The new test framework reduced our test setup time from 30 minutes to 3 minutes. The automatic cleanup alone saved us hours of debugging!" - *Development Team*

> "Writing integration tests is actually enjoyable now. The smart change detection means I only run what I need to test." - *QA Engineer*

---

**Need help?** Check out our [troubleshooting guide](troubleshooting/) or review the [quick start](quick-start.md) for a hands-on introduction!