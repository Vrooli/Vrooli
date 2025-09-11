#!/usr/bin/env bash
set -euo pipefail

# Integration tests for Pandas-AI

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCRIPT_DIR}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

PANDAS_AI_PORT="${PANDAS_AI_PORT:-8095}"
PANDAS_AI_URL="http://localhost:${PANDAS_AI_PORT}"

# Test 1: Multiple query types
test_query_types() {
    log::info "Testing different query types..."
    
    local queries=("describe" "mean" "sum" "count" "max" "min")
    local failed=0
    
    for query in "${queries[@]}"; do
        local response
        response=$(timeout 5 curl -s -X POST "${PANDAS_AI_URL}/analyze" \
            -H "Content-Type: application/json" \
            -d "{\"query\": \"${query}\", \"data\": {\"values\": [1,2,3,4,5]}}" 2>/dev/null)
        
        if [[ $? -eq 0 ]] && echo "${response}" | grep -q "success.*true"; then
            log::success "Query '${query}' works"
        else
            log::error "Query '${query}' failed"
            failed=1
        fi
    done
    
    return ${failed}
}

# Test 2: Different data structures
test_data_structures() {
    log::info "Testing different data structures..."
    
    # Test with multiple columns
    local multi_col='{"col1": [1,2,3], "col2": [4,5,6], "col3": [7,8,9]}'
    local response
    response=$(timeout 5 curl -s -X POST "${PANDAS_AI_URL}/analyze" \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"describe\", \"data\": ${multi_col}}" 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "${response}" | grep -q "success.*true"; then
        log::success "Multi-column data works"
        return 0
    else
        log::error "Multi-column data failed"
        return 1
    fi
}

# Test 3: Error handling
test_error_handling() {
    log::info "Testing error handling..."
    
    # Test with invalid data
    local response
    response=$(timeout 5 curl -s -X POST "${PANDAS_AI_URL}/analyze" \
        -H "Content-Type: application/json" \
        -d '{"query": "describe"}' 2>/dev/null)
    
    if echo "${response}" | grep -q "error"; then
        log::success "Error handling works"
        return 0
    else
        log::error "Error handling failed"
        return 1
    fi
}

# Test 4: Performance (response within timeout)
test_performance() {
    log::info "Testing performance..."
    
    local start_time=$(date +%s)
    timeout 5 curl -s -X POST "${PANDAS_AI_URL}/analyze" \
        -H "Content-Type: application/json" \
        -d '{"query": "describe", "data": {"test": [1,2,3,4,5]}}' > /dev/null 2>&1
    local end_time=$(date +%s)
    
    local duration=$((end_time - start_time))
    
    if [[ ${duration} -le 5 ]]; then
        log::success "Response time acceptable (${duration}s)"
        return 0
    else
        log::error "Response too slow (${duration}s)"
        return 1
    fi
}

# Main integration test execution
main() {
    log::header "Pandas-AI Integration Tests"
    
    local failed=0
    
    # Run all integration tests
    test_query_types || failed=1
    test_data_structures || failed=1
    test_error_handling || failed=1
    test_performance || failed=1
    
    if [[ ${failed} -eq 0 ]]; then
        log::success "All integration tests passed!"
        exit 0
    else
        log::error "Some integration tests failed"
        exit 1
    fi
}

main "$@"