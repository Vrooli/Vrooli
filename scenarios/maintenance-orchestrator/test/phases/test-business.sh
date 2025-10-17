#!/bin/bash
set -euo pipefail

echo "=== Business Logic Tests ==="

SCENARIO_NAME="maintenance-orchestrator"
failed=0

# Get dynamic ports
API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || echo "")
if [ -z "$API_PORT" ]; then
  echo "❌ Could not discover API_PORT - is the scenario running?"
  exit 1
fi

API_BASE_URL="http://localhost:$API_PORT/api/v1"

echo "Testing business workflows..."

# Test 1: Discovery and listing
echo "Test 1: Scenario discovery"
response=$(curl -sf "$API_BASE_URL/scenarios")
scenario_count=$(echo "$response" | jq '.scenarios | length')
if [ "$scenario_count" -gt 0 ]; then
  echo "✅ Discovery working: Found $scenario_count scenarios"
else
  echo "❌ Discovery failed: No scenarios found"
  failed=1
fi

# Test 2: All scenarios start inactive
echo "Test 2: Default inactive state"
active_count=$(curl -sf "$API_BASE_URL/status" | jq '.activeScenarios')
if [ "$active_count" -eq 0 ]; then
  echo "✅ All scenarios start inactive (count: $active_count)"
else
  echo "❌ Scenarios should start inactive but found $active_count active"
  failed=1
fi

# Test 3: Activate a scenario (using a known scenario ID)
echo "Test 3: Scenario activation"
# Get first scenario ID
first_scenario=$(echo "$response" | jq -r '.scenarios[0].id')
echo "   Activating: $first_scenario"

activate_response=$(curl -sf -X POST "$API_BASE_URL/scenarios/$first_scenario/activate")
if echo "$activate_response" | jq -e '.success == true' > /dev/null; then
  echo "✅ Activation successful"
else
  echo "❌ Activation failed"
  failed=1
fi

# Verify activation
active_count=$(curl -sf "$API_BASE_URL/status" | jq '.activeScenarios')
if [ "$active_count" -eq 1 ]; then
  echo "✅ Verified: 1 scenario active"
else
  echo "❌ Expected 1 active scenario, found $active_count"
  failed=1
fi

# Test 4: Deactivate scenario
echo "Test 4: Scenario deactivation"
deactivate_response=$(curl -sf -X POST "$API_BASE_URL/scenarios/$first_scenario/deactivate")
if echo "$deactivate_response" | jq -e '.success == true' > /dev/null; then
  echo "✅ Deactivation successful"
else
  echo "❌ Deactivation failed"
  failed=1
fi

# Verify deactivation
active_count=$(curl -sf "$API_BASE_URL/status" | jq '.activeScenarios')
if [ "$active_count" -eq 0 ]; then
  echo "✅ Verified: 0 scenarios active"
else
  echo "❌ Expected 0 active scenarios, found $active_count"
  failed=1
fi

# Test 5: Preset application
echo "Test 5: Preset functionality"
presets_response=$(curl -sf "$API_BASE_URL/presets")
preset_count=$(echo "$presets_response" | jq '.presets | length')
if [ "$preset_count" -ge 3 ]; then
  echo "✅ Found $preset_count presets (requirement: ≥3)"
else
  echo "❌ Expected at least 3 presets, found $preset_count"
  failed=1
fi

# Test 6: Stop-all functionality
echo "Test 6: Emergency stop-all"
# First activate a scenario
curl -sf -X POST "$API_BASE_URL/scenarios/$first_scenario/activate" > /dev/null

# Then stop all
stop_response=$(curl -sf -X POST "$API_BASE_URL/stop-all")
if echo "$stop_response" | jq -e '.success == true' > /dev/null; then
  echo "✅ Stop-all executed successfully"

  # Verify all inactive
  active_count=$(curl -sf "$API_BASE_URL/status" | jq '.activeScenarios')
  if [ "$active_count" -eq 0 ]; then
    echo "✅ Verified: All scenarios stopped"
  else
    echo "❌ Expected 0 active after stop-all, found $active_count"
    failed=1
  fi
else
  echo "❌ Stop-all failed"
  failed=1
fi

# Test 7: CLI business workflows
echo "Test 7: CLI business operations"
if command -v maintenance-orchestrator &> /dev/null; then
  # Test preset listing
  if maintenance-orchestrator preset list > /dev/null 2>&1; then
    echo "✅ CLI preset list works"
  else
    echo "❌ CLI preset list failed"
    failed=1
  fi
else
  echo "⚠️  CLI not installed - skipping CLI tests"
fi

exit $failed
