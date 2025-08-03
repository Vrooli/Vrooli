# 🎨 Testing Patterns

Common patterns and examples for the Vrooli test framework.

## 📚 Pattern Categories

### 🧪 **Unit Testing Patterns**
- [Basic Unit Tests](unit-testing.md) - Simple function and module testing
- [Service Mocking](service-mocking.md) - Testing with service dependencies
- [Configuration Testing](configuration.md) - Testing configuration handling

### 🔌 **Integration Testing Patterns**
- [Service Integration](integration.md) - Testing real services
- [Multi-Service Workflows](workflows.md) - End-to-end scenarios
- [API Testing](api-testing.md) - Testing REST APIs and webhooks

### 🚀 **Advanced Patterns**
- [Performance Testing](performance.md) - Load and benchmark testing
- [Error Handling](error-handling.md) - Testing failure scenarios
- [Security Testing](security.md) - Testing security features

## 🎯 Quick Pattern Reference

### Essential Setup Patterns

```bash
# Basic unit test
setup() {
    vrooli_setup_unit_test
}

# Service-specific test
setup() {
    vrooli_setup_service_test "ollama"
}

# Multi-service integration
setup() {
    vrooli_setup_integration_test "postgres" "redis"
}

# Performance test (minimal overhead)
setup() {
    vrooli_setup_performance_test
}
```

### Common Assertion Patterns

```bash
# Command testing
assert_command_success my_command
assert_command_failure invalid_command
assert_output_contains "expected text"

# JSON API testing
assert_json_has_key "$.status"
assert_json_equals "$.status" "healthy"
assert_json_array_length "$.items" 5

# File system testing
assert_file_exists "/path/to/file"
assert_file_contains "/path/to/file" "content"
assert_directory_not_empty "/tmp/output"

# Service testing
assert_service_healthy "localhost:8080"
assert_port_open 5432
assert_process_running "my_daemon"
```

### Resource Management Patterns

```bash
# Dynamic port allocation
setup() {
    vrooli_setup_unit_test
    local port=$(vrooli_port_allocate "my-service")
    export MY_SERVICE_PORT="$port"
}

# Temporary file creation
@test "file processing test" {
    local input_file=$(vrooli_isolation_create_file "input.txt")
    echo "test data" > "$input_file"
    
    # Process file - cleanup is automatic
    process_file "$input_file"
}

# Custom cleanup registration
setup() {
    vrooli_setup_unit_test
    vrooli_isolation_register_cleanup "my_custom_cleanup"
}

my_custom_cleanup() {
    # Custom cleanup logic here
    echo "Cleaning up custom resources..."
}
```

## 📋 Pattern Selection Guide

### Choose Your Pattern Based on What You're Testing

| Testing Goal | Pattern Type | Setup Function |
|--------------|--------------|----------------|
| Pure functions, algorithms | Unit | `vrooli_setup_unit_test` |
| Single service behavior | Service | `vrooli_setup_service_test "service"` |
| Multiple services working together | Integration | `vrooli_setup_integration_test "svc1" "svc2"` |
| End-to-end user workflows | Workflow | `vrooli_setup_integration_test` + scenarios |
| Performance characteristics | Performance | `vrooli_setup_performance_test` |

### Pattern Complexity Guide

| Complexity | Lines of Code | When to Use |
|------------|---------------|-------------|
| **Simple** | 5-15 lines | Basic function testing, configuration validation |
| **Medium** | 15-30 lines | Service integration, API testing, file processing |
| **Complex** | 30-50 lines | Multi-service workflows, performance testing |

## 🔍 Pattern Examples by Use Case

### Testing Configuration Changes
```bash
@test "configuration reload updates behavior" {
    vrooli_setup_unit_test
    
    # Create test configuration
    local config=$(vrooli_isolation_create_file "test.yaml")
    echo "debug: true" > "$config"
    
    # Test configuration loading
    assert_command_success load_config "$config"
    assert_config_equals "debug" "true"
    
    # Test configuration change
    echo "debug: false" > "$config"
    assert_command_success reload_config "$config"
    assert_config_equals "debug" "false"
}
```

### Testing API Endpoints
```bash
@test "API returns correct status and data" {
    vrooli_setup_service_test "api-server"
    
    local port=$(vrooli_config_get_port "api-server")
    local base_url="http://localhost:$port"
    
    # Test endpoint availability
    assert_service_healthy "localhost:$port"
    
    # Test API response
    assert_command_success curl -sf "$base_url/api/status"
    assert_json_equals "$.status" "healthy"
    assert_json_has_key "$.version"
    assert_json_has_key "$.uptime"
}
```

### Testing Error Conditions
```bash
@test "service handles invalid input gracefully" {
    vrooli_setup_service_test "my-service"
    
    # Test various invalid inputs
    assert_command_failure my_service --invalid-flag
    assert_output_contains "Invalid flag"
    
    assert_command_failure my_service --input ""
    assert_output_contains "Input required"
    
    # Ensure service is still healthy after errors
    assert_service_healthy "localhost:$(vrooli_config_get_port 'my-service')"
}
```

### Testing File Processing
```bash
@test "batch file processing works correctly" {
    vrooli_setup_unit_test
    
    # Create test files
    local input_dir=$(vrooli_isolation_create_tmpdir "input")
    local output_dir=$(vrooli_isolation_create_tmpdir "output")
    
    # Create test data
    echo "file1 content" > "$input_dir/file1.txt"
    echo "file2 content" > "$input_dir/file2.txt"
    
    # Process files
    assert_command_success process_directory "$input_dir" "$output_dir"
    
    # Verify results
    assert_file_exists "$output_dir/file1.processed"
    assert_file_exists "$output_dir/file2.processed"
    assert_file_contains "$output_dir/file1.processed" "processed"
}
```

## 🎯 Best Practices Summary

### ✅ Do This
- **Use specific setup functions** for your test type
- **Use rich assertions** instead of manual checks
- **Test one thing per test** for clarity
- **Use descriptive test names** that explain what's being tested
- **Let the framework handle cleanup** (don't write manual teardown)

### ❌ Avoid This
- **Don't mix unit and integration** tests in the same file
- **Don't use manual cleanup** when automatic cleanup is available
- **Don't hard-code ports or paths** - use configuration and isolation
- **Don't test implementation details** - test behavior
- **Don't write overly complex tests** - break them down

### 📏 Sizing Guidelines

| Test Type | Typical Length | Max Recommended |
|-----------|----------------|------------------|
| Unit test | 10-20 lines | 40 lines |
| Service test | 15-30 lines | 50 lines |
| Integration test | 20-40 lines | 60 lines |
| Workflow test | 30-50 lines | 80 lines |

## 🚀 Next Steps

1. **Start with [Unit Testing Patterns](unit-testing.md)** for basic testing
2. **Move to [Integration Patterns](integration.md)** for service testing
3. **Explore [Advanced Patterns](workflows.md)** for complex scenarios
4. **Check [API Reference](../reference/)** for complete function lists

**Need help?** Check the [troubleshooting guide](../troubleshooting/) or [quick start](../quick-start.md) for setup help!