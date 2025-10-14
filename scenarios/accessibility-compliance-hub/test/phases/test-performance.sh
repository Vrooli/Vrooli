#!/bin/bash
# Performance tests for accessibility-compliance-hub scenario

set -euo pipefail

# Colors for output
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# Configuration
readonly API_BASE="http://localhost:${API_PORT:-18000}"
readonly PERFORMANCE_THRESHOLD_MS=2000  # API response time target from PRD

# Test counter
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
test_start() {
    local test_name="$1"
    echo -e "\n${YELLOW}Testing: ${test_name}...${NC}"
    ((TESTS_TOTAL++))
}

test_pass() {
    local details="${1:-}"
    echo -e "${GREEN}✓ PASSED${NC} ${details}"
    ((TESTS_PASSED++))
}

test_fail() {
    local reason="${1:-Unknown reason}"
    echo -e "${RED}✗ FAILED: ${reason}${NC}"
    ((TESTS_FAILED++))
}

# Test API response time
test_api_response_time() {
    test_start "API response time under ${PERFORMANCE_THRESHOLD_MS}ms"

    local start_time
    local end_time
    start_time=$(date +%s%3N)
    curl -sf "${API_BASE}/health" > /dev/null 2>&1
    end_time=$(date +%s%3N)

    local response_time=$((end_time - start_time))

    if [ $response_time -lt $PERFORMANCE_THRESHOLD_MS ]; then
        test_pass "(${response_time}ms)"
    else
        test_fail "Response time ${response_time}ms exceeds ${PERFORMANCE_THRESHOLD_MS}ms threshold"
    fi
}

# Test concurrent requests
test_concurrent_load() {
    test_start "Handles 10 concurrent requests"

    local pids=()
    local start_time
    local end_time
    start_time=$(date +%s%3N)

    for _ in {1..10}; do
        curl -sf "${API_BASE}/health" > /dev/null 2>&1 &
        pids+=($!)
    done

    # Wait for all requests to complete
    for pid in "${pids[@]}"; do
        wait "$pid"
    done

    end_time=$(date +%s%3N)
    local total_time=$((end_time - start_time))
    local avg_time=$((total_time / 10))

    if [ $avg_time -lt $PERFORMANCE_THRESHOLD_MS ]; then
        test_pass "(avg ${avg_time}ms per request)"
    else
        test_fail "Average time ${avg_time}ms exceeds threshold"
    fi
}

# Test memory usage (basic check)
test_memory_stability() {
    test_start "API process memory stability"

    # Check if API process exists and get memory usage
    local pid
    local mem_kb
    pid=$(pgrep -f "accessibility-compliance-hub-api" | head -1)

    if [ -z "$pid" ]; then
        test_fail "API process not found"
        return
    fi

    mem_kb=$(ps -p "$pid" -o rss= 2>/dev/null || echo "0")
    local mem_mb=$((mem_kb / 1024))

    # Reasonable threshold: under 500MB for a Go API prototype
    if [ $mem_mb -lt 500 ]; then
        test_pass "(${mem_mb}MB)"
    else
        test_fail "Memory usage ${mem_mb}MB seems high"
    fi
}

# Main test execution
main() {
    echo -e "${YELLOW}=== Accessibility Compliance Hub Performance Tests ===${NC}"
    echo -e "${YELLOW}Performance Target: < ${PERFORMANCE_THRESHOLD_MS}ms API response time (from PRD)${NC}\n"

    # Check if API is running
    if ! curl -sf "${API_BASE}/health" > /dev/null 2>&1; then
        echo -e "${RED}Cannot run tests - API is not available${NC}"
        exit 1
    fi

    # Run tests
    test_api_response_time
    test_concurrent_load
    test_memory_stability

    # Print summary
    echo -e "\n${YELLOW}=== Test Summary ===${NC}"
    echo -e "Total tests: ${TESTS_TOTAL}"
    echo -e "${GREEN}Passed: ${TESTS_PASSED}${NC}"
    echo -e "${RED}Failed: ${TESTS_FAILED}${NC}"

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All performance tests passed!${NC}"
        exit 0
    else
        echo -e "\n${RED}Some performance tests failed!${NC}"
        exit 1
    fi
}

main "$@"
