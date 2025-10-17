#!/bin/bash
set -euo pipefail

echo "=== Integration Tests ==="

SCENARIO_NAME="maintenance-orchestrator"
failed=0

# Get dynamic ports using vrooli command
echo "Discovering service ports..."
API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || echo "")
UI_PORT=$(vrooli scenario port "$SCENARIO_NAME" UI_PORT 2>/dev/null || echo "")

if [ -z "$API_PORT" ]; then
  echo "❌ Could not discover API_PORT - is the scenario running?"
  echo "   Start with: vrooli scenario start $SCENARIO_NAME"
  exit 1
fi

echo "✅ API Port: $API_PORT"
echo "✅ UI Port: $UI_PORT"

API_BASE_URL="http://localhost:$API_PORT"
UI_BASE_URL="http://localhost:$UI_PORT"

# Test health endpoint
echo "Testing health endpoint..."
if curl -sf "$API_BASE_URL/health" > /dev/null; then
  echo "✅ Health endpoint responding"
else
  echo "❌ Health endpoint not responding"
  failed=1
fi

# Test API endpoints
echo "Testing API v1 endpoints..."

# Test scenarios endpoint
if curl -sf "$API_BASE_URL/api/v1/scenarios" | jq -e '.scenarios' > /dev/null 2>&1; then
  scenario_count=$(curl -sf "$API_BASE_URL/api/v1/scenarios" | jq '.scenarios | length')
  echo "✅ Scenarios endpoint: Found $scenario_count scenarios"
else
  echo "❌ Scenarios endpoint failed"
  failed=1
fi

# Test status endpoint
if curl -sf "$API_BASE_URL/api/v1/status" | jq -e '.totalScenarios' > /dev/null 2>&1; then
  total=$(curl -sf "$API_BASE_URL/api/v1/status" | jq '.totalScenarios')
  echo "✅ Status endpoint: $total total scenarios"
else
  echo "❌ Status endpoint failed"
  failed=1
fi

# Test presets endpoint
if curl -sf "$API_BASE_URL/api/v1/presets" | jq -e '.presets' > /dev/null 2>&1; then
  preset_count=$(curl -sf "$API_BASE_URL/api/v1/presets" | jq '.presets | length')
  echo "✅ Presets endpoint: Found $preset_count presets"
else
  echo "❌ Presets endpoint failed"
  failed=1
fi

# Test UI accessibility
echo "Testing UI accessibility..."
ui_temp=$(mktemp)
if /usr/bin/curl -sf "$UI_BASE_URL" > "$ui_temp" 2>/dev/null && /usr/bin/grep -q "Maintenance Orchestrator" "$ui_temp"; then
  echo "✅ UI accessible and contains expected content"
else
  echo "❌ UI not accessible or missing content"
  failed=1
fi
rm -f "$ui_temp"

# Test CLI integration
echo "Testing CLI integration..."
if command -v maintenance-orchestrator &> /dev/null; then
  echo "✅ CLI installed and accessible"

  # Test CLI commands
  if maintenance-orchestrator status --json > /dev/null 2>&1; then
    echo "✅ CLI status command works"
  else
    echo "❌ CLI status command failed"
    failed=1
  fi

  if maintenance-orchestrator list > /dev/null 2>&1; then
    echo "✅ CLI list command works"
  else
    echo "❌ CLI list command failed"
    failed=1
  fi
else
  echo "⚠️  CLI not installed - run: cd cli && ./install.sh"
fi

exit $failed
