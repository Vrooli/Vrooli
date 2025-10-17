#!/bin/bash

# Open Data Cube Smoke Tests
# Quick validation of basic functionality (<30s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
source "${RESOURCE_DIR}/lib/core.sh"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Test function wrapper
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    echo -n "  ${test_name}... "
    
    if eval "$test_cmd" &>/dev/null; then
        echo "✓"
        ((TESTS_PASSED++))
        return 0
    else
        echo "✗"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Main smoke test execution
main() {
    echo "Running Open Data Cube smoke tests..."
    echo ""
    
    local start_time=$(date +%s)
    
    # Test 1: Check if Docker is available
    run_test "Docker availability" "command -v docker"
    
    # Test 2: Check if containers are running
    run_test "API container running" "docker ps --format '{{.Names}}' | grep -q '${ODC_API_CONTAINER}'"
    
    # Test 3: Check database container
    run_test "Database container running" "docker ps --format '{{.Names}}' | grep -q '${ODC_DB_CONTAINER}'"
    
    # Test 4: Check OWS container
    run_test "OWS container running" "docker ps --format '{{.Names}}' | grep -q '${ODC_OWS_CONTAINER}'"
    
    # Test 5: Check Redis container
    run_test "Redis container running" "docker ps --format '{{.Names}}' | grep -q '${ODC_REDIS_CONTAINER}'"
    
    # Test 6: Database connectivity
    run_test "Database connectivity" "docker exec ${ODC_DB_CONTAINER} pg_isready -U datacube"
    
    # Test 7: Redis connectivity
    run_test "Redis connectivity" "docker exec ${ODC_REDIS_CONTAINER} redis-cli ping"
    
    # Test 8: API health check
    run_test "API health check" "timeout 5 curl -sf 'http://localhost:${ODC_PORT}/health'"
    
    # Test 9: Datacube CLI availability
    run_test "Datacube CLI available" "docker exec ${ODC_API_CONTAINER} datacube --version"
    
    # Test 10: Product listing works
    run_test "Product listing" "docker exec ${ODC_API_CONTAINER} datacube product list"
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "----------------------------------------"
    echo "Smoke Test Results:"
    echo "  Passed: ${TESTS_PASSED}"
    echo "  Failed: ${TESTS_FAILED}"
    echo "  Duration: ${duration}s"
    
    if [[ ${TESTS_FAILED} -eq 0 ]]; then
        echo "  Status: SUCCESS"
        echo "----------------------------------------"
        return 0
    else
        echo "  Status: FAILURE"
        echo "----------------------------------------"
        return 1
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi