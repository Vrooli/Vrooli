#!/usr/bin/env bash
# API tests for study-buddy scenario

set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Load environment
if [ -f "$SCENARIO_DIR/.env" ]; then
    export $(grep -v '^#' "$SCENARIO_DIR/.env" | xargs)
fi

# Get actual running ports dynamically
PORTS_JSON=$(vrooli scenario port study-buddy --json 2>/dev/null || echo '{}')
API_PORT=$(echo "$PORTS_JSON" | jq -r '.ports[]? | select(.key=="API_PORT") | .port' 2>/dev/null || echo "")

# Fallback to environment or defaults
API_PORT="${API_PORT:-${API_PORT_ENV:-18775}}"
API_URL="http://localhost:$API_PORT"

echo "🔌 Running API tests..."

# Test flashcard generation (P0 requirement)
echo -n "Testing flashcard generation endpoint... "
response=$(curl -sf -X POST "$API_URL/api/flashcards/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "subject_id": "test-subject",
        "content": "The mitochondria is the powerhouse of the cell",
        "card_count": 3,
        "difficulty_level": "intermediate"
    }' 2>/dev/null || echo '{"error": "Failed"}')

if echo "$response" | grep -q '"cards"'; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    echo "Response: $response"
    exit 1
fi

# Test due cards retrieval (P0 requirement)
echo -n "Testing due cards retrieval... "
response=$(curl -sf "$API_URL/api/study/due-cards?user_id=test-user&limit=10" 2>/dev/null || echo '{"error": "Failed"}')

if echo "$response" | grep -q '"cards"'; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    echo "Response: $response"
    exit 1
fi

# Test study session start (P0 requirement)
echo -n "Testing study session start... "
response=$(curl -sf -X POST "$API_URL/api/study/session/start" \
    -H "Content-Type: application/json" \
    -d '{
        "user_id": "test-user",
        "subject_id": "test-subject",
        "session_type": "review",
        "target_duration": 25
    }' 2>/dev/null || echo '{"error": "Failed"}')

if echo "$response" | grep -q '"session_id"'; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    echo "Response: $response"
    exit 1
fi

# Test flashcard answer submission (P0 requirement)
echo -n "Testing flashcard answer submission... "
response=$(curl -sf -X POST "$API_URL/api/study/answer" \
    -H "Content-Type: application/json" \
    -d '{
        "session_id": "test-session",
        "flashcard_id": "test-card",
        "user_response": "good",
        "response_time": 5000,
        "user_id": "test-user"
    }' 2>/dev/null || echo '{"error": "Failed"}')

if echo "$response" | grep -q '"xp_earned"'; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    echo "Response: $response"
    exit 1
fi

# Test study materials endpoint
echo -n "Testing study materials endpoint... "
if curl -sf "$API_URL/api/study/materials" | grep -q "materials"; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    exit 1
fi

echo "✅ All API tests passed!"
exit 0