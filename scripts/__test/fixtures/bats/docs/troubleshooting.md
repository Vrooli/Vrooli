# Troubleshooting Guide: Common Issues and Solutions

Quick solutions for common problems when using the Vrooli BATS testing infrastructure.

## 🚨 Quick Diagnostic Commands

### Health Check Your Test Environment
```bash
# Check if infrastructure is loaded correctly
source "/path/to/bats-fixtures/core/common_setup.bash"
setup_standard_mocks
mock::list_loaded

# Verify essential functions exist
assert_function_exists "setup_standard_mocks"
assert_function_exists "assert_success"
assert_function_exists "mock::load"

# Check directory structure
ls -la /path/to/bats-fixtures/
ls -la /path/to/bats-fixtures/mocks/system/
ls -la /path/to/bats-fixtures/docs/
```

### Test Environment Variables
```bash
# Check essential environment variables
echo "TEST_TMPDIR: ${TEST_TMPDIR:-NOT_SET}"
echo "MOCK_RESPONSES_DIR: ${MOCK_RESPONSES_DIR:-NOT_SET}"
echo "VROOLI_TEST_FIXTURES_DIR: ${VROOLI_TEST_FIXTURES_DIR:-NOT_SET}"

# For resource tests
echo "RESOURCE_NAME: ${RESOURCE_NAME:-NOT_SET}"
echo "TEST_NAMESPACE: ${TEST_NAMESPACE:-NOT_SET}"
```

## ❌ Common Error Messages

### "Mock not found for category:name"

**Error:**
```
[MOCK_REGISTRY] WARNING: Mock not found for system:docker
```

**Causes & Solutions:**

1. **File doesn't exist:**
```bash
# Check if mock file exists
ls -la /path/to/bats-fixtures/mocks/system/docker.bash

# If missing, check correct path
find /path/to/bats-fixtures -name "docker.bash" -type f
```

2. **Wrong category:**
```bash
# Resource auto-detection failed
mock::detect_resource_category "my-resource"

# Use correct category
mock::load system docker     # Not resource docker
mock::load resource ollama    # Not system ollama
```

3. **Path configuration wrong:**
```bash
# Check VROOLI_TEST_FIXTURES_DIR
echo "$VROOLI_TEST_FIXTURES_DIR"

# Should point to bats-fixtures directory
export VROOLI_TEST_FIXTURES_DIR="/correct/path/to/bats-fixtures"
```

### "Command call tracking not available"

**Error:**
```
Warning: Command call tracking not available
```

**Causes & Solutions:**

1. **Setup not called:**
```bash
# Bad: No setup
@test "my test" {
    run docker ps
    assert_command_called "docker"  # Fails
}

# Good: Proper setup
setup() {
    setup_standard_mocks  # Sets up tracking
}
```

2. **Wrong setup mode:**
```bash
# Bad: Manual mock loading without tracking
mock::load system docker

# Good: Use setup functions
setup_standard_mocks  # Includes tracking setup
```

3. **Directory cleanup:**
```bash
# Check if tracking directory exists
echo "MOCK_RESPONSES_DIR: $MOCK_RESPONSES_DIR"
ls -la "$MOCK_RESPONSES_DIR"

# Recreate if missing
mkdir -p "$MOCK_RESPONSES_DIR"
```

### "Expected environment variable to be set"

**Error:**
```
Expected environment variable to be set: OLLAMA_PORT
```

**Causes & Solutions:**

1. **Wrong setup mode:**
```bash
# Bad: Generic setup for resource test
setup_standard_mocks

# Good: Resource-specific setup
setup_resource_test "ollama"
```

2. **Resource name mismatch:**
```bash
# Bad: Wrong resource name
setup_resource_test "olama"  # Typo

# Good: Correct resource name
setup_resource_test "ollama"
```

3. **Custom environment not set:**
```bash
# If using custom ports
OLLAMA_PORT="9999" setup_resource_test "ollama"
assert_env_equals "OLLAMA_PORT" "9999"
```

### "Invalid JSON"

**Error:**
```
Invalid JSON: <html><body>Error 404</body></html>
```

**Causes & Solutions:**

1. **HTTP error returned HTML:**
```bash
# Check HTTP status first
assert_http_status "$API_URL" "200"
# Then check JSON
assert_json_valid "$response"
```

2. **Mock not configured:**
```bash
# Set proper mock response
mock::http::set_endpoint_response "$API_URL" '{"status":"ok"}' 200

# Instead of getting default HTML error
```

3. **Wrong endpoint:**
```bash
# Bad: Wrong endpoint
curl -s "$OLLAMA_BASE_URL/wrong-endpoint"

# Good: Check available endpoints
curl -s "$OLLAMA_BASE_URL/health"
curl -s "$OLLAMA_BASE_URL/api/tags"
```

### "Function already loaded"

**Error:**
```
bash: declare: docker: cannot assign - readonly
```

**Causes & Solutions:**

1. **Multiple loads (this is actually OK):**
```bash
# Mock registry prevents duplicates automatically
mock::load system docker  # First load
mock::load system docker  # Second load - ignored, not error
```

2. **Conflicting function definitions:**
```bash
# Check if function already exists
if declare -f docker >/dev/null; then
    echo "Docker function already defined"
fi

# Unset if needed (rarely necessary)
unset -f docker
```

### "Port not in use" / "Port not open"

**Error:**
```
Expected port to be in use: 11434
Port not open: localhost:11434
```

**Causes & Solutions:**

1. **Mock not configured:**
```bash
# Configure port mock
mock::http::set_endpoint_state "http://localhost:11434" "healthy"

# Or set container running
mock::docker::set_container_state "ollama_container" "running"
```

2. **Wrong port number:**
```bash
# Check expected port
echo "OLLAMA_PORT: $OLLAMA_PORT"
assert_env_equals "OLLAMA_PORT" "11434"  # Verify expectation
```

3. **Service not started in mock:**
```bash
# Ensure resource is configured as healthy
setup_resource_test "ollama"
assert_resource_healthy "ollama"  # This sets up port mocking
```

## 🔧 Performance Issues

### "Tests are slow"

**Symptoms:**
- Tests take > 5 seconds to start
- Individual assertions take > 100ms
- Memory usage > 100MB for simple tests

**Solutions:**

1. **Use minimal setup:**
```bash
# Bad: Over-engineering
setup_integration_test "ollama" "whisper" "n8n" "qdrant"

# Good: Use only what you need
setup_standard_mocks
```

2. **Enable performance mode:**
```bash
setup() {
    export TEST_PERFORMANCE_MODE="true"
    setup_standard_mocks
}
```

3. **Cache expensive operations:**
```bash
# Bad: Multiple API calls
assert_json_field_equals "$(curl -s $API)" ".status" "ok"
assert_json_field_exists "$(curl -s $API)" ".version"

# Good: Cache response
local response=$(curl -s "$API")
assert_json_field_equals "$response" ".status" "ok"
assert_json_field_exists "$response" ".version"
```

### "Memory usage too high"

**Solutions:**

1. **Check loaded mocks:**
```bash
# See what's loaded
mock::list_loaded

# Only load what you need
setup_standard_mocks  # Not setup_integration_test
```

2. **Clean up properly:**
```bash
teardown() {
    cleanup_mocks  # Always call this
}
```

3. **Use in-memory temp directories:**
```bash
# Should automatically use /dev/shm or /tmp
echo "TEST_TMPDIR: $TEST_TMPDIR"
```

## 🐛 Debugging Techniques

### Enable Debug Mode
```bash
# Enable various debug modes
export MOCK_DEBUG="true"
export MOCK_TRACE_COMMANDS="true" 
export MOCK_PERFORMANCE_TIMING="true"
export BATS_VERBOSE_RUN="true"

# Run single test with verbose output
bats -t my_test.bats
```

### Inspect Mock State
```bash
@test "debug mock state" {
    setup_resource_test "ollama"
    
    # Check loaded mocks
    mock::list_loaded
    
    # Check environment
    env | grep -E "(OLLAMA|TEST|MOCK)" | sort
    
    # Check mock responses
    ls -la "$MOCK_RESPONSES_DIR"
    
    # Check command tracking
    if [[ -f "${MOCK_RESPONSES_DIR}/command_calls.log" ]]; then
        echo "=== Command Calls ==="
        cat "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
}
```

### Test Mock Responses
```bash
@test "debug API responses" {
    setup_resource_test "ollama"
    
    # Test each endpoint manually
    echo "=== Health Check ==="
    curl -v "$OLLAMA_BASE_URL/health"
    
    echo "=== API Tags ==="
    curl -v "$OLLAMA_BASE_URL/api/tags"
    
    echo "=== Container Status ==="
    docker inspect "$OLLAMA_CONTAINER_NAME" || echo "Container not found"
}
```

### Validate Assertions
```bash
@test "debug assertion behavior" {
    local test_json='{"status":"ok","version":"1.0.0"}'
    
    # Test JSON parsing
    echo "JSON: $test_json"
    echo "Status: $(echo "$test_json" | jq -r '.status')"
    echo "Version: $(echo "$test_json" | jq -r '.version')"
    
    # Test assertions
    assert_json_valid "$test_json"
    assert_json_field_equals "$test_json" ".status" "ok"
    assert_json_field_exists "$test_json" ".version"
}
```

## 🔍 Environment Issues

### "Command not found: bats"

**Solutions:**
```bash
# Check BATS installation
which bats

# Install if missing (Ubuntu/Debian)
sudo apt-get install bats

# Or install from GitHub
git clone https://github.com/bats-core/bats-core.git
cd bats-core
sudo ./install.sh /usr/local
```

### "Permission denied" errors

**Solutions:**
```bash
# Check file permissions
ls -la /path/to/bats-fixtures/
chmod +x /path/to/bats-fixtures/**/*.bash

# Check test file permissions
chmod +x your_test.bats

# Check temporary directory permissions
ls -la "$TEST_TMPDIR"
```

### "No such file or directory"

**Solutions:**
```bash
# Check relative vs absolute paths
# Bad: Relative path
source "bats-fixtures/core/common_setup.bash"

# Good: Absolute path
source "/full/path/to/bats-fixtures/core/common_setup.bash"

# Or use environment variable
source "$VROOLI_TEST_FIXTURES_DIR/core/common_setup.bash"
```

## 🔄 Integration Issues

### "Docker commands fail"

**Solutions:**
```bash
# Check if Docker mock is loaded
mock::is_loaded system docker

# Manually load if needed
mock::load system docker

# Test Docker mock
run docker --version
assert_success
assert_output_contains "Docker version"
```

### "HTTP requests timeout"

**Solutions:**
```bash
# Check HTTP mock mode
echo "HTTP_MOCK_MODE: ${HTTP_MOCK_MODE:-normal}"

# Disable slow mode if enabled
export HTTP_MOCK_MODE="normal"

# Check endpoint configuration
curl -v "$API_URL" || echo "Endpoint not mocked"
```

### "Resource dependencies fail"

**Solutions:**
```bash
# For integration tests, ensure all resources are configured
setup_integration_test "ollama" "whisper" "n8n"

# Check each resource individually
assert_resource_healthy "ollama"
assert_resource_healthy "whisper"
assert_resource_healthy "n8n"

# Check cross-resource communication
assert_http_endpoint_reachable "$OLLAMA_BASE_URL/health"
assert_http_endpoint_reachable "$WHISPER_BASE_URL/health"
```

## 📊 Common Test Patterns Issues

### "Assertions fail unexpectedly"

**Debug approach:**
```bash
@test "debug failing assertion" {
    # Add debug output before assertion
    echo "Output: '$output'"
    echo "Status: $status"
    
    # Break down complex assertions
    assert_success
    assert_not_empty "$output"
    assert_output_contains "expected_text"
    
    # Use more specific assertions
    assert_equals "$status" "0"
    assert_string_contains "$output" "specific_substring"
}
```

### "Environment not isolated"

**Solutions:**
```bash
# Ensure unique test namespace
echo "TEST_NAMESPACE: $TEST_NAMESPACE"

# Check container naming
echo "Container names:"
env | grep "_CONTAINER_NAME"

# Verify cleanup
teardown() {
    cleanup_mocks
    # Additional cleanup if needed
    unset CUSTOM_VAR
}
```

## 💡 Best Practices for Troubleshooting

### 1. Start Simple
```bash
# Begin with minimal test
@test "basic functionality" {
    setup_standard_mocks
    run docker --version
    assert_success
}

# Then add complexity gradually
```

### 2. Use Debug Output
```bash
@test "debug test" {
    setup_resource_test "ollama"
    
    # Debug output
    echo "=== Environment ==="
    env | grep OLLAMA
    
    echo "=== Mock State ==="
    mock::list_loaded
    
    echo "=== Actual Test ==="
    assert_resource_healthy "ollama"
}
```

### 3. Test Assumptions
```bash
@test "validate assumptions" {
    # Test your assumptions explicitly
    assert_command_exists "curl"
    assert_command_exists "jq"
    assert_command_exists "docker"
    
    # Then proceed with actual test
}
```

### 4. Isolate Problems
```bash
# Test components in isolation
@test "test mock loading only" {
    mock::load system docker
    assert_function_exists "docker"
}

@test "test environment setup only" {
    setup_resource_test "ollama"
    assert_env_set "OLLAMA_PORT"
}

@test "test full integration" {
    setup_resource_test "ollama"
    assert_resource_healthy "ollama"
}
```

## 📞 Getting Help

### Check Documentation First
1. [Setup Guide](setup-guide.md) - For setup mode issues
2. [Assertions Reference](assertions.md) - For assertion problems
3. [Mock Registry](mock-registry.md) - For mock loading issues
4. [Resource Testing](resource-testing.md) - For resource-specific problems

### Create Minimal Reproduction
```bash
#!/usr/bin/env bats
# Minimal test case for bug report

bats_require_minimum_version 1.5.0
source "/path/to/bats-fixtures/core/common_setup.bash"

setup() {
    setup_standard_mocks
}

@test "reproduce issue" {
    # Minimal steps to reproduce the problem
    run docker ps
    assert_success  # This fails, but why?
}
```

### Gather Debug Information
```bash
# System information
echo "BATS version: $(bats --version)"
echo "Bash version: $BASH_VERSION"
echo "OS: $(uname -a)"

# Test environment
echo "Test fixtures dir: $VROOLI_TEST_FIXTURES_DIR"
echo "Working directory: $(pwd)"
echo "Test temp dir: $TEST_TMPDIR"

# Mock state
mock::list_loaded
```

Remember: Most issues are configuration or path problems. Start with the basics and work your way up to complex scenarios.