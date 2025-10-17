#!/bin/bash

# OpenEMS Unit Tests
# Library function validation (<60s)

set -eo pipefail  # Remove 'u' to avoid unbound variable issues

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Set default ports for testing
export OPENEMS_PORT="${OPENEMS_PORT:-8084}"
export OPENEMS_JSONRPC_PORT="${OPENEMS_JSONRPC_PORT:-8085}"
export OPENEMS_MODBUS_PORT="${OPENEMS_MODBUS_PORT:-502}"
export OPENEMS_BACKEND_PORT="${OPENEMS_BACKEND_PORT:-8086}"

# Source configuration if available (but don't fail if it doesn't work)
source "${RESOURCE_DIR}/config/defaults.sh" 2>/dev/null || true

# Source libraries for testing
source "${RESOURCE_DIR}/lib/core.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0

# Test function
run_test() {
    local test_name="$1"
    local command="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "  Testing $test_name... "
    
    if eval "$command" &>/dev/null; then
        echo "✅"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo "❌"
        return 1
    fi
}

echo "🧪 OpenEMS Unit Tests"
echo "==================="

# Test 1: Configuration validation (check port is set and is a number)
run_test "port configuration" "[[ -n '$OPENEMS_PORT' && '$OPENEMS_PORT' =~ ^[0-9]+$ ]]"

# Test 2: Data directory creation
run_test "data directory creation" "mkdir -p ${RESOURCE_DIR}/data/test && [[ -d ${RESOURCE_DIR}/data/test ]]"

# Test 3: Default config creation
run_test "default config creation" "openems::create_default_config"

# Test 4: DER configuration creation
run_test "DER config generation" "openems::configure_der wind 15"

# Test 5: Solar simulation data
run_test "solar simulation data" "openems::simulate_solar 6000"

# Test 6: Load simulation data
run_test "load simulation data" "openems::simulate_load 4000"

# Test 7: Content listing function
run_test "content list function" "content::list"

# Test 8: Configuration file handling
echo '{"test": "data"}' > /tmp/unit_test.json
run_test "config file addition" "content::add unit_test /tmp/unit_test.json"

# Test 9: Configuration retrieval
run_test "config retrieval" "content::get unit_test | grep -q 'test'"

# Test 10: Configuration removal
run_test "config removal" "content::remove unit_test"

# Test 11: Status display function
run_test "status display" "status::show"

# Test 12: Verbose status
run_test "verbose status" "status::show --verbose"

# Test 13: JSON validation for DER config
run_test "DER JSON structure" "[[ -f ${RESOURCE_DIR}/data/configs/der_wind.json ]]"

# Test 14: Simulation data validation (check both locations)
run_test "simulation data exists" "[[ -f ${RESOURCE_DIR}/data/edge/data/solar_sim.json ]] || [[ -f /tmp/openems_solar_sim.json ]]"

# Test 15: Clean test data
run_test "cleanup test data" "rm -rf ${RESOURCE_DIR}/data/test && rm -f /tmp/unit_test.json"

# Summary
echo ""
echo "📊 Unit Test Results"
echo "==================="
echo "Tests Run: $TESTS_RUN"
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $((TESTS_RUN - TESTS_PASSED))"

if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
    echo "✅ All unit tests passed!"
    exit 0
else
    echo "❌ Some unit tests failed"
    exit 1
fi