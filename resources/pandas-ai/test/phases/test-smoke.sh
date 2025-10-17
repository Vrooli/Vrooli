#!/usr/bin/env bash
set -euo pipefail

# Smoke tests for Pandas-AI - Quick health validation (<30s)

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCRIPT_DIR}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

PANDAS_AI_PORT="${PANDAS_AI_PORT:-8095}"
PANDAS_AI_URL="http://localhost:${PANDAS_AI_PORT}"

# Test 1: Service is running
test_service_running() {
    log::info "Checking if Pandas-AI service is running..."
    
    if pgrep -f "pandas-ai.*server.py" > /dev/null; then
        log::success "Service process is running"
        return 0
    else
        log::error "Service process not found"
        return 1
    fi
}

# Test 2: Health endpoint responds
test_health_endpoint() {
    log::info "Testing health endpoint..."
    
    if timeout 5 curl -sf "${PANDAS_AI_URL}/health" > /dev/null; then
        log::success "Health endpoint responding"
        return 0
    else
        log::error "Health endpoint not responding"
        return 1
    fi
}

# Test 3: Basic analysis works
test_basic_analysis() {
    log::info "Testing basic analysis functionality..."
    
    local response
    response=$(timeout 5 curl -s -X POST "${PANDAS_AI_URL}/analyze" \
        -H "Content-Type: application/json" \
        -d '{"query": "describe", "data": {"test": [1,2,3]}}' 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "${response}" | grep -q "success.*true"; then
        log::success "Basic analysis working"
        return 0
    else
        log::error "Basic analysis failed"
        return 1
    fi
}

# Test 4: Port is accessible
test_port_accessible() {
    log::info "Checking port ${PANDAS_AI_PORT} accessibility..."
    
    if nc -z localhost "${PANDAS_AI_PORT}" 2>/dev/null; then
        log::success "Port ${PANDAS_AI_PORT} is accessible"
        return 0
    else
        log::error "Port ${PANDAS_AI_PORT} is not accessible"
        return 1
    fi
}

# Main smoke test execution
main() {
    log::header "Pandas-AI Smoke Tests"
    
    local failed=0
    
    # Run all smoke tests
    test_service_running || failed=1
    test_port_accessible || failed=1
    test_health_endpoint || failed=1
    test_basic_analysis || failed=1
    
    if [[ ${failed} -eq 0 ]]; then
        log::success "All smoke tests passed!"
        exit 0
    else
        log::error "Some smoke tests failed"
        exit 1
    fi
}

main "$@"