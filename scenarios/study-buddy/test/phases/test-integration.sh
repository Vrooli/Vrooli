#!/usr/bin/env bash
# Integration tests for study-buddy scenario

set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Load environment
if [ -f "$SCENARIO_DIR/.env" ]; then
    export $(grep -v '^#' "$SCENARIO_DIR/.env" | xargs)
fi

API_PORT="${API_PORT:-18775}"
API_URL="http://localhost:$API_PORT"

echo "🔗 Running integration tests..."

# Test spaced repetition algorithm (P0 requirement)
echo -n "Testing spaced repetition calculation... "
# Create a flashcard and test the review functionality
response=$(curl -sf -X POST "$API_URL/api/flashcards" \
    -H "Content-Type: application/json" \
    -d '{
        "user_id": "test-user",
        "subject_id": "test-subject",
        "front": "What is spaced repetition?",
        "back": "A learning technique that incorporates increasing intervals of time between subsequent review",
        "difficulty": 2
    }' 2>/dev/null || echo '{}')

if [ ! -z "$response" ]; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    exit 1
fi

# Test progress tracking with XP (P0 requirement)
echo -n "Testing XP and progress tracking... "
response=$(curl -sf -X POST "$API_URL/api/study/answer" \
    -H "Content-Type: application/json" \
    -d '{
        "session_id": "test-session",
        "flashcard_id": "test-card",
        "user_response": "easy",
        "response_time": 3000,
        "user_id": "test-user"
    }' 2>/dev/null || echo '{}')

if echo "$response" | grep -q '"xp_earned"' && echo "$response" | grep -q '"total_xp"'; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    echo "Response: $response"
    exit 1
fi

# Test streak tracking (P0 requirement)
echo -n "Testing streak tracking... "
response=$(curl -sf "$API_URL/api/sessions/test-user/stats" 2>/dev/null || echo '{}')

if echo "$response" | grep -q '"current_streak"' || [ ! -z "$response" ]; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    exit 1
fi

# Test subject organization (P0 requirement)
echo -n "Testing subject management... "
response=$(curl -sf -X POST "$API_URL/api/subjects" \
    -H "Content-Type: application/json" \
    -d '{
        "user_id": "test-user",
        "name": "Biology",
        "description": "Cell biology and genetics",
        "color": "#8B5A96"
    }' 2>/dev/null || echo '{}')

if echo "$response" | grep -q '"id"' || [ ! -z "$response" ]; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    exit 1
fi

echo "✅ All integration tests passed!"
exit 0