# Setup Guide: Choosing the Right Test Environment

This guide explains the three setup modes available in the Vrooli BATS testing infrastructure and helps you choose the right one for your needs.

## Overview: Three Setup Modes

| Mode | Purpose | Performance | Complexity | Use Case |
|------|---------|-------------|------------|----------|
| **Standard** | Basic system mocking | Fastest | Lowest | Unit tests, simple scripts |
| **Resource** | Single resource testing | Fast | Medium | Resource integration tests |
| **Integration** | Multi-resource testing | Moderate | Higher | End-to-end workflows |

## 🚀 Standard Setup Mode

**When to use:** Testing scripts that interact with basic system commands (Docker, HTTP, system tools).

### Example Use Cases
- Testing Docker deployment scripts
- Testing HTTP API clients
- Testing system administration scripts
- Unit testing shell functions

### Setup Pattern
```bash
#!/usr/bin/env bats
source "/path/to/bats-fixtures/core/common_setup.bash"

setup() {
    setup_standard_mocks
}

teardown() {
    cleanup_mocks
}

@test "example test" {
    # Your test here
}
```

### What You Get
```bash
# Mocked Commands Available:
docker, curl, wget, jq, systemctl, git, ssh, openssl

# Environment Variables:
$TEST_TMPDIR          # Temporary directory for test files
$MOCK_RESPONSES_DIR   # Directory for mock tracking

# Basic Assertions:
assert_success, assert_failure, assert_output_contains
```

### Complete Example
```bash
#!/usr/bin/env bats

bats_require_minimum_version 1.5.0
source "/home/matthalloran8/Vrooli/scripts/tests/bats-fixtures/core/common_setup.bash"

setup() {
    setup_standard_mocks
}

teardown() {
    cleanup_mocks
}

@test "docker commands work" {
    run docker --version
    assert_success
    assert_output_contains "Docker version"
}

@test "HTTP requests work" {
    run curl -s http://example.com/health
    assert_success
    assert_json_valid "$output"
}

@test "system commands work" {
    run systemctl status docker
    assert_success
    assert_output_contains "active (running)"
}
```

## 🎯 Resource Setup Mode

**When to use:** Testing integration with a specific Vrooli resource (Ollama, Whisper, N8N, etc.).

### Example Use Cases
- Testing Ollama model interactions
- Testing Whisper transcription workflows
- Testing N8N automation triggers
- Resource health monitoring

### Setup Pattern
```bash
#!/usr/bin/env bats
source "/path/to/bats-fixtures/core/common_setup.bash"

setup() {
    setup_resource_test "resource_name"
}

teardown() {
    cleanup_mocks
}

@test "example test" {
    # Resource-specific environment is configured
    # Resource-specific mocks are loaded
    # Resource health can be checked
}
```

### Supported Resources
```bash
# AI Resources
setup_resource_test "ollama"           # Ollama LLM service
setup_resource_test "whisper"          # Whisper transcription
setup_resource_test "comfyui"          # ComfyUI image generation
setup_resource_test "unstructured-io"  # Document processing

# Automation Resources  
setup_resource_test "n8n"              # N8N automation
setup_resource_test "node-red"         # Node-RED flows
setup_resource_test "huginn"           # Huginn agents
setup_resource_test "windmill"         # Windmill workflows

# Storage Resources
setup_resource_test "qdrant"           # Vector database
setup_resource_test "minio"            # Object storage
setup_resource_test "postgres"         # PostgreSQL
setup_resource_test "redis"            # Redis cache

# Agent Resources
setup_resource_test "agent-s2"         # Agent S2
setup_resource_test "browserless"      # Browserless automation
setup_resource_test "claude-code"      # Claude Code agent

# Other Resources
setup_resource_test "searxng"          # Search engine
setup_resource_test "judge0"           # Code execution
```

### What You Get (Resource-Specific)
```bash
# For Ollama:
$OLLAMA_PORT=11434
$OLLAMA_BASE_URL="http://localhost:11434"
$OLLAMA_CONTAINER_NAME="test_12345_ollama"

# For Whisper:
$WHISPER_PORT=8090
$WHISPER_BASE_URL="http://localhost:8090"
$WHISPER_CONTAINER_NAME="test_12345_whisper"

# For N8N:
$N8N_PORT=5678
$N8N_BASE_URL="http://localhost:5678"
$N8N_CONTAINER_NAME="test_12345_n8n"

# Universal:
$RESOURCE_NAME           # The resource being tested
$TEST_NAMESPACE          # Unique test namespace
$CONTAINER_NAME_PREFIX   # Prefix for container names
```

### Complete Example
```bash
#!/usr/bin/env bats

bats_require_minimum_version 1.5.0
source "/home/matthalloran8/Vrooli/scripts/tests/bats-fixtures/core/common_setup.bash"

setup() {
    setup_resource_test "ollama"
}

teardown() {
    cleanup_mocks
}

@test "ollama environment is configured" {
    assert_env_set "OLLAMA_PORT"
    assert_env_set "OLLAMA_BASE_URL"
    assert_env_equals "OLLAMA_PORT" "11434"
}

@test "ollama service is healthy" {
    assert_resource_healthy "ollama"
}

@test "ollama container is running" {
    assert_docker_container_running "$OLLAMA_CONTAINER_NAME"
}

@test "ollama API responds correctly" {
    run curl -s "$OLLAMA_BASE_URL/api/tags"
    assert_success
    assert_json_valid "$output"
    assert_json_field_exists "$output" ".models"
}
```

## 🔗 Integration Setup Mode

**When to use:** Testing workflows that involve multiple resources working together.

### Example Use Cases
- AI pipeline: Whisper → Ollama → N8N
- Data pipeline: Postgres → Qdrant → Search
- Automation workflows with multiple tools
- End-to-end system testing

### Setup Pattern
```bash
#!/usr/bin/env bats
source "/path/to/bats-fixtures/core/common_setup.bash"

setup() {
    setup_integration_test "resource1" "resource2" "resource3"
}

teardown() {
    cleanup_mocks
}

@test "example test" {
    # All resources are configured
    # Cross-resource communication works
    # Complex workflows can be tested
}
```

### What You Get
- **All environment variables** for each resource
- **All resource-specific mocks** loaded
- **Coordinated test namespacing** (same namespace for all resources)
- **Port conflict avoidance** (each resource gets unique ports)

### Complete Example
```bash
#!/usr/bin/env bats

bats_require_minimum_version 1.5.0
source "/home/matthalloran8/Vrooli/scripts/tests/bats-fixtures/core/common_setup.bash"

setup() {
    setup_integration_test "ollama" "whisper" "n8n"
}

teardown() {
    cleanup_mocks
}

@test "all resources are configured" {
    assert_env_set "OLLAMA_PORT"
    assert_env_set "WHISPER_PORT"
    assert_env_set "N8N_PORT"
    
    # All resources share the same test namespace
    assert_env_set "TEST_NAMESPACE"
}

@test "all resources are healthy" {
    assert_resource_healthy "ollama"
    assert_resource_healthy "whisper" 
    assert_resource_healthy "n8n"
}

@test "cross-resource communication works" {
    # Test that N8N can call Ollama via Whisper
    local workflow_json='{
        "nodes": [
            {"type": "whisper", "url": "'$WHISPER_BASE_URL'"},
            {"type": "ollama", "url": "'$OLLAMA_BASE_URL'"}
        ]
    }'
    
    assert_json_valid "$workflow_json"
    
    # Test N8N workflow creation
    run curl -X POST "$N8N_BASE_URL/api/v1/workflows" \
        -H "Content-Type: application/json" \
        -d "$workflow_json"
    assert_success
}

@test "containers are isolated by namespace" {
    # All containers should use the same test namespace
    assert_docker_container_running "${TEST_NAMESPACE}_ollama"
    assert_docker_container_running "${TEST_NAMESPACE}_whisper"
    assert_docker_container_running "${TEST_NAMESPACE}_n8n"
}
```

## ⚡ Performance Considerations

### Startup Time Comparison
```bash
# Standard: ~50ms (minimal mocks)
setup_standard_mocks

# Resource: ~150ms (standard + resource-specific)
setup_resource_test "ollama"  

# Integration: ~300ms (all resources)
setup_integration_test "ollama" "whisper" "n8n"
```

### Memory Usage
- **Standard**: ~5MB (basic mocks only)
- **Resource**: ~10-15MB (adds resource mocks + environment)
- **Integration**: ~25-50MB (scales with number of resources)

### Optimization Tips
```bash
# 1. Use the minimal setup that meets your needs
setup_standard_mocks  # Not setup_resource_test if you don't need it

# 2. Use resource mode for single-resource tests
setup_resource_test "ollama"  # Not integration for single resource

# 3. Clean up properly to avoid memory leaks
teardown() {
    cleanup_mocks  # Always call this
}

# 4. Use performance mode for speed-critical tests
export TEST_PERFORMANCE_MODE="true"  # Disables sleep, reduces delays
```

## 🎯 Decision Tree

```
What are you testing?
├─ Shell script that uses Docker/HTTP/basic commands
│   └─ → setup_standard_mocks
├─ Integration with ONE Vrooli resource
│   └─ → setup_resource_test "resource_name"
├─ Workflow involving MULTIPLE resources
│   └─ → setup_integration_test "res1" "res2" "res3"
└─ Custom/complex scenario
    └─ → Start with setup_standard_mocks, then manually load specific mocks
```

## 🚀 Next Steps

1. **Try the examples**: Start with [basic examples](examples/) in the documentation
2. **Learn assertions**: Check out the [Assertions Reference](assertions.md)
3. **Understand mocks**: Read about the [Mock Registry](mock-registry.md)
4. **Advanced patterns**: Explore [Resource Testing](resource-testing.md)