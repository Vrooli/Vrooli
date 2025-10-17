#!/usr/bin/env bash
# mcrcon integration tests - end-to-end functionality (<120s)

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

# Main integration test
main() {
    echo "Running mcrcon integration tests..."
    echo "=============================="
    
    # Test 1: CLI help displays all required commands
    run_test "CLI help completeness" "'${RESOURCE_DIR}/cli.sh' help | grep -q 'manage'"
    
    # Test 2: Info command returns valid JSON
    run_test "info JSON output" "'${RESOURCE_DIR}/cli.sh' info --json | jq -e '.startup_order'"
    
    # Test 3: Status command works
    run_test "status command" "'${RESOURCE_DIR}/cli.sh' status"
    
    # Test 4: Credentials command works
    run_test "credentials command" "'${RESOURCE_DIR}/cli.sh' credentials"
    
    # Test 5: Content list works (even with no servers)
    run_test "content list" "'${RESOURCE_DIR}/cli.sh' content list"
    
    # Test 6: Test that install command exists
    run_test "install command exists" "'${RESOURCE_DIR}/cli.sh' help | grep -q 'install'"
    
    # Test 7: Runtime configuration is valid
    run_test "runtime.json validation" "jq -e '.startup_order and .dependencies and .priority' '${RESOURCE_DIR}/config/runtime.json'"
    
    # Test 8: Schema configuration is valid JSON
    run_test "schema.json validation" "jq -e '.properties.server' '${RESOURCE_DIR}/config/schema.json'"
    
    # Test 9: Server discovery command exists
    run_test "discover command" "'${RESOURCE_DIR}/cli.sh' help | grep -q 'discover'"
    
    # Test 10: Player management commands exist
    run_test "player commands" "'${RESOURCE_DIR}/cli.sh' help | grep -q 'player'"
    
    # Test 11: Execute-all command exists
    run_test "execute-all command" "'${RESOURCE_DIR}/cli.sh' help | grep -q 'execute-all'"
    
    # Test 12: Content discovery works
    run_test "content discover" "'${RESOURCE_DIR}/cli.sh' content discover"
    
    # Summary
    echo "=============================="
    echo "Integration Tests: ${TESTS_PASSED}/${TESTS_RUN} passed"
    
    if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
        echo "All integration tests passed!"
        exit 0
    else
        echo "Some integration tests failed"
        exit 1
    fi
}

main "$@"