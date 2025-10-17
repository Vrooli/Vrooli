#!/usr/bin/env bash
################################################################################
# Node-RED Test Library - v2.0 Contract Compliant
# 
# Test implementations for Node-RED resource validation
################################################################################

set -euo pipefail

# Test implementation functions that are called by CLI
node_red::test::smoke() {
    local test_script="${NODE_RED_CLI_DIR}/test/phases/test-smoke.sh"
    
    if [[ ! -f "$test_script" ]]; then
        log::error "Smoke test script not found"
        return 1
    fi
    
    log::info "Running Node-RED smoke tests..."
    timeout 30 bash "$test_script"
}

node_red::test::integration() {
    local test_script="${NODE_RED_CLI_DIR}/test/phases/test-integration.sh"
    
    if [[ ! -f "$test_script" ]]; then
        log::error "Integration test script not found"
        return 1
    fi
    
    log::info "Running Node-RED integration tests..."
    timeout 120 bash "$test_script"
}

node_red::test::unit() {
    local test_script="${NODE_RED_CLI_DIR}/test/phases/test-unit.sh"
    
    if [[ ! -f "$test_script" ]]; then
        log::error "Unit test script not found"
        return 1
    fi
    
    log::info "Running Node-RED unit tests..."
    timeout 60 bash "$test_script"
}

node_red::test::all() {
    local failed=0
    
    log::header "Running all Node-RED tests"
    
    # Run tests in order: smoke -> unit -> integration
    if ! node_red::test::smoke; then
        log::error "Smoke tests failed"
        ((failed++))
    fi
    
    if ! node_red::test::unit; then
        log::error "Unit tests failed"
        ((failed++))
    fi
    
    if ! node_red::test::integration; then
        log::error "Integration tests failed"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All tests passed!"
        return 0
    else
        log::error "$failed test suite(s) failed"
        return 1
    fi
}

# Performance test helper
node_red::test::performance() {
    log::header "Node-RED Performance Test"
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${NODE_RED_CONTAINER_NAME:-node-red}$"; then
        log::error "Node-RED must be running for performance tests"
        return 1
    fi
    
    local url="http://localhost:${NODE_RED_PORT:-1880}"
    
    # Test 1: API Response Time
    log::test "API response time"
    local total_time=0
    local iterations=10
    
    for ((i=1; i<=iterations; i++)); do
        local start_time end_time duration
        start_time=$(date +%s%N)
        timeout 5 curl -sf "${url}/flows" > /dev/null 2>&1
        end_time=$(date +%s%N)
        duration=$(( (end_time - start_time) / 1000000 ))
        total_time=$((total_time + duration))
    done
    
    local avg_time=$((total_time / iterations))
    
    if [[ $avg_time -lt 100 ]]; then
        log::success "Excellent performance: ${avg_time}ms average"
    elif [[ $avg_time -lt 200 ]]; then
        log::success "Good performance: ${avg_time}ms average"
    elif [[ $avg_time -lt 500 ]]; then
        log::warning "Acceptable performance: ${avg_time}ms average"
    else
        log::error "Poor performance: ${avg_time}ms average"
        return 1
    fi
    
    # Test 2: Memory Usage
    log::test "Memory usage"
    local mem_usage
    mem_usage=$(docker stats --no-stream --format "{{.MemUsage}}" "${NODE_RED_CONTAINER_NAME:-node-red}" | cut -d'/' -f1 | sed 's/[^0-9.]//g')
    
    if (( $(echo "$mem_usage < 512" | bc -l) )); then
        log::success "Memory usage acceptable: ${mem_usage}MB"
    else
        log::warning "High memory usage: ${mem_usage}MB"
    fi
    
    return 0
}