# Resource Testing Guide: Best Practices for Vrooli Resources

Comprehensive guide for testing AI, automation, storage, and other Vrooli resources.

## 🎯 Overview

This guide covers testing patterns for all Vrooli resource categories:
- **AI Resources** (Ollama, Whisper, ComfyUI)
- **Automation Resources** (N8N, Node-RED, Huginn) 
- **Storage Resources** (Qdrant, MinIO, PostgreSQL)
- **Agent Resources** (Agent-S2, Browserless, Claude Code)
- **Search Resources** (SearXNG)
- **Execution Resources** (Judge0)

## 🏗️ Resource Categories

### AI Resources
AI-powered services for language, speech, and image processing.

| Resource | Purpose | Default Port | Key APIs |
|----------|---------|--------------|----------|
| **Ollama** | LLM inference | 11434 | `/api/generate`, `/api/tags` |
| **Whisper** | Speech-to-text | 8090 | `/transcribe`, `/translate` |
| **ComfyUI** | Image generation | 8188 | `/api/prompt`, `/api/queue` |
| **Unstructured-IO** | Document processing | 8000 | `/general/v0/general` |

### Automation Resources
Workflow and process automation tools.

| Resource | Purpose | Default Port | Key APIs |
|----------|---------|--------------|----------|
| **N8N** | Workflow automation | 5678 | `/api/v1/workflows`, `/healthz` |
| **Node-RED** | Visual programming | 1880 | `/flows`, `/settings` |
| **Huginn** | Agent automation | 3000 | `/agents`, `/events` |
| **Windmill** | Developer automation | 8000 | `/api/scripts`, `/api/flows` |

### Storage Resources
Data storage and management services.

| Resource | Purpose | Default Port | Key APIs |
|----------|---------|--------------|----------|
| **Qdrant** | Vector database | 6333 | `/collections`, `/points` |
| **MinIO** | Object storage | 9000 | `/minio/health/live` |
| **PostgreSQL** | Relational database | 5432 | SQL interface |
| **Redis** | Key-value cache | 6379 | Redis protocol |

## 🚀 Basic Resource Testing Patterns

### Standard Resource Test Template
```bash
#!/usr/bin/env bats
# Template for testing any Vrooli resource

bats_require_minimum_version 1.5.0
# Source from environment variable or relative path
source "${VROOLI_TEST_FIXTURES_DIR:-./fixtures/bats}/core/common_setup.bash"

setup() {
    setup_resource_test "RESOURCE_NAME"
}

teardown() {
    cleanup_mocks
}

@test "resource environment configured" {
    assert_env_set "RESOURCE_NAME_PORT"
    assert_env_set "RESOURCE_NAME_BASE_URL"
    assert_env_set "RESOURCE_NAME_CONTAINER_NAME"
}

@test "resource is healthy" {
    assert_resource_healthy "RESOURCE_NAME"
}

@test "resource container is running" {
    assert_docker_container_running "$RESOURCE_NAME_CONTAINER_NAME"
}

@test "resource API responds" {
    assert_http_status "$RESOURCE_NAME_BASE_URL/health" "200"
}
```

## 🤖 AI Resource Testing

### Ollama Testing Pattern
```bash
#!/usr/bin/env bats
# Ollama LLM Service Testing

# Source from environment variable or relative path
source "${VROOLI_TEST_FIXTURES_DIR:-./fixtures/bats}/core/common_setup.bash"

setup() {
    setup_resource_test "ollama"
}

teardown() {
    cleanup_mocks
}

@test "ollama environment configured" {
    assert_env_equals "OLLAMA_PORT" "11434"
    assert_env_equals "OLLAMA_BASE_URL" "http://localhost:11434"
    assert_string_contains "$OLLAMA_CONTAINER_NAME" "ollama"
}

@test "ollama models API" {
    run curl -s "$OLLAMA_BASE_URL/api/tags"
    assert_success
    assert_json_valid "$output"
    assert_json_field_exists "$output" ".models"
    
    # Test model availability
    local model_count=$(echo "$output" | jq '.models | length')
    assert_greater_than "$model_count" "0"
}

@test "ollama generation API" {
    local request='{
        "model": "llama3.1:8b",
        "prompt": "Hello world",
        "stream": false
    }'
    
    run curl -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d "$request"
    
    assert_success
    assert_json_valid "$output"
    assert_json_field_exists "$output" ".response"
    assert_json_field_equals "$output" ".done" "true"
}

@test "ollama pull API" {
    run curl -X POST "$OLLAMA_BASE_URL/api/pull" \
        -H "Content-Type: application/json" \
        -d '{"name": "llama3.1:8b"}'
    
    assert_success
    assert_json_valid "$output"
    assert_json_field_exists "$output" ".status"
}

@test "ollama error handling" {
    # Test invalid model request
    run curl -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model": "nonexistent", "prompt": "test"}'
    
    # Should handle gracefully (mock returns valid response)
    assert_success
}
```

### Whisper Testing Pattern
```bash
@test "whisper transcription API" {
    local request='{
        "audio_data": "mock_audio_base64",
        "language": "en"
    }'
    
    run curl -X POST "$WHISPER_BASE_URL/transcribe" \
        -H "Content-Type: application/json" \
        -d "$request"
    
    assert_success
    assert_json_valid "$output"
    assert_json_field_exists "$output" ".text"
    assert_json_field_exists "$output" ".language"
    assert_json_field_exists "$output" ".duration"
}

@test "whisper translation API" {
    local request='{
        "audio_data": "mock_audio_base64",
        "target_language": "en"
    }'
    
    run curl -X POST "$WHISPER_BASE_URL/translate" \
        -H "Content-Type: application/json" \
        -d "$request"
    
    assert_success
    assert_json_valid "$output"
    assert_json_field_exists "$output" ".text"
    assert_json_field_equals "$output" ".language" "en"
}
```

## 🔄 Automation Resource Testing

### N8N Testing Pattern
```bash
@test "n8n workflow management" {
    # Test workflow listing
    run curl -s "$N8N_BASE_URL/api/v1/workflows"
    assert_success
    assert_json_valid "$output"
    
    # Test workflow creation
    local workflow='{
        "name": "Test Workflow",
        "nodes": [
            {"type": "webhook", "position": [100, 100]},
            {"type": "set", "position": [300, 100]}
        ],
        "connections": {}
    }'
    
    run curl -X POST "$N8N_BASE_URL/api/v1/workflows" \
        -H "Content-Type: application/json" \
        -d "$workflow"
    
    assert_success
    assert_json_valid "$output"
}

@test "n8n health and status" {
    run curl -s "$N8N_BASE_URL/healthz"
    assert_success
    assert_json_field_equals "$output" ".status" "ok"
}

@test "n8n execution API" {
    # Test workflow execution (mock)
    local execution_data='{"test": true}'
    
    run curl -X POST "$N8N_BASE_URL/api/v1/workflows/1/execute" \
        -H "Content-Type: application/json" \
        -d "$execution_data"
    
    assert_success
}
```

### Node-RED Testing Pattern
```bash
@test "node-red flow management" {
    # Test flow retrieval
    run curl -s "$NODE_RED_BASE_URL/flows"
    assert_success
    assert_json_valid "$output"
    
    # Test settings
    run curl -s "$NODE_RED_BASE_URL/settings"
    assert_success
    assert_json_valid "$output"
    assert_json_field_exists "$output" ".version"
}
```

## 💾 Storage Resource Testing

### Qdrant Testing Pattern
```bash
@test "qdrant collections API" {
    # Test collection listing
    run curl -s "$QDRANT_BASE_URL/collections"
    assert_success
    assert_json_valid "$output"
    assert_json_field_exists "$output" ".result.collections"
    
    # Test collection creation
    local collection_config='{
        "vectors": {
            "size": 384,
            "distance": "Cosine"
        }
    }'
    
    run curl -X PUT "$QDRANT_BASE_URL/collections/test_collection" \
        -H "Content-Type: application/json" \
        -d "$collection_config"
    
    assert_success
    assert_json_field_equals "$output" ".status" "ok"
}

@test "qdrant health check" {
    run curl -s "$QDRANT_BASE_URL/health"
    assert_success
    assert_json_field_equals "$output" ".status" "ok"
    assert_json_field_exists "$output" ".version"
}

@test "qdrant points API" {
    local points_data='{
        "points": [
            {"id": 1, "vector": [0.1, 0.2, 0.3], "payload": {"test": true}}
        ]
    }'
    
    run curl -X PUT "$QDRANT_BASE_URL/collections/test/points" \
        -H "Content-Type: application/json" \
        -d "$points_data"
    
    assert_success
}
```

### MinIO Testing Pattern
```bash
@test "minio health and liveness" {
    run curl -s "$MINIO_BASE_URL/minio/health/live"
    assert_success
    assert_json_field_equals "$output" ".status" "ok"
}

@test "minio readiness check" {
    run curl -s "$MINIO_BASE_URL/minio/health/ready"
    assert_success
}
```

## 🔗 Integration Testing Patterns

### AI Pipeline Integration
```bash
#!/usr/bin/env bats
# Test AI processing pipeline: Audio → Whisper → Ollama

setup() {
    setup_integration_test "whisper" "ollama"
}

@test "audio processing pipeline" {
    # Step 1: Transcribe audio with Whisper
    local audio_request='{"audio_data": "mock_audio", "language": "en"}'
    local transcription=$(curl -X POST "$WHISPER_BASE_URL/transcribe" \
        -H "Content-Type: application/json" \
        -d "$audio_request")
    
    assert_json_valid "$transcription"
    local transcribed_text=$(echo "$transcription" | jq -r '.text')
    assert_not_empty "$transcribed_text"
    
    # Step 2: Process with Ollama
    local ollama_request='{
        "model": "llama3.1:8b",
        "prompt": "Summarize: '"$transcribed_text"'",
        "stream": false
    }'
    
    local summary=$(curl -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d "$ollama_request")
    
    assert_json_valid "$summary"
    assert_json_field_equals "$summary" ".done" "true"
    
    local summary_text=$(echo "$summary" | jq -r '.response')
    assert_not_empty "$summary_text"
}
```

### Automation + Storage Integration
```bash
@test "data pipeline: N8N + Qdrant" {
    setup_integration_test "n8n" "qdrant"
    
    # Create N8N workflow that writes to Qdrant
    local workflow='{
        "name": "Data Processing Pipeline",
        "nodes": [
            {"type": "webhook", "url": "/webhook/data"},
            {"type": "qdrant-insert", "collection": "processed_data"}
        ]
    }'
    
    # Create workflow
    run curl -X POST "$N8N_BASE_URL/api/v1/workflows" \
        -H "Content-Type: application/json" \
        -d "$workflow"
    assert_success
    
    # Test data flow
    local test_data='{"vector": [0.1, 0.2, 0.3], "metadata": {"source": "test"}}'
    
    # Simulate workflow execution
    run curl -X POST "$N8N_BASE_URL/webhook/data" \
        -H "Content-Type: application/json" \
        -d "$test_data"
    assert_success
}
```

## 🎭 Resource State Management

### Setting Resource States
```bash
setup() {
    setup_resource_test "ollama"
    
    # Configure specific resource state
    mock::http::set_endpoint_state "$OLLAMA_BASE_URL" "healthy"
    mock::docker::set_container_state "$OLLAMA_CONTAINER_NAME" "running"
}

@test "resource with custom state" {
    assert_resource_healthy "ollama"
}
```

### Testing Failure Scenarios
```bash
@test "resource failure handling" {
    # Simulate unhealthy resource
    mock::http::set_endpoint_state "$OLLAMA_BASE_URL" "unhealthy"
    
    # Test graceful failure
    run curl -s "$OLLAMA_BASE_URL/health"
    assert_failure
    
    # Verify error handling
    local status_code=$(curl -s -o /dev/null -w "%{http_code}" "$OLLAMA_BASE_URL/health")
    assert_equals "$status_code" "503"
}

@test "container failure scenarios" {
    # Simulate stopped container
    mock::docker::set_container_state "$OLLAMA_CONTAINER_NAME" "stopped"
    
    # Test container not running
    run docker inspect "$OLLAMA_CONTAINER_NAME" --format '{{.State.Status}}'
    assert_output_equals "stopped"
}
```

## ⚡ Performance Testing

### Resource Startup Time
```bash
@test "resource startup performance" {
    local start_time=$(date +%s%3N)
    
    setup_resource_test "ollama"
    assert_resource_healthy "ollama"
    
    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    
    # Should start in under 500ms with mocks
    assert_less_than "$duration" "500"
}
```

### API Response Time
```bash
@test "API response performance" {
    setup_resource_test "ollama"
    
    local start_time=$(date +%s%3N)
    
    run curl -s "$OLLAMA_BASE_URL/api/tags"
    assert_success
    
    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    
    # API should respond in under 100ms with mocks
    assert_less_than "$duration" "100"
}
```

### Load Testing Pattern
```bash
@test "concurrent API requests" {
    setup_resource_test "ollama"
    
    # Simulate multiple concurrent requests
    for i in {1..10}; do
        curl -s "$OLLAMA_BASE_URL/health" > /dev/null &
    done
    wait
    
    # Resource should still be healthy
    assert_resource_healthy "ollama"
}
```

## 🔧 Custom Resource Testing

### Adding New Resource Support
```bash
# 1. Add to resource category detection
mock::detect_resource_category() {
    case "$resource" in
        "my-new-resource")
            echo "custom"
            ;;
        # ... existing cases
    esac
}

# 2. Create resource mock file
# /mocks/resources/custom/my-new-resource.bash

# 3. Add environment configuration
mock::configure_resource_environment() {
    case "$resource" in
        "my-new-resource")
            export MY_RESOURCE_PORT="${MY_RESOURCE_PORT:-8080}"
            export MY_RESOURCE_BASE_URL="http://localhost:${MY_RESOURCE_PORT}"
            ;;
        # ... existing cases
    esac
}

# 4. Test the new resource
@test "my new resource works" {
    setup_resource_test "my-new-resource"
    assert_resource_healthy "my-new-resource"
}
```

## 📊 Resource Monitoring

### Health Check Aggregation
```bash
@test "multi-resource health dashboard" {
    setup_integration_test "ollama" "whisper" "qdrant" "n8n"
    
    local health_report=""
    for resource in "ollama" "whisper" "qdrant" "n8n"; do
        if assert_resource_healthy "$resource" 2>/dev/null; then
            health_report+="$resource: healthy\n"
        else
            health_report+="$resource: unhealthy\n"
        fi
    done
    
    echo -e "$health_report"
    
    # Ensure critical resources are healthy
    assert_resource_healthy "ollama"
    assert_resource_healthy "qdrant"
}
```

### Resource Dependencies
```bash
@test "resource dependency chain" {
    setup_integration_test "postgres" "qdrant" "ollama" "n8n"
    
    # Test dependency order
    assert_resource_healthy "postgres"    # Foundation
    assert_resource_healthy "qdrant"      # Depends on storage
    assert_resource_healthy "ollama"      # Independent
    assert_resource_healthy "n8n"         # Orchestrates others
    
    # Test cross-dependencies
    assert_resource_chain_working "ollama" "qdrant"
    assert_resource_chain_working "qdrant" "n8n"
}
```

## 🎯 Best Practices

### 1. Test Structure
```bash
# Good: Logical test organization
@test "environment setup"      # Test configuration
@test "health checks"          # Test basic functionality  
@test "API functionality"      # Test core features
@test "error handling"         # Test failure scenarios
@test "integration"            # Test with other resources
```

### 2. Assertion Patterns
```bash
# Good: Specific, descriptive assertions
assert_env_equals "OLLAMA_PORT" "11434"
assert_json_field_exists "$response" ".models[0].name"
assert_resource_healthy "ollama"

# Bad: Generic assertions
assert_success
assert_not_empty "$output"
```

### 3. Performance Considerations
```bash
# Good: Minimal setup for simple tests
setup_resource_test "ollama"

# Bad: Over-engineering for simple tests
setup_integration_test "ollama" "whisper" "n8n" "qdrant" "postgres"
```

### 4. Mock Realism
```bash
# Good: Realistic mock responses
mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" \
    '{"models":[{"name":"llama3.1:8b","size":4900000000}]}'

# Bad: Overly simple mocks
mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" '{"status":"ok"}'
```

## 📚 Next Steps

- **Start with examples**: Try [resource examples](examples/resource-test.bats)
- **Learn setup modes**: Review [Setup Guide](setup-guide.md)
- **Understand mocks**: Read [Mock Registry](mock-registry.md)
- **Debug issues**: Check [Troubleshooting](troubleshooting.md)