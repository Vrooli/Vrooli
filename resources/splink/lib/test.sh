#!/usr/bin/env bash
# Splink Test Library - Validation and testing functions

set -euo pipefail

# Source core functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/core.sh"

# Run tests based on type
run_tests() {
    local test_type="${1:-all}"
    
    case "$test_type" in
        smoke)
            run_smoke_tests
            ;;
        integration)
            run_integration_tests
            ;;
        unit)
            run_unit_tests
            ;;
        all)
            run_all_tests
            ;;
        *)
            echo "Error: Unknown test type: $test_type"
            echo "Available types: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Run smoke tests (quick health check)
run_smoke_tests() {
    echo "Running Splink smoke tests..."
    echo "=============================="
    
    local passed=0
    local failed=0
    
    # Test 1: Service health check
    echo -n "Testing health endpoint... "
    if timeout 5 curl -sf "http://localhost:${SPLINK_PORT}/health" &> /dev/null; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 2: API responsiveness
    echo -n "Testing API responsiveness... "
    local start_time=$(date +%s%N)
    if timeout 2 curl -sf "http://localhost:${SPLINK_PORT}/health" &> /dev/null; then
        local end_time=$(date +%s%N)
        local response_time=$(( (end_time - start_time) / 1000000 ))
        if [ $response_time -lt 500 ]; then
            echo "✓ PASSED (${response_time}ms)"
            ((passed++))
        else
            echo "✗ FAILED (${response_time}ms > 500ms)"
            ((failed++))
        fi
    else
        echo "✗ FAILED (timeout)"
        ((failed++))
    fi
    
    # Test 3: Container status
    echo -n "Testing container status... "
    if docker ps -f name="$SPLINK_CONTAINER" --format "{{.Status}}" | grep -q Up; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    echo ""
    echo "Smoke Test Results: $passed passed, $failed failed"
    
    if [ $failed -gt 0 ]; then
        return 1
    fi
    return 0
}

# Run integration tests
run_integration_tests() {
    echo "Running Splink integration tests..."
    echo "===================================="
    
    local passed=0
    local failed=0
    
    # Test 1: Deduplication endpoint
    echo -n "Testing deduplication endpoint... "
    local response=$(curl -X POST "http://localhost:${SPLINK_PORT}/linkage/deduplicate" \
        -H "Content-Type: application/json" \
        -d '{"dataset_id": "test_dataset", "settings": {}}' \
        -s -o /dev/null -w "%{http_code}" 2>/dev/null || echo "000")
    
    if [[ "$response" == "200" ]] || [[ "$response" == "202" ]] || [[ "$response" == "400" ]]; then
        echo "✓ PASSED (HTTP $response)"
        ((passed++))
    else
        echo "✗ FAILED (HTTP $response)"
        ((failed++))
    fi
    
    # Test 2: Link endpoint
    echo -n "Testing link endpoint... "
    response=$(curl -X POST "http://localhost:${SPLINK_PORT}/linkage/link" \
        -H "Content-Type: application/json" \
        -d '{"dataset1_id": "test1", "dataset2_id": "test2", "settings": {}}' \
        -s -o /dev/null -w "%{http_code}" 2>/dev/null || echo "000")
    
    if [[ "$response" == "200" ]] || [[ "$response" == "202" ]] || [[ "$response" == "400" ]]; then
        echo "✓ PASSED (HTTP $response)"
        ((passed++))
    else
        echo "✗ FAILED (HTTP $response)"
        ((failed++))
    fi
    
    # Test 3: Jobs listing
    echo -n "Testing jobs endpoint... "
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        "http://localhost:${SPLINK_PORT}/linkage/jobs" 2>/dev/null || echo "000")
    
    if [[ "$response" == "200" ]]; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED (HTTP $response)"
        ((failed++))
    fi
    
    # Test 4: Database connectivity (if configured)
    echo -n "Testing database connectivity... "
    if [[ -n "${POSTGRES_HOST:-}" ]]; then
        # This would require actual database testing
        echo "⊗ SKIPPED (not implemented)"
    else
        echo "⊗ SKIPPED (no database configured)"
    fi
    
    echo ""
    echo "Integration Test Results: $passed passed, $failed failed"
    
    if [ $failed -gt 0 ]; then
        return 1
    fi
    return 0
}

# Run unit tests
run_unit_tests() {
    echo "Running Splink unit tests..."
    echo "============================"
    
    local passed=0
    local failed=0
    
    # Test 1: Configuration validation
    echo -n "Testing configuration files... "
    if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]] && \
       [[ -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 2: Port allocation
    echo -n "Testing port configuration... "
    if [[ "$SPLINK_PORT" == "8096" ]]; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED (expected 8096, got $SPLINK_PORT)"
        ((failed++))
    fi
    
    # Test 3: Directory structure
    echo -n "Testing directory structure... "
    if [[ -d "${SCRIPT_DIR}/lib" ]] && \
       [[ -d "${SCRIPT_DIR}/config" ]] && \
       [[ -d "${SCRIPT_DIR}/test" ]]; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    # Test 4: CLI commands
    echo -n "Testing CLI command structure... "
    if "${SCRIPT_DIR}/cli.sh" help &> /dev/null; then
        echo "✓ PASSED"
        ((passed++))
    else
        echo "✗ FAILED"
        ((failed++))
    fi
    
    echo ""
    echo "Unit Test Results: $passed passed, $failed failed"
    
    if [ $failed -gt 0 ]; then
        return 1
    fi
    return 0
}

# Run all tests
run_all_tests() {
    echo "Running all Splink tests..."
    echo "==========================="
    echo ""
    
    local total_failed=0
    
    # Run smoke tests
    if ! run_smoke_tests; then
        ((total_failed++))
    fi
    echo ""
    
    # Run integration tests
    if ! run_integration_tests; then
        ((total_failed++))
    fi
    echo ""
    
    # Run unit tests
    if ! run_unit_tests; then
        ((total_failed++))
    fi
    echo ""
    
    echo "============================="
    if [ $total_failed -eq 0 ]; then
        echo "All tests PASSED ✓"
        return 0
    else
        echo "$total_failed test suite(s) FAILED ✗"
        return 1
    fi
}