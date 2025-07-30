# Mock Pattern Standardization Guide

This guide establishes consistent mock patterns across all Vrooli resource tests to ensure reliability, maintainability, and predictable behavior.

## 🎯 Overview

Standardized mocking enables:
- **Isolated Testing**: No external dependencies during tests
- **Predictable Behavior**: Consistent mock responses across all resources
- **Fast Execution**: No network calls or real Docker operations
- **Reliable Results**: Tests pass consistently regardless of environment

## 🏗️ Mock Architecture

### **Three-Tier Mock System**
1. **system_mocks.bash** - Core system commands (docker, curl, jq)
2. **mock_helpers.bash** - Configuration and utility mocks
3. **resource_mocks.bash** - Resource-specific mock setups

### **Shared Infrastructure Integration**
All tests must use the standardized mock framework via `common_setup.bash`:

```bash
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks - REQUIRED for all tests
    setup_standard_mocks
    
    # Resource-specific setup follows...
}
```

## 📋 **MANDATORY MOCK PATTERNS**

### **1. Standard Setup Pattern**
Every test file must follow this exact pattern:

```bash
#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

setup() {
    # 1. Load shared infrastructure (REQUIRED)
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # 2. Setup standard mocks (REQUIRED)
    setup_standard_mocks
    
    # 3. Set test environment variables
    export RESOURCE_PORT="8080"
    export RESOURCE_CONTAINER_NAME="resource-test"
    export RESOURCE_BASE_URL="http://localhost:8080"
    
    # 4. Configure resource-specific mocks
    mock::docker::set_container_state "resource-test" "running"
    mock::http::set_endpoint_state "http://localhost:8080" "healthy"
    
    # 5. Load configuration and functions
    source "${RESOURCE_DIR}/config/defaults.sh"
    source "${RESOURCE_DIR}/lib/module.sh"
}

teardown() {
    # Clean up test environment (REQUIRED)
    cleanup_mocks
}
```

### **2. Docker Mock Standardization**
All resources must use consistent Docker mocking:

```bash
# Container state management
mock::docker::set_container_state "container-name" "running|stopped|missing"
mock::docker::set_container_id "container-name" "abc123def456"
mock::docker::set_container_stats "container-name" "cpu_percent" "memory_used" "memory_limit"

# Image management
mock::docker::set_image_exists "image:tag" "true|false"
mock::docker::set_pull_success "image:tag"
mock::docker::set_pull_failure "image:tag" "error_message"

# Container operations
mock::docker::set_run_success "image:tag" "container_id"
mock::docker::set_run_failure "image:tag" "error_message"
```

### **3. HTTP Mock Standardization**
All resources must use consistent HTTP endpoint mocking:

```bash
# Basic endpoint responses
mock::http::set_endpoint_response "http://localhost:8080/health" "200" '{"status":"healthy"}'
mock::http::set_endpoint_response "http://localhost:8080/api/v1/info" "404" '{"error":"Not Found"}'

# Dynamic response sequences (for retry testing)
mock::http::set_endpoint_sequence "http://localhost:8080/health" "503,503,200" "unhealthy,unhealthy,healthy"

# Response delays (for timeout testing)
mock::http::set_endpoint_delay "http://localhost:8080/slow" 10

# Connection failures
mock::http::set_endpoint_response "http://localhost:8080/fail" "connection_refused" ""
```

### **4. System Command Mock Standardization**
All resources must use consistent system command mocking:

```bash
# Process management
mock::system::set_process_running "process_name" "true|false"
mock::system::set_port_status "8080" "available|in_use|in_use_other"

# File system operations
mock::system::set_file_exists "/path/to/file" "true|false"
mock::system::set_directory_writable "/path/to/dir" "true|false"

# Network operations
mock::system::set_network_reachable "hostname:port" "true|false"
```

## 🔧 **RESOURCE-SPECIFIC MOCK PATTERNS**

### **Configuration Mock Pattern**
```bash
# In setup() function - configure healthy defaults
export RESOURCE_PORT="8080"
export RESOURCE_CONTAINER_NAME="resource-test"
export RESOURCE_BASE_URL="http://localhost:8080"

# Set healthy state by default
mock::docker::set_container_state "$RESOURCE_CONTAINER_NAME" "running"
mock::http::set_endpoint_response "$RESOURCE_BASE_URL/health" "200" '{"status":"healthy"}'
mock::http::set_endpoint_response "$RESOURCE_BASE_URL/api/version" "200" '{"version":"1.0.0"}'
```

### **Resource Type-Specific Patterns**

#### **AI Resources Pattern**
```bash
# AI service specific mocks
mock::http::set_endpoint_response "http://localhost:11434/api/tags" "200" '{"models":["llama3.1:8b","codellama:7b"]}'
mock::http::set_endpoint_response "http://localhost:11434/api/generate" "200" '{"response":"Generated text"}'
mock::system::set_gpu_available "true"
```

#### **Automation Resources Pattern**
```bash
# Automation service specific mocks
mock::http::set_endpoint_response "http://localhost:5678/api/workflows" "200" '{"workflows":[{"id":"wf1","name":"Test"}]}'
mock::http::set_endpoint_response "http://localhost:5678/api/executions" "200" '{"executions":[]}'
mock::system::set_database_connection "true"
```

#### **Storage Resources Pattern**
```bash
# Storage service specific mocks
mock::http::set_endpoint_response "http://localhost:9000/minio/health/ready" "200" "OK"
mock::http::set_endpoint_response "http://localhost:9000/api/buckets" "200" '{"buckets":["test-bucket"]}'
mock::system::set_disk_space "/data" "10GB" "available"
```

#### **Agent Resources Pattern**
```bash
# Agent service specific mocks
mock::http::set_endpoint_response "http://localhost:4113/health" "200" '{"status":"ready","version":"1.0.0"}'
mock::http::set_endpoint_response "http://localhost:4113/ai/task" "200" '{"task_id":"task123","status":"completed"}'
mock::system::set_display_available "true"
```

## 📝 **STANDARDIZED TEST SCENARIOS**

### **Health Check Test Pattern**
```bash
@test "resource::health_check should return healthy when service is running" {
    # Arrange - service is healthy (from setup)
    
    # Act
    run resource::health_check
    
    # Assert
    [ "$status" -eq 0 ]
    [[ "$output" =~ "healthy" ]]
}

@test "resource::health_check should handle service unavailable" {
    # Arrange - mock unhealthy service
    mock::http::set_endpoint_response "$RESOURCE_BASE_URL/health" "503" '{"status":"unhealthy"}'
    
    # Act
    run resource::health_check
    
    # Assert
    [ "$status" -ne 0 ]
    [[ "$output" =~ "unhealthy" || "$output" =~ "unavailable" ]]
}
```

### **Installation Test Pattern**
```bash
@test "resource::install should complete successfully" {
    # Arrange
    export YES="yes"
    mock::docker::set_pull_success "$RESOURCE_IMAGE"
    mock::docker::set_run_success "$RESOURCE_IMAGE" "container123"
    
    # Act
    run resource::install
    
    # Assert
    [ "$status" -eq 0 ]
    [[ "$output" =~ "installed" || "$output" =~ "complete" ]]
}

@test "resource::install should handle Docker pull failure" {
    # Arrange
    export YES="yes"
    mock::docker::set_pull_failure "$RESOURCE_IMAGE" "network error"
    
    # Act
    run resource::install
    
    # Assert
    [ "$status" -ne 0 ]
    [[ "$output" =~ "failed" || "$output" =~ "error" ]]
}
```

### **API Integration Test Pattern**
```bash
@test "resource::api_request should handle authentication" {
    # Arrange
    export RESOURCE_API_KEY="test-key"
    mock::http::set_endpoint_response "$RESOURCE_BASE_URL/api/secure" "200" '{"authorized":true}'
    
    # Act
    run resource::api_request "GET" "/api/secure"
    
    # Assert
    [ "$status" -eq 0 ]
    [[ "$output" =~ "authorized" ]]
    
    # Verify auth header was sent (optional but recommended)
    grep -q "Authorization: Bearer test-key" "$MOCK_RESPONSES_DIR/http_requests.log"
}
```

## 🚨 **CRITICAL STANDARDIZATION REQUIREMENTS**

### **1. Mock State Management**
```bash
# DO: Use state-based mocking
mock::docker::set_container_state "myservice" "running"

# DON'T: Mock individual commands inconsistently
docker() { echo "Container running"; }
```

### **2. Endpoint Response Consistency**
```bash
# DO: Use consistent JSON response format
mock::http::set_endpoint_response "http://localhost:8080/health" "200" '{"status":"healthy","timestamp":"2024-01-01T00:00:00Z"}'

# DON'T: Use inconsistent formats
curl() { echo "OK"; }  # Too simplistic
```

### **3. Error Scenario Coverage**
```bash
# DO: Test multiple error scenarios
@test "should handle connection refused" {
    mock::http::set_endpoint_response "http://localhost:8080/api" "connection_refused" ""
    # Test implementation
}

@test "should handle timeout" {
    mock::http::set_endpoint_delay "http://localhost:8080/api" 10
    # Test implementation
}

@test "should handle server error" {
    mock::http::set_endpoint_response "http://localhost:8080/api" "500" '{"error":"Internal Server Error"}'
    # Test implementation
}
```

### **4. Resource Cleanup**
```bash
# DO: Always implement proper cleanup
teardown() {
    cleanup_mocks
    rm -rf "/tmp/test-*"
}

# DON'T: Leave test artifacts
teardown() {
    # Empty or missing cleanup
}
```

## 📊 **MOCK VALIDATION CHECKLIST**

### **Setup Validation**
- [ ] Uses `setup_standard_mocks()` from common_setup.bash
- [ ] Sets consistent environment variables
- [ ] Configures healthy default state
- [ ] Loads required configuration files

### **Mock Configuration**
- [ ] Docker container states properly mocked
- [ ] HTTP endpoints have realistic responses
- [ ] Error scenarios comprehensively covered
- [ ] System commands appropriately mocked

### **Test Implementation**
- [ ] Follows AAA pattern (Arrange, Act, Assert)
- [ ] Uses descriptive test names
- [ ] Tests both success and failure scenarios
- [ ] Proper assertions with meaningful messages

### **Cleanup Validation**
- [ ] Implements `teardown()` with `cleanup_mocks()`
- [ ] Removes temporary files and directories
- [ ] Resets environment state
- [ ] No test pollution between runs

## 🎯 **MIGRATION STRATEGY**

### **Phase 1: Assessment** (Per Resource)
1. Identify current mock patterns in use
2. Compare against standardized patterns
3. Plan migration approach
4. Estimate effort required

### **Phase 2: Implementation** (Per Resource)
1. Update setup() to use `setup_standard_mocks()`
2. Replace custom mocks with standardized patterns
3. Add missing error scenario mocks
4. Implement proper cleanup patterns

### **Phase 3: Validation** (Per Resource)
1. Run all tests to verify functionality
2. Check mock framework compliance
3. Validate error scenario coverage
4. Ensure no test pollution

## 🔍 **QUALITY ASSURANCE**

### **Mock Framework Compliance**
```bash
# Verify standardized mock usage
grep -r "setup_standard_mocks" resource/*/test/*.bats
grep -r "cleanup_mocks" resource/*/test/*.bats
grep -r "mock::docker::" resource/*/test/*.bats
```

### **Test Isolation Verification**
```bash
# Verify tests can run independently
bats resource/lib/test.bats --tap
bats resource/lib/test.bats::specific_test --tap
```

### **Performance Validation**
```bash
# Verify test execution time
time bats resource/lib/*.bats
```

## 📚 **REFERENCE IMPLEMENTATIONS**

### **Excellent Examples to Follow**
- **qdrant/lib/common.bats** - Perfect standardization example
- **browserless/lib/api.bats** - Excellent HTTP mock usage
- **ollama/lib/models.bats** - Complex workflow mocking
- **unstructured-io/config/defaults.bats** - Comprehensive configuration mocking

### **Templates Available**
- All templates in `/scripts/resources/tests/bats-fixtures/setup/resource_templates/`
- Use as starting point for consistent implementation
- Customize for resource-specific requirements

## 🚀 **EXPECTED OUTCOMES**

After standardization:
- **Consistent Behavior**: All tests use identical mock patterns
- **Improved Reliability**: Predictable test results across environments
- **Enhanced Maintainability**: Unified mock framework easier to update
- **Faster Development**: Standardized patterns accelerate new test creation
- **Better Quality**: Comprehensive error scenario coverage across all resources

This standardization ensures a robust, maintainable, and reliable testing ecosystem across all Vrooli resources.