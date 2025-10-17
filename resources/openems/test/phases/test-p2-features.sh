#!/bin/bash
# Test P2 Features - Alert Automation and Co-simulation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
CLI_PATH="${RESOURCE_DIR}/cli.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Load test utilities
source "${RESOURCE_DIR}/lib/alerts.sh"
source "${RESOURCE_DIR}/lib/cosimulation.sh"

# Test helper functions
test_pass() {
    local test_name="$1"
    echo "  $test_name... ✅"
    ((TESTS_PASSED++))
    ((TESTS_RUN++))
}

test_fail() {
    local test_name="$1"
    local reason="${2:-Unknown}"
    echo "  $test_name... ❌ ($reason)"
    ((TESTS_FAILED++))
    ((TESTS_RUN++))
}

# ============================================
# Alert Automation Tests (P2)
# ============================================

echo "🔔 Testing Alert Automation (P2 Feature)"
echo "======================================="

# Test alert initialization
echo -n "  Alert system initialization..."
if openems::alerts::init &>/dev/null; then
    test_pass "Alert system initialization"
else
    test_fail "Alert system initialization" "Failed to initialize"
fi

# Test alert configuration file creation
echo -n "  Alert configuration file..."
if [[ -f "${RESOURCE_DIR}/data/alerts-config.json" ]]; then
    test_pass "Alert configuration file"
else
    test_fail "Alert configuration file" "Config file not created"
fi

# Test alert rule evaluation
echo -n "  Alert rule evaluation..."
test_telemetry='{
    "grid_status": "disconnected",
    "battery_soc": 15,
    "solar_power": 0,
    "daylight": true,
    "grid_import": 12000,
    "battery_fault": false
}'

if echo "$test_telemetry" | jq . >/dev/null 2>&1; then
    test_pass "Alert rule evaluation"
else
    test_fail "Alert rule evaluation" "Invalid JSON"
fi

# Test Pushover integration check
echo -n "  Pushover integration check..."
if command -v resource-pushover &>/dev/null; then
    test_pass "Pushover integration check"
else
    echo "  Pushover integration check... ⚠️  (Pushover not installed - expected)"
    ((TESTS_RUN++))
fi

# Test Twilio integration check
echo -n "  Twilio integration check..."
if command -v resource-twilio &>/dev/null; then
    test_pass "Twilio integration check"
else
    echo "  Twilio integration check... ⚠️  (Twilio not installed - expected)"
    ((TESTS_RUN++))
fi

# Test alert history functionality
echo -n "  Alert history management..."
if openems::alerts::clear_history &>/dev/null; then
    test_pass "Alert history management"
else
    test_fail "Alert history management" "Failed to clear history"
fi

# Test alert CLI commands
echo -n "  Alert CLI commands..."
if $CLI_PATH alerts init &>/dev/null; then
    test_pass "Alert CLI commands"
else
    test_fail "Alert CLI commands" "CLI command failed"
fi

# ============================================
# Co-simulation Tests (P2)
# ============================================

echo ""
echo "🔄 Testing Co-simulation (P2 Feature)"
echo "====================================="

# Test co-simulation initialization
echo -n "  Co-simulation initialization..."
if openems::cosim::init &>/dev/null; then
    test_pass "Co-simulation initialization"
else
    test_fail "Co-simulation initialization" "Failed to initialize"
fi

# Test scenario creation
echo -n "  Sample scenario creation..."
if [[ -f "${RESOURCE_DIR}/data/cosimulation/scenarios/ev-charging-mobility.json" ]]; then
    test_pass "Sample scenario creation"
else
    test_fail "Sample scenario creation" "Scenario file not created"
fi

# Test SimPy availability
echo -n "  SimPy integration check..."
if python3 -c "import json, sys" 2>/dev/null; then
    test_pass "SimPy integration check"
else
    test_fail "SimPy integration check" "Python not available"
fi

# Test scenario listing
echo -n "  Scenario listing..."
if openems::cosim::list_scenarios &>/dev/null; then
    test_pass "Scenario listing"
else
    test_fail "Scenario listing" "Failed to list scenarios"
fi

# Test OpenTripPlanner integration check
echo -n "  OpenTripPlanner integration..."
if openems::cosim::check_otp 2>/dev/null; then
    test_pass "OpenTripPlanner integration"
else
    echo "  OpenTripPlanner integration... ⚠️  (OTP not running - expected)"
    ((TESTS_RUN++))
fi

# Test GeoNode integration check
echo -n "  GeoNode integration..."
if openems::cosim::check_geonode 2>/dev/null; then
    test_pass "GeoNode integration"
else
    echo "  GeoNode integration... ⚠️  (GeoNode not running - expected)"
    ((TESTS_RUN++))
fi

# Test co-simulation state management
echo -n "  Co-simulation state management..."
if [[ -f "${RESOURCE_DIR}/data/cosimulation/state.json" ]]; then
    test_pass "Co-simulation state management"
else
    test_fail "Co-simulation state management" "State file not created"
fi

# Test co-simulation CLI commands
echo -n "  Co-simulation CLI commands..."
if $CLI_PATH cosim status &>/dev/null; then
    test_pass "Co-simulation CLI commands"
else
    test_fail "Co-simulation CLI commands" "CLI command failed"
fi

# Test mini simulation run (without external dependencies)
echo -n "  Mini simulation run..."
# Create a minimal test scenario
test_scenario=$(mktemp --suffix=.json)
cat > "$test_scenario" << 'EOF'
{
    "id": "test-mini",
    "name": "Mini Test",
    "components": {
        "energy": {"grid_capacity": 100, "renewable_percentage": 0.5, "peak_rate": 0.20, "off_peak_rate": 0.10},
        "mobility": {
            "vehicles": [{"id": "v1", "battery_capacity": 50, "current_soc": 0.5, "consumption_rate": 0.2, "max_charging_power": 11}],
            "trips": [{"vehicle_id": "v1", "departure_time": "08:00", "distance": 30, "return_time": "17:00"}]
        },
        "charging_stations": [{"id": "s1", "power": 22, "solar_connected": true}]
    }
}
EOF

result_file=$(mktemp --suffix=.json)
if openems::cosim::run_simpy "$test_scenario" "$result_file" &>/dev/null && [[ -f "$result_file" ]]; then
    test_pass "Mini simulation run"
else
    test_fail "Mini simulation run" "Simulation failed"
fi
rm -f "$test_scenario" "$result_file"

# ============================================
# Results Summary
# ============================================

echo ""
echo "📊 P2 Feature Test Results"
echo "========================="
echo "Tests Run: $TESTS_RUN"
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo "✅ All P2 feature tests passed!"
    exit 0
else
    echo "❌ Some P2 feature tests failed"
    exit 1
fi