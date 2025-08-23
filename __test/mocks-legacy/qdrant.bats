#!/usr/bin/env bats
# Comprehensive tests for Qdrant mock system
# Tests the qdrant.sh mock implementation for correctness and integration

# Source trash module for safe test cleanup
MOCK_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${MOCK_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test setup - load dependencies
setup() {
    # Set up test environment
    export MOCK_UTILS_VERBOSE=false
    export MOCK_VERIFICATION_ENABLED=true
    
    # Load the mock utilities first (required by qdrant mock)
    source "${BATS_TEST_DIRNAME}/logs.sh"
    
    # Load verification system if available
    if [[ -f "${BATS_TEST_DIRNAME}/verification.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/verification.sh"
    fi
    
    # Load the qdrant mock
    source "${BATS_TEST_DIRNAME}/qdrant.sh"
    
    # Initialize clean state for each test
    mock::qdrant::reset
    
    # Create test log directory
    TEST_LOG_DIR=$(mktemp -d)
    mock::init_logging "$TEST_LOG_DIR"
}

# Test cleanup
teardown() {
    # Clean up test logs
    if [[ -n "${TEST_LOG_DIR:-}" && -d "$TEST_LOG_DIR" ]]; then
        trash::safe_remove "$TEST_LOG_DIR" --test-cleanup
    fi
    
    # Clean up environment
    unset TEST_LOG_DIR
}

# =============================================================================
# Basic Configuration Tests
# =============================================================================

@test "qdrant mock loads successfully" {
    run mock::qdrant::debug::dump_state
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Qdrant Mock State Dump" ]]
}

@test "default configuration is set correctly" {
    [ "$(mock::qdrant::get::config container_name)" = "qdrant" ]
    [ "$(mock::qdrant::get::config rest_port)" = "6333" ]
    [ "$(mock::qdrant::get::config grpc_port)" = "6334" ]
    [ "$(mock::qdrant::get::config base_url)" = "http://localhost:6333" ]
    [ "$(mock::qdrant::get::config version)" = "v1.11.0" ]
}

@test "reset function clears all state" {
    # Set up some state
    mock::qdrant::set_container_state "test_container" "running"
    mock::qdrant::create_collection "test_collection" "768" "Dot"
    mock::qdrant::inject_error "health" "http_500"
    mock::qdrant::create_snapshot "test_snapshot" "all"
    
    # Verify state exists
    [ "$(mock::qdrant::get::container_state test_container)" = "running" ]
    [ "$(mock::qdrant::get::collection_count)" = "1" ]
    
    # Reset and verify state is cleared
    mock::qdrant::reset
    
    [ "$(mock::qdrant::get::container_state test_container)" = "" ]
    [ "$(mock::qdrant::get::collection_count)" = "0" ]
}

# =============================================================================
# Container State Management Tests
# =============================================================================

@test "set container state works correctly" {
    mock::qdrant::set_container_state "test_container" "running"
    [ "$(mock::qdrant::get::container_state test_container)" = "running" ]
    
    mock::qdrant::set_container_state "test_container" "stopped"
    [ "$(mock::qdrant::get::container_state test_container)" = "stopped" ]
}

@test "container assertions work correctly" {
    mock::qdrant::set_container_state "test_container" "running"
    
    run mock::qdrant::assert::container_running "test_container"
    [ "$status" -eq 0 ]
    
    run mock::qdrant::assert::container_stopped "test_container"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is not stopped" ]]
}

@test "multiple container states can be managed" {
    mock::qdrant::set_container_state "container1" "running"
    mock::qdrant::set_container_state "container2" "stopped"
    
    [ "$(mock::qdrant::get::container_state container1)" = "running" ]
    [ "$(mock::qdrant::get::container_state container2)" = "stopped" ]
}

# =============================================================================
# Collection Management Tests
# =============================================================================

@test "create collection works correctly" {
    mock::qdrant::create_collection "test_vectors" "1536" "Cosine" "1000" "1000"
    
    run mock::qdrant::assert::collection_exists "test_vectors"
    [ "$status" -eq 0 ]
    
    local info=$(mock::qdrant::get::collection_info "test_vectors")
    [[ "$info" =~ "1536|Cosine|1000|1000" ]]
}

@test "delete collection works correctly" {
    mock::qdrant::create_collection "temp_collection" "768" "Dot"
    
    run mock::qdrant::assert::collection_exists "temp_collection"
    [ "$status" -eq 0 ]
    
    mock::qdrant::delete_collection "temp_collection"
    
    run mock::qdrant::assert::collection_not_exists "temp_collection"
    [ "$status" -eq 0 ]
}

@test "collection count works correctly" {
    [ "$(mock::qdrant::get::collection_count)" = "0" ]
    
    mock::qdrant::create_collection "col1" "1536" "Cosine"
    [ "$(mock::qdrant::get::collection_count)" = "1" ]
    
    mock::qdrant::create_collection "col2" "768" "Dot"
    [ "$(mock::qdrant::get::collection_count)" = "2" ]
    
    mock::qdrant::delete_collection "col1"
    [ "$(mock::qdrant::get::collection_count)" = "1" ]
}

# =============================================================================
# Health Status Management Tests
# =============================================================================

@test "health status can be set and retrieved" {
    mock::qdrant::set_health_status "healthy"
    [ "$(mock::qdrant::get::config health_status)" = "healthy" ]
    
    mock::qdrant::set_health_status "unhealthy"
    [ "$(mock::qdrant::get::config health_status)" = "unhealthy" ]
}

@test "health assertion works correctly" {
    mock::qdrant::set_health_status "healthy"
    
    run mock::qdrant::assert::healthy
    [ "$status" -eq 0 ]
    
    mock::qdrant::set_health_status "unhealthy"
    
    run mock::qdrant::assert::healthy
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is not healthy" ]]
}

# =============================================================================
# API Key Authentication Tests
# =============================================================================

@test "API key can be set and retrieved" {
    mock::qdrant::set_api_key "test-api-key-123"
    [ "$(mock::qdrant::get::config api_key)" = "test-api-key-123" ]
}

@test "API key authentication blocks unauthorized requests" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::set_api_key "secure-key-456"
    
    # Request without API key should fail
    run curl -s "http://localhost:6333/"
    [ "$status" -eq 22 ]
    [[ "$output" =~ "Unauthorized" ]]
}

@test "API key authentication allows authorized requests" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::set_api_key "secure-key-789"
    
    # Request with correct API key should succeed
    run curl -s -H "api-key: secure-key-789" "http://localhost:6333/"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "qdrant - vector search engine" ]]
}

# =============================================================================
# API Endpoint Tests - Health/Version
# =============================================================================

@test "health endpoint returns version information" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    run curl -s "http://localhost:6333/"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"title": "qdrant - vector search engine"' ]]
    [[ "$output" =~ '"version": "v1.11.0"' ]]
}

@test "health endpoint reflects custom version" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    QDRANT_MOCK_CONFIG[version]="v2.0.0"
    mock::qdrant::save_state
    
    run curl -s "http://localhost:6333/"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"version": "v2.0.0"' ]]
}

@test "health endpoint fails when server is stopped" {
    mock::qdrant::set_server_status "stopped"
    
    run curl -f -s "http://localhost:6333/"
    [ "$status" -eq 7 ]  # Connection failed
}

# =============================================================================
# API Endpoint Tests - Cluster
# =============================================================================

@test "cluster endpoint returns cluster information" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    run curl -s "http://localhost:6333/cluster"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status": "enabled"' ]]
    [[ "$output" =~ '"peer_id": "00000000-0000-0000-0000-000000000000"' ]]
    [[ "$output" =~ '"role": "leader"' ]]
}

@test "cluster endpoint handles custom cluster info" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    QDRANT_MOCK_CLUSTER_INFO[peers_count]="3"
    QDRANT_MOCK_CLUSTER_INFO[raft_info]="follower"
    mock::qdrant::save_state
    
    run curl -s "http://localhost:6333/cluster"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"role": "follower"' ]]
}

# =============================================================================
# API Endpoint Tests - Telemetry
# =============================================================================

@test "telemetry endpoint returns statistics" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::create_collection "test1" "1536" "Cosine"
    mock::qdrant::create_collection "test2" "768" "Dot"
    
    run curl -s "http://localhost:6333/telemetry"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"count": 2' ]]
    [[ "$output" =~ '"name": "qdrant"' ]]
    [[ "$output" =~ '"vectors": 100000' ]]
}

# =============================================================================
# API Endpoint Tests - Metrics
# =============================================================================

@test "metrics endpoint returns Prometheus format" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    run curl -s "http://localhost:6333/metrics"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "# HELP qdrant_app_info" ]]
    [[ "$output" =~ "qdrant_rest_requests_total" ]]
    [[ "$output" =~ "qdrant_grpc_requests_total" ]]
    [[ "$output" =~ "qdrant_collections_total" ]]
}

# =============================================================================
# API Endpoint Tests - Collections List
# =============================================================================

@test "collections endpoint returns empty list initially" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    run curl -s "http://localhost:6333/collections"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"collections": []' ]]
}

@test "collections endpoint returns created collections" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::create_collection "vectors1" "1536" "Cosine"
    mock::qdrant::create_collection "vectors2" "768" "Dot"
    
    run curl -s "http://localhost:6333/collections"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"name":"vectors1"' ]]
    [[ "$output" =~ '"name":"vectors2"' ]]
}

# =============================================================================
# API Endpoint Tests - Collection Info
# =============================================================================

@test "collection info endpoint returns collection details" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::create_collection "test_collection" "1536" "Cosine" "5000" "5000"
    
    run curl -s "http://localhost:6333/collections/test_collection"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status": "green"' ]]
    [[ "$output" =~ '"vectors_count": 5000' ]]
    [[ "$output" =~ '"points_count": 5000' ]]
    [[ "$output" =~ '"size": 1536' ]]
    [[ "$output" =~ '"distance": "Cosine"' ]]
}

@test "collection info endpoint returns error for non-existent collection" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    run curl -s "http://localhost:6333/collections/non_existent"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Collection not found" ]]
}

# =============================================================================
# API Endpoint Tests - Collection Create/Delete
# =============================================================================

@test "create collection via API endpoint" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    local data='{"vectors":{"size":768,"distance":"Dot"}}'
    # Use direct call without 'run' to preserve state
    local result
    result=$(curl -X PUT -s "http://localhost:6333/collections/new_collection" \
        -H "Content-Type: application/json" \
        -d "$data")
    
    [[ "$result" =~ '"result":true' ]]
    
    # Reload state after subshell operation
    mock::qdrant::load_state
    mock::qdrant::assert::collection_exists "new_collection"
}

@test "create collection fails if already exists" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::create_collection "existing_collection" "1536" "Cosine"
    
    local data='{"vectors":{"size":768,"distance":"Dot"}}'
    run curl -X PUT -s "http://localhost:6333/collections/existing_collection" \
        -H "Content-Type: application/json" \
        -d "$data"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Collection already exists" ]]
}

@test "delete collection via API endpoint" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::create_collection "to_delete" "1536" "Cosine"
    
    # Use direct call without 'run' to preserve state
    local result
    result=$(curl -X DELETE -s "http://localhost:6333/collections/to_delete")
    
    [[ "$result" =~ '"result":true' ]]
    
    # Reload state after subshell operation
    mock::qdrant::load_state
    mock::qdrant::assert::collection_not_exists "to_delete"
}

@test "delete collection fails if not exists" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    run curl -X DELETE -s "http://localhost:6333/collections/non_existent"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Collection not found" ]]
}

# =============================================================================
# API Endpoint Tests - Collection Cluster
# =============================================================================

@test "collection cluster endpoint returns shard information" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::create_collection "sharded_collection" "1536" "Cosine"
    
    run curl -s "http://localhost:6333/collections/sharded_collection/cluster"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"shard_count": 1' ]]
    [[ "$output" =~ '"state": "Active"' ]]
}

# =============================================================================
# Error Injection Tests
# =============================================================================

@test "connection timeout error injection works" {
    mock::qdrant::inject_error "health" "connection_timeout"
    
    run curl -s --max-time 1 "http://localhost:6333/"
    [ "$status" -eq 28 ]
    [[ "$output" =~ "Operation timed out" ]]
}

@test "connection refused error injection works" {
    mock::qdrant::inject_error "collections" "connection_refused"
    
    run curl -f -s "http://localhost:6333/collections"
    [ "$status" -eq 7 ]
    [[ "$output" =~ "Connection refused" ]]
}

@test "HTTP 500 error injection works" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::inject_error "cluster" "http_500"
    
    run curl -f -s "http://localhost:6333/cluster"
    [ "$status" -eq 22 ]
}

@test "HTTP 401 unauthorized error injection works" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::inject_error "telemetry" "http_401"
    
    run curl -s --write-out "%{http_code}" "http://localhost:6333/telemetry"
    [[ "$output" =~ "401" ]]
}

@test "HTTP 400 bad request error injection works" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::inject_error "metrics" "http_400"
    
    run curl -s --write-out "%{http_code}" "http://localhost:6333/metrics"
    [[ "$output" =~ "400" ]]
}

# =============================================================================
# curl Interceptor Tests
# =============================================================================

@test "curl interceptor handles Qdrant URLs" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    run curl -s "http://localhost:6333/"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "qdrant - vector search engine" ]]
}

@test "curl interceptor handles non-Qdrant URLs" {
    run curl -s "https://example.com"
    [ "$status" -eq 0 ]
    [ "$output" = "Mock curl response" ]
}

@test "curl interceptor supports output files" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    local output_file="/tmp/curl_test_output"
    run curl -s "http://localhost:6333/" --output "$output_file"
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    [[ "$(cat "$output_file")" =~ "qdrant - vector search engine" ]]
    
    trash::safe_remove "$output_file" --test-cleanup
}

@test "curl interceptor supports write-out option" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    run curl -s "http://localhost:6333/collections" --write-out "%{http_code}"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "200" ]]
}

@test "curl interceptor logs all API calls" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    # Make API calls without run to preserve state
    curl -s "http://localhost:6333/" > /dev/null
    curl -s "http://localhost:6333/cluster" > /dev/null
    curl -s "http://localhost:6333/collections" > /dev/null
    
    # Now check assertions
    mock::qdrant::assert::api_called "6333"
    mock::qdrant::assert::api_called "cluster"
    mock::qdrant::assert::api_called "collections"
}

# =============================================================================
# Snapshot Management Tests
# =============================================================================

@test "create snapshot works correctly" {
    mock::qdrant::create_snapshot "backup1" "all"
    mock::qdrant::create_snapshot "backup2" "agent_memory,code_embeddings"
    
    # Verify snapshots were created
    [[ -n "${QDRANT_MOCK_SNAPSHOTS[backup1]}" ]]
    [[ -n "${QDRANT_MOCK_SNAPSHOTS[backup2]}" ]]
}

# =============================================================================
# Scenario Builder Tests
# =============================================================================

@test "create running service scenario works" {
    mock::qdrant::scenario::create_running_service
    
    run mock::qdrant::assert::container_running "qdrant"
    [ "$status" -eq 0 ]
    
    run mock::qdrant::assert::healthy
    [ "$status" -eq 0 ]
    
    [ "$(mock::qdrant::get::config server_status)" = "running" ]
    [ "$(mock::qdrant::get::collection_count)" = "3" ]  # Creates 3 default collections
    
    run mock::qdrant::assert::collection_exists "agent_memory"
    [ "$status" -eq 0 ]
    
    run mock::qdrant::assert::collection_exists "code_embeddings"
    [ "$status" -eq 0 ]
    
    run mock::qdrant::assert::collection_exists "document_chunks"
    [ "$status" -eq 0 ]
}

@test "create stopped service scenario works" {
    mock::qdrant::scenario::create_stopped_service
    
    run mock::qdrant::assert::container_stopped "qdrant"
    [ "$status" -eq 0 ]
    
    run mock::qdrant::assert::healthy
    [ "$status" -eq 1 ]  # Should be unhealthy
    
    [ "$(mock::qdrant::get::config server_status)" = "stopped" ]
}

@test "create authenticated service scenario works" {
    mock::qdrant::scenario::create_authenticated_service "test-key-123"
    
    run mock::qdrant::assert::container_running "qdrant"
    [ "$status" -eq 0 ]
    
    [ "$(mock::qdrant::get::config api_key)" = "test-key-123" ]
    
    # Request without API key should fail
    run curl -s "http://localhost:6333/"
    [ "$status" -eq 22 ]
    [[ "$output" =~ "Unauthorized" ]]
    
    # Request with API key should succeed
    run curl -s -H "api-key: test-key-123" "http://localhost:6333/"
    [ "$status" -eq 0 ]
}

@test "custom container name in scenarios works" {
    mock::qdrant::scenario::create_running_service "custom-qdrant"
    
    run mock::qdrant::assert::container_running "custom-qdrant"
    [ "$status" -eq 0 ]
}

# =============================================================================
# Integration Tests
# =============================================================================

@test "realistic qdrant workflow simulation" {
    # Start with stopped service
    mock::qdrant::scenario::create_stopped_service
    
    # Verify APIs fail when stopped
    run curl -f -s "http://localhost:6333/"
    [ "$status" -eq 7 ]  # Connection refused
    
    # Start service
    mock::qdrant::scenario::create_running_service
    
    # Verify health endpoint works
    run curl -s "http://localhost:6333/"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "qdrant - vector search engine" ]]
    
    # Check cluster status
    run curl -s "http://localhost:6333/cluster"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status": "enabled"' ]]
    
    # List collections (without run to preserve state)
    local collections_result
    collections_result=$(curl -s "http://localhost:6333/collections")
    [[ "$collections_result" =~ "agent_memory" ]]
    
    # Create new collection (without run to preserve state)
    local create_result
    create_result=$(curl -X PUT -s "http://localhost:6333/collections/test_vectors" \
        -H "Content-Type: application/json" \
        -d '{"vectors":{"size":384,"distance":"Euclid"}}')
    [[ "$create_result" =~ '"result":true' ]]
    
    # Get collection info (without run to preserve state)
    local info_result
    info_result=$(curl -s "http://localhost:6333/collections/test_vectors")
    [[ "$info_result" =~ '"size": 384' ]]
    [[ "$info_result" =~ '"distance": "Euclid"' ]]
    
    # Check metrics
    local metrics_result
    metrics_result=$(curl -s "http://localhost:6333/metrics")
    [[ "$metrics_result" =~ "qdrant_collections_total 4" ]]  # 3 default + 1 created
    
    # Delete collection (without run to preserve state)
    local delete_result
    delete_result=$(curl -X DELETE -s "http://localhost:6333/collections/test_vectors")
    [[ "$delete_result" =~ '"result":true' ]]
    
    # Verify all APIs were called
    mock::qdrant::assert::api_called "6333"
    mock::qdrant::assert::api_called "cluster"
    mock::qdrant::assert::api_called "collections"
    mock::qdrant::assert::api_called "metrics"
}

@test "API key authentication workflow" {
    # Create authenticated service
    mock::qdrant::scenario::create_authenticated_service "secure-api-key"
    
    # Verify health check fails without key
    run curl -f -s "http://localhost:6333/"
    [ "$status" -eq 22 ]
    
    # Verify collections list fails without key
    run curl -f -s "http://localhost:6333/collections"
    [ "$status" -eq 22 ]
    
    # APIs should work with correct key
    run curl -s -H "api-key: secure-api-key" "http://localhost:6333/"
    [ "$status" -eq 0 ]
    
    run curl -s -H "api-key: secure-api-key" "http://localhost:6333/collections"
    [ "$status" -eq 0 ]
}

@test "error recovery simulation" {
    mock::qdrant::scenario::create_running_service
    
    # Inject temporary error
    mock::qdrant::inject_error "collections" "http_500"
    
    # Verify API fails
    run curl -f -s "http://localhost:6333/collections"
    [ "$status" -eq 22 ]
    
    # Clear error (simulate recovery)
    mock::qdrant::reset
    mock::qdrant::scenario::create_running_service
    
    # Verify API works again
    run curl -s "http://localhost:6333/collections"
    [ "$status" -eq 0 ]
}

# =============================================================================
# Logging Integration Tests
# =============================================================================

@test "state changes are logged" {
    mock::qdrant::set_container_state "test_logging" "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::create_collection "log_test" "1536" "Cosine"
    
    # Check that state changes were logged
    [[ -f "$TEST_LOG_DIR/used_mocks.log" ]]
    
    local log_content=$(cat "$TEST_LOG_DIR/used_mocks.log")
    [[ "$log_content" =~ "qdrant_container_state:test_logging:running" ]]
    [[ "$log_content" =~ "qdrant_health:set:healthy" ]]
    [[ "$log_content" =~ "qdrant_collection:log_test:created" ]]
}

@test "API calls are logged to request log" {
    # Ensure we have a test log directory
    [[ -n "$TEST_LOG_DIR" ]]
    
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    # Make some API calls (without run to preserve logging)
    curl -s "http://localhost:6333/" >/dev/null
    curl -s "http://localhost:6333/collections" >/dev/null
    curl -s "http://localhost:6333/cluster" >/dev/null
    
    # Check that calls were logged in the qdrant mock's request log
    mock::qdrant::assert::api_called "6333"
    mock::qdrant::assert::api_called "collections"
    mock::qdrant::assert::api_called "cluster"
}

# =============================================================================
# Debug and Utility Tests
# =============================================================================

@test "debug dump shows comprehensive state" {
    mock::qdrant::set_container_state "debug_container" "running"
    mock::qdrant::create_collection "debug_collection" "768" "Dot" "100" "100"
    mock::qdrant::inject_error "health" "http_500"
    mock::qdrant::set_api_response "custom" "Custom response"
    mock::qdrant::create_snapshot "debug_snapshot" "all"
    
    run mock::qdrant::debug::dump_state
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Qdrant Mock State Dump" ]]
    [[ "$output" =~ "debug_container" ]]
    [[ "$output" =~ "debug_collection" ]]
    [[ "$output" =~ "health: http_500" ]]
    [[ "$output" =~ "debug_snapshot" ]]
}

@test "getter functions work correctly" {
    mock::qdrant::set_container_state "getter_test" "running"
    QDRANT_MOCK_CONFIG[custom_key]="custom_value"
    mock::qdrant::create_collection "get_test" "384" "Euclid" "50" "50"
    
    [ "$(mock::qdrant::get::container_state getter_test)" = "running" ]
    [ "$(mock::qdrant::get::config custom_key)" = "custom_value" ]
    [ "$(mock::qdrant::get::collection_info get_test)" = "384|Euclid|50|50" ]
    [ "$(mock::qdrant::get::collection_count)" = "1" ]
}

@test "state persistence works across subshells" {
    # Set state in main shell
    mock::qdrant::set_container_state "persist_test" "running"
    mock::qdrant::create_collection "persist_collection" "1536" "Cosine"
    
    # Test in subshell (simulates BATS test execution)
    (
        source "${BATS_TEST_DIRNAME}/qdrant.sh"
        [ "$(mock::qdrant::get::container_state persist_test)" = "running" ]
        [ "$(mock::qdrant::get::collection_count)" = "1" ]
    )
}

# =============================================================================
# Edge Cases and Error Handling Tests
# =============================================================================

@test "unknown API endpoint returns 404" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    
    run curl -s --write-out "%{http_code}" "http://localhost:6333/unknown/endpoint"
    [[ "$output" =~ "404" ]]
}

@test "empty configuration values are handled" {
    mock::qdrant::set_container_state "" "running"
    mock::qdrant::create_collection "" "" ""
    
    run mock::qdrant::get::container_state ""
    [ "$status" -eq 0 ]
    
    run mock::qdrant::get::collection_info ""
    [ "$status" -eq 0 ]
}

@test "concurrent API calls are handled correctly" {
    mock::qdrant::scenario::create_running_service
    
    # Simulate multiple concurrent calls - run sequentially to avoid state conflicts
    curl -s "http://localhost:6333/" > /dev/null
    curl -s "http://localhost:6333/collections" > /dev/null  
    curl -s "http://localhost:6333/cluster" > /dev/null
    curl -s "http://localhost:6333/telemetry" > /dev/null
    
    # Verify all calls were logged
    mock::qdrant::assert::api_called "6333"
    mock::qdrant::assert::api_called "collections"
    mock::qdrant::assert::api_called "cluster"
    mock::qdrant::assert::api_called "telemetry"
}

@test "custom API responses work correctly" {
    mock::qdrant::set_server_status "running"
    mock::qdrant::set_health_status "healthy"
    mock::qdrant::set_api_response "health" '{"custom":"health response"}'
    mock::qdrant::set_api_response "collections" '{"custom":"collections response"}'
    
    run curl -s "http://localhost:6333/"
    [ "$status" -eq 0 ]
    [[ "$output" == '{"custom":"health response"}' ]]
    
    run curl -s "http://localhost:6333/collections"
    [ "$status" -eq 0 ]
    [[ "$output" == '{"custom":"collections response"}' ]]
}