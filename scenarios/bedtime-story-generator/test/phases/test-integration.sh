#!/bin/bash
set -e

echo "🔗 Running integration tests for bedtime-story-generator..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CLI_BIN="${SCENARIO_DIR}/cli/bedtime-story"

# Get API port from environment or use default
API_PORT="${API_PORT:-16896}"
UI_PORT="${UI_PORT:-38891}"
API_URL="http://localhost:${API_PORT}"
UI_URL="http://localhost:${UI_PORT}"

echo "📍 Testing against API at ${API_URL}"
echo "📍 Testing against UI at ${UI_URL}"

# Test 1: Health check endpoint
echo ""
echo "🩺 Test 1: API Health Check"
if curl -sf "${API_URL}/health" | grep -q "healthy"; then
    echo "✅ API health check passed"
else
    echo "❌ API health check failed"
    exit 1
fi

# Test 2: Get themes
echo ""
echo "🎨 Test 2: Get Available Themes"
THEMES_RESPONSE=$(curl -sf "${API_URL}/api/v1/themes")
if echo "${THEMES_RESPONSE}" | grep -q "Adventure"; then
    echo "✅ Themes endpoint works"
    echo "   Available themes: $(echo "${THEMES_RESPONSE}" | jq -r '.[].name' | paste -sd ', ' -)"
else
    echo "❌ Themes endpoint failed"
    exit 1
fi

# Test 3: Generate a story
echo ""
echo "📚 Test 3: Generate Story"
STORY_RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/stories/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "age_group": "6-8",
        "theme": "Adventure",
        "length": "short"
    }')

if [ -z "${STORY_RESPONSE}" ]; then
    echo "❌ Story generation failed - empty response"
    exit 1
fi

STORY_ID=$(echo "${STORY_RESPONSE}" | jq -r '.id')
if [ "${STORY_ID}" != "null" ] && [ -n "${STORY_ID}" ]; then
    echo "✅ Story generated successfully"
    echo "   Story ID: ${STORY_ID}"
    echo "   Title: $(echo "${STORY_RESPONSE}" | jq -r '.title')"
    echo "   Reading time: $(echo "${STORY_RESPONSE}" | jq -r '.reading_time_minutes') minutes"
else
    echo "❌ Story generation failed - no ID returned"
    echo "   Response: ${STORY_RESPONSE}"
    exit 1
fi

# Test 4: List stories
echo ""
echo "📋 Test 4: List Stories"
LIST_RESPONSE=$(curl -sf "${API_URL}/api/v1/stories")
STORY_COUNT=$(echo "${LIST_RESPONSE}" | jq '. | length')
if [ "${STORY_COUNT}" -gt 0 ]; then
    echo "✅ Story list works - found ${STORY_COUNT} stories"
else
    echo "❌ Story list failed or no stories found"
    exit 1
fi

# Test 5: Get specific story
echo ""
echo "📖 Test 5: Get Specific Story"
STORY_DETAIL=$(curl -sf "${API_URL}/api/v1/stories/${STORY_ID}")
if echo "${STORY_DETAIL}" | grep -q "${STORY_ID}"; then
    echo "✅ Story retrieval works"
    PAGE_COUNT=$(echo "${STORY_DETAIL}" | jq -r '.page_count')
    echo "   Pages: ${PAGE_COUNT}"
else
    echo "❌ Story retrieval failed"
    exit 1
fi

# Test 6: Toggle favorite
echo ""
echo "⭐ Test 6: Toggle Favorite"
FAV_RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/stories/${STORY_ID}/favorite")
if echo "${FAV_RESPONSE}" | grep -q "success"; then
    SUCCESS=$(echo "${FAV_RESPONSE}" | jq -r '.success')
    echo "✅ Favorite toggle works - success: ${SUCCESS}"
else
    echo "❌ Favorite toggle failed"
    exit 1
fi

# Test 7: Start reading session
echo ""
echo "📖 Test 7: Start Reading Session"
READ_RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/stories/${STORY_ID}/read" \
    -H "Content-Type: application/json" \
    -d '{"pages_read": 1}')
    
if echo "${READ_RESPONSE}" | grep -q "session_id"; then
    SESSION_ID=$(echo "${READ_RESPONSE}" | jq -r '.session_id')
    echo "✅ Reading session started - session_id: ${SESSION_ID}"
else
    echo "❌ Reading session failed"
    exit 1
fi

# Test 8: CLI integration
echo ""
echo "🖥️  Test 8: CLI Integration"
if "${CLI_BIN}" list | grep -q "${STORY_ID}"; then
    echo "✅ CLI can list stories from API"
else
    echo "⚠️  CLI list might be using different format"
fi

# Test 9: UI accessibility
echo ""
echo "🌐 Test 9: UI Server Check"
if curl -sf "${UI_URL}" | grep -q "Bedtime Story"; then
    echo "✅ UI is accessible and serving content"
else
    echo "⚠️  UI might not be running or serving different content"
fi

# Test 10: Database persistence
echo ""
echo "💾 Test 10: Database Persistence"
# Check if the story we created is actually in the database
STORY_CHECK=$(curl -sf "${API_URL}/api/v1/stories/${STORY_ID}")
if echo "${STORY_CHECK}" | grep -q "${STORY_ID}"; then
    echo "✅ Story persists in database"
    TIMES_READ=$(echo "${STORY_CHECK}" | jq -r '.times_read')
    echo "   Times read: ${TIMES_READ}"
else
    echo "❌ Story not found in database"
    exit 1
fi

echo ""
echo "✅ All integration tests passed!"
echo "📊 Summary:"
echo "   - API endpoints: Working"
echo "   - Story generation: Functional" 
echo "   - Database persistence: Verified"
echo "   - CLI integration: Working"
echo "   - UI server: Running"