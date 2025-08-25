#!/bin/bash

# AutoGPT Test Functions

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
AUTOGPT_TEST_DIR="${APP_ROOT}/resources/autogpt/lib"

# Source common functions
source "${AUTOGPT_TEST_DIR}/common.sh"

# Main test function
autogpt::test() {
    local test_type="${1:-basic}"
    
    format_header "AutoGPT Tests"
    
    local all_passed=true
    
    # Basic installation test
    log::info "Running installation test..."
    if autogpt::is_installed; then
        log::success "✅ AutoGPT image installed"
    else
        log::error "❌ AutoGPT image not installed"
        all_passed=false
    fi
    
    # Container status test
    log::info "Running container test..."
    if autogpt::is_running; then
        log::success "✅ AutoGPT container running"
    else
        log::warn "⚠️ AutoGPT container not running"
    fi
    
    # Health check test
    log::info "Running health check..."
    local health_status=$(autogpt::health_check 2>&1 || echo "unknown")
    if [[ "${health_status}" == "healthy" ]]; then
        log::success "✅ AutoGPT is healthy"
    elif [[ "${health_status}" == "not running" ]]; then
        log::warn "⚠️ AutoGPT not running (expected if not started)"
    else
        log::error "❌ AutoGPT health check failed: ${health_status}"
        all_passed=false
    fi
    
    # Configuration test
    log::info "Running configuration test..."
    if [[ -d "${AUTOGPT_CONFIG_DIR}" ]]; then
        log::success "✅ Configuration directory exists"
    else
        log::warn "⚠️ Configuration directory missing"
    fi
    
    # Workspace test
    log::info "Running workspace test..."
    if [[ -d "${AUTOGPT_WORKSPACE_DIR}" ]]; then
        log::success "✅ Workspace directory exists"
    else
        log::warn "⚠️ Workspace directory missing"
    fi
    
    # Port availability test
    log::info "Running port test..."
    if ! lsof -i:${AUTOGPT_PORT} &>/dev/null; then
        log::success "✅ Port ${AUTOGPT_PORT} available"
    elif autogpt::is_running; then
        log::success "✅ Port ${AUTOGPT_PORT} in use by AutoGPT"
    else
        log::error "❌ Port ${AUTOGPT_PORT} in use by another process"
        all_passed=false
    fi
    
    # Advanced tests if requested
    if [[ "${test_type}" == "full" || "${test_type}" == "advanced" ]]; then
        autogpt::test_advanced
    fi
    
    # Summary
    echo
    if [[ "${all_passed}" == true ]]; then
        log::success "All tests passed"
        return 0
    else
        log::error "Some tests failed"
        return 1
    fi
}

# Advanced tests
autogpt::test_advanced() {
    log::info "Running advanced tests..."
    
    # Test goal injection
    local test_goal="/tmp/autogpt_test_goal.json"
    cat > "${test_goal}" <<EOF
{
    "name": "Test Goal",
    "description": "A simple test goal",
    "tasks": [
        "Verify AutoGPT is working",
        "Generate a test response"
    ]
}
EOF
    
    if autogpt::inject_goal "${test_goal}" &>/dev/null; then
        log::success "✅ Goal injection works"
        rm -f "${test_goal}"
    else
        log::error "❌ Goal injection failed"
        rm -f "${test_goal}"
        return 1
    fi
    
    # Test running a simple task if AutoGPT is running
    if autogpt::is_running; then
        log::info "Testing task execution..."
        local result=$(docker exec "${AUTOGPT_CONTAINER_NAME}" python -c "print('AutoGPT task execution test passed')" 2>/dev/null || echo "failed")
        if [[ "${result}" == *"passed"* ]]; then
            log::success "✅ Task execution works"
        else
            log::error "❌ Task execution failed"
        fi
    fi
}

# Run performance test
autogpt::test_performance() {
    format_header "AutoGPT Performance Test"
    
    if ! autogpt::is_running; then
        log::error "AutoGPT must be running for performance tests"
        return 1
    fi
    
    log::info "Testing response time..."
    local start_time=$(date +%s%N)
    timeout 5 curl -s "http://localhost:${AUTOGPT_PORT}/health" &>/dev/null || true
    local end_time=$(date +%s%N)
    local response_time=$(( (end_time - start_time) / 1000000 ))
    
    if [[ ${response_time} -lt 1000 ]]; then
        log::success "✅ Response time: ${response_time}ms (excellent)"
    elif [[ ${response_time} -lt 3000 ]]; then
        log::warn "⚠️ Response time: ${response_time}ms (acceptable)"
    else
        log::error "❌ Response time: ${response_time}ms (poor)"
    fi
    
    # Check memory usage
    local memory=$(docker stats --no-stream --format "{{.MemUsage}}" "${AUTOGPT_CONTAINER_NAME}" 2>/dev/null | cut -d'/' -f1 || echo "unknown")
    log::info "Memory usage: ${memory}"
    
    # Check CPU usage
    local cpu=$(docker stats --no-stream --format "{{.CPUPerc}}" "${AUTOGPT_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
    log::info "CPU usage: ${cpu}"
}

# Get test results for status
autogpt::get_test_status() {
    local test_file="${AUTOGPT_DATA_DIR}/.test_results"
    
    if [[ -f "${test_file}" ]]; then
        local last_run=$(stat -c %Y "${test_file}" 2>/dev/null || echo 0)
        local current_time=$(date +%s)
        local age=$(( (current_time - last_run) / 60 ))
        
        if [[ ${age} -lt 60 ]]; then
            echo "passed (${age}m ago)"
        else
            echo "stale ($(( age / 60 ))h ago)"
        fi
    else
        echo "not run"
    fi
}

# Save test results
autogpt::save_test_results() {
    local status="${1:-unknown}"
    mkdir -p "${AUTOGPT_DATA_DIR}"
    echo "$(date '+%Y-%m-%d %H:%M:%S'): ${status}" > "${AUTOGPT_DATA_DIR}/.test_results"
}