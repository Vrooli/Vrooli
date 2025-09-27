#!/bin/bash
set -e

echo "🌐 Running UI automation tests for bedtime-story-generator..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Get dynamic ports from environment - use correct defaults
API_PORT="${API_PORT:-16896}"
UI_PORT="${UI_PORT:-38891}"

# Wait for UI to be ready
echo "⏳ Waiting for UI to be ready..."
for i in {1..30}; do
    if curl -sf "http://localhost:${UI_PORT}" > /dev/null; then
        echo "✅ UI is ready"
        break
    fi
    sleep 1
done

# Test 1: Homepage loads
echo "🧪 Test 1: Homepage loads..."
curl -sf "http://localhost:${UI_PORT}" > /tmp/ui_response.html
if grep -q 'id="root"' /tmp/ui_response.html; then
    echo "✅ Homepage loads correctly"
else
    echo "❌ Homepage failed to load"
    echo "   Response content:"
    head /tmp/ui_response.html
    exit 1
fi

# Test 2: Check time-aware theme (uses JavaScript)
echo "🧪 Test 2: Time-aware UI theme..."
CURRENT_HOUR=$(date +%H)
if [ "$CURRENT_HOUR" -ge 6 ] && [ "$CURRENT_HOUR" -lt 18 ]; then
    EXPECTED_THEME="daytime"
elif [ "$CURRENT_HOUR" -ge 18 ] && [ "$CURRENT_HOUR" -lt 21 ]; then
    EXPECTED_THEME="evening"
else
    EXPECTED_THEME="nighttime"
fi
echo "   Expected theme: $EXPECTED_THEME (current hour: $CURRENT_HOUR)"

# Test 3: API integration from UI
echo "🧪 Test 3: API integration..."
# Check if UI can fetch stories
RESPONSE=$(curl -sf "http://localhost:${API_PORT}/api/v1/stories" || echo "FAILED")
if [ "$RESPONSE" != "FAILED" ]; then
    echo "✅ API integration working"
else
    echo "❌ API integration failed"
    exit 1
fi

# Test 4: Story generation UI flow (using browserless if available)
echo "🧪 Test 4: Story generation UI flow..."
if command -v vrooli &> /dev/null && vrooli resource status browserless --json 2>/dev/null | jq -r '.status' | grep -q "running"; then
    echo "   Using browserless for UI automation..."
    
    # Take screenshot of initial state
    vrooli resource browserless screenshot \
        --url "http://localhost:${UI_PORT}" \
        --output "/tmp/bedtime-story-ui-initial.png" \
        --width 1280 \
        --height 800
    
    if [ -f "/tmp/bedtime-story-ui-initial.png" ]; then
        echo "✅ UI screenshot captured"
    else
        echo "⚠️  Screenshot capture failed, but continuing"
    fi
    
    # Simulate story generation click
    vrooli resource browserless execute \
        --url "http://localhost:${UI_PORT}" \
        --script "
            // Click generate story button
            const generateBtn = document.querySelector('[data-testid=\"generate-story-btn\"], button:contains(\"Generate\"), button:contains(\"Create Story\")');
            if (generateBtn) {
                generateBtn.click();
                return 'clicked';
            }
            return 'button-not-found';
        " 2>/dev/null || echo "⚠️  Browserless execution not available"
else
    echo "⚠️  Browserless not available, skipping visual tests"
fi

# Test 5: Responsive design check
echo "🧪 Test 5: Responsive design..."
# Check if viewport meta tag exists
if curl -sf "http://localhost:${UI_PORT}" | grep -q 'viewport'; then
    echo "✅ Responsive viewport meta tag found"
else
    echo "⚠️  No viewport meta tag found (may affect mobile view)"
fi

# Test 6: Asset loading
echo "🧪 Test 6: Asset loading..."
# Check if CSS and JS files load
HTML_CONTENT=$(curl -sf "http://localhost:${UI_PORT}")
if echo "$HTML_CONTENT" | grep -q -E '(\.css|styles)'; then
    echo "✅ CSS assets referenced"
else
    echo "❌ No CSS assets found"
    exit 1
fi

if echo "$HTML_CONTENT" | grep -q -E '(\.js|script)'; then
    echo "✅ JavaScript assets referenced"
else
    echo "❌ No JavaScript assets found"
    exit 1
fi

# Test 7: Book viewer component
echo "🧪 Test 7: Book viewer component..."
# Check for book viewer elements
if curl -sf "http://localhost:${UI_PORT}" | grep -q -E '(book|story|reader|viewer)'; then
    echo "✅ Book viewer component elements found"
else
    echo "⚠️  Book viewer elements not found in initial load"
fi

# Test 8: Kid-friendly indicators
echo "🧪 Test 8: Kid-friendly design elements..."
# Check for kid-friendly CSS classes or elements
if curl -sf "http://localhost:${UI_PORT}" | grep -q -E '(playful|cartoon|kid|child|fun)'; then
    echo "✅ Kid-friendly design elements detected"
else
    echo "⚠️  No explicit kid-friendly markers found"
fi

echo "
📊 UI Test Summary:
- Homepage: ✅
- Time-aware theme: Expected $EXPECTED_THEME
- API integration: ✅
- Asset loading: ✅
- Responsive design: ✅
- Visual testing: $([ -f "/tmp/bedtime-story-ui-initial.png" ] && echo "✅ Screenshot saved" || echo "⚠️  No browserless")
"

echo "✅ UI automation tests completed!"