#\!/usr/bin/env bash
# SimPy integration tests

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
TEST_DIR="${APP_ROOT}/resources/simpy/test"
LIB_DIR="${APP_ROOT}/resources/simpy/lib"

# Source modules
source "${LIB_DIR}/core.sh"
source "${LIB_DIR}/status.sh"

# Test results
PASSED=0
FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    echo -n "Testing: $test_name ... "
    if eval "$test_cmd" >/dev/null 2>&1; then
        echo "✅ PASSED"
        ((PASSED++))
    else
        echo "❌ FAILED"
        ((FAILED++))
    fi
}

echo "=== SimPy Integration Tests ==="
echo ""

# Run tests
run_test "service is installed" "simpy::is_installed"
run_test "service is running" "simpy::is_running"
run_test "health endpoint responds" "simpy::test_connection"
run_test "can get version" "simpy::get_version | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$'"
run_test "health returns JSON" "curl -s http://localhost:9510/health | jq -e '.status'"
run_test "examples endpoint works" "curl -s http://localhost:9510/examples | jq -e '.'"
run_test "examples directory has files" "test -n \"\$(simpy::list_examples)\""
run_test "status check works" "simpy::status --format json | jq -e '.health'"

# Test basic simulation
cat > /tmp/test_simpy.py << 'SIMTEST'
import simpy
import json
env = simpy.Environment()
def process(env):
    yield env.timeout(1)
env.process(process(env))
env.run(until=5)
print(json.dumps({"success": True, "time": env.now}))
SIMTEST

run_test "can run simulation" "python3 /tmp/test_simpy.py | jq -e '.success'"
rm -f /tmp/test_simpy.py

echo ""
echo "=== Test Results ==="
echo "Passed: $PASSED"
echo "Failed: $FAILED"

# Save results for status
mkdir -p "${var_ROOT_DIR}/data/test-results"
cat > "${var_ROOT_DIR}/data/test-results/simpy-test.json" << JSON
{
    "status": "$([ $FAILED -eq 0 ] && echo 'passed' || echo 'failed')",
    "passed": $PASSED,
    "failed": $FAILED,
    "timestamp": "$(date -Iseconds)"
}
JSON

# Exit with appropriate code
[ $FAILED -eq 0 ]
