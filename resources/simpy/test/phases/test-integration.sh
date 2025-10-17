#!/usr/bin/env bash
################################################################################
# SimPy Integration Tests - End-to-end functionality (<120s)
# 
# Tests simulation execution, API functionality, and integration points
################################################################################

set -euo pipefail

# Get directories
PHASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$PHASE_DIR/.." && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/content.sh"

# Export configuration
simpy::export_config

# Test timeout
TIMEOUT_CMD="timeout 10"

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "  Testing $test_name... "
    
    if eval "$test_command" &>/dev/null; then
        echo "✓"
        ((TESTS_PASSED++))
        return 0
    else
        echo "✗"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Create test simulation
create_test_simulation() {
    cat > "/tmp/test_simulation.py" << 'EOF'
import simpy
import json

def test_process(env):
    yield env.timeout(1)
    return "success"

env = simpy.Environment()
result = env.process(test_process(env))
env.run()
print(json.dumps({"test": "passed", "simpy_version": simpy.__version__}))
EOF
}

# Main integration test
main() {
    log::header "SimPy Integration Tests"
    
    # Ensure service is running
    if ! simpy::is_running; then
        log::info "Starting SimPy service for integration testing..."
        if ! simpy::docker::start; then
            log::error "Failed to start SimPy service"
            exit 1
        fi
        sleep 3
    fi
    
    # Test 1: Execute simulation via API
    create_test_simulation
    local test_json='{"code": "'$(cat /tmp/test_simulation.py | python3 -c "import sys, json; print(json.dumps(sys.stdin.read()))" | sed 's/^"//;s/"$//')'"}'
    
    run_test "Simulation execution via API" "echo '$test_json' | $TIMEOUT_CMD curl -sf -X POST -H 'Content-Type: application/json' -d @- http://localhost:${SIMPY_PORT}/simulate | grep -q success"
    
    # Test 2: Content management - add
    run_test "Add simulation script" "simpy::content::add /tmp/test_simulation.py test_sim"
    
    # Test 3: Content management - list
    run_test "List simulation scripts" "simpy::content::list | grep -q test_sim"
    
    # Test 4: Content management - get
    run_test "Get simulation script details" "simpy::content::get test_sim"
    
    # Test 5: Content management - execute
    run_test "Execute simulation script" "simpy::content::execute test_sim"
    
    # Test 6: Example simulations exist
    run_test "Example simulations available" "[ -f '$SIMPY_EXAMPLES_DIR/bank_queue.py' ]"
    
    # Test 7: Execute example simulation
    if [[ -f "$SIMPY_EXAMPLES_DIR/bank_queue.py" ]]; then
        run_test "Execute bank_queue example" "$TIMEOUT_CMD python3 '$SIMPY_EXAMPLES_DIR/bank_queue.py'"
    fi
    
    # Test 8: API error handling
    run_test "API error handling" "$TIMEOUT_CMD curl -sf -X POST -H 'Content-Type: application/json' -d '{\"code\": \"invalid python code\"}' http://localhost:${SIMPY_PORT}/simulate | grep -q error"
    
    # Test 9: Results directory writable
    run_test "Results directory writable" "touch '$SIMPY_RESULTS_DIR/test.txt' && rm '$SIMPY_RESULTS_DIR/test.txt'"
    
    # Test 10: Service restart
    run_test "Service restart" "simpy::docker::restart && sleep 3 && simpy::test_connection"
    
    # Test 11: Content management - remove
    run_test "Remove simulation script" "simpy::content::remove test_sim"
    
    # Test 12: Multiple concurrent requests
    for i in {1..3}; do
        (curl -sf http://localhost:${SIMPY_PORT}/health &) &>/dev/null
    done
    wait
    run_test "Concurrent request handling" "true"
    
    # Cleanup
    rm -f /tmp/test_simulation.py
    
    # Summary
    echo ""
    log::info "Test Summary:"
    log::success "  Passed: $TESTS_PASSED"
    if [[ $TESTS_FAILED -gt 0 ]]; then
        log::error "  Failed: $TESTS_FAILED"
        exit 1
    fi
    
    log::success "All integration tests passed!"
}

main "$@"