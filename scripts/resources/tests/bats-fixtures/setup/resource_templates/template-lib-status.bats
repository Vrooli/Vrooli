#!/usr/bin/env bats
# Tests for RESOURCE_NAME lib/status.sh health monitoring functions
#
# Template Usage:
# 1. Copy this file to RESOURCE_NAME/lib/status.bats
# 2. Replace RESOURCE_NAME with your resource name (e.g., "ollama", "n8n")
# 3. Replace STATUS_FUNCTIONS with your actual status function names
# 4. Implement resource-specific health monitoring tests
# 5. Remove this header comment block

bats_require_minimum_version 1.5.0

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export RESOURCE_NAME_PORT="8080"  # Replace with actual port
    export RESOURCE_NAME_CONTAINER_NAME="RESOURCE_NAME-test"
    export RESOURCE_NAME_BASE_URL="http://localhost:8080"
    export RESOURCE_NAME_HEALTH_ENDPOINT="/health"  # Replace with actual endpoint
    
    # Get resource directory path
    RESOURCE_NAME_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    
    # Configure default healthy state
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    mock::http::set_endpoint_response "http://localhost:8080/health" "200" '{"status":"healthy","version":"1.0.0"}'
    
    # Load configuration and the functions to test
    source "${RESOURCE_NAME_DIR}/config/defaults.sh"
    source "${RESOURCE_NAME_DIR}/config/messages.sh"
    RESOURCE_NAME::export_config
    RESOURCE_NAME::messages::init
    
    # Load the status monitoring functions
    source "${RESOURCE_NAME_DIR}/lib/status.sh"
}

teardown() {
    # Clean up test environment
    cleanup_mocks
}

# Test basic status checking

@test "RESOURCE_NAME::status should report healthy when service is running" {
    # Service is healthy (from setup)
    run RESOURCE_NAME::status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "healthy" || "$output" =~ "running" ]]
}

@test "RESOURCE_NAME::status should report unhealthy when service is down" {
    # Mock unhealthy service
    mock::docker::set_container_state "RESOURCE_NAME-test" "stopped"
    mock::http::set_endpoint_response "http://localhost:8080/health" "connection_refused" ""
    
    run RESOURCE_NAME::status
    [ "$status" -ne 0 ]
    [[ "$output" =~ "unhealthy" || "$output" =~ "stopped" || "$output" =~ "down" ]]
}

@test "RESOURCE_NAME::status should handle missing container" {
    # Mock missing container
    mock::docker::set_container_state "RESOURCE_NAME-test" "missing"
    
    run RESOURCE_NAME::status
    [ "$status" -ne 0 ]
    [[ "$output" =~ "not found" || "$output" =~ "missing" ]]
}

# Test detailed status information

@test "RESOURCE_NAME::detailed_status should provide comprehensive information" {
    run RESOURCE_NAME::detailed_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Container:" ]]
    [[ "$output" =~ "Health:" ]]
    [[ "$output" =~ "Port:" ]]
    [[ "$output" =~ "URL:" ]]
}

@test "RESOURCE_NAME::detailed_status should include version information" {
    # Mock version info
    mock::http::set_endpoint_response "http://localhost:8080/version" "200" '{"version":"1.2.3"}'
    
    run RESOURCE_NAME::detailed_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "version" || "$output" =~ "1.2.3" ]]
}

@test "RESOURCE_NAME::detailed_status should show resource usage" {
    # Mock container stats
    mock::docker::set_container_stats "RESOURCE_NAME-test" "50" "256MB" "1GB"
    
    run RESOURCE_NAME::detailed_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CPU" || "$output" =~ "Memory" ]]
}

# Test health check functions

@test "RESOURCE_NAME::health_check should validate service health" {
    run RESOURCE_NAME::health_check
    [ "$status" -eq 0 ]
    [[ "$output" =~ "healthy" ]]
}

@test "RESOURCE_NAME::health_check should detect service degradation" {
    # Mock degraded service
    mock::http::set_endpoint_response "http://localhost:8080/health" "503" '{"status":"degraded","errors":["high_latency"]}'
    
    run RESOURCE_NAME::health_check
    [ "$status" -ne 0 ]
    [[ "$output" =~ "degraded" || "$output" =~ "error" ]]
}

@test "RESOURCE_NAME::health_check should timeout appropriately" {
    # Mock slow response
    mock::http::set_endpoint_delay "http://localhost:8080/health" 10
    
    run timeout 5 RESOURCE_NAME::health_check
    [ "$status" -ne 0 ]
}

@test "RESOURCE_NAME::health_check should validate dependencies" {
    # Mock dependency failure (customize based on your resource)
    # For example, if resource depends on a database:
    # mock::http::set_endpoint_response "http://localhost:8080/health" "200" '{"status":"healthy","dependencies":{"database":"unhealthy"}}'
    
    run RESOURCE_NAME::health_check
    [ "$status" -eq 0 ]  # Modify based on how your resource handles dependency failures
}

# Test readiness checking

@test "RESOURCE_NAME::readiness_check should confirm service is ready" {
    run RESOURCE_NAME::readiness_check
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ready" ]]
}

@test "RESOURCE_NAME::readiness_check should detect initialization state" {
    # Mock initializing service
    mock::http::set_endpoint_response "http://localhost:8080/health" "503" '{"status":"initializing"}'
    
    run RESOURCE_NAME::readiness_check
    [ "$status" -ne 0 ]
    [[ "$output" =~ "initializing" || "$output" =~ "not ready" ]]
}

@test "RESOURCE_NAME::wait_for_ready should wait for service initialization" {
    # Mock service becoming ready
    mock::http::set_endpoint_sequence "http://localhost:8080/health" "503,503,200" "initializing,initializing,healthy"
    
    run timeout 10 RESOURCE_NAME::wait_for_ready
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::wait_for_ready should respect timeout" {
    # Mock service never becoming ready
    mock::http::set_endpoint_response "http://localhost:8080/health" "503" "initializing"
    
    run timeout 3 RESOURCE_NAME::wait_for_ready 2
    [ "$status" -ne 0 ]
}

# Test performance monitoring

@test "RESOURCE_NAME::performance_check should report metrics" {
    # Mock performance metrics
    mock::http::set_endpoint_response "http://localhost:8080/metrics" "200" 'cpu_usage=45.2,memory_usage=67.8,response_time=120ms'
    
    run RESOURCE_NAME::performance_check
    [ "$status" -eq 0 ]
    [[ "$output" =~ "cpu" || "$output" =~ "memory" || "$output" =~ "response" ]]
}

@test "RESOURCE_NAME::performance_check should detect performance issues" {
    # Mock high resource usage
    mock::docker::set_container_stats "RESOURCE_NAME-test" "95" "2GB" "2GB"
    
    run RESOURCE_NAME::performance_check
    [ "$status" -ne 0 ]
    [[ "$output" =~ "high" || "$output" =~ "warning" ]]
}

@test "RESOURCE_NAME::benchmark should measure response times" {
    run RESOURCE_NAME::benchmark
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ms" || "$output" =~ "response time" ]]
}

# Test connectivity checking

@test "RESOURCE_NAME::connectivity_check should verify network access" {
    run RESOURCE_NAME::connectivity_check
    [ "$status" -eq 0 ]
    [[ "$output" =~ "connected" || "$output" =~ "reachable" ]]
}

@test "RESOURCE_NAME::connectivity_check should detect network issues" {
    # Mock network failure
    mock::http::set_endpoint_response "http://localhost:8080/health" "connection_refused" ""
    
    run RESOURCE_NAME::connectivity_check
    [ "$status" -ne 0 ]
    [[ "$output" =~ "connection" || "$output" =~ "network" ]]
}

@test "RESOURCE_NAME::port_check should verify port accessibility" {
    run RESOURCE_NAME::port_check
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::port_check should detect port conflicts" {
    # Mock port in use by different service
    mock::system::set_port_status "8080" "in_use_other"
    
    run RESOURCE_NAME::port_check
    [ "$status" -ne 0 ]
    [[ "$output" =~ "conflict" || "$output" =~ "in use" ]]
}

# Test resource-specific status checks

@test "RESOURCE_NAME specific status checks" {
    # Example for AI resources:
    # run RESOURCE_NAME::model_status
    # [ "$status" -eq 0 ]
    # [[ "$output" =~ "models loaded" ]]
    
    # Example for automation resources:
    # run RESOURCE_NAME::workflow_status
    # [ "$status" -eq 0 ]
    # [[ "$output" =~ "workflows active" ]]
    
    # Example for storage resources:
    # run RESOURCE_NAME::storage_status
    # [ "$status" -eq 0 ]
    # [[ "$output" =~ "storage available" ]]
    
    # Replace this with actual resource-specific tests
    skip "Implement resource-specific status checks"
}

# Test status reporting formats

@test "RESOURCE_NAME::status --json should output valid JSON" {
    run RESOURCE_NAME::status --json
    [ "$status" -eq 0 ]
    echo "$output" | jq . >/dev/null  # Validate JSON syntax
}

@test "RESOURCE_NAME::status --brief should output concise information" {
    run RESOURCE_NAME::status --brief
    [ "$status" -eq 0 ]
    # Brief output should be significantly shorter
    [ ${#output} -lt 200 ]
}

@test "RESOURCE_NAME::status --verbose should output detailed information" {
    run RESOURCE_NAME::status --verbose
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Container ID:" ]]
    [[ "$output" =~ "Configuration:" ]]
    [[ "$output" =~ "Dependencies:" ]]
}

# Test error handling and edge cases

@test "RESOURCE_NAME::status should handle Docker daemon down" {
    # Mock Docker daemon unavailable
    docker() { echo "Cannot connect to the Docker daemon"; return 1; }
    export -f docker
    
    run RESOURCE_NAME::status
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Docker" || "$output" =~ "daemon" ]]
}

@test "RESOURCE_NAME::status should handle partial service failure" {
    # Mock container running but health check failing
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    mock::http::set_endpoint_response "http://localhost:8080/health" "500" "Internal Server Error"
    
    run RESOURCE_NAME::status
    [ "$status" -ne 0 ]
    [[ "$output" =~ "running" && "$output" =~ "unhealthy" ]]
}

@test "RESOURCE_NAME::status should handle configuration changes" {
    # Test with different port
    export RESOURCE_NAME_PORT="9090"
    mock::http::set_endpoint_response "http://localhost:9090/health" "200" "healthy"
    
    run RESOURCE_NAME::status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "9090" ]]
}

# Test status caching (if implemented)

@test "RESOURCE_NAME::status should cache recent results" {
    # First call
    run RESOURCE_NAME::status --cache
    [ "$status" -eq 0 ]
    
    # Second call should use cache
    mock::http::set_endpoint_response "http://localhost:8080/health" "connection_refused" ""
    run RESOURCE_NAME::status --cache
    [ "$status" -eq 0 ]  # Should use cached result
}

@test "RESOURCE_NAME::status should refresh expired cache" {
    # Mock expired cache
    export RESOURCE_NAME_STATUS_CACHE_TTL=1
    run RESOURCE_NAME::status --cache
    [ "$status" -eq 0 ]
    
    sleep 2
    
    # Should refresh cache
    run RESOURCE_NAME::status --cache
    [ "$status" -eq 0 ]
}

# Test monitoring alerts (if implemented)

@test "RESOURCE_NAME::check_alerts should detect threshold violations" {
    # Mock high resource usage triggering alert
    mock::docker::set_container_stats "RESOURCE_NAME-test" "90" "1.8GB" "2GB"
    
    run RESOURCE_NAME::check_alerts
    [ "$status" -ne 0 ]
    [[ "$output" =~ "alert" || "$output" =~ "threshold" ]]
}

# Add more resource-specific status monitoring tests here:

# Example templates for different resource types:

# For AI resources:
# @test "RESOURCE_NAME::gpu_status should report GPU utilization" { ... }
# @test "RESOURCE_NAME::model_health should validate loaded models" { ... }

# For automation resources:
# @test "RESOURCE_NAME::execution_status should report workflow state" { ... }
# @test "RESOURCE_NAME::queue_status should show pending tasks" { ... }

# For storage resources:
# @test "RESOURCE_NAME::disk_status should report storage usage" { ... }
# @test "RESOURCE_NAME::backup_status should verify backup health" { ... }