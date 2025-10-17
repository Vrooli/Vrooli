#!/usr/bin/env bash
################################################################################
# AutoGPT Test Library - v2.0 Contract Compliant
# Test implementations for AutoGPT resource
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"
TEST_DIR="${RESOURCE_DIR}/test"

# Source common functions
source "${SCRIPT_DIR}/common.sh"

################################################################################
# Test Orchestration
################################################################################

# Main test runner - v2.0 compliant
autogpt::test::run() {
    local phase="${1:-all}"
    
    # Delegate to test runner script
    if [[ -f "${TEST_DIR}/run-tests.sh" ]]; then
        bash "${TEST_DIR}/run-tests.sh" "${phase}"
        return $?
    else
        log::error "Test runner not found at: ${TEST_DIR}/run-tests.sh"
        return 1
    fi
}

# Smoke test - Quick health validation (<30s)
autogpt::test::smoke() {
    if [[ -f "${TEST_DIR}/phases/test-smoke.sh" ]]; then
        bash "${TEST_DIR}/phases/test-smoke.sh"
        return $?
    else
        # Fallback inline smoke test
        log::info "Running inline smoke test"
        
        if ! autogpt_container_running; then
            log::error "AutoGPT container not running"
            return 1
        fi
        
        if timeout 5 curl -sf "http://localhost:${AUTOGPT_PORT_API}/health" > /dev/null 2>&1; then
            log::success "Health check passed"
            return 0
        else
            log::error "Health check failed"
            return 1
        fi
    fi
}

# Integration test - End-to-end functionality (<120s)
autogpt::test::integration() {
    if [[ -f "${TEST_DIR}/phases/test-integration.sh" ]]; then
        bash "${TEST_DIR}/phases/test-integration.sh"
        return $?
    else
        log::error "Integration test not found"
        return 1
    fi
}

# Unit test - Library functions (<60s)
autogpt::test::unit() {
    if [[ -f "${TEST_DIR}/phases/test-unit.sh" ]]; then
        bash "${TEST_DIR}/phases/test-unit.sh"
        return $?
    else
        log::error "Unit test not found"
        return 1
    fi
}

# All tests
autogpt::test::all() {
    local failed=0
    
    log::header "Running all AutoGPT tests"
    
    # Run tests in order
    autogpt::test::smoke || ((failed++))
    autogpt::test::integration || ((failed++))
    autogpt::test::unit || ((failed++))
    
    if [[ ${failed} -eq 0 ]]; then
        log::success "All tests passed"
        return 0
    else
        log::error "${failed} test suites failed"
        return ${failed}
    fi
}

# Legacy wrapper for backward compatibility
autogpt::test() {
    local test_type="${1:-all}"
    
    format_header "AutoGPT Tests"
    
    # Map legacy test types to v2.0 phases
    case "${test_type}" in
        basic|smoke)
            autogpt::test::smoke
            ;;
        full|advanced|integration)
            autogpt::test::integration
            ;;
        unit)
            autogpt::test::unit
            ;;
        all)
            autogpt::test::all
            ;;
        *)
            log::error "Unknown test type: ${test_type}"
            log::info "Available: smoke, integration, unit, all"
            return 1
            ;;
    esac
}

################################################################################
# Test Utilities
################################################################################

# Create test agent configuration
autogpt::test::create_test_agent() {
    local name="${1:-test-agent}"
    local config_file="/tmp/autogpt-test-${name}.yaml"
    
    cat > "${config_file}" <<EOF
name: ${name}
description: Test agent for validation
goal: Validate AutoGPT functionality
model: gpt-3.5-turbo
max_iterations: 5
memory_backend: local
tools:
  - web_search
  - file_operations
EOF
    
    echo "${config_file}"
}

# Cleanup test artifacts
autogpt::test::cleanup() {
    log::info "Cleaning up test artifacts"
    
    # Remove test agents
    find "${AUTOGPT_AGENTS_DIR}" -name "test-*" -delete 2>/dev/null || true
    
    # Remove temporary files
    rm -f /tmp/autogpt-test-*.yaml
    
    log::success "Test cleanup complete"
}

# Validate configuration
autogpt::test::validate_config() {
    local errors=0
    
    # Check required variables
    local required_vars=(
        "AUTOGPT_CONTAINER_NAME"
        "AUTOGPT_IMAGE"
        "AUTOGPT_PORT_API"
        "AUTOGPT_DATA_DIR"
    )
    
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            log::error "Required variable not set: ${var}"
            ((errors++))
        fi
    done
    
    # Check port is valid
    if [[ ! "${AUTOGPT_PORT_API}" =~ ^[0-9]+$ ]] || 
       [[ ${AUTOGPT_PORT_API} -lt 1024 ]] || 
       [[ ${AUTOGPT_PORT_API} -gt 65535 ]]; then
        log::error "Invalid port: ${AUTOGPT_PORT_API}"
        ((errors++))
    fi
    
    return ${errors}
}

# Performance benchmark
autogpt::test::benchmark() {
    log::header "AutoGPT Performance Benchmark"
    
    if ! autogpt_container_running; then
        log::error "AutoGPT must be running for benchmarks"
        return 1
    fi
    
    # Test health check response time
    log::info "Testing health check response time..."
    local start_time end_time duration
    start_time=$(date +%s%N)
    timeout 5 curl -sf "http://localhost:${AUTOGPT_PORT_API}/health" > /dev/null 2>&1
    end_time=$(date +%s%N)
    duration=$((($end_time - $start_time) / 1000000))
    log::info "Health check response: ${duration}ms"
    
    if [[ ${duration} -lt 1000 ]]; then
        log::success "Response time: ${duration}ms (excellent)"
    elif [[ ${duration} -lt 3000 ]]; then
        log::warn "Response time: ${duration}ms (acceptable)"
    else
        log::error "Response time: ${duration}ms (poor)"
    fi
    
    # Check resource usage
    local memory=$(docker stats --no-stream --format "{{.MemUsage}}" "${AUTOGPT_CONTAINER_NAME}" 2>/dev/null | cut -d'/' -f1 || echo "unknown")
    local cpu=$(docker stats --no-stream --format "{{.CPUPerc}}" "${AUTOGPT_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
    
    log::info "Memory usage: ${memory}"
    log::info "CPU usage: ${cpu}"
    
    log::success "Benchmark complete"
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

################################################################################
# Export Functions
################################################################################

export -f autogpt::test::run
export -f autogpt::test::smoke
export -f autogpt::test::integration
export -f autogpt::test::unit
export -f autogpt::test::all
export -f autogpt::test::create_test_agent
export -f autogpt::test::cleanup
export -f autogpt::test::validate_config
export -f autogpt::test::benchmark

# Legacy exports for backward compatibility
export -f autogpt::test