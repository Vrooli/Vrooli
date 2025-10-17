#!/usr/bin/env bash
################################################################################
# Haystack Test Library - v2.0 Universal Contract Compliant
# 
# Test implementation for Haystack resource
################################################################################

set -euo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HAYSTACK_DIR="${APP_ROOT}/resources/haystack"
HAYSTACK_TEST_DIR="${HAYSTACK_DIR}/test"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# ============================================================================
# Test Functions
# ============================================================================

# Run smoke tests
haystack::test_smoke() {
    log::info "Running Haystack smoke test..."
    
    if [[ -f "${HAYSTACK_TEST_DIR}/phases/test-smoke.sh" ]]; then
        bash "${HAYSTACK_TEST_DIR}/phases/test-smoke.sh"
    else
        # Fallback simple smoke test
        local port
        port=$(haystack::get_port 2>/dev/null || echo "8075")
        
        if timeout 5 curl -sf "http://localhost:${port}/health" &>/dev/null; then
            log::success "Haystack is running and healthy"
            return 0
        else
            log::error "Haystack health check failed"
            return 1
        fi
    fi
}

# Run integration tests
haystack::test_integration() {
    log::info "Running Haystack integration tests..."
    
    if [[ -f "${HAYSTACK_TEST_DIR}/phases/test-integration.sh" ]]; then
        bash "${HAYSTACK_TEST_DIR}/phases/test-integration.sh"
    else
        log::warning "Integration test file not found"
        return 1
    fi
}

# Run unit tests
haystack::test_unit() {
    log::info "Running Haystack unit tests..."
    
    if [[ -f "${HAYSTACK_TEST_DIR}/phases/test-unit.sh" ]]; then
        bash "${HAYSTACK_TEST_DIR}/phases/test-unit.sh"
    else
        log::warning "Unit tests not yet implemented"
        return 0
    fi
}

# Run all tests
haystack::test_all() {
    log::info "Running all Haystack tests..."
    
    if [[ -f "${HAYSTACK_TEST_DIR}/run-tests.sh" ]]; then
        bash "${HAYSTACK_TEST_DIR}/run-tests.sh" all
    else
        # Run individual test phases
        local failed=0
        
        haystack::test_smoke || ((failed++))
        haystack::test_integration || ((failed++))
        haystack::test_unit || ((failed++))
        
        if [[ ${failed} -eq 0 ]]; then
            log::success "All tests passed"
            return 0
        else
            log::error "${failed} test phases failed"
            return 1
        fi
    fi
}

# Main test dispatcher
haystack::test() {
    local phase="${1:-all}"
    
    case "${phase}" in
        smoke)
            haystack::test_smoke
            ;;
        integration)
            haystack::test_integration
            ;;
        unit)
            haystack::test_unit
            ;;
        all)
            haystack::test_all
            ;;
        *)
            log::error "Unknown test phase: ${phase}"
            echo "Usage: test [smoke|integration|unit|all]"
            return 1
            ;;
    esac
}

# Validate Haystack installation
haystack::validate_installation() {
    log::info "Validating Haystack installation..."
    
    local issues=()
    
    # Define paths if not already set
    local HAYSTACK_VENV_DIR="${HAYSTACK_VENV_DIR:-${HAYSTACK_DIR}/venv}"
    local HAYSTACK_SCRIPTS_DIR="${HAYSTACK_SCRIPTS_DIR:-${HAYSTACK_DIR}/scripts}"
    
    # Check Python environment
    if [[ ! -d "${HAYSTACK_VENV_DIR}" ]]; then
        issues+=("Virtual environment not found")
    fi
    
    if [[ ! -f "${HAYSTACK_VENV_DIR}/bin/python" ]]; then
        issues+=("Python binary not found in venv")
    fi
    
    # Check server script
    if [[ ! -f "${HAYSTACK_SCRIPTS_DIR}/server.py" ]]; then
        issues+=("Server script not found")
    fi
    
    # Check required Python packages
    if [[ -f "${HAYSTACK_VENV_DIR}/bin/python" ]]; then
        local packages=("fastapi" "uvicorn" "pydantic")
        for pkg in "${packages[@]}"; do
            if ! "${HAYSTACK_VENV_DIR}/bin/python" -c "import ${pkg}" 2>/dev/null; then
                issues+=("Python package '${pkg}' not installed")
            fi
        done
    fi
    
    # Report results
    if [[ ${#issues[@]} -eq 0 ]]; then
        log::success "Installation validation passed"
        return 0
    else
        log::error "Installation validation failed:"
        for issue in "${issues[@]}"; do
            log::error "  - ${issue}"
        done
        return 1
    fi
}

# Performance test
haystack::test_performance() {
    log::info "Running performance tests..."
    
    local port
    port=$(haystack::get_port 2>/dev/null || echo "8075")
    
    if ! haystack::is_running; then
        log::error "Haystack not running, cannot test performance"
        return 1
    fi
    
    # Test health check response time
    local start_time end_time duration
    start_time=$(date +%s%N)
    timeout 5 curl -sf "http://localhost:${port}/health" &>/dev/null
    end_time=$(date +%s%N)
    duration=$((($end_time - $start_time) / 1000000))  # Convert to milliseconds
    
    if [[ ${duration} -lt 500 ]]; then
        log::success "Health check response time: ${duration}ms (< 500ms target)"
    else
        log::warning "Health check response time: ${duration}ms (exceeds 500ms target)"
    fi
    
    # Test document indexing
    local test_doc='{"documents":[{"content":"Test document for performance testing","metadata":{"test":true}}]}'
    
    start_time=$(date +%s%N)
    curl -sf -X POST "http://localhost:${port}/index" \
        -H "Content-Type: application/json" \
        -d "${test_doc}" &>/dev/null
    end_time=$(date +%s%N)
    duration=$((($end_time - $start_time) / 1000000))
    
    log::info "Document indexing time: ${duration}ms"
    
    # Test query performance
    start_time=$(date +%s%N)
    curl -sf -X POST "http://localhost:${port}/query" \
        -H "Content-Type: application/json" \
        -d '{"query":"test","top_k":10}' &>/dev/null
    end_time=$(date +%s%N)
    duration=$((($end_time - $start_time) / 1000000))
    
    if [[ ${duration} -lt 2000 ]]; then
        log::success "Query response time: ${duration}ms (< 2000ms target)"
    else
        log::warning "Query response time: ${duration}ms (exceeds 2000ms target)"
    fi
    
    return 0
}