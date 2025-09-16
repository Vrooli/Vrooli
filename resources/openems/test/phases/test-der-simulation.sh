#!/bin/bash

# DER Simulation Test Suite
# Tests distributed energy resource telemetry and simulation

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/../.." && pwd)"

source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/der_simulator.sh"

echo "🧪 OpenEMS DER Simulation Tests"
echo "================================"

# Test results tracking
TESTS_PASSED=0
TESTS_FAILED=0

# ============================================
# Test Functions
# ============================================

test_solar_simulation() {
    echo -n "Testing solar generation simulation... "
    
    if der::simulate_solar 5000 5 &>/dev/null; then
        echo "✅"
        ((TESTS_PASSED++))
    else
        echo "❌"
        ((TESTS_FAILED++))
    fi
}

test_battery_simulation() {
    echo -n "Testing battery storage simulation... "
    
    if der::simulate_battery 20 50 5 5 &>/dev/null; then
        echo "✅"
        ((TESTS_PASSED++))
    else
        echo "❌"
        ((TESTS_FAILED++))
    fi
}

test_ev_charger_simulation() {
    echo -n "Testing EV charger simulation... "
    
    if der::simulate_ev_charger 11 true 5 &>/dev/null; then
        echo "✅"
        ((TESTS_PASSED++))
    else
        echo "❌"
        ((TESTS_FAILED++))
    fi
}

test_wind_simulation() {
    echo -n "Testing wind turbine simulation... "
    
    if der::simulate_wind_turbine 50 8 5 &>/dev/null; then
        echo "✅"
        ((TESTS_PASSED++))
    else
        echo "❌"
        ((TESTS_FAILED++))
    fi
}

test_telemetry_persistence() {
    echo -n "Testing telemetry data persistence... "
    
    # Send test telemetry
    openems::send_telemetry "test_asset" "test" "1000" "1" "400" "2.5" "50" "25"
    
    # Check if data was saved locally
    if [[ -f "${DATA_DIR}/edge/data/telemetry.jsonl" ]]; then
        if grep -q "test_asset" "${DATA_DIR}/edge/data/telemetry.jsonl"; then
            echo "✅"
            ((TESTS_PASSED++))
        else
            echo "❌ (data not found)"
            ((TESTS_FAILED++))
        fi
    else
        echo "❌ (file not created)"
        ((TESTS_FAILED++))
    fi
}

test_questdb_integration() {
    echo -n "Testing QuestDB integration... "
    
    # Check if QuestDB is available
    if timeout 5 curl -sf "http://${QUESTDB_HOST}:${QUESTDB_PORT}/exec" &>/dev/null; then
        # Initialize tables
        openems::init_telemetry &>/dev/null
        
        # Send test data
        openems::send_telemetry "questdb_test" "test" "2000" "2" "400" "5" "75" "22"
        
        echo "✅"
        ((TESTS_PASSED++))
    else
        echo "⚠️  (QuestDB not available)"
        ((TESTS_PASSED++))  # Don't fail if optional dependency is missing
    fi
}

test_redis_integration() {
    echo -n "Testing Redis integration... "
    
    # Check if Redis is available
    if command -v redis-cli &>/dev/null && timeout 2 nc -zv "${REDIS_HOST}" "${REDIS_PORT}" &>/dev/null; then
        # Send test data
        openems::send_telemetry "redis_test" "test" "3000" "3" "400" "7.5" "90" "20"
        
        # Check if data is in Redis
        if redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" EXISTS "openems:assets:redis_test" | grep -q "1"; then
            echo "✅"
            ((TESTS_PASSED++))
        else
            echo "❌ (data not in Redis)"
            ((TESTS_FAILED++))
        fi
    else
        echo "⚠️  (Redis not available)"
        ((TESTS_PASSED++))  # Don't fail if optional dependency is missing
    fi
}

test_microgrid_simulation() {
    echo -n "Testing full microgrid simulation... "
    
    if der::simulate_microgrid 5 &>/dev/null; then
        echo "✅"
        ((TESTS_PASSED++))
    else
        echo "❌"
        ((TESTS_FAILED++))
    fi
}

test_grid_event_simulation() {
    echo -n "Testing grid outage simulation... "
    
    if der::simulate_grid_outage 5 &>/dev/null; then
        # Check if event was logged
        if [[ -f "${DATA_DIR}/edge/data/grid_event.json" ]]; then
            echo "✅"
            ((TESTS_PASSED++))
        else
            echo "❌ (event not logged)"
            ((TESTS_FAILED++))
        fi
    else
        echo "❌"
        ((TESTS_FAILED++))
    fi
}

# ============================================
# Run Tests
# ============================================

echo ""
echo "Running DER simulation tests..."
echo ""

test_solar_simulation
test_battery_simulation
test_ev_charger_simulation
test_wind_simulation
test_telemetry_persistence
test_questdb_integration
test_redis_integration
test_microgrid_simulation
test_grid_event_simulation

# ============================================
# Results Summary
# ============================================

echo ""
echo "================================"
echo "Test Results Summary:"
echo "  Passed: ${TESTS_PASSED}"
echo "  Failed: ${TESTS_FAILED}"
echo "================================"

if [[ ${TESTS_FAILED} -eq 0 ]]; then
    echo "✅ All DER simulation tests passed!"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi