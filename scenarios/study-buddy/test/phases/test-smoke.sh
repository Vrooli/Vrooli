#!/usr/bin/env bash
# Smoke tests for study-buddy scenario

set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Load environment
if [ -f "$SCENARIO_DIR/.env" ]; then
    export $(grep -v '^#' "$SCENARIO_DIR/.env" | xargs)
fi

# Get actual running ports dynamically
PORTS_JSON=$(vrooli scenario port study-buddy --json 2>/dev/null || echo '{}')
API_PORT=$(echo "$PORTS_JSON" | jq -r '.ports[]? | select(.key=="API_PORT") | .port' 2>/dev/null || echo "")
UI_PORT=$(echo "$PORTS_JSON" | jq -r '.ports[]? | select(.key=="UI_PORT") | .port' 2>/dev/null || echo "")

# Fallback to environment or defaults
API_PORT="${API_PORT:-${API_PORT_ENV:-18775}}"
UI_PORT="${UI_PORT:-${UI_PORT_ENV:-36804}}"

echo "🔥 Running smoke tests..."

# Test 1: API health check
echo -n "Testing API health endpoint... "
if curl -sf "http://localhost:$API_PORT/health" | grep -q "healthy"; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    exit 1
fi

# Test 2: UI availability
echo -n "Testing UI availability... "
if curl -sf "http://localhost:$UI_PORT/" | grep -q "Study Buddy"; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    exit 1
fi

# Test 3: CLI availability
echo -n "Testing CLI availability... "
if study-buddy help >/dev/null 2>&1; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    exit 1
fi

echo "✅ All smoke tests passed!"
exit 0