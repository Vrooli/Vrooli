# Assertions Reference: 35+ Testing Functions

Complete reference for all assertion functions available in the Vrooli BATS testing infrastructure.

## 📋 Quick Reference

| Category | Count | Key Functions |
|----------|-------|---------------|
| **[Standard BATS](#standard-bats-assertions)** | 4 | `assert_success`, `assert_failure`, `assert_output`, `assert_equal` |
| **[Output Testing](#output-assertions)** | 5 | `assert_output_contains`, `assert_output_matches`, `assert_output_empty` |
| **[File System](#file-system-assertions)** | 7 | `assert_file_exists`, `assert_dir_exists`, `assert_file_contains` |
| **[Environment](#environment-assertions)** | 4 | `assert_env_set`, `assert_env_equals`, `assert_env_unset` |
| **[Commands](#command-assertions)** | 3 | `assert_command_exists`, `assert_function_exists`, `assert_command_called` |
| **[JSON Testing](#json-assertions)** | 4 | `assert_json_valid`, `assert_json_field_equals`, `assert_json_field_exists` |
| **[Network Testing](#network-assertions)** | 4 | `assert_port_in_use`, `assert_http_endpoint_reachable`, `assert_http_status` |
| **[Resource Testing](#resource-assertions)** | 5 | `assert_resource_healthy`, `assert_docker_container_running` |
| **[Data Structures](#data-structure-assertions)** | 2 | `assert_arrays_equal`, `assert_array_contains` |
| **[Mock Testing](#mock-assertions)** | 2 | `assert_mock_used`, `assert_mock_response_used` |

---

## Standard BATS Assertions

These are built into BATS and available everywhere:

### Basic Status Checks
```bash
@test "basic command success" {
    run docker --version
    assert_success        # Exit code = 0
    assert_failure        # Exit code != 0
}
```

### Output Comparison
```bash
@test "exact output matching" {
    run echo "hello world"
    assert_output "hello world"           # Exact match
    assert_equal "$output" "hello world"  # Alternative syntax
}
```

### Line-by-Line Output
```bash
@test "multi-line output" {
    run docker ps
    assert_line --index 0 --partial "CONTAINER"
    assert_line --index 1 "container_name"
}
```

---

## Output Assertions

### `assert_output_contains(text)`
Test that output contains specific text (case-sensitive).
```bash
@test "output contains text" {
    run curl -s http://localhost:8080/health
    assert_output_contains "status"
    assert_output_contains "healthy"
}
```

### `assert_output_not_contains(text)`
Test that output does NOT contain specific text.
```bash
@test "no error messages" {
    run my_script.sh
    assert_success
    assert_output_not_contains "ERROR"
    assert_output_not_contains "FAILED"
}
```

### `assert_output_matches(regex)`
Test output against regex pattern.
```bash
@test "output format validation" {
    run docker --version
    assert_output_matches "Docker version [0-9]+\.[0-9]+\.[0-9]+"
}
```

### `assert_output_empty()`
Test that command produced no output.
```bash
@test "silent operation" {
    run my_silent_command
    assert_success
    assert_output_empty
}
```

### `assert_output_line_count(count)`
Test exact number of output lines.
```bash
@test "expected line count" {
    run ls /usr/bin
    assert_output_line_count 10  # Exactly 10 lines
}
```

---

## File System Assertions

### `assert_file_exists(path)`
Test that file exists.
```bash
@test "config file created" {
    run setup_script.sh
    assert_file_exists "/etc/myapp/config.yml"
}
```

### `assert_file_not_exists(path)`
Test that file does NOT exist.
```bash
@test "temp file cleaned up" {
    setup_and_cleanup.sh
    assert_file_not_exists "/tmp/myapp.lock"
}
```

### `assert_dir_exists(path)`
Test that directory exists.
```bash
@test "data directory created" {
    run mkdir -p /var/lib/myapp
    assert_dir_exists "/var/lib/myapp"
}
```

### `assert_dir_not_exists(path)`
Test that directory does NOT exist.
```bash
@test "temporary directory removed" {
    cleanup_temp_dirs.sh
    assert_dir_not_exists "/tmp/build_$$"
}
```

### `assert_file_contains(file, text)`
Test that file contains specific text.
```bash
@test "configuration applied" {
    run configure_app.sh
    assert_file_contains "/etc/myapp/config.yml" "database_host: localhost"
}
```

### `assert_file_empty(file)`
Test that file exists but is empty.
```bash
@test "log file initialized" {
    run touch /var/log/myapp.log
    assert_file_exists "/var/log/myapp.log"
    assert_file_empty "/var/log/myapp.log"
}
```

### `assert_file_permissions(file, perms)`
Test file permissions (octal format).
```bash
@test "secure file permissions" {
    run install_credentials.sh
    assert_file_permissions "/etc/myapp/secrets.key" "600"
    assert_file_permissions "/etc/myapp/config.yml" "644"
}
```

---

## Environment Assertions

### `assert_env_set(variable)`
Test that environment variable is set (non-empty).
```bash
@test "required environment configured" {
    setup_resource_test "ollama"
    assert_env_set "OLLAMA_PORT"
    assert_env_set "OLLAMA_BASE_URL"
}
```

### `assert_env_equals(variable, value)`
Test that environment variable equals specific value.
```bash
@test "correct port assignment" {
    setup_resource_test "ollama"
    assert_env_equals "OLLAMA_PORT" "11434"
    assert_env_equals "RESOURCE_NAME" "ollama"
}
```

### `assert_env_unset(variable)`
Test that environment variable is NOT set.
```bash
@test "sensitive data not exposed" {
    assert_env_unset "DATABASE_PASSWORD"
    assert_env_unset "API_SECRET_KEY"
}
```

---

## Command Assertions

### `assert_command_exists(command)`
Test that command is available in PATH.
```bash
@test "required tools installed" {
    assert_command_exists "docker"
    assert_command_exists "curl"
    assert_command_exists "jq"
}
```

### `assert_function_exists(function)`
Test that bash function is defined.
```bash
@test "helper functions loaded" {
    source helper_functions.sh
    assert_function_exists "setup_database"
    assert_function_exists "cleanup_temp_files"
}
```

### `assert_command_called(command, [args])`
Test that mock command was called (requires mock tracking).
```bash
@test "script calls expected commands" {
    setup_standard_mocks
    run my_deployment_script.sh
    
    assert_command_called "docker" "build"
    assert_command_called "curl" "-X POST"
}
```

---

## JSON Assertions

### `assert_json_valid(json_string)`
Test that string is valid JSON.
```bash
@test "API returns valid JSON" {
    run curl -s http://localhost:8080/api/status
    assert_success
    assert_json_valid "$output"
}
```

### `assert_json_field_equals(json, field, value)`
Test that JSON field equals specific value.
```bash
@test "API status check" {
    local response=$(curl -s http://localhost:8080/health)
    assert_json_field_equals "$response" ".status" "healthy"
    assert_json_field_equals "$response" ".version" "1.0.0"
}
```

### `assert_json_field_exists(json, field)`
Test that JSON field exists (not null).
```bash
@test "required fields present" {
    local config=$(cat config.json)
    assert_json_field_exists "$config" ".database.host"
    assert_json_field_exists "$config" ".api.endpoints"
}
```

### `assert_json_schema_valid(json, schema_file)`
Test JSON against schema file (basic validation).
```bash
@test "configuration schema valid" {
    local config=$(cat config.json)
    assert_json_schema_valid "$config" "/schema/config.schema.json"
}
```

---

## Network Assertions

### `assert_port_available(port)`
Test that port is NOT in use.
```bash
@test "port available for service" {
    assert_port_available "8080"
    # Now we can safely start our service
}
```

### `assert_port_in_use(port)`
Test that port IS in use.
```bash
@test "service started successfully" {
    start_my_service.sh
    assert_port_in_use "8080"
}
```

### `assert_http_endpoint_reachable(url, [timeout])`
Test that HTTP endpoint responds (any status).
```bash
@test "service endpoint accessible" {
    start_service.sh
    assert_http_endpoint_reachable "http://localhost:8080/health" 10
}
```

### `assert_http_status(url, status, [timeout])`
Test that HTTP endpoint returns specific status.
```bash
@test "API endpoints return correct status" {
    assert_http_status "http://localhost:8080/health" "200"
    assert_http_status "http://localhost:8080/nonexistent" "404"
}
```

---

## Resource Assertions

### `assert_resource_healthy(resource, [port], [url])`
Comprehensive health check for Vrooli resources.
```bash
@test "ollama resource health" {
    setup_resource_test "ollama"
    assert_resource_healthy "ollama"
    # Checks: port in use, HTTP endpoint, container running
}
```

### `assert_docker_container_running(container_name)`
Test that Docker container is running.
```bash
@test "containers are running" {
    setup_integration_test "ollama" "whisper"
    assert_docker_container_running "$OLLAMA_CONTAINER_NAME"
    assert_docker_container_running "$WHISPER_CONTAINER_NAME"
}
```

### `assert_docker_container_healthy(container_name)`
Test Docker container health status.
```bash
@test "container health checks pass" {
    start_services.sh
    assert_docker_container_healthy "myapp_web"
    assert_docker_container_healthy "myapp_db"
}
```

### `assert_api_response_valid(endpoint, [fields], [timeout])`
Test API endpoint returns valid JSON with expected fields.
```bash
@test "API response structure" {
    assert_api_response_valid "http://localhost:8080/api/status" "status,version,uptime"
}
```

### `assert_resource_chain_working(source, target, [data])`
Test data flow between resources (integration testing).
```bash
@test "AI pipeline integration" {
    setup_integration_test "whisper" "ollama" "n8n"
    assert_resource_chain_working "whisper" "ollama" "test_audio_data"
}
```

---

## Data Structure Assertions

### `assert_arrays_equal(array1_name, array2_name)`
Test that two arrays contain same elements in same order.
```bash
@test "configuration arrays match" {
    local expected=("host1" "host2" "host3")
    local actual=($(get_configured_hosts))
    assert_arrays_equal expected actual
}
```

### `assert_array_contains(array_name, element)`
Test that array contains specific element.
```bash
@test "required services configured" {
    local services=($(get_enabled_services))
    assert_array_contains services "docker"
    assert_array_contains services "nginx"
}
```

---

## Mock Assertions

### `assert_mock_used(mock_name)`
Test that specific mock was used during test.
```bash
@test "script uses expected tools" {
    setup_standard_mocks
    run deployment_script.sh
    assert_mock_used "docker"
    assert_mock_used "curl"
}
```

### `assert_mock_response_used(mock_file)`
Test that specific mock response file was accessed.
```bash
@test "mock responses utilized" {
    setup_resource_test "ollama"
    run test_ollama_integration.sh
    assert_mock_response_used "ollama_health_response.json"
}
```

---

## 🎯 Common Patterns

### Health Check Pattern
```bash
@test "complete service health check" {
    setup_resource_test "ollama"
    
    # Environment
    assert_env_set "OLLAMA_PORT"
    assert_env_equals "OLLAMA_PORT" "11434"
    
    # Network
    assert_port_in_use "$OLLAMA_PORT"
    assert_http_status "$OLLAMA_BASE_URL/health" "200"
    
    # Container
    assert_docker_container_running "$OLLAMA_CONTAINER_NAME"
    
    # API
    local response=$(curl -s "$OLLAMA_BASE_URL/api/tags")
    assert_json_valid "$response"
    assert_json_field_exists "$response" ".models"
}
```

### Configuration Validation Pattern
```bash
@test "configuration validation" {
    run configure_service.sh
    assert_success
    
    # Files created
    assert_file_exists "/etc/myapp/config.yml"
    assert_file_permissions "/etc/myapp/config.yml" "644"
    
    # Content validation
    assert_file_contains "/etc/myapp/config.yml" "port: 8080"
    
    # JSON structure
    local config=$(yq eval -o=json /etc/myapp/config.yml)
    assert_json_valid "$config"
    assert_json_field_equals "$config" ".server.port" "8080"
}
```

### Integration Testing Pattern
```bash
@test "multi-service integration" {
    setup_integration_test "postgres" "redis" "api"
    
    # All services healthy
    assert_resource_healthy "postgres"
    assert_resource_healthy "redis"
    assert_resource_healthy "api"
    
    # Cross-service communication
    assert_api_response_valid "$API_BASE_URL/health" "database,cache,status"
    
    # Data flow test
    local test_data='{"test": true}'
    run curl -X POST "$API_BASE_URL/data" -d "$test_data"
    assert_success
    assert_json_valid "$output"
}
```

---

## 🚀 Performance Tips

1. **Use specific assertions**: `assert_env_equals` is faster than `assert_env_set` + manual comparison
2. **Batch JSON checks**: Parse JSON once, then multiple field checks
3. **Cache command outputs**: Store `curl` results in variables for multiple assertions
4. **Use mock tracking**: `assert_command_called` is faster than parsing logs manually

## 🔧 Troubleshooting

| Error | Likely Cause | Solution |
|-------|--------------|----------|
| "Command call tracking not available" | Mocks not loaded | Call `setup_standard_mocks` in setup() |
| "Invalid JSON" | API returned HTML/error | Check `assert_http_status` first |
| "Expected environment variable" | Resource not configured | Use correct `setup_resource_test` |
| "Port not in use" | Service not started | Check container/service status |

## 📚 Next Steps

- **Learn setup modes**: [Setup Guide](setup-guide.md)
- **Understand mocks**: [Mock Registry](mock-registry.md)  
- **See examples**: [Examples Directory](examples/)
- **Resource patterns**: [Resource Testing](resource-testing.md)