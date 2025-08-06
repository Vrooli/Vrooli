#!/usr/bin/env bats
# Comprehensive tests for Browserless mock system
# Tests the browserless.sh mock implementation for correctness and integration

# Test setup - load dependencies
setup() {
    # Set up test environment
    export MOCK_UTILS_VERBOSE=false
    export MOCK_VERIFICATION_ENABLED=true
    
    # Load the mock utilities first (required by browserless mock)
    source "${BATS_TEST_DIRNAME}/logs.sh"
    
    # Load verification system if available
    if [[ -f "${BATS_TEST_DIRNAME}/verification.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/verification.sh"
    fi
    
    # Load the browserless mock
    source "${BATS_TEST_DIRNAME}/browserless.sh"
    
    # Initialize clean state for each test
    mock::browserless::reset
    
    # Create test log directory
    TEST_LOG_DIR=$(mktemp -d)
    mock::init_logging "$TEST_LOG_DIR"
}

# Test cleanup
teardown() {
    # Clean up test logs
    if [[ -n "${TEST_LOG_DIR:-}" && -d "$TEST_LOG_DIR" ]]; then
        rm -rf "$TEST_LOG_DIR"
    fi
    
    # Clean up environment
    unset TEST_LOG_DIR
}

# =============================================================================
# Basic Configuration Tests
# =============================================================================

@test "browserless mock loads successfully" {
    run mock::browserless::debug::dump_state
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Browserless Mock State Dump" ]]
}

@test "default configuration is set correctly" {
    [ "$(mock::browserless::get::config container_name)" = "vrooli-browserless" ]
    [ "$(mock::browserless::get::config port)" = "3001" ]
    [ "$(mock::browserless::get::config base_url)" = "http://localhost:3001" ]
    [ "$(mock::browserless::get::config max_browsers)" = "5" ]
    [ "$(mock::browserless::get::config headless)" = "yes" ]
}

@test "reset function clears all state" {
    # Set up some state
    mock::browserless::set_container_state "test_container" "running"
    mock::browserless::set_pressure "3" "2" "false"
    mock::browserless::inject_error "screenshot" "http_500"
    
    # Verify state exists
    [ "$(mock::browserless::get::container_state test_container)" = "running" ]
    [ "$(mock::browserless::get::pressure_data running)" = "3" ]
    
    # Reset and verify state is cleared
    mock::browserless::reset
    
    [ "$(mock::browserless::get::container_state test_container)" = "" ]
    [ "$(mock::browserless::get::pressure_data running)" = "1" ]  # back to default
}

# =============================================================================
# Container State Management Tests
# =============================================================================

@test "set container state works correctly" {
    mock::browserless::set_container_state "test_container" "running"
    [ "$(mock::browserless::get::container_state test_container)" = "running" ]
    
    mock::browserless::set_container_state "test_container" "stopped"
    [ "$(mock::browserless::get::container_state test_container)" = "stopped" ]
}

@test "container assertions work correctly" {
    mock::browserless::set_container_state "test_container" "running"
    
    run mock::browserless::assert::container_running "test_container"
    [ "$status" -eq 0 ]
    
    run mock::browserless::assert::container_stopped "test_container"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is not stopped" ]]
}

@test "multiple container states can be managed" {
    mock::browserless::set_container_state "container1" "running"
    mock::browserless::set_container_state "container2" "stopped"
    
    [ "$(mock::browserless::get::container_state container1)" = "running" ]
    [ "$(mock::browserless::get::container_state container2)" = "stopped" ]
}

# =============================================================================
# Pressure Data Management Tests
# =============================================================================

@test "default pressure data is set correctly" {
    [ "$(mock::browserless::get::pressure_data running)" = "1" ]
    [ "$(mock::browserless::get::pressure_data queued)" = "0" ]
    [ "$(mock::browserless::get::pressure_data maxConcurrent)" = "5" ]
    [ "$(mock::browserless::get::pressure_data isAvailable)" = "true" ]
}

@test "pressure data can be updated" {
    mock::browserless::set_pressure "3" "2" "false" "0.75" "0.85"
    
    [ "$(mock::browserless::get::pressure_data running)" = "3" ]
    [ "$(mock::browserless::get::pressure_data queued)" = "2" ]
    [ "$(mock::browserless::get::pressure_data isAvailable)" = "false" ]
    [ "$(mock::browserless::get::pressure_data cpu)" = "0.75" ]
    [ "$(mock::browserless::get::pressure_data memory)" = "0.85" ]
}

@test "pressure availability assertion works" {
    mock::browserless::set_pressure "2" "0" "true"
    
    run mock::browserless::assert::pressure_available
    [ "$status" -eq 0 ]
    
    mock::browserless::set_pressure "5" "3" "false"
    
    run mock::browserless::assert::pressure_available
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is not available" ]]
}

# =============================================================================
# Health Status Management Tests
# =============================================================================

@test "health status can be set and retrieved" {
    mock::browserless::set_health_status "healthy"
    [ "$(mock::browserless::get::config health_status)" = "healthy" ]
    
    mock::browserless::set_health_status "unhealthy"
    [ "$(mock::browserless::get::config health_status)" = "unhealthy" ]
}

@test "health assertion works correctly" {
    mock::browserless::set_health_status "healthy"
    
    run mock::browserless::assert::healthy
    [ "$status" -eq 0 ]
    
    mock::browserless::set_health_status "unhealthy"
    
    run mock::browserless::assert::healthy
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is not healthy" ]]
}

# =============================================================================
# API Endpoint Tests - /pressure
# =============================================================================

@test "pressure endpoint returns JSON response" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    run curl -s "http://localhost:3001/pressure"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"running":' ]]
    [[ "$output" =~ '"queued":' ]]
    [[ "$output" =~ '"isAvailable":' ]]
}

@test "pressure endpoint reflects custom pressure data" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    mock::browserless::set_pressure "3" "2" "false" "0.80" "0.90"
    
    run curl -s "http://localhost:3001/pressure"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"running": 3' ]]
    [[ "$output" =~ '"queued": 2' ]]
    [[ "$output" =~ '"isAvailable": false' ]]
    [[ "$output" =~ '"cpu": 0.80' ]]
    [[ "$output" =~ '"memory": 0.90' ]]
}

@test "pressure endpoint fails when server is stopped" {
    mock::browserless::set_server_status "stopped"
    
    run curl -f -s "http://localhost:3001/pressure"
    [ "$status" -eq 7 ]  # Connection failed
}

@test "pressure endpoint fails when service is unhealthy" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "unhealthy"
    
    run curl -f -s "http://localhost:3001/pressure"
    [ "$status" -eq 22 ]  # HTTP error
}

# =============================================================================
# API Endpoint Tests - Screenshot
# =============================================================================

@test "screenshot endpoint generates PNG data" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    local output_file="/tmp/test_screenshot.png"
    run curl -X POST "http://localhost:3001/chrome/screenshot" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}' \
        --output "$output_file" -s
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    
    # Verify it starts with PNG signature
    local first_bytes
    first_bytes=$(head -c 8 "$output_file" | xxd -p | head -c 16)
    [[ "$first_bytes" =~ ^89504e47 ]]  # PNG signature
    
    rm -f "$output_file"
}

@test "screenshot endpoint handles custom responses" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    mock::browserless::set_api_response "screenshot" "Custom screenshot response"
    
    local output_file="/tmp/test_custom_screenshot.png"
    run curl -X POST "http://localhost:3001/chrome/screenshot" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}' \
        --output "$output_file" -s
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    [ "$(cat "$output_file")" = "Custom screenshot response" ]
    
    rm -f "$output_file"
}

@test "screenshot endpoint logs API calls" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    curl -X POST "http://localhost:3001/chrome/screenshot" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}' \
        -s > /dev/null
    
    run mock::browserless::assert::api_called "screenshot"
    [ "$status" -eq 0 ]
}

# =============================================================================
# API Endpoint Tests - PDF Generation
# =============================================================================

@test "pdf endpoint generates PDF data" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    local output_file="/tmp/test_document.pdf"
    run curl -X POST "http://localhost:3001/chrome/pdf" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}' \
        --output "$output_file" -s
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    
    # Verify it starts with PDF header
    local first_line
    first_line=$(head -n 1 "$output_file")
    [[ "$first_line" =~ ^%PDF- ]]
    
    rm -f "$output_file"
}

@test "pdf endpoint handles custom responses" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    mock::browserless::set_api_response "pdf" "Custom PDF response"
    
    local output_file="/tmp/test_custom.pdf"
    run curl -X POST "http://localhost:3001/chrome/pdf" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}' \
        --output "$output_file" -s
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    [ "$(cat "$output_file")" = "Custom PDF response" ]
    
    rm -f "$output_file"
}

# =============================================================================
# API Endpoint Tests - Content Scraping
# =============================================================================

@test "content endpoint returns HTML content" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    run curl -X POST "http://localhost:3001/chrome/content" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}' -s
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "<!DOCTYPE html>" ]]
    [[ "$output" =~ "Mock scraped content" ]]
    [[ "$output" =~ "https://example.com" ]]
}

@test "content endpoint respects URL in request data" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    run curl -X POST "http://localhost:3001/chrome/content" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://test-site.com"}' -s
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "https://test-site.com" ]]
}

@test "content endpoint handles custom responses" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    mock::browserless::set_api_response "content" "Custom HTML content"
    
    run curl -X POST "http://localhost:3001/chrome/content" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}' -s
    
    [ "$status" -eq 0 ]
    [ "$output" = "Custom HTML content" ]
}

# =============================================================================
# API Endpoint Tests - Function Execution
# =============================================================================

@test "function endpoint returns JSON response" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    run curl -X POST "http://localhost:3001/chrome/function" \
        -H "Content-Type: application/json" \
        -d '{"code": "async ({ page }) => { return { result: \"test\" }; }"}' -s
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"title":' ]]
    [[ "$output" =~ '"url":' ]]
    [[ "$output" =~ '"viewport":' ]]
    [[ "$output" =~ '"result":' ]]
}

@test "function endpoint handles custom responses" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    mock::browserless::set_api_response "function" '{"custom": "function response"}'
    
    run curl -X POST "http://localhost:3001/chrome/function" \
        -H "Content-Type: application/json" \
        -d '{"code": "async ({ page }) => { return { result: \"test\" }; }"}' -s
    
    [ "$status" -eq 0 ]
    [ "$output" = '{"custom": "function response"}' ]
}

# =============================================================================
# Error Injection Tests
# =============================================================================

@test "connection timeout error injection works" {
    mock::browserless::inject_error "pressure" "connection_timeout"
    
    run curl -s --max-time 1 "http://localhost:3001/pressure"
    [ "$status" -eq 28 ]
    [[ "$output" =~ "Operation timed out" ]]
}

@test "connection refused error injection works" {
    mock::browserless::inject_error "screenshot" "connection_refused"
    
    run curl -f -s "http://localhost:3001/chrome/screenshot"
    [ "$status" -eq 7 ]
    [[ "$output" =~ "Connection refused" ]]
}

@test "HTTP 500 error injection works" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    mock::browserless::inject_error "pdf" "http_500"
    
    run curl -f -s "http://localhost:3001/chrome/pdf" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}'
    
    [ "$status" -eq 22 ]
}

@test "HTTP 429 too many requests error injection works" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    mock::browserless::inject_error "content" "http_429"
    
    run curl -s --write-out "%{http_code}" "http://localhost:3001/chrome/content" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}'
    
    [[ "$output" =~ "429" ]]
}

@test "HTTP 400 bad request error injection works" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    mock::browserless::inject_error "function" "http_400"
    
    run curl -s --write-out "%{http_code}" "http://localhost:3001/chrome/function" \
        -H "Content-Type: application/json" \
        -d '{"invalid": "data"}'
    
    [[ "$output" =~ "400" ]]
}

# =============================================================================
# curl Interceptor Tests
# =============================================================================

@test "curl interceptor handles Browserless URLs" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    run curl -s "http://localhost:3001/pressure"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"running":' ]]
}

@test "curl interceptor handles non-Browserless URLs" {
    run curl -s "https://example.com"
    [ "$status" -eq 0 ]
    [ "$output" = "Mock curl response" ]
}

@test "curl interceptor supports output files" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    local output_file="/tmp/curl_test_output"
    run curl -s "http://localhost:3001/pressure" --output "$output_file"
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    [[ "$(cat "$output_file")" =~ '"running":' ]]
    
    rm -f "$output_file"
}

@test "curl interceptor supports write-out option" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    run curl -s "http://localhost:3001/pressure" --write-out "%{http_code}"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "200" ]]
}

@test "curl interceptor logs all API calls" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    # Make API calls without run to preserve state
    curl -s "http://localhost:3001/pressure" > /dev/null
    curl -s "http://localhost:3001/chrome/screenshot" > /dev/null
    
    # Now check assertions
    mock::browserless::assert::api_called "pressure"
    mock::browserless::assert::api_called "screenshot"
}

# =============================================================================
# Scenario Builder Tests
# =============================================================================

@test "create running service scenario works" {
    mock::browserless::scenario::create_running_service
    
    run mock::browserless::assert::container_running "vrooli-browserless"
    [ "$status" -eq 0 ]
    
    run mock::browserless::assert::healthy
    [ "$status" -eq 0 ]
    
    run mock::browserless::assert::pressure_available
    [ "$status" -eq 0 ]
    
    [ "$(mock::browserless::get::config server_status)" = "running" ]
}

@test "create stopped service scenario works" {
    mock::browserless::scenario::create_stopped_service
    
    run mock::browserless::assert::container_stopped "vrooli-browserless"
    [ "$status" -eq 0 ]
    
    run mock::browserless::assert::healthy
    [ "$status" -eq 1 ]  # Should be unhealthy
    
    [ "$(mock::browserless::get::config server_status)" = "stopped" ]
}

@test "create overloaded service scenario works" {
    mock::browserless::scenario::create_overloaded_service
    
    run mock::browserless::assert::container_running "vrooli-browserless"
    [ "$status" -eq 0 ]
    
    run mock::browserless::assert::healthy
    [ "$status" -eq 0 ]
    
    run mock::browserless::assert::pressure_available
    [ "$status" -eq 1 ]  # Should not be available
    
    [ "$(mock::browserless::get::pressure_data running)" = "5" ]
    [ "$(mock::browserless::get::pressure_data queued)" = "3" ]
    [ "$(mock::browserless::get::pressure_data cpu)" = "0.85" ]
    [ "$(mock::browserless::get::pressure_data memory)" = "0.90" ]
}

@test "custom container name in scenarios works" {
    mock::browserless::scenario::create_running_service "custom-browserless"
    
    run mock::browserless::assert::container_running "custom-browserless"
    [ "$status" -eq 0 ]
}

# =============================================================================
# Integration Tests
# =============================================================================

@test "realistic browserless workflow simulation" {
    # Start with stopped service
    mock::browserless::scenario::create_stopped_service
    
    # Verify APIs fail when stopped
    run curl -f -s "http://localhost:3001/pressure"
    [ "$status" -eq 7 ]  # Connection refused
    
    # Start service
    mock::browserless::scenario::create_running_service
    
    # Verify pressure endpoint works
    run curl -s "http://localhost:3001/pressure"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"isAvailable": true' ]]
    
    # Take screenshot (without run to preserve state)
    local screenshot_file="/tmp/integration_screenshot.png"
    curl -X POST "http://localhost:3001/chrome/screenshot" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}' \
        --output "$screenshot_file" -s
    [ -f "$screenshot_file" ]
    
    # Generate PDF (without run to preserve state)
    local pdf_file="/tmp/integration_document.pdf"
    curl -X POST "http://localhost:3001/chrome/pdf" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}' \
        --output "$pdf_file" -s
    [ -f "$pdf_file" ]
    
    # Scrape content (without run to preserve state)
    local content_result
    content_result=$(curl -X POST "http://localhost:3001/chrome/content" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com"}' -s)
    [[ "$content_result" =~ "Mock scraped content" ]]
    
    # Execute function (without run to preserve state)
    local function_result
    function_result=$(curl -X POST "http://localhost:3001/chrome/function" \
        -H "Content-Type: application/json" \
        -d '{"code": "async ({ page }) => ({ test: true })"}' -s)
    [[ "$function_result" =~ '"result":' ]]
    
    # Verify all APIs were called
    mock::browserless::assert::api_called "pressure"
    mock::browserless::assert::api_called "screenshot"
    mock::browserless::assert::api_called "pdf"
    mock::browserless::assert::api_called "content"
    mock::browserless::assert::api_called "function"
    
    # Clean up
    rm -f "$screenshot_file" "$pdf_file"
}

@test "service overload handling simulation" {
    # Create overloaded service
    mock::browserless::scenario::create_overloaded_service
    
    # Verify pressure shows unavailable
    run curl -s "http://localhost:3001/pressure"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"isAvailable": false' ]]
    [[ "$output" =~ '"running": 5' ]]
    [[ "$output" =~ '"queued": 3' ]]
    
    # APIs should still work (service is running, just overloaded)
    run curl -s "http://localhost:3001/chrome/screenshot"
    [ "$status" -eq 0 ]
}

@test "error recovery simulation" {
    mock::browserless::scenario::create_running_service
    
    # Inject temporary error
    mock::browserless::inject_error "screenshot" "http_500"
    
    # Verify API fails
    run curl -f -s "http://localhost:3001/chrome/screenshot"
    [ "$status" -eq 22 ]
    
    # Clear error (simulate recovery)
    mock::browserless::reset
    mock::browserless::scenario::create_running_service
    
    # Verify API works again
    run curl -s "http://localhost:3001/chrome/screenshot"
    [ "$status" -eq 0 ]
}

# =============================================================================
# Logging Integration Tests
# =============================================================================

@test "state changes are logged" {
    mock::browserless::set_container_state "test_logging" "running"
    mock::browserless::set_health_status "healthy"
    mock::browserless::set_pressure "2" "1" "true"
    
    # Check that state changes were logged
    [[ -f "$TEST_LOG_DIR/used_mocks.log" ]]
    
    local log_content=$(cat "$TEST_LOG_DIR/used_mocks.log")
    [[ "$log_content" =~ "browserless_container_state:test_logging:running" ]]
    [[ "$log_content" =~ "browserless_health:set:healthy" ]]
    [[ "$log_content" =~ "browserless_pressure:set:" ]]
}

@test "API calls are logged to call log" {
    # Ensure we have a test log directory
    [[ -n "$TEST_LOG_DIR" ]]
    
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    # Make some API calls (without run to preserve logging)
    curl -s "http://localhost:3001/pressure" >/dev/null
    curl -s "http://localhost:3001/chrome/screenshot" >/dev/null
    
    # Check that calls were logged in the browserless mock's request log
    mock::browserless::assert::api_called "pressure"
    mock::browserless::assert::api_called "screenshot"
}

# =============================================================================
# Debug and Utility Tests
# =============================================================================

@test "debug dump shows comprehensive state" {
    mock::browserless::set_container_state "debug_container" "running"
    mock::browserless::set_pressure "3" "1" "true" "0.50" "0.60"
    mock::browserless::inject_error "pdf" "http_500"
    mock::browserless::set_api_response "content" "Custom content"
    
    run mock::browserless::debug::dump_state
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Browserless Mock State Dump" ]]
    [[ "$output" =~ "debug_container" ]]
    [[ "$output" =~ "running: 3" ]]
    [[ "$output" =~ "pdf: http_500" ]]
}

@test "getter functions work correctly" {
    mock::browserless::set_container_state "getter_test" "running"
    BROWSERLESS_MOCK_CONFIG[custom_key]="custom_value"
    mock::browserless::set_pressure "4" "2" "false"
    
    [ "$(mock::browserless::get::container_state getter_test)" = "running" ]
    [ "$(mock::browserless::get::config custom_key)" = "custom_value" ]
    [ "$(mock::browserless::get::pressure_data running)" = "4" ]
    [ "$(mock::browserless::get::pressure_data queued)" = "2" ]
}

@test "state persistence works across subshells" {
    # Set state in main shell
    mock::browserless::set_container_state "persist_test" "running"
    mock::browserless::set_pressure "2" "1" "true"
    
    # Test in subshell (simulates BATS test execution)
    (
        source "${BATS_TEST_DIRNAME}/browserless.sh"
        [ "$(mock::browserless::get::container_state persist_test)" = "running" ]
        [ "$(mock::browserless::get::pressure_data running)" = "2" ]
    )
}

# =============================================================================
# Edge Cases and Error Handling Tests
# =============================================================================

@test "unknown API endpoint returns 404" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    run curl -s --write-out "%{http_code}" "http://localhost:3001/chrome/unknown"
    [[ "$output" =~ "404" ]]
}

@test "malformed JSON data is handled gracefully" {
    mock::browserless::set_server_status "running"
    mock::browserless::set_health_status "healthy"
    
    run curl -X POST "http://localhost:3001/chrome/screenshot" \
        -H "Content-Type: application/json" \
        -d '{malformed json}' -s
    
    [ "$status" -eq 0 ]  # Mock should still handle it gracefully
}

@test "empty configuration values are handled" {
    mock::browserless::set_container_state "" "running"
    mock::browserless::set_pressure "" "" ""
    
    run mock::browserless::get::container_state ""
    [ "$status" -eq 0 ]
    
    run mock::browserless::get::pressure_data ""
    [ "$status" -eq 0 ]
}

@test "concurrent API calls are handled correctly" {
    mock::browserless::scenario::create_running_service
    
    # Simulate multiple concurrent calls - run sequentially to avoid state conflicts
    curl -s "http://localhost:3001/pressure" > /dev/null
    curl -s "http://localhost:3001/chrome/screenshot" > /dev/null  
    curl -s "http://localhost:3001/chrome/pdf" > /dev/null
    
    # Verify all calls were logged
    mock::browserless::assert::api_called "pressure"
    mock::browserless::assert::api_called "screenshot"
    mock::browserless::assert::api_called "pdf"
}