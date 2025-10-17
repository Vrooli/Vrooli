#!/bin/bash
# Home Assistant Test Functions

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HOME_ASSISTANT_TEST_DIR="${APP_ROOT}/resources/home-assistant"

# Source dependencies
source "${HOME_ASSISTANT_TEST_DIR}/lib/core.sh"
source "${HOME_ASSISTANT_TEST_DIR}/lib/health.sh"
source "${HOME_ASSISTANT_TEST_DIR}/lib/inject.sh"

#######################################
# Run smoke tests - Quick validation
# Returns: 0 if all pass, 1 otherwise
#######################################
home_assistant::test::smoke() {
    log::header "Home Assistant Smoke Tests"
    
    local failed=0
    
    # Test 1: Container exists
    log::info "Test: Container exists..."
    if docker::container_exists "$HOME_ASSISTANT_CONTAINER_NAME"; then
        log::success "✓ Container exists"
    else
        log::error "✗ Container does not exist"
        ((failed++))
    fi
    
    # Test 2: Container is running
    log::info "Test: Container is running..."
    if docker::is_running "$HOME_ASSISTANT_CONTAINER_NAME"; then
        log::success "✓ Container is running"
    else
        log::error "✗ Container is not running"
        ((failed++))
    fi
    
    # Test 3: Health endpoint responds
    log::info "Test: Health endpoint responds..."
    if timeout 5 curl -sf "http://localhost:${HOME_ASSISTANT_PORT}/api/" &>/dev/null; then
        log::success "✓ API endpoint responds"
    else
        # Check if it returns 401 (auth required but running)
        local response
        response=$(timeout 5 curl -s -o /dev/null -w "%{http_code}" "http://localhost:${HOME_ASSISTANT_PORT}/api/" 2>/dev/null)
        if [[ "$response" == "401" ]]; then
            log::success "✓ API endpoint responds (auth required)"
        else
            log::error "✗ API endpoint not responding (code: $response)"
            ((failed++))
        fi
    fi
    
    # Test 4: Web UI accessible
    log::info "Test: Web UI accessible..."
    if timeout 5 curl -sf "http://localhost:${HOME_ASSISTANT_PORT}/" &>/dev/null; then
        log::success "✓ Web UI is accessible"
    else
        log::error "✗ Web UI is not accessible"
        ((failed++))
    fi
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All smoke tests passed!"
        return 0
    else
        log::error "$failed smoke test(s) failed"
        return 1
    fi
}

#######################################
# Run integration tests - Full functionality
# Returns: 0 if all pass, 1 otherwise
#######################################
home_assistant::test::integration() {
    log::header "Home Assistant Integration Tests"
    
    local failed=0
    
    # Test 1: API health check
    log::info "Test: API health check..."
    if home_assistant::health::is_healthy; then
        log::success "✓ Health check passes"
    else
        log::error "✗ Health check failed"
        ((failed++))
    fi
    
    # Test 2: Get detailed status
    log::info "Test: Get detailed status..."
    local status
    status=$(home_assistant::health::get_status)
    if [[ -n "$status" ]]; then
        log::success "✓ Status retrieval works"
        echo "$status" | jq . 2>/dev/null || echo "$status"
    else
        log::error "✗ Failed to get status"
        ((failed++))
    fi
    
    # Test 3: Configuration persistence
    log::info "Test: Configuration directory exists..."
    if [[ -d "$HOME_ASSISTANT_CONFIG_DIR" ]]; then
        log::success "✓ Configuration directory exists"
        
        # Check for key config files
        if [[ -f "$HOME_ASSISTANT_CONFIG_DIR/configuration.yaml" ]]; then
            log::success "✓ Main configuration file exists"
        else
            log::warning "⚠ Main configuration file not yet created"
        fi
    else
        log::error "✗ Configuration directory missing"
        ((failed++))
    fi
    
    # Test 4: Container restart resilience
    log::info "Test: Container restart resilience..."
    if docker::is_running "$HOME_ASSISTANT_CONTAINER_NAME"; then
        log::info "Restarting container..."
        docker restart "$HOME_ASSISTANT_CONTAINER_NAME" >/dev/null 2>&1
        sleep 5
        
        if home_assistant::health::wait_for_healthy 30; then
            log::success "✓ Restart successful and healthy"
        else
            log::error "✗ Failed to become healthy after restart"
            ((failed++))
        fi
    else
        log::warning "⚠ Skipping restart test - container not running"
    fi
    
    # Test 5: API discovery endpoint
    log::info "Test: API discovery endpoint..."
    local discovery_response
    discovery_response=$(timeout 5 curl -s "http://localhost:${HOME_ASSISTANT_PORT}/api/discovery_info" 2>/dev/null)
    if [[ -n "$discovery_response" ]]; then
        log::success "✓ Discovery endpoint accessible"
    else
        log::warning "⚠ Discovery endpoint not accessible (may require auth)"
    fi
    
    # Test 6: Backup functionality
    log::info "Test: Backup functionality..."
    local backup_output
    backup_output=$(home_assistant::backup 2>&1)
    local backup_file=$(echo "$backup_output" | grep -o "/.*backup_.*\.tar\.gz" | tail -1)
    if [[ -f "$backup_file" ]]; then
        log::success "✓ Backup created successfully: $(basename "$backup_file")"
    else
        log::error "✗ Failed to create backup"
        ((failed++))
    fi
    
    # Test 7: List backups
    log::info "Test: List backups..."
    local backup_list
    backup_list=$(home_assistant::backup::list 2>&1)
    if echo "$backup_list" | grep -q "backup_"; then
        log::success "✓ Backup listing works"
    else
        log::error "✗ Failed to list backups"
        ((failed++))
    fi
    
    # Test 8: Webhook support
    log::info "Test: Webhook endpoint..."
    local webhook_response
    webhook_response=$(timeout 5 curl -sf -X POST "http://localhost:${HOME_ASSISTANT_PORT}/api/webhook/test_webhook" \
        -H "Content-Type: application/json" \
        -d '{"test": "data"}' \
        -w "%{http_code}" -o /dev/null 2>/dev/null || echo "000")
    if [[ "$webhook_response" == "200" ]]; then
        log::success "✓ Webhook endpoint responds correctly"
    else
        log::warning "⚠ Webhook endpoint returned: $webhook_response"
    fi
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All integration tests passed!"
        return 0
    else
        log::error "$failed integration test(s) failed"
        return 1
    fi
}

#######################################
# Run unit tests - Library functions
# Returns: 0 if all pass, 1 otherwise
#######################################
home_assistant::test::unit() {
    log::header "Home Assistant Unit Tests"
    
    local failed=0
    
    # Test 1: Port registry integration
    log::info "Test: Port registry integration..."
    local port
    port=$(home_assistant::get_port)
    if [[ "$port" == "$HOME_ASSISTANT_PORT" ]]; then
        log::success "✓ Port retrieval works: $port"
    else
        log::error "✗ Port mismatch: expected $HOME_ASSISTANT_PORT, got $port"
        ((failed++))
    fi
    
    # Test 2: Configuration export
    log::info "Test: Configuration export..."
    home_assistant::export_config
    if [[ -n "$HOME_ASSISTANT_CONTAINER_NAME" ]] && [[ -n "$HOME_ASSISTANT_PORT" ]]; then
        log::success "✓ Configuration exported successfully"
    else
        log::error "✗ Configuration export incomplete"
        ((failed++))
    fi
    
    # Test 3: API info generation
    log::info "Test: API info generation..."
    local api_info
    api_info=$(home_assistant::get_api_info)
    if echo "$api_info" | jq . >/dev/null 2>&1; then
        log::success "✓ API info is valid JSON"
        echo "$api_info" | jq .
    else
        log::error "✗ API info is not valid JSON"
        ((failed++))
    fi
    
    # Test 4: Directory initialization
    log::info "Test: Directory initialization..."
    home_assistant::init
    if [[ -d "$HOME_ASSISTANT_CONFIG_DIR" ]]; then
        log::success "✓ Directories initialized"
    else
        log::error "✗ Failed to initialize directories"
        ((failed++))
    fi
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All unit tests passed!"
        return 0
    else
        log::error "$failed unit test(s) failed"
        return 1
    fi
}

#######################################
# Run all tests
# Returns: 0 if all pass, 1 otherwise
#######################################
home_assistant::test::all() {
    log::header "Running All Home Assistant Tests"
    
    local overall_failed=0
    
    # Run test suites in order
    if ! home_assistant::test::smoke; then
        ((overall_failed++))
    fi
    
    if ! home_assistant::test::integration; then
        ((overall_failed++))
    fi
    
    if ! home_assistant::test::unit; then
        ((overall_failed++))
    fi
    
    # Final summary
    echo
    log::header "Test Suite Summary"
    if [[ $overall_failed -eq 0 ]]; then
        log::success "✓ All test suites passed!"
        return 0
    else
        log::error "✗ $overall_failed test suite(s) failed"
        return 1
    fi
}

# Export functions
export -f home_assistant::test::smoke
export -f home_assistant::test::integration
export -f home_assistant::test::unit
export -f home_assistant::test::all