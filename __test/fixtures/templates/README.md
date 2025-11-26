# Vrooli Test Templates

Simple, copy-paste ready templates for writing BATS tests.

## Quick Start

1. **Copy the appropriate template** to your test directory
2. **Rename the file** to match what you're testing
3. **Replace the example tests** with your actual tests
4. **Run your test** with `bats your-test.bats`

## Available Templates

### 📝 basic.bats
**Use for:** Simple unit tests with system mocks (Docker, HTTP, file operations)

```bash
cp fixtures/templates/basic.bats fixtures/tests/core/my-test.bats
```

**What you get:**
- System command mocks (docker, curl, etc.)
- File operation helpers
- Basic assertions
- Temporary directory setup

### ⚙️ service.bats
**Use for:** Testing a specific service (Ollama, PostgreSQL, N8N, etc.)

```bash
cp fixtures/templates/service.bats fixtures/tests/services/ollama-test.bats
```

**What you get:**
- Service-specific environment variables
- Service health checks
- Docker container testing
- Configuration validation

**Remember to change:**
- `SERVICE_NAME="ollama"` → your service name
- Update test descriptions and assertions

### 🔗 integration.bats
**Use for:** Testing multiple services working together

```bash
cp fixtures/templates/integration.bats fixtures/tests/workflows/ai-storage.bats
```

**What you get:**
- Multi-service environment setup
- Integration test patterns
- Data pipeline testing
- Cross-service validation

**Remember to change:**
- `SERVICES=("ollama" "postgres" "node-red")` → your services
- Update integration scenarios

## Writing Good Tests

### Test Naming
```bash
# Good: Descriptive and specific
@test "ollama generates response from natural language prompt"
@test "postgres connection handles authentication failure gracefully"

# Bad: Vague and unclear  
@test "ollama works"
@test "test database"
```

### Assertions
```bash
# Use specific assertions
assert_output_contains "expected text"
assert_json_field_equals "$json" ".status" "success"
assert_file_exists "$expected_file"

# Avoid vague patterns
[[ "$output" =~ "something" ]]  # Too vague
```

### Test Structure
```bash
@test "clear description of what is being tested" {
    # 1. Setup (if needed beyond global setup)
    local test_data="example"
    
    # 2. Execute the command
    run your_command "$test_data"
    
    # 3. Verify results
    assert_success
    assert_output_contains "expected result"
}
```

## Common Patterns

### Testing Docker Commands
```bash
@test "docker container starts successfully" {
    run docker run -d nginx
    assert_success
    assert_output_contains "container_id"
}
```

### Testing HTTP APIs
```bash
@test "API returns valid health status" {
    run curl -s http://localhost:8080/health
    assert_success
    assert_json_valid "$output"
    assert_json_field_equals "$output" ".status" "healthy"
}
```

### Testing File Operations
```bash
@test "configuration file is created correctly" {
    local config_file="$VROOLI_TEST_TMPDIR/config.yml"
    
    run generate_config "$config_file"
    assert_success
    assert_file_exists "$config_file"
    assert_file_contains "$config_file" "version: 1.0"
}
```

### Testing Environment Variables
```bash
@test "service environment is properly configured" {
    assert_env_set "API_PORT"
    assert_env_equals "API_PORT" "8080"
}
```

## Debugging Tests

### Enable Debug Output
```bash
# In your test file, add this to setup():
vrooli_log_enable_debug

# Or run with debug:
VROOLI_CONFIG_DEBUG=true bats your-test.bats
```

### Check Test Logs
```bash
# Test logs are written to:
cat /tmp/vrooli-test-*.log
```

### Validate Test Environment
```bash
@test "debug: show test environment" {
    echo "Test namespace: $TEST_NAMESPACE"
    echo "Temp directory: $VROOLI_TEST_TMPDIR"
    echo "Available functions: $(declare -F | grep assert_ | wc -l)"
}
```

## Need Help?

- **Check existing tests** in `fixtures/tests/` for examples
- **Read assertions reference** in the main documentation
- **Use auto-setup** if unsure: `vrooli_auto_setup` in setup()

## Migration from Old Tests

If you have existing BATS tests:

1. **Replace setup functions:**
   ```bash
   # Old
   setup_standard_mock_framework
   
   # New
   vrooli_setup_unit_test
   ```

2. **Update teardown:**
   ```bash
   # Old
   cleanup_standard_mock_framework
   
   # New
   vrooli_cleanup_test
   ```

3. **Use new assertions:**
   ```bash
   # Old
   [[ "$output" =~ "text" ]]
   
   # New
   assert_output_contains "text"
   ```

The new system is backward compatible, so old tests should still work while you migrate.
