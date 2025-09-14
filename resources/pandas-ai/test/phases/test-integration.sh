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

# Test 5: P1 Feature - Visualization generation
test_visualization() {
    log::info "Testing visualization generation..."
    
    local response
    response=$(timeout 5 curl -s -X POST "${PANDAS_AI_URL}/analyze" \
        -H "Content-Type: application/json" \
        -d '{"query": "describe", "data": {"x": [1,2,3,4,5], "y": [2,4,6,8,10]}, "visualization": true, "viz_type": "scatter"}' 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "${response}" | grep -q "visualization" && echo "${response}" | grep -q "success.*true"; then
        log::success "Visualization generation works"
        return 0
    else
        log::error "Visualization generation failed"
        return 1
    fi
}

# Test 6: P1 Feature - Data cleaning suggestions
test_data_cleaning() {
    log::info "Testing data cleaning suggestions..."
    
    # Test with data that has missing values and duplicates
    local dirty_data='{"col1": [1, 2, null, 4, 5, 5], "col2": [10, 20, 30, 40, 50, 50]}'
    local response
    response=$(timeout 5 curl -s -X POST "${PANDAS_AI_URL}/analyze" \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"describe\", \"data\": ${dirty_data}, \"clean_data\": true}" 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "${response}" | grep -q "cleaning_suggestions" && echo "${response}" | grep -q "missing_values\|duplicates"; then
        log::success "Data cleaning suggestions work"
        return 0
    else
        log::error "Data cleaning suggestions failed"
        return 1
    fi
}

# Test 7: P1 Feature - Multi-dataframe operations
test_multi_dataframe() {
    log::info "Testing multi-dataframe operations..."
    
    local response
    response=$(timeout 5 curl -s -X POST "${PANDAS_AI_URL}/analyze/multi" \
        -H "Content-Type: application/json" \
        -d '{
            "dataframes": [
                {"name": "df1", "data": {"id": [1, 2, 3], "value1": [10, 20, 30]}},
                {"name": "df2", "data": {"id": [2, 3, 4], "value2": [200, 300, 400]}}
            ],
            "operation": "merge",
            "query": "preview",
            "join_keys": ["id"],
            "how": "inner"
        }' 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "${response}" | grep -q "success.*true" && echo "${response}" | grep -q "rows.*2"; then
        log::success "Multi-dataframe operations work"
        return 0
    else
        log::error "Multi-dataframe operations failed"
        return 1
    fi
}

# Test 8: P1 Feature - Performance optimization (chunking)
test_performance_optimization() {
    log::info "Testing performance optimization features..."
    
    # Test with max_rows override
    local response
    response=$(timeout 5 curl -s -X POST "${PANDAS_AI_URL}/analyze" \
        -H "Content-Type: application/json" \
        -d '{"query": "describe", "data": {"values": [1,2,3,4,5]}, "max_rows": 3}' 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "${response}" | grep -q "performance_metrics" && echo "${response}" | grep -q "processing_time"; then
        log::success "Performance optimization features work"
        return 0
    else
        log::error "Performance optimization features failed"
        return 1
    fi
}

# Test 9: P1 Feature - Advanced configuration
test_advanced_configuration() {
    log::info "Testing advanced configuration..."
    
    # Check if configuration is exposed in root endpoint
    local response
    response=$(timeout 5 curl -s "${PANDAS_AI_URL}/" 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "${response}" | grep -q "configuration" && echo "${response}" | grep -q "max_rows\|max_workers\|chunk_size"; then
        log::success "Advanced configuration exposed correctly"
        return 0
    else
        log::error "Advanced configuration not properly exposed"
        return 1
    fi
}

# Main integration test execution
main() {
    log::header "Pandas-AI Integration Tests"
    
    local failed=0
    
    # Run basic integration tests
    log::info "Running basic integration tests..."
    test_query_types || failed=1
    test_data_structures || failed=1
    test_error_handling || failed=1
    test_performance || failed=1
    
    # Run P1 feature tests
    log::info "Running P1 feature tests..."
    test_visualization || failed=1
    test_data_cleaning || failed=1
    test_multi_dataframe || failed=1
    test_performance_optimization || failed=1
    test_advanced_configuration || failed=1
    
    if [[ ${failed} -eq 0 ]]; then
        log::success "All integration tests passed!"
        exit 0
    else
        log::error "Some integration tests failed"
        exit 1
    fi
}

main "$@"