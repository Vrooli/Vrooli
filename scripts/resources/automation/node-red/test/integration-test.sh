#!/usr/bin/env bash
# Node-RED Integration Test
# Comprehensive testing using enhanced integration test framework

set -euo pipefail

# Source enhanced integration test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NODE_RED_ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_TESTS_LIB_DIR}/enhanced-integration-test-lib.sh"

# Load Node-RED configuration
# shellcheck disable=SC1091
source "${NODE_RED_ROOT_DIR}/config/defaults.sh"
node_red::export_config

#######################################
# Test Configuration
#######################################
SERVICE_TESTS=(
    "test_node_red_installation"
    "test_web_and_api_accessibility"
    "test_container_health"
    "test_configuration_validation"
    "test_fixtures_and_examples"
)

#######################################
# Node-RED Installation Test
#######################################
test_node_red_installation() {
    test::header "Node-RED Installation Test"
    
    # Test Docker daemon
    if ! docker::check_daemon; then
        test::fail "Docker daemon not accessible"
        return 1
    fi
    test::pass "Docker daemon accessible"
    
    # Check if Node-RED is installed
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        test::info "Installing Node-RED for testing..."
        if ! node_red::install; then
            test::fail "Node-RED installation failed"
            return 1
        fi
    fi
    test::pass "Node-RED container exists"
    
    # Check if Node-RED is running
    if ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        test::info "Starting Node-RED..."
        if ! node_red::start; then
            test::fail "Failed to start Node-RED"
            return 1
        fi
    fi
    test::pass "Node-RED is running"
    
    return 0
}

#######################################
# Consolidated Web and API Accessibility Test
#######################################
test_web_and_api_accessibility() {
    test::header "Web Interface & API Accessibility Test"
    
    # Wait for Node-RED to be ready
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -sf "http://localhost:$NODE_RED_PORT" >/dev/null 2>&1; then
            break
        fi
        test::info "Waiting for Node-RED to be ready (attempt $attempt/$max_attempts)..."
        sleep 2
        ((attempt++))
    done
    
    if [[ $attempt -gt $max_attempts ]]; then
        test::fail "Node-RED not accessible after ${max_attempts} attempts"
        return 1
    fi
    test::pass "Node-RED is accessible"
    
    # Test main web interface
    local response_code content
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$NODE_RED_PORT" 2>/dev/null || echo "000")
    
    if [[ "$response_code" != "200" ]]; then
        test::fail "Web interface not accessible (HTTP $response_code)"
        return 1
    fi
    test::pass "Web interface accessible (HTTP $response_code)"
    
    # Test content validation
    content=$(curl -s "http://localhost:$NODE_RED_PORT" 2>/dev/null || echo "")
    if [[ "$content" == *"Node-RED"* ]]; then
        test::pass "Web interface contains expected Node-RED content"
    else
        test::warn "Web interface content validation failed"
    fi
    
    # Test API endpoints
    local flows_response settings_response
    flows_response=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$NODE_RED_PORT/flows" 2>/dev/null || echo "000")
    settings_response=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$NODE_RED_PORT/settings" 2>/dev/null || echo "000")
    
    if [[ "$flows_response" == "200" ]]; then
        test::pass "Flows API accessible (HTTP $flows_response)"
    else
        test::warn "Flows API not accessible (HTTP $flows_response)"
    fi
    
    if [[ "$settings_response" == "200" ]]; then
        test::pass "Settings API accessible (HTTP $settings_response)"
    else
        test::warn "Settings API not accessible (HTTP $settings_response)"
    fi
    
    return 0
}

#######################################
# Container Health Test
#######################################
test_container_health() {
    test::header "Container Health Test"
    
    # Test container status
    if docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        test::pass "Node-RED container is running"
    else
        test::fail "Node-RED container is not running"
        return 1
    fi
    
    # Test container resource usage
    local stats
    stats=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemUsage}}" "$NODE_RED_CONTAINER_NAME" 2>/dev/null || echo "")
    
    if [[ -n "$stats" ]]; then
        local cpu mem
        cpu=$(echo "$stats" | awk '{print $1}')
        mem=$(echo "$stats" | awk '{print $2}')
        test::info "Container resource usage - CPU: $cpu, Memory: $mem"
        test::pass "Container resource monitoring successful"
    else
        test::warn "Unable to retrieve container resource usage"
    fi
    
    # Test container logs
    local log_count
    log_count=$(docker logs --tail 10 "$NODE_RED_CONTAINER_NAME" 2>/dev/null | wc -l)
    
    if [[ "$log_count" -gt 0 ]]; then
        test::pass "Container logs accessible ($log_count recent lines)"
    else
        test::warn "No recent container logs found"
    fi
    
    return 0
}

#######################################
# Configuration Validation Test
#######################################
test_configuration_validation() {
    test::header "Configuration Validation Test"
    
    # Test data directory
    if [[ -d "$NODE_RED_DATA_DIR" ]]; then
        test::pass "Node-RED data directory exists: $NODE_RED_DATA_DIR"
    else
        test::fail "Node-RED data directory missing: $NODE_RED_DATA_DIR"
        return 1
    fi
    
    # Test network configuration
    if docker network inspect "$NODE_RED_NETWORK_NAME" >/dev/null 2>&1; then
        test::pass "Node-RED network exists: $NODE_RED_NETWORK_NAME"
    else
        test::warn "Node-RED network not found: $NODE_RED_NETWORK_NAME"
    fi
    
    # Test port accessibility
    if http::port_accessible "localhost" "$NODE_RED_PORT"; then
        test::pass "Node-RED port accessible: $NODE_RED_PORT"
    else
        test::fail "Node-RED port not accessible: $NODE_RED_PORT"
        return 1
    fi
    
    return 0
}

#######################################
# Consolidated Fixtures and Examples Test
#######################################
test_fixtures_and_examples() {
    test::header "Fixtures & Examples Validation Test"
    
    # Test example flows directory
    local example_count=0
    local invalid_count=0
    
    if [[ -d "${NODE_RED_ROOT_DIR}/examples" ]]; then
        example_count=$(find "${NODE_RED_ROOT_DIR}/examples" -name "*.json" -type f 2>/dev/null | wc -l)
        test::info "Found $example_count example flow files"
        
        # Validate JSON in all example flows
        while IFS= read -r -d '' flow_file; do
            if ! jq empty "$flow_file" 2>/dev/null; then
                test::warn "Invalid JSON in example flow: $(basename "$flow_file")"
                ((invalid_count++))
            fi
        done < <(find "${NODE_RED_ROOT_DIR}/examples" -name "*.json" -type f -print0 2>/dev/null)
        
        if [[ "$invalid_count" -eq 0 ]]; then
            test::pass "All $example_count example flows contain valid JSON"
        else
            test::fail "$invalid_count of $example_count example flows contain invalid JSON"
            return 1
        fi
    else
        test::info "No examples directory found - skipping flow validation"
    fi
    
    # Test default flows if they exist
    local default_flows="${NODE_RED_ROOT_DIR}/examples/default-flows.json"
    if [[ -f "$default_flows" ]]; then
        if jq empty "$default_flows" 2>/dev/null; then
            test::pass "Default flows file is valid JSON"
        else
            test::fail "Default flows file contains invalid JSON"
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Service-specific setup and teardown
#######################################
setup_node_red_test_environment() {
    test::info "Setting up Node-RED test environment..."
    
    # Ensure Node-RED is installed and running for tests
    test_node_red_installation
    
    # Wait a bit for full initialization
    sleep 3
}

cleanup_node_red_test_environment() {
    test::info "Cleaning up Node-RED test environment..."
    
    # Keep Node-RED running for manual inspection if needed
    # Uncomment the following line if you want to stop it after tests
    # node_red::stop >/dev/null 2>&1
}

#######################################
# Main execution
#######################################
main() {
    test::suite_header "Node-RED Integration Tests"
    
    # Setup test environment
    setup_node_red_test_environment
    
    # Run all tests
    test::run_test_suite SERVICE_TESTS
    
    # Cleanup
    cleanup_node_red_test_environment
    
    # Show summary
    test::suite_summary
}

# Load the Node-RED core module for testing
# shellcheck disable=SC1091
source "${NODE_RED_ROOT_DIR}/lib/core.sh"

# Run main function
main "$@"