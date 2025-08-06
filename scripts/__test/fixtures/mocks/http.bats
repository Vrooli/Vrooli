#!/usr/bin/env bats
# Comprehensive tests for HTTP mock system
# Tests the http.sh mock implementation for correctness and integration

# Test setup - load dependencies
setup() {
    # Set up test environment
    export MOCK_UTILS_VERBOSE=false
    export MOCK_VERIFICATION_ENABLED=true
    
    # Load the mock utilities first (required by http mock)
    source "${BATS_TEST_DIRNAME}/logs.sh"
    
    # Load verification system if available
    if [[ -f "${BATS_TEST_DIRNAME}/verification.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/verification.sh"
    fi
    
    # Load the http mock
    source "${BATS_TEST_DIRNAME}/http.sh"
    
    # Initialize clean state for each test
    mock::http::reset
    
    # Create test log directory
    TEST_LOG_DIR=$(mktemp -d)
    mock::init_logging "$TEST_LOG_DIR"
}

# Wrapper for run command that reloads http state afterward
run_http_command() {
    run "$@"
    # Reload state from file after http commands that might modify state
    if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
        eval "$(cat "$HTTP_MOCK_STATE_FILE")" 2>/dev/null || true
    fi
}

# Test cleanup
teardown() {
    # Clean up test logs
    if [[ -n "${TEST_LOG_DIR:-}" && -d "$TEST_LOG_DIR" ]]; then
        rm -rf "$TEST_LOG_DIR"
    fi
    
    # Clean up environment
    unset HTTP_MOCK_MODE
    unset TEST_LOG_DIR
}

# =============================================================================
# Basic HTTP Command Tests - curl
# =============================================================================

@test "curl with no URL should fail" {
    run curl
    [ "$status" -eq 2 ]
    [[ "$output" =~ "no URL specified" ]]
}

@test "curl with valid URL should return default response" {
    run curl http://example.com/some/path
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Mock response" ]]
    [[ "$output" =~ "example.com" ]]
}

@test "curl with health endpoint should return health response" {
    run curl http://localhost:8080/health
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status":"healthy"' ]]
    [[ "$output" =~ '"timestamp"' ]]
}

@test "curl with version endpoint should return version info" {
    run curl http://api.service.com/version
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"version":"1.0.0"' ]]
    [[ "$output" =~ '"build"' ]]
    [[ "$output" =~ '"commit":"abc123"' ]]
}

@test "curl with metrics endpoint should return prometheus format" {
    run curl http://localhost:9090/metrics
    [ "$status" -eq 0 ]
    [[ "$output" =~ "# HELP http_requests_total" ]]
    [[ "$output" =~ "# TYPE http_requests_total counter" ]]
    [[ "$output" =~ "http_requests_total{method=\"GET\",status=\"200\"} 42" ]]
}

@test "curl with ping endpoint should return pong" {
    run curl http://service.com/ping
    [ "$status" -eq 0 ]
    [[ "$output" == "pong" ]]
}

# =============================================================================
# curl Flag and Option Tests
# =============================================================================

@test "curl with -s (silent) flag should suppress verbose output" {
    run curl -s http://example.com/path
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Mock response" ]]
}

@test "curl with -i (include headers) should show headers" {
    run curl -i http://example.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HTTP/1.1 200 OK" ]]
    [[ "$output" =~ "Content-Type: application/json" ]]
    [[ "$output" =~ "Server: MockServer/1.0" ]]
}

@test "curl with -I (HEAD method) should return headers only" {
    run curl -I http://example.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HTTP/1.1 200 OK" ]]
    [[ "$output" != *'"status":"ok"'* ]]
}

@test "curl with -X POST should use POST method" {
    run curl -X POST http://api.service.com/api/v1/users
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status":"created"' ]]
}

@test "curl with -d data should use POST method automatically" {
    run curl -d '{"name":"test"}' http://api.service.com/api/v1/users
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status":"created"' ]]
}

@test "curl with -w write-out should show status code" {
    run curl -w "%{http_code}" http://example.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ "200" ]]
}

@test "curl with -f (fail) should fail on HTTP errors" {
    mock::http::set_endpoint_response "http://error.com" '{"error":"server error"}' 500
    
    run curl -f http://error.com
    [ "$status" -eq 22 ]
    [[ "$output" =~ "returned error: 500" ]]
}

@test "curl with -o output file should write to file" {
    local temp_file=$(mktemp)
    
    run curl -o "$temp_file" http://example.com/path
    [ "$status" -eq 0 ]
    [[ -f "$temp_file" ]]
    [[ "$(cat "$temp_file")" =~ "Mock response" ]]
    
    rm -f "$temp_file"
}

# =============================================================================
# HTTP Method Tests
# =============================================================================

@test "GET request to API endpoint should return data" {
    run curl http://api.service.com/api/v1/users
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"data":[]' ]]
    [[ "$output" =~ '"total":0' ]]
}

@test "POST request to API endpoint should return created response" {
    run curl -X POST http://api.service.com/api/v1/users
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status":"created"' ]]
    [[ "$output" =~ '"id"' ]]
}

@test "PUT request to API endpoint should return updated response" {
    run curl -X PUT http://api.service.com/api/v1/users/123
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status":"updated"' ]]
    [[ "$output" =~ '"updated"' ]]
}

@test "DELETE request to API endpoint should return deleted response" {
    run curl -X DELETE http://api.service.com/api/v1/users/123
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status":"deleted"' ]]
}

@test "method validation should return 405 for wrong method" {
    mock::http::set_endpoint_method "http://strict.com/api" "POST"
    
    run curl -X GET http://strict.com/api
    [ "$status" -eq 1 ]  # curl returns 1 for HTTP error without -f
    [[ "$output" =~ '"error":"Method not allowed"' ]]
    [[ "$output" =~ '"expected":"POST"' ]]
    [[ "$output" =~ '"received":"GET"' ]]
}

# =============================================================================
# Resource-Specific Pattern Tests
# =============================================================================

@test "ollama API tags endpoint should return models" {
    run curl http://localhost:11434/api/tags
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"models"' ]]
    [[ "$output" =~ '"name":"llama3.1:8b"' ]]
    [[ "$output" =~ '"name":"deepseek-r1:8b"' ]]
}

@test "ollama generate endpoint should handle POST only" {
    run curl -X POST http://localhost:11434/api/generate
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"response":"Hello! This is a mock response from Ollama."' ]]
    [[ "$output" =~ '"done":true' ]]
    
    run curl -X GET http://localhost:11434/api/generate
    [ "$status" -eq 1 ]
    [[ "$output" =~ '"error":"Method not allowed"' ]]
}

@test "whisper transcribe endpoint should return transcription" {
    run curl -X POST http://localhost:8090/transcribe
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"text":"This is a mock transcription result from Whisper."' ]]
    [[ "$output" =~ '"language":"en"' ]]
    [[ "$output" =~ '"duration":5.2' ]]
}

@test "n8n workflows endpoint should return workflow data" {
    run curl http://localhost:5678/api/v1/workflows
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"data"' ]]
    [[ "$output" =~ '"name":"Test Workflow"' ]]
    [[ "$output" =~ '"active":true' ]]
}

@test "qdrant collections endpoint should return collections" {
    run curl http://localhost:6333/collections
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"result"' ]]
    [[ "$output" =~ '"collections"' ]]
    [[ "$output" =~ '"name":"test_collection"' ]]
}

@test "minio health endpoint should return status" {
    run curl http://localhost:9000/minio/health/live
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status":"ok"' ]]
    [[ "$output" =~ '"timestamp"' ]]
}

# =============================================================================
# wget Command Tests
# =============================================================================

@test "wget with no URL should fail" {
    run wget
    [ "$status" -eq 1 ]
    [[ "$output" =~ "missing URL" ]]
}

@test "wget with valid URL should download content" {
    run wget -q http://example.com/path
    [ "$status" -eq 0 ]
    [[ -f "index.html" ]]
    [[ "$(cat index.html)" =~ "Mock response" ]]
    
    rm -f index.html
}

@test "wget with -O output file should save to specified file" {
    local temp_file=$(mktemp)
    
    run wget -q -O "$temp_file" http://example.com/path
    [ "$status" -eq 0 ]
    [[ -f "$temp_file" ]]
    [[ "$(cat "$temp_file")" =~ "Mock response" ]]
    
    rm -f "$temp_file"
}

@test "wget with -O - should output to stdout" {
    run wget -q -O - http://example.com/path
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Mock response" ]]
}

@test "wget without -q should show progress" {
    run wget -O /dev/null http://example.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Resolving" ]]
    [[ "$output" =~ "Connecting" ]]
    [[ "$output" =~ "HTTP request sent" ]]
    [[ "$output" =~ "saved" ]]
}

# =============================================================================
# nc (netcat) Command Tests
# =============================================================================

@test "nc port test should succeed for healthy endpoints" {
    mock::http::set_endpoint_state "http://localhost:8080" "healthy"
    
    run nc -z localhost 8080
    [ "$status" -eq 0 ]
}

@test "nc port test should fail for unavailable endpoints" {
    mock::http::set_endpoint_state "http://localhost:9999" "unavailable"
    
    run nc -z localhost 9999
    [ "$status" -eq 1 ]
}

@test "nc with verbose flag should show connection details" {
    mock::http::set_endpoint_state "http://localhost:8080" "healthy"
    
    run nc -zv localhost 8080
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Connection to localhost 8080 port" ]]
    [[ "$output" =~ "succeeded" ]]
}

@test "nc should succeed for common ports by default" {
    run nc -z localhost 80
    [ "$status" -eq 0 ]
    
    run nc -z localhost 443
    [ "$status" -eq 0 ]
    
    run nc -z localhost 8080
    [ "$status" -eq 0 ]
}

# =============================================================================
# Mock State Management Tests
# =============================================================================

@test "set_endpoint_state should configure endpoint behavior" {
    mock::http::set_endpoint_state "http://test.com/health" "healthy"
    
    run curl http://test.com/health
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status":"healthy"' ]]
}

@test "set_endpoint_response should override default responses" {
    mock::http::set_endpoint_response "http://custom.com" '{"custom":"response"}' 201
    
    run curl http://custom.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"custom":"response"' ]]
}

@test "set_endpoint_delay should add delays in slow mode" {
    mock::http::set_endpoint_delay "http://slow.com" "1"
    export HTTP_MOCK_MODE="slow"
    
    local start_time=$(date +%s)
    run curl http://slow.com
    local end_time=$(date +%s)
    
    [ "$status" -eq 0 ]
    local duration=$((end_time - start_time))
    [[ $duration -ge 1 ]]
}

@test "set_endpoint_sequence should return responses in order" {
    mock::http::set_endpoint_sequence "http://sequence.com" '{"call":1}:200,{"call":2}:201,{"call":3}:202'
    
    # First call
    run curl http://sequence.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"call":1' ]]
    
    # Second call  
    run curl http://sequence.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"call":2' ]]
    
    # Third call
    run curl http://sequence.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"call":3' ]]
    
    # Fourth call should repeat last response
    run curl http://sequence.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"call":3' ]]
}

@test "set_endpoint_unreachable should mark base URL and common paths" {
    mock::http::set_endpoint_unreachable "http://down.com"
    
    run curl http://down.com
    [ "$status" -eq 7 ]  # Connection refused
    [[ "$output" =~ "Connection refused" ]]
    
    run curl http://down.com/health
    [ "$status" -eq 7 ]
    [[ "$output" =~ "Connection refused" ]]
    
    run curl http://down.com/api
    [ "$status" -eq 7 ]
    [[ "$output" =~ "Connection refused" ]]
}

# =============================================================================
# HTTP Mock Mode Tests
# =============================================================================

@test "offline mode should return DNS resolution errors" {
    export HTTP_MOCK_MODE="offline"
    
    run curl http://example.com
    [ "$status" -eq 6 ]
    [[ "$output" =~ "Could not resolve host" ]]
}

@test "error mode should return connection errors" {
    export HTTP_MOCK_MODE="error"
    
    run curl http://example.com
    [ "$status" -eq 7 ]
    [[ "$output" =~ "Failed to connect" ]]
}

@test "slow mode should apply configured delays" {
    export HTTP_MOCK_MODE="slow"
    mock::http::set_endpoint_delay "http://slow.com" "1"
    
    local start_time=$(date +%s)
    run curl http://slow.com
    local end_time=$(date +%s)
    
    [ "$status" -eq 0 ]
    local duration=$((end_time - start_time))
    [[ $duration -ge 1 ]]
}

# =============================================================================
# Error Injection Tests
# =============================================================================

@test "injected DNS resolution error should override normal behavior" {
    mock::http::inject_error "curl" "dns_resolution"
    
    run curl http://example.com
    [ "$status" -eq 6 ]
    [[ "$output" =~ "Could not resolve host" ]]
}

@test "injected connection timeout error should return timeout" {
    mock::http::inject_error "curl" "connection_timeout"
    
    run curl http://example.com
    [ "$status" -eq 28 ]
    [[ "$output" =~ "Connection timed out" ]]
}

@test "injected connection refused error should return connection error" {
    mock::http::inject_error "curl" "connection_refused"
    
    run curl http://example.com
    [ "$status" -eq 7 ]
    [[ "$output" =~ "Connection refused" ]]
}

@test "injected SSL error should return SSL error" {
    mock::http::inject_error "curl" "ssl_error"
    
    run curl https://secure.com
    [ "$status" -eq 35 ]
    [[ "$output" =~ "SSL connect error" ]]
}

@test "injected HTTP error should return HTTP error" {
    mock::http::inject_error "curl" "http_error"
    
    run curl http://example.com
    [ "$status" -eq 22 ]
    [[ "$output" =~ "returned error: 500" ]]
}

# =============================================================================
# Call Tracking Tests
# =============================================================================

@test "HTTP calls should be tracked correctly" {
    run_http_command curl http://track.com
    run_http_command curl http://track.com
    run_http_command curl http://different.com
    
    local track_count=$(mock::http::get::call_count "http://track.com")
    local different_count=$(mock::http::get::call_count "http://different.com")
    
    [[ "$track_count" == "2" ]]
    [[ "$different_count" == "1" ]]
}

@test "get response and status helpers should work correctly" {
    mock::http::set_endpoint_response "http://test.com" '{"test":"data"}' 201
    
    local response=$(mock::http::get::response "http://test.com")
    local status=$(mock::http::get::status_code "http://test.com")
    local state=$(mock::http::get::endpoint_state "http://test.com")
    
    [[ "$response" =~ '"test":"data"' ]]
    [[ "$status" == "201" ]]
}

# =============================================================================
# Scenario Builder Tests
# =============================================================================

@test "create_healthy_services should set up healthy service stack" {
    mock::http::scenario::create_healthy_services "http://test"
    
    # Verify all services are healthy
    run curl http://test:8080/health
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status":"healthy"' ]]
    [[ "$output" =~ '"service":"api"' ]]
    
    run curl http://test:5432/health
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"service":"db"' ]]
    
    run curl http://test:6379/health
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"service":"cache"' ]]
}

@test "create_mixed_health should set up mixed health scenario" {
    mock::http::scenario::create_mixed_health "http://mixed"
    
    # Healthy services should work
    run curl http://mixed:8080/health
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"status":"healthy"' ]]
    
    # Unhealthy service should return 503
    run curl http://mixed:5432/health
    [ "$status" -eq 1 ]  # curl returns 1 for HTTP errors
    
    # Unavailable service should fail to connect
    run curl http://mixed:9200/health
    [ "$status" -eq 7 ]
    [[ "$output" =~ "Connection refused" ]]
}

@test "create_api_versions should set up versioned API endpoints" {
    mock::http::scenario::create_api_versions "http://api"
    
    # v1 API should indicate deprecation
    run curl http://api/api/v1/users
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"deprecated":true' ]]
    [[ "$output" =~ '"migrate_to":"/api/v2/users"' ]]
    
    # v2 API should return current version
    run curl http://api/api/v2/users
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"version":"2.0"' ]]
    [[ "$output" =~ '"name":"Test User"' ]]
    
    # v3 API should indicate beta
    run curl http://api/api/v3/users
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"version":"3.0-beta"' ]]
    [[ "$output" =~ '"profile"' ]]
}

# =============================================================================
# Assertion Helper Tests
# =============================================================================

@test "assert_endpoint_called should succeed when endpoint was called" {
    run_http_command curl http://called.com
    run_http_command curl http://called.com
    
    run mock::http::assert::endpoint_called "http://called.com" 2
    [ "$status" -eq 0 ]
}

@test "assert_endpoint_called should fail when endpoint wasn't called enough" {
    run_http_command curl http://under.com
    
    run mock::http::assert::endpoint_called "http://under.com" 5
    [ "$status" -eq 1 ]
    [[ "$output" =~ "called 1 times, expected at least 5" ]]
}

@test "assert_endpoint_not_called should succeed when endpoint wasn't called" {
    run mock::http::assert::endpoint_not_called "http://never.com"
    [ "$status" -eq 0 ]
}

@test "assert_endpoint_not_called should fail when endpoint was called" {
    run_http_command curl http://was_called.com
    
    run mock::http::assert::endpoint_not_called "http://was_called.com"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "was called 1 times but should not have been called" ]]
}

@test "assert_endpoint_healthy should succeed for healthy endpoints" {
    mock::http::set_endpoint_state "http://healthy.com" "healthy"
    
    run mock::http::assert::endpoint_healthy "http://healthy.com"
    [ "$status" -eq 0 ]
}

@test "assert_endpoint_healthy should fail for unhealthy endpoints" {
    mock::http::set_endpoint_state "http://unhealthy.com" "unhealthy"
    
    run mock::http::assert::endpoint_healthy "http://unhealthy.com"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is not healthy" ]]
}

@test "assert_endpoint_unavailable should succeed for unavailable endpoints" {
    mock::http::set_endpoint_state "http://down.com" "unavailable"
    
    run mock::http::assert::endpoint_unavailable "http://down.com"
    [ "$status" -eq 0 ]
}

@test "assert_response_contains should succeed when content is found" {
    mock::http::set_endpoint_response "http://content.com" '{"find":"this content"}'
    
    run mock::http::assert::response_contains "http://content.com" "this content"
    [ "$status" -eq 0 ]
}

@test "assert_response_contains should fail when content is not found" {
    mock::http::set_endpoint_response "http://nocontent.com" '{"other":"content"}'
    
    run mock::http::assert::response_contains "http://nocontent.com" "missing content"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "does not contain 'missing content'" ]]
}

@test "assert_status_code should succeed for correct status" {
    mock::http::set_endpoint_response "http://status.com" '{"test":"data"}' 201
    
    run mock::http::assert::status_code "http://status.com" "201"
    [ "$status" -eq 0 ]
}

@test "assert_status_code should fail for incorrect status" {
    mock::http::set_endpoint_response "http://wrongstatus.com" '{"test":"data"}' 404
    
    run mock::http::assert::status_code "http://wrongstatus.com" "200"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is 404, expected 200" ]]
}

# =============================================================================
# Debug and Utility Tests
# =============================================================================

@test "debug dump should show current HTTP state" {
    mock::http::set_endpoint_state "http://debug.com" "healthy"
    mock::http::set_endpoint_response "http://debug.com/api" '{"debug":"data"}' 200
    mock::http::inject_error "curl" "connection_timeout"
    
    run mock::http::debug::dump_state
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HTTP Mock State Dump" ]]
    [[ "$output" =~ "debug_com" ]]
    [[ "$output" =~ "healthy" ]]
    [[ "$output" =~ "connection_timeout" ]]
}

@test "debug list_endpoints should show configured endpoints" {
    mock::http::set_endpoint_state "http://list1.com" "healthy"
    mock::http::set_endpoint_state "http://list2.com" "unhealthy"
    run_http_command curl http://list1.com
    
    run mock::http::debug::list_endpoints
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Configured HTTP Endpoints" ]]
    [[ "$output" =~ "list1_com" ]]
    [[ "$output" =~ "healthy" ]]
    [[ "$output" =~ "list2_com" ]]
    [[ "$output" =~ "unhealthy" ]]
    [[ "$output" =~ "(1 calls)" ]]
}

@test "reset function should clear all HTTP state" {
    # Set up some state
    mock::http::set_endpoint_state "http://reset.com" "healthy"
    mock::http::set_endpoint_response "http://reset.com/api" '{"data":"test"}'
    mock::http::inject_error "curl" "timeout"
    run_http_command curl http://reset.com
    
    # Verify state exists
    local key=$(_sanitize_url_key 'http://reset.com')
    [[ -n "${MOCK_HTTP_ENDPOINTS[$key]}" ]]
    [[ "$(mock::http::get::call_count 'http://reset.com')" == "1" ]]
    
    # Reset and verify state is cleared
    mock::http::reset
    
    [[ -z "${MOCK_HTTP_ENDPOINTS[$key]}" ]]
    [[ "$(mock::http::get::call_count 'http://reset.com')" == "0" ]]
}

# =============================================================================
# URL Sanitization Tests
# =============================================================================

@test "URL sanitization should handle special characters" {
    mock::http::set_endpoint_response "https://test.com:8080/api?param=value" '{"special":"url"}'
    
    run curl "https://test.com:8080/api?param=value"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"special":"url"' ]]
}

@test "URL sanitization should handle different protocols" {
    mock::http::set_endpoint_response "ftp://files.com/data" '{"protocol":"ftp"}'
    
    run curl "ftp://files.com/data"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"protocol":"ftp"' ]]
}

# =============================================================================
# Integration Tests
# =============================================================================

@test "multiple HTTP tools should share state" {
    mock::http::set_endpoint_response "http://shared.com" '{"shared":"state"}'
    
    # Test with curl
    run curl http://shared.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"shared":"state"' ]]
    
    # Test with wget 
    run wget -q -O - http://shared.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"shared":"state"' ]]
    
    # Verify both calls were tracked
    local call_count=$(mock::http::get::call_count "http://shared.com")
    [[ "$call_count" == "2" ]]
}

@test "complex API interaction should work end-to-end" {
    # Set up API endpoints
    mock::http::set_endpoint_response "http://api.test/auth" '{"token":"abc123"}' 200
    mock::http::set_endpoint_response "http://api.test/users" '{"users":[{"id":1,"name":"John"}]}' 200
    mock::http::set_endpoint_method "http://api.test/users" "GET"
    
    # Authenticate
    run curl http://api.test/auth
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"token":"abc123"' ]]
    
    # Get users
    run curl -H "Authorization: Bearer abc123" http://api.test/users
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"name":"John"' ]]
    
    # Verify both endpoints were called
    [[ "$(mock::http::get::call_count 'http://api.test/auth')" == "1" ]]
    [[ "$(mock::http::get::call_count 'http://api.test/users')" == "1" ]]
}

# =============================================================================
# Logging Integration Tests
# =============================================================================

@test "HTTP commands should be logged to http_calls.log" {
    # Ensure we have a test log directory
    [[ -n "$TEST_LOG_DIR" ]]
    
    # Run some HTTP commands
    curl http://log.test >/dev/null
    wget -q http://log.test
    
    # Check that commands were logged to the correct file (centralized logging routes HTTP commands to http_calls.log)
    [[ -f "$TEST_LOG_DIR/http_calls.log" ]]
    
    local log_content=$(cat "$TEST_LOG_DIR/http_calls.log")
    [[ "$log_content" =~ "curl: http://log.test" ]]
    [[ "$log_content" =~ "wget: -q http://log.test" ]]
}

@test "HTTP state changes should be logged" {
    mock::http::set_endpoint_state "http://log.test" "healthy"
    
    # Check that state change was logged to the correct file (centralized logging uses used_mocks.log)
    [[ -f "$TEST_LOG_DIR/used_mocks.log" ]]
    
    local log_content=$(cat "$TEST_LOG_DIR/used_mocks.log")
    [[ "$log_content" =~ "http_endpoint_state:http://log.test:healthy" ]]
}

# =============================================================================
# Verification Integration Tests
# =============================================================================

@test "HTTP commands should trigger verification recording" {
    # Skip if verification system not available
    if ! command -v mock::verify::record_call &>/dev/null; then
        skip "Verification system not available"
    fi
    
    # Reset verification state
    mock::verify::reset
    
    # Run an HTTP command
    curl http://verify.test >/dev/null
    
    # Verify the call was recorded (basic check)
    run mock::verify::was_called "curl" "http://verify.test"
    [ "$status" -eq 0 ]
}

# =============================================================================
# Performance and Edge Case Tests
# =============================================================================

@test "large response should be handled correctly" {
    local large_response='{"data":"'$(printf 'x%.0s' {1..10000})'"}'
    mock::http::set_endpoint_response "http://large.com" "$large_response"
    
    run curl http://large.com
    [ "$status" -eq 0 ]
    [[ ${#output} -gt 10000 ]]
}

@test "empty response should be handled correctly" {
    mock::http::set_endpoint_response "http://empty.com" ""
    
    run curl http://empty.com
    [ "$status" -eq 0 ]
    [[ -z "$output" ]]
}

@test "invalid JSON response should be handled correctly" {
    mock::http::set_endpoint_response "http://invalid.com" "not json at all"
    
    run curl http://invalid.com
    [ "$status" -eq 0 ]
    [[ "$output" == "not json at all" ]]
}

@test "many concurrent mock setups should work" {
    # Set up many endpoints
    for i in {1..50}; do
        mock::http::set_endpoint_response "http://bulk$i.com" "{\"id\":$i}"
    done
    
    # Test a few random ones
    run curl http://bulk25.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"id":25' ]]
    
    run curl http://bulk1.com
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"id":1' ]]
}

# =============================================================================
# Error Recovery Tests
# =============================================================================

@test "mock should recover from malformed state file" {
    # Corrupt the state file
    if [[ -n "${HTTP_MOCK_STATE_FILE}" ]]; then
        echo "corrupted data" > "$HTTP_MOCK_STATE_FILE"
    fi
    
    # Mock should still work (fallback to defaults)
    run curl http://recovery.test/test
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"message":"Mock response for test"' ]]
}

@test "mock should handle missing state file gracefully" {
    # Remove state file
    if [[ -n "${HTTP_MOCK_STATE_FILE}" && -f "$HTTP_MOCK_STATE_FILE" ]]; then
        rm -f "$HTTP_MOCK_STATE_FILE"
    fi
    
    # Mock should still work
    run curl http://missing.test/test
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"message":"Mock response for test"' ]]
}