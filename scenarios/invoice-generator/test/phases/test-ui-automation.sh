#!/bin/bash
# UI automation tests for invoice-generator

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🖥️  Running Invoice Generator UI automation tests..."

# Verify UI is running
UI_PORT="${UI_PORT:-35470}"
API_PORT="${API_PORT:-19571}"

echo "📡 Checking UI health endpoint..."
if ! curl -sf "http://localhost:${UI_PORT}/health" > /dev/null; then
    echo "❌ UI health check failed - is the UI running?"
    exit 1
fi
echo "✅ UI health check passed"

echo "📡 Checking API health endpoint..."
if ! curl -sf "http://localhost:${API_PORT}/health" > /dev/null; then
    echo "❌ API health check failed - is the API running?"
    exit 1
fi
echo "✅ API health check passed"

# Screenshot validation using browserless
echo "📸 Capturing UI screenshot for validation..."
OUTPUT_FILE="/tmp/invoice-generator-ui-test.png"

if vrooli resource browserless screenshot \
    --url "http://localhost:${UI_PORT}" \
    --output "$OUTPUT_FILE" \
    --wait-for "body" \
    --viewport-width 1920 \
    --viewport-height 1080 2>&1; then

    # Verify screenshot was created and has reasonable file size
    if [ ! -f "$OUTPUT_FILE" ]; then
        echo "❌ Screenshot file was not created"
        exit 1
    fi

    FILE_SIZE=$(stat -c%s "$OUTPUT_FILE" 2>/dev/null || stat -f%z "$OUTPUT_FILE" 2>/dev/null || echo "0")
    if [ "$FILE_SIZE" -lt 10000 ]; then
        echo "❌ Screenshot file too small ($FILE_SIZE bytes) - UI may not have loaded"
        exit 1
    fi

    echo "✅ UI screenshot captured successfully (${FILE_SIZE} bytes)"
    echo "   📁 Screenshot saved to: $OUTPUT_FILE"
else
    echo "⚠️  Screenshot capture failed - browserless may not be available"
    echo "   Continuing tests without screenshot validation..."
fi

# Basic UI endpoint tests
echo "🔍 Testing UI serves index page..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${UI_PORT}/")
if [ "$HTTP_CODE" != "200" ]; then
    echo "❌ UI index page returned HTTP $HTTP_CODE"
    exit 1
fi
echo "✅ UI index page accessible (HTTP 200)"

# Verify UI can reach API
echo "🔗 Verifying UI can communicate with API..."
UI_HEALTH=$(curl -sf "http://localhost:${UI_PORT}/health" | jq -r '.api_connectivity.connected' 2>/dev/null || echo "false")
if [ "$UI_HEALTH" != "true" ]; then
    echo "❌ UI cannot communicate with API"
    exit 1
fi
echo "✅ UI successfully communicates with API"

testing::phase::end_with_summary "Invoice Generator UI automation tests completed"
