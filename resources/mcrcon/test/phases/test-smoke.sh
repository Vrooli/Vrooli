#!/usr/bin/env bash
# mcrcon smoke tests - quick health validation (<30s)

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly TEST_DIR="$(dirname "$SCRIPT_DIR")"
readonly RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

# Test function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "  Testing $test_name... "
    
    if eval "$test_command" > /dev/null 2>&1; then
        echo "✓"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo "✗"
        return 1
    fi
}

# Main smoke test
main() {
    echo "Running mcrcon smoke tests..."
    echo "=============================="
    
    # Test 1: CLI exists and is executable
    run_test "CLI exists" "test -f '${RESOURCE_DIR}/cli.sh' && test -x '${RESOURCE_DIR}/cli.sh'"
    
    # Test 2: Help command works
    run_test "help command" "'${RESOURCE_DIR}/cli.sh' help"
    
    # Test 3: Info command works
    run_test "info command" "'${RESOURCE_DIR}/cli.sh' info"
    
    # Test 4: Runtime configuration exists
    run_test "runtime.json exists" "test -f '${RESOURCE_DIR}/config/runtime.json'"
    
    # Test 5: Health check endpoint (if service is running)
    if timeout 1 curl -sf "http://localhost:${MCRCON_HEALTH_PORT}/health" > /dev/null 2>&1; then
        run_test "health endpoint" "timeout 5 curl -sf 'http://localhost:${MCRCON_HEALTH_PORT}/health'"
    else
        echo "  Skipping health endpoint test (service not running)"
    fi
    
    # Test 6: Core library exists
    run_test "core library exists" "test -f '${RESOURCE_DIR}/lib/core.sh'"
    
    # Test 7: Test library exists
    run_test "test library exists" "test -f '${RESOURCE_DIR}/lib/test.sh'"
    
    # Test 8: Configuration loads successfully
    run_test "configuration loads" "source '${RESOURCE_DIR}/config/defaults.sh'"
    
    # Summary
    echo "=============================="
    echo "Smoke Tests: ${TESTS_PASSED}/${TESTS_RUN} passed"
    
    if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
        echo "All smoke tests passed!"
        exit 0
    else
        echo "Some smoke tests failed"
        exit 1
    fi
}

main "$@"