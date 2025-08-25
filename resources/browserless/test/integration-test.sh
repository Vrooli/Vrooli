#!/usr/bin/env bash
# Browserless Integration Tests
# Comprehensive testing using enhanced-integration-test-lib.sh

set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
TEST_DIR="${APP_ROOT}/resources/browserless/test"
BROWSERLESS_DIR="${APP_ROOT}/resources/browserless"

# Source test framework
# shellcheck disable=SC1091
source "${BROWSERLESS_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_TESTS_LIB_DIR}/enhanced-integration-test-lib.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${BROWSERLESS_DIR}/config/defaults.sh"
# Export configuration
browserless::export_config

# Source management functions
# shellcheck disable=SC1091
source "${BROWSERLESS_DIR}/lib/core.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_DIR}/lib/health.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_DIR}/lib/recovery.sh"

# Test configuration
export SERVICE_NAME="browserless"
export SERVICE_PORT="${BROWSERLESS_PORT}"
export SERVICE_CONTAINER="${BROWSERLESS_CONTAINER_NAME}"
export SERVICE_HEALTH_ENDPOINT="/pressure"
export SERVICE_API_BASE="${BROWSERLESS_BASE_URL}"

# Define test suites
SERVICE_TESTS=(
    "test_browserless_installation"
    "test_browserless_health_endpoint"
    "test_browserless_api_endpoints"
    "test_container_health"
    "test_configuration_validation"
    "test_browserless_screenshot"
    "test_browserless_pdf"
    "test_browserless_scraping"
    "test_browserless_function"
    "test_browserless_pressure"
    "test_backup_recovery"
    "run_browserless_fixture_tests"
)

#######################################
# Test Browserless installation
#######################################
test_browserless_installation() {
    test::start "Browserless Installation"
    
    # Check if already installed
    if docker::container_exists "$SERVICE_CONTAINER"; then
        test::pass "Browserless already installed"
        
        # Ensure it's running
        if docker::is_running "$SERVICE_CONTAINER"; then
            test::pass "Container is running"
        else
            # Try to start it
            if browserless::start; then
                test::pass "Container started successfully"
            else
                test::fail "Failed to start existing container"
                return 1
            fi
        fi
    else
        # Install Browserless
        if browserless::install; then
            test::pass "Installation completed"
        else
            test::fail "Installation failed"
            return 1
        fi
    fi
    
    # Wait for service to be ready
    if browserless::wait_for_ready; then
        test::pass "Service is ready"
    else
        test::fail "Service failed to become ready"
        return 1
    fi
    
    test::success "Browserless installation test completed"
}

#######################################
# Test health endpoint
#######################################
test_browserless_health_endpoint() {
    test::start "Health Endpoint"
    
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" "${SERVICE_API_BASE}${SERVICE_HEALTH_ENDPOINT}")
    
    if [[ "$response" == "200" ]]; then
        test::pass "Health endpoint returned 200"
    else
        test::fail "Health endpoint returned $response"
        return 1
    fi
    
    # Check pressure data structure
    local pressure_data
    pressure_data=$(curl -s "${SERVICE_API_BASE}${SERVICE_HEALTH_ENDPOINT}")
    
    if echo "$pressure_data" | jq -e '.running' >/dev/null 2>&1; then
        test::pass "Pressure data contains 'running' field"
    else
        test::fail "Pressure data missing 'running' field"
        return 1
    fi
    
    if echo "$pressure_data" | jq -e '.maxConcurrent' >/dev/null 2>&1; then
        test::pass "Pressure data contains 'maxConcurrent' field"
    else
        test::fail "Pressure data missing 'maxConcurrent' field"
        return 1
    fi
    
    test::success "Health endpoint test completed"
}

#######################################
# Test API endpoints
#######################################
test_browserless_api_endpoints() {
    test::start "API Endpoints"
    
    local endpoints=(
        "/metrics"
        "/config"
        "/json/version"
    )
    
    for endpoint in "${endpoints[@]}"; do
        local response
        response=$(curl -s -o /dev/null -w "%{http_code}" "${SERVICE_API_BASE}${endpoint}")
        
        if [[ "$response" == "200" ]]; then
            test::pass "Endpoint $endpoint returned 200"
        else
            test::warn "Endpoint $endpoint returned $response (might not be available in this version)"
        fi
    done
    
    test::success "API endpoints test completed"
}

#######################################
# Test container health
#######################################
test_container_health() {
    test::start "Container Health"
    
    # Check if container exists
    if ! docker::container_exists "$SERVICE_CONTAINER"; then
        test::fail "Container does not exist"
        return 1
    fi
    
    # Check if container is running
    if docker::is_running "$SERVICE_CONTAINER"; then
        test::pass "Container is running"
    else
        test::fail "Container is not running"
        return 1
    fi
    
    # Check resource usage
    local stats
    stats=$(docker stats --no-stream --format "json" "$SERVICE_CONTAINER" 2>/dev/null || echo "{}")
    
    if [[ -n "$stats" ]] && [[ "$stats" != "{}" ]]; then
        test::pass "Container stats retrieved successfully"
        
        # Parse and display key metrics
        local cpu_percent mem_usage
        cpu_percent=$(echo "$stats" | jq -r '.CPUPerc' 2>/dev/null | tr -d '%' || echo "0")
        mem_usage=$(echo "$stats" | jq -r '.MemUsage' 2>/dev/null || echo "Unknown")
        
        test::info "CPU Usage: ${cpu_percent}%"
        test::info "Memory Usage: $mem_usage"
    else
        test::warn "Unable to retrieve container stats"
    fi
    
    test::success "Container health test completed"
}

#######################################
# Test configuration validation
#######################################
test_configuration_validation() {
    test::start "Configuration Validation"
    
    # Check if configuration is applied
    local config_data
    config_data=$(curl -s "${SERVICE_API_BASE}/config" 2>/dev/null || echo "{}")
    
    if [[ -n "$config_data" ]] && [[ "$config_data" != "{}" ]]; then
        test::pass "Configuration endpoint accessible"
        
        # Verify some expected configuration values
        local max_concurrent
        max_concurrent=$(echo "$config_data" | jq -r '.maxConcurrentSessions // .concurrent // 0' 2>/dev/null || echo "0")
        
        if [[ "$max_concurrent" -gt 0 ]]; then
            test::pass "Max concurrent sessions configured: $max_concurrent"
        else
            test::info "Max concurrent sessions not found in config (might use different field name)"
        fi
    else
        test::warn "Configuration endpoint not available (might not be supported in this version)"
    fi
    
    test::success "Configuration validation completed"
}

#######################################
# Test screenshot functionality
#######################################
test_browserless_screenshot() {
    test::start "Screenshot API"
    
    local test_url="https://example.com"
    local temp_file="/tmp/browserless_test_screenshot_$$.png"
    local response
    
    response=$(curl -s -X POST \
        "${SERVICE_API_BASE}/screenshot" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"$test_url\"}" \
        -w "\n%{http_code}" \
        -o "$temp_file" \
        2>/dev/null | tail -n1)
    
    if [[ "$response" == "200" ]]; then
        test::pass "Screenshot API returned 200"
        
        # Check if file was created and has content
        if [[ -f "$temp_file" ]] && [[ -s "$temp_file" ]]; then
            local file_size
            file_size=$(stat -f%z "$temp_file" 2>/dev/null || stat -c%s "$temp_file" 2>/dev/null || echo "0")
            test::pass "Screenshot file created (size: $file_size bytes)"
            rm -f "$temp_file"
        else
            test::fail "Screenshot file is empty or missing"
            rm -f "$temp_file"
            return 1
        fi
    else
        test::fail "Screenshot API returned $response"
        rm -f "$temp_file"
        return 1
    fi
    
    test::success "Screenshot API test completed"
}

#######################################
# Test PDF generation
#######################################
test_browserless_pdf() {
    test::start "PDF API"
    
    local test_url="https://example.com"
    local temp_file="/tmp/browserless_test_document_$$.pdf"
    local response
    
    response=$(curl -s -X POST \
        "${SERVICE_API_BASE}/pdf" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"$test_url\"}" \
        -w "\n%{http_code}" \
        -o "$temp_file" \
        2>/dev/null | tail -n1)
    
    if [[ "$response" == "200" ]]; then
        test::pass "PDF API returned 200"
        
        # Check if file was created and has content
        if [[ -f "$temp_file" ]] && [[ -s "$temp_file" ]]; then
            local file_size
            file_size=$(stat -f%z "$temp_file" 2>/dev/null || stat -c%s "$temp_file" 2>/dev/null || echo "0")
            test::pass "PDF file created (size: $file_size bytes)"
            
            # Verify it's actually a PDF
            if file "$temp_file" | grep -q "PDF"; then
                test::pass "File is a valid PDF"
            else
                test::warn "File might not be a valid PDF"
            fi
            rm -f "$temp_file"
        else
            test::fail "PDF file is empty or missing"
            rm -f "$temp_file"
            return 1
        fi
    else
        test::fail "PDF API returned $response"
        rm -f "$temp_file"
        return 1
    fi
    
    test::success "PDF API test completed"
}

#######################################
# Test web scraping
#######################################
test_browserless_scraping() {
    test::start "Scraping API"
    
    local test_url="https://example.com"
    local response_code
    local response_body
    
    response_body=$(curl -s -X POST \
        "${SERVICE_API_BASE}/content" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"$test_url\"}" \
        2>/dev/null)
    
    response_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
        "${SERVICE_API_BASE}/content" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"$test_url\"}")
    
    if [[ "$response_code" == "200" ]]; then
        test::pass "Content API returned 200"
        
        # Check if response contains expected content
        if echo "$response_body" | grep -qi "example" 2>/dev/null; then
            test::pass "Content contains expected text"
        else
            test::warn "Content might not be complete"
        fi
    else
        test::fail "Content API returned $response_code"
        return 1
    fi
    
    test::success "Scraping API test completed"
}

#######################################
# Test function execution
#######################################
test_browserless_function() {
    test::start "Function API"
    
    local code='() => { return document.title; }'
    local response
    local response_code
    
    response=$(curl -s -X POST \
        "${SERVICE_API_BASE}/function" \
        -H "Content-Type: application/json" \
        -d "{\"code\":\"$code\",\"context\":{}}" \
        2>/dev/null)
    
    response_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
        "${SERVICE_API_BASE}/function" \
        -H "Content-Type: application/json" \
        -d "{\"code\":\"$code\",\"context\":{}}")
    
    if [[ "$response_code" == "200" ]]; then
        test::pass "Function API returned 200"
        test::info "Function result: $(echo "$response" | head -c 100)"
    else
        test::warn "Function API returned $response_code (may not be available in all versions)"
    fi
    
    test::success "Function API test completed"
}

#######################################
# Test browser pressure
#######################################
test_browserless_pressure() {
    test::start "Browser Pressure"
    
    local pressure_data
    pressure_data=$(curl -s "${SERVICE_API_BASE}/pressure")
    
    if [[ -z "$pressure_data" ]]; then
        test::fail "Unable to get pressure data"
        return 1
    fi
    
    local running queued max_concurrent
    running=$(echo "$pressure_data" | jq -r '.running // -1' 2>/dev/null)
    queued=$(echo "$pressure_data" | jq -r '.queued // -1' 2>/dev/null)
    max_concurrent=$(echo "$pressure_data" | jq -r '.maxConcurrent // -1' 2>/dev/null)
    
    if [[ "$running" -ge 0 ]]; then
        test::pass "Running browsers: $running"
    else
        test::fail "Unable to get running browser count"
    fi
    
    if [[ "$queued" -ge 0 ]]; then
        test::pass "Queued requests: $queued"
    else
        test::fail "Unable to get queued request count"
    fi
    
    if [[ "$max_concurrent" -gt 0 ]]; then
        test::pass "Max concurrent browsers: $max_concurrent"
    else
        test::fail "Unable to get max concurrent browser count"
    fi
    
    # Test pressure under load
    test::info "Testing pressure under load..."
    
    # Send multiple concurrent requests
    for i in {1..3}; do
        curl -s -X POST \
            "${SERVICE_API_BASE}/screenshot" \
            -H "Content-Type: application/json" \
            -d '{"url":"https://example.com"}' \
            -o /dev/null &
    done
    
    # Wait a moment for requests to register
    sleep 2
    
    # Check pressure again
    pressure_data=$(curl -s "${SERVICE_API_BASE}/pressure")
    running=$(echo "$pressure_data" | jq -r '.running // 0' 2>/dev/null)
    
    if [[ "$running" -gt 0 ]]; then
        test::pass "Browsers actively processing requests: $running"
    else
        test::info "No active browsers (requests may have completed quickly)"
    fi
    
    # Wait for background jobs to complete
    wait
    
    test::success "Browser pressure test completed"
}

#######################################
# Test backup and recovery
#######################################
test_backup_recovery() {
    test::start "Backup and Recovery"
    
    # Create a backup
    local backup_label="test_backup_$$"
    if browserless::create_backup "$backup_label"; then
        test::pass "Backup created successfully"
    else
        test::fail "Failed to create backup"
        return 1
    fi
    
    # List backups to verify
    local backup_list
    backup_list=$(backup::list "browserless" 2>/dev/null || echo "")
    
    if echo "$backup_list" | grep -q "$backup_label"; then
        test::pass "Backup appears in list"
    else
        test::fail "Backup not found in list"
        return 1
    fi
    
    # Validate the backup
    if browserless::validate_backup; then
        test::pass "Backup validation successful"
    else
        test::warn "Backup validation had warnings"
    fi
    
    test::success "Backup and recovery test completed"
}

#######################################
# Run fixture-based tests
#######################################
run_browserless_fixture_tests() {
    test::start "Fixture Tests"
    
    # Test with known good URLs
    local test_sites=("https://example.com" "https://www.google.com" "https://httpbin.org/html")
    
    for site in "${test_sites[@]}"; do
        test::info "Testing with $site"
        
        # Test screenshot
        local response
        response=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
            "${SERVICE_API_BASE}/screenshot" \
            -H "Content-Type: application/json" \
            -d "{\"url\":\"$site\"}" \
            --max-time 30)
        
        if [[ "$response" == "200" ]]; then
            test::pass "Successfully captured screenshot of $site"
        else
            test::warn "Failed to capture $site (response: $response)"
        fi
    done
    
    # Test with different options
    test::info "Testing with custom options"
    
    local custom_response
    custom_response=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
        "${SERVICE_API_BASE}/screenshot" \
        -H "Content-Type: application/json" \
        -d '{
            "url": "https://example.com",
            "options": {
                "fullPage": true,
                "type": "png",
                "quality": 80
            }
        }' \
        --max-time 30)
    
    if [[ "$custom_response" == "200" ]]; then
        test::pass "Successfully captured with custom options"
    else
        test::warn "Failed with custom options (response: $custom_response)"
    fi
    
    test::success "Fixture tests completed"
}

#######################################
# Main test execution
#######################################
main() {
    test::header "Browserless Integration Tests"
    
    # Initialize test environment
    test::init
    
    # Check Docker availability
    if ! docker info >/dev/null 2>&1; then
        test::fail "Docker is not running or not accessible"
        exit 1
    fi
    
    # Check if Browserless is installed
    if ! docker::container_exists "$SERVICE_CONTAINER"; then
        log::info "Browserless not installed, installing for tests..."
        if ! browserless::install; then
            test::fail "Failed to install Browserless for testing"
            exit 1
        fi
    fi
    
    # Ensure container is running
    if ! docker::is_running "$SERVICE_CONTAINER"; then
        log::info "Starting Browserless container..."
        if ! browserless::start; then
            test::fail "Failed to start Browserless container"
            exit 1
        fi
    fi
    
    # Wait for service to be ready
    log::info "Waiting for Browserless to be ready..."
    if ! browserless::wait_for_ready; then
        test::fail "Browserless failed to become ready"
        exit 1
    fi
    
    # Run all tests
    local failed_tests=0
    local skipped_tests=0
    
    for test_func in "${SERVICE_TESTS[@]}"; do
        echo
        if declare -f "$test_func" >/dev/null; then
            if ! $test_func; then
                ((failed_tests++))
            fi
        else
            test::warn "Test function $test_func not found, skipping"
            ((skipped_tests++))
        fi
    done
    
    # Summary
    echo
    test::header "Test Summary"
    local total_tests=${#SERVICE_TESTS[@]}
    local passed_tests=$((total_tests - failed_tests - skipped_tests))
    
    echo "Total Tests: $total_tests"
    echo "Passed: $passed_tests"
    echo "Failed: $failed_tests"
    echo "Skipped: $skipped_tests"
    
    if [[ $failed_tests -eq 0 ]]; then
        test::success "All tests passed!"
        exit 0
    else
        test::fail "$failed_tests test(s) failed"
        exit 1
    fi
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi