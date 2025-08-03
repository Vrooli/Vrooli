# 🚀 Quick Start Guide

Get up and running with the Vrooli test framework in 5 minutes!

## ⚡ TL;DR - Just Run Tests

```bash
# Navigate to test directory
cd scripts/__test

# Run everything (recommended first try)
./runners/run-all.sh

# Run only fast unit tests
./runners/run-unit.sh

# Test only what changed
./runners/run-changed.sh
```

## 📋 Prerequisites

### Required Tools
- **Bash 4.0+** (check: `bash --version`)
- **BATS** (check: `bats --version`)
- **Git** (for change detection)
- **Basic Unix tools**: `curl`, `jq`, `nc` (for integration tests)

### Quick Install (Ubuntu/Debian)
```bash
# Install BATS and dependencies
sudo apt update
sudo apt install bats curl jq netcat-openbsd

# Verify installation
bats --version
```

### Quick Install (macOS)
```bash
# Using Homebrew
brew install bats-core curl jq netcat

# Verify installation
bats --version
```

## 🎯 Your First Test Run

### Step 1: Basic Test Execution
```bash
cd scripts/__test

# See what tests are available
./runners/run-unit.sh --list

# Run a quick smoke test
./runners/run-unit.sh --filter "core"
```

**Expected Output:**
```
🧪 Unit Test Runner
Found 5 unit test files
✅ core_config: 2.3s, 8 tests passed
✅ core_utils: 1.8s, 12 tests passed
...
Unit Test Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Total:    5
  Passed:   5 ✅
  Failed:   0 ❌
  Success:  100%
```

### Step 2: Integration Tests (Optional)
```bash
# Check if services are available for integration testing
./integration/health-check.sh

# Run integration tests (slower, requires services)
./runners/run-integration.sh --list
./runners/run-integration.sh ollama  # Test specific service
```

### Step 3: Smart Change Detection
```bash
# Make a small change to any file
echo "# test comment" >> fixtures/setup.bash

# Test only what's affected by your change
./runners/run-changed.sh --dry-run    # See what would run
./runners/run-changed.sh              # Actually run tests
```

## ✍️ Writing Your First Test

### Create a Simple Unit Test

1. **Create test file:**
```bash
cat > fixtures/tests/my_first_test.bats << 'EOF'
#!/usr/bin/env bats
# My First Test - Testing basic functionality

# Load the test framework (this line is required)
source "${VROOLI_TEST_ROOT}/fixtures/setup.bash"

setup() {
    # Set up test environment (runs before each test)
    vrooli_setup_unit_test
}

@test "basic math works correctly" {
    # Test that 2 + 2 equals 4
    result=$((2 + 2))
    assert_equals "$result" "4"
}

@test "string manipulation works" {
    # Test string operations
    text="hello world"
    assert_contains "$text" "hello"
    assert_contains "$text" "world"
}

@test "command execution works" {
    # Test running commands
    assert_command_success echo "test"
    assert_output_equals "test"
}
EOF
```

2. **Run your test:**
```bash
./runners/run-unit.sh fixtures/tests/my_first_test.bats
```

3. **Expected output:**
```
✅ my_first_test: 0.8s, 3 tests passed
```

### Create an Integration Test

1. **Create integration test:**
```bash
mkdir -p integration/services
cat > integration/services/my_service.sh << 'EOF'
#!/usr/bin/env bash
# Test a real service

source "${VROOLI_TEST_ROOT}/shared/logging.bash"
source "${VROOLI_TEST_ROOT}/shared/utils.bash"

# Test if a service is healthy
test_service_health() {
    vrooli_log_info "Testing service health..."
    
    # Example: test that port 80 is open (adjust as needed)
    if nc -z localhost 80 2>/dev/null; then
        vrooli_log_success "Service is healthy"
        return 0
    else
        vrooli_log_warn "Service not available (this is expected in most cases)"
        return 0  # Don't fail - service might not be running
    fi
}

# Run the test
test_service_health
EOF
```

2. **Make it executable and run:**
```bash
chmod +x integration/services/my_service.sh
./runners/run-integration.sh my_service
```

## 🎮 Essential Commands

### Running Tests

| Command | Purpose | Example |
|---------|---------|---------|
| `run-all.sh` | Run everything | `./runners/run-all.sh --verbose` |
| `run-unit.sh` | Unit tests only | `./runners/run-unit.sh --parallel` |
| `run-integration.sh` | Integration tests | `./runners/run-integration.sh ollama` |
| `run-changed.sh` | Changed files only | `./runners/run-changed.sh --dry-run` |

### Common Options

| Option | Effect | Example |
|--------|--------|---------|
| `--verbose` | Show detailed output | `./runners/run-unit.sh --verbose` |
| `--parallel` | Run tests in parallel | `./runners/run-unit.sh --parallel --jobs 8` |
| `--filter PATTERN` | Filter by name pattern | `./runners/run-all.sh --filter "ollama"` |
| `--dry-run` | Show what would run | `./runners/run-changed.sh --dry-run` |
| `--fail-fast` | Stop on first failure | `./runners/run-all.sh --fail-fast` |

### Debugging and Cleanup

| Command | Purpose |
|---------|---------|
| `./cleanup-all.sh --status` | Show cleanup status |
| `./cleanup-all.sh --clean` | Clean test resources |
| `./cleanup-all.sh --full` | Full cleanup including Docker |
| `bats --verbose test.bats` | Debug specific test |

## 🧪 Test Framework Features

### Rich Assertions (Available in all tests)
```bash
# Basic assertions
assert_equals "actual" "expected"
assert_not_equals "actual" "unexpected"
assert_contains "hello world" "hello"

# Command testing
assert_command_success echo "hello"
assert_command_failure false
assert_output_contains "hello"
assert_output_not_contains "goodbye"

# File system
assert_file_exists "/tmp/test.txt"
assert_directory_exists "/tmp/test_dir"

# JSON testing
assert_json_has_key "$.status"
assert_json_equals "$.status" "healthy"

# Service testing
assert_service_healthy "localhost:8080"
assert_port_open 8080
```

### Automatic Test Isolation
```bash
setup() {
    vrooli_setup_unit_test  # Automatic isolation and cleanup
}

# Or for service-specific tests
setup() {
    vrooli_setup_service_test "ollama"  # Service-specific setup
}

# Or for integration tests
setup() {
    vrooli_setup_integration_test "postgres" "redis"  # Multi-service setup
}
```

### Configuration Access
```bash
# Get service configuration
port=$(vrooli_config_get_port "ollama")
timeout=$(vrooli_config_get_timeout "ollama")

# Get general configuration
debug=$(vrooli_config_get_bool "debug" "false")
max_parallel=$(vrooli_config_get_int "max_parallel_tests" "4")
```

## 🔧 Common Patterns

### Testing a Service
```bash
@test "ollama service responds to API calls" {
    vrooli_setup_service_test "ollama"
    
    # Get service configuration
    local port=$(vrooli_config_get_port "ollama")
    local base_url="http://localhost:$port"
    
    # Test service health
    assert_service_healthy "localhost:$port"
    
    # Test API endpoint
    assert_command_success curl -sf "$base_url/api/version"
    assert_json_has_key "$.version"
}
```

### Testing Configuration
```bash
@test "configuration loading works correctly" {
    vrooli_setup_unit_test
    
    # Create test config
    local config_file=$(vrooli_isolation_create_file "test-config")
    echo "test_value: 42" > "$config_file"
    
    # Test config loading
    source_config "$config_file"
    assert_equals "$test_value" "42"
}
```

### Testing File Operations
```bash
@test "file processing works correctly" {
    vrooli_setup_unit_test
    
    # Create test file in isolated directory
    local test_file=$(vrooli_isolation_create_file "input.txt")
    echo "test data" > "$test_file"
    
    # Process file
    process_file "$test_file" > "$test_file.out"
    
    # Verify results
    assert_file_exists "$test_file.out"
    assert_file_contains "$test_file.out" "processed"
}
```

## 🆘 Troubleshooting Quick Fixes

### Tests Not Found
```bash
# Problem: "No tests found"
# Solution: Check your path and file patterns
ls fixtures/tests/*.bats  # Should show test files
find . -name "*.bats"     # Find all test files
```

### Permission Errors
```bash
# Problem: Permission denied
# Solution: Make scripts executable
chmod +x runners/*.sh
chmod +x integration/**/*.sh
```

### Service Connection Errors
```bash
# Problem: "Connection refused" in integration tests
# Solution: Check if services are running
./integration/health-check.sh              # Check all services
nc -z localhost 11434                      # Check specific port
docker ps                                  # Check Docker containers
```

### Test Isolation Issues
```bash
# Problem: Tests interfere with each other
# Solution: Clean up between runs
./cleanup-all.sh --clean                   # Clean test resources
./cleanup-all.sh --status                  # Check what's left
```

### Slow Test Execution
```bash
# Problem: Tests are too slow
# Solution: Use parallel execution and filters
./runners/run-unit.sh --parallel           # Parallel unit tests
./runners/run-changed.sh                   # Only test changes
./runners/run-all.sh --filter "core"       # Filter by pattern
```

## 🎯 Next Steps

### Explore Advanced Features
1. **[Pattern Examples](patterns/)** - Learn common testing patterns
2. **[API Reference](reference/)** - Complete function documentation
3. **[Configuration Guide](reference/configuration.md)** - Advanced configuration

### Integrate with Development Workflow
```bash
# Add to your development script
#!/bin/bash
# my-dev-workflow.sh

echo "Running tests for changed files..."
if ./runners/run-changed.sh; then
    echo "✅ All tests passed - ready to commit!"
else
    echo "❌ Tests failed - please fix before committing"
    exit 1
fi
```

### Set Up Pre-commit Hooks
```bash
# .git/hooks/pre-commit
#!/bin/bash
cd scripts/__test
exec ./runners/run-changed.sh --fail-fast
```

## 🎉 You're Ready!

You now have:
- ✅ A working test environment
- ✅ Your first custom test
- ✅ Knowledge of essential commands
- ✅ Debugging techniques

**Happy Testing!** 🧪

For more advanced topics, check out:
- **[Testing Patterns](patterns/)** - Real-world examples
- **[Troubleshooting Guide](troubleshooting/)** - Solutions to common issues
- **[Complete API Reference](reference/)** - All available functions