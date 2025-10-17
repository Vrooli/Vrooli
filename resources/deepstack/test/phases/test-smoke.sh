#!/usr/bin/env bash
# DeepStack Resource - Smoke Tests
# Quick health validation (must complete in <30s)

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly TEST_DIR="$(dirname "${SCRIPT_DIR}")"
readonly RESOURCE_DIR="$(dirname "${TEST_DIR}")"

# Load test library
source "${RESOURCE_DIR}/lib/test.sh"

# Timeout for smoke tests
SMOKE_TIMEOUT=30

# Run smoke tests with timeout
run_smoke_tests() {
    local start_time=$(date +%s)
    local failures=0
    
    echo "DeepStack Smoke Tests"
    echo "===================="
    
    # Test 1: Docker availability
    echo ""
    echo "Test 1: Docker availability"
    test_docker_running || ((failures++))
    
    # Test 2: Service status check
    echo ""
    echo "Test 2: Service status"
    if docker ps | grep -q "${DEEPSTACK_CONTAINER_NAME}"; then
        echo "✓ DeepStack container is running"
        
        # Test 3: Health endpoint
        echo ""
        echo "Test 3: Health endpoint"
        test_health_check || ((failures++))
        
        # Test 4: Port accessibility
        echo ""
        echo "Test 4: Port accessibility"
        if nc -z localhost "${DEEPSTACK_PORT}" 2>/dev/null; then
            echo "✓ Port ${DEEPSTACK_PORT} is accessible"
        else
            echo "✗ Port ${DEEPSTACK_PORT} is not accessible" >&2
            ((failures++))
        fi
    else
        echo "ℹ DeepStack is not running (install and start first)"
        echo "  Run: $(basename "${RESOURCE_DIR}/cli.sh") manage install"
        echo "  Run: $(basename "${RESOURCE_DIR}/cli.sh") manage start --wait"
        return 2
    fi
    
    # Test 5: Response time check
    echo ""
    echo "Test 5: Response time"
    local response_start=$(date +%s%3N)
    if timeout 5 curl -sf "http://${DEEPSTACK_HOST}:${DEEPSTACK_PORT}/v1/vision/detection" &> /dev/null; then
        local response_end=$(date +%s%3N)
        local response_time=$((response_end - response_start))
        if [[ $response_time -lt 5000 ]]; then
            echo "✓ API response time: ${response_time}ms"
        else
            echo "⚠ Slow API response: ${response_time}ms" >&2
        fi
    else
        echo "✗ API not responding within 5s" >&2
        ((failures++))
    fi
    
    # Check total execution time
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Test duration: ${duration}s"
    
    if [[ $duration -gt $SMOKE_TIMEOUT ]]; then
        echo "⚠ Warning: Smoke tests exceeded ${SMOKE_TIMEOUT}s timeout" >&2
    fi
    
    if [[ $failures -gt 0 ]]; then
        echo ""
        echo "✗ Smoke tests failed with $failures error(s)" >&2
        return 1
    else
        echo ""
        echo "✓ All smoke tests passed"
        return 0
    fi
}

# Run with timeout enforcement
if timeout "$SMOKE_TIMEOUT" bash -c "$(declare -f run_smoke_tests); $(declare -f test_docker_running); $(declare -f test_health_check); source '${RESOURCE_DIR}/config/defaults.sh'; run_smoke_tests"; then
    exit 0
else
    exit_code=$?
    if [[ $exit_code -eq 124 ]]; then
        echo "✗ Smoke tests timed out after ${SMOKE_TIMEOUT}s" >&2
    fi
    exit $exit_code
fi