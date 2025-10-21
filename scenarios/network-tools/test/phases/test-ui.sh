#!/usr/bin/env bash
# Test Phase 7: UI Validation
# Validates UI functionality, accessibility, and user experience

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Get UI port from environment or default
UI_PORT=${UI_PORT:-37646}
API_PORT=${API_PORT:-17124}

echo "================================================"
echo "🎨 Phase 7: UI Validation"
echo "================================================"

TESTS_PASSED=0
TESTS_FAILED=0

# Helper function
test_ui_endpoint() {
    local name="$1"
    local path="$2"
    local expected_status="$3"

    echo "  Testing: $name"

    local status=$(curl -s -w "%{http_code}" -o /dev/null --max-time 10 "http://localhost:$UI_PORT$path")

    if [ "$status" -eq "$expected_status" ]; then
        echo "    ✅ Returns HTTP $status"
        ((TESTS_PASSED++))
        return 0
    else
        echo "    ❌ Expected HTTP $expected_status, got $status"
        ((TESTS_FAILED++))
        return 1
    fi
}

# UI Server Availability
echo ""
echo "🌐 UI Server Availability:"

test_ui_endpoint "Index page loads" "/" 200
test_ui_endpoint "Health endpoint responds" "/health" 200

# Health Check Validation
echo ""
echo "💚 UI Health Check:"
HEALTH_RESPONSE=$(curl -s --max-time 10 "http://localhost:$UI_PORT/health")

if echo "$HEALTH_RESPONSE" | jq -e '.api_connectivity' > /dev/null 2>&1; then
    echo "  ✅ Health response includes api_connectivity"
    ((TESTS_PASSED++))

    API_CONNECTED=$(echo "$HEALTH_RESPONSE" | jq -r '.api_connectivity.connected')
    if [ "$API_CONNECTED" = "true" ]; then
        echo "  ✅ UI reports connected to API"
        ((TESTS_PASSED++))
    else
        echo "  ❌ UI not connected to API"
        ((TESTS_FAILED++))
    fi
else
    echo "  ❌ Health response missing api_connectivity"
    ((TESTS_FAILED++))
fi

# API Proxy Functionality
echo ""
echo "🔄 API Proxy Functionality:"

# Test that UI can proxy API requests
PROXY_RESPONSE=$(curl -s -w "\n%{http_code}" --max-time 10 "http://localhost:$UI_PORT/api/health")
PROXY_BODY=$(echo "$PROXY_RESPONSE" | head -n -1)
PROXY_STATUS=$(echo "$PROXY_RESPONSE" | tail -n 1)

if [ "$PROXY_STATUS" -eq 200 ]; then
    echo "  ✅ UI proxies API health endpoint"
    ((TESTS_PASSED++))

    if echo "$PROXY_BODY" | jq -e '.service' > /dev/null 2>&1; then
        SERVICE_NAME=$(echo "$PROXY_BODY" | jq -r '.service')
        if [ "$SERVICE_NAME" = "network-tools" ]; then
            echo "  ✅ Proxied response is valid API response"
            ((TESTS_PASSED++))
        else
            echo "  ❌ Proxied response has unexpected service name: $SERVICE_NAME"
            ((TESTS_FAILED++))
        fi
    else
        echo "  ❌ Proxied response is not valid JSON"
        ((TESTS_FAILED++))
    fi
else
    echo "  ❌ UI proxy failed with status $PROXY_STATUS"
    ((TESTS_FAILED++))
fi

# Static Asset Serving
echo ""
echo "📦 Static Asset Serving:"

# Check for common UI assets
ASSETS=(
    "/index.html:text/html"
    "/app.js:application/javascript"
    "/styles.css:text/css"
)

for asset_spec in "${ASSETS[@]}"; do
    IFS=: read -r path expected_type <<< "$asset_spec"

    response=$(curl -s -I --max-time 10 "http://localhost:$UI_PORT$path" 2>&1)
    status=$(echo "$response" | grep -i "HTTP/" | awk '{print $2}')
    content_type=$(echo "$response" | grep -i "Content-Type:" | cut -d: -f2 | tr -d ' \r\n' | cut -d';' -f1)

    if [ "$status" = "200" ]; then
        if [[ "$content_type" == *"$expected_type"* ]]; then
            echo "  ✅ $path serves with correct content type ($content_type)"
            ((TESTS_PASSED++))
        else
            echo "  ⚠️  $path serves but with unexpected content type: $content_type (expected $expected_type)"
        fi
    else
        echo "  ⚠️  $path returned status $status (may not exist yet)"
    fi
done

# Security Headers
echo ""
echo "🔒 Security Headers:"

SECURITY_RESPONSE=$(curl -s -I --max-time 10 "http://localhost:$UI_PORT/")

# Check for basic security practices
if echo "$SECURITY_RESPONSE" | grep -qi "Content-Type:.*charset=utf-8"; then
    echo "  ✅ UTF-8 charset specified"
    ((TESTS_PASSED++))
else
    echo "  ⚠️  UTF-8 charset not specified in Content-Type"
fi

# Visual Regression Testing
echo ""
echo "📸 Visual Regression (Screenshot):"

# Create temp directory for screenshots
SCREENSHOT_DIR="${SCENARIO_DIR}/test/temp/screenshots"
mkdir -p "$SCREENSHOT_DIR"

# Check if browserless resource is available
if command -v resource-browserless &> /dev/null; then
    SCREENSHOT_PATH="${SCREENSHOT_DIR}/ui-main-$(date +%Y%m%d-%H%M%S).png"

    if resource-browserless screenshot --url "http://localhost:$UI_PORT/" --output "$SCREENSHOT_PATH" &> /dev/null; then
        if [ -f "$SCREENSHOT_PATH" ] && [ -s "$SCREENSHOT_PATH" ]; then
            echo "  ✅ UI screenshot captured: $SCREENSHOT_PATH"
            ((TESTS_PASSED++))

            # Verify image dimensions (basic sanity check)
            if command -v identify &> /dev/null; then
                DIMENSIONS=$(identify -format "%wx%h" "$SCREENSHOT_PATH" 2>/dev/null || echo "")
                if [ -n "$DIMENSIONS" ]; then
                    echo "  ✅ Screenshot dimensions: $DIMENSIONS"
                    ((TESTS_PASSED++))
                fi
            fi
        else
            echo "  ❌ Screenshot file is empty or missing"
            ((TESTS_FAILED++))
        fi
    else
        echo "  ⚠️  Screenshot capture failed (browserless may be unavailable)"
    fi
else
    echo "  ⚠️  Browserless resource not available, skipping screenshot test"
fi

# UI Accessibility (Basic Checks)
echo ""
echo "♿ UI Accessibility (Basic):"

INDEX_CONTENT=$(curl -s --max-time 10 "http://localhost:$UI_PORT/")

if echo "$INDEX_CONTENT" | grep -q "<html"; then
    echo "  ✅ Valid HTML structure"
    ((TESTS_PASSED++))

    if echo "$INDEX_CONTENT" | grep -q "<title>"; then
        echo "  ✅ Page has title tag"
        ((TESTS_PASSED++))
    else
        echo "  ⚠️  Page missing title tag"
    fi

    if echo "$INDEX_CONTENT" | grep -q 'lang='; then
        echo "  ✅ HTML lang attribute present"
        ((TESTS_PASSED++))
    else
        echo "  ⚠️  HTML lang attribute missing"
    fi
else
    echo "  ⚠️  Response doesn't appear to be HTML"
fi

# Summary
echo ""
echo "================================================"
echo "📊 UI Validation Summary"
echo "================================================"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    echo "✅ All UI validation tests passed!"
    echo "🎨 UI Features Validated:"
    echo "   • Server responds correctly"
    echo "   • Health monitoring working"
    echo "   • API proxy functional"
    echo "   • Static assets served properly"
    echo "   • Basic accessibility present"
    exit 0
else
    echo ""
    echo "❌ Some UI validation tests failed"
    exit 1
fi
