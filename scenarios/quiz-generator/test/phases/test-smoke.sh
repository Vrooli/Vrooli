#!/bin/bash
set -euo pipefail

# Smoke tests for quiz-generator scenario
# Tests basic functionality to ensure the system is working

echo "🔥 Running smoke tests for quiz-generator..."

# Use ports from environment (set by lifecycle) with fallbacks for standalone testing
API_PORT="${API_PORT:-16470}"
UI_PORT="${UI_PORT:-3251}"

API_URL="http://localhost:${API_PORT}"
UI_URL="http://localhost:${UI_PORT}"

echo "Testing against API: ${API_URL}"

# Test 1: Health check
echo -n "Testing health endpoint... "
if curl -sf "${API_URL}/api/health" > /dev/null; then
    echo "✅ PASS"
else
    echo "❌ FAIL: Health endpoint not responding"
    exit 1
fi

# Test 2: Generate quiz endpoint (with longer timeout for Ollama LLM)
echo -n "Testing quiz generation... "
QUIZ_RESPONSE=$(curl -sf --max-time 60 -X POST "${API_URL}/api/v1/quiz/generate" \
    -H "Content-Type: application/json" \
    -d '{"content": "The sun is a star.", "question_count": 2}' || echo "FAILED")

if [[ "$QUIZ_RESPONSE" == "FAILED" ]]; then
    echo "❌ FAIL: Quiz generation failed"
    exit 1
fi

QUIZ_ID=$(echo "$QUIZ_RESPONSE" | jq -r '.quiz_id // empty')
if [[ -n "$QUIZ_ID" ]]; then
    echo "✅ PASS (Quiz ID: ${QUIZ_ID})"
else
    echo "❌ FAIL: No quiz ID returned"
    exit 1
fi

# Test 3: Create manual quiz
echo -n "Testing manual quiz creation... "
CREATE_RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/quiz" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Test Quiz",
        "questions": [{
            "type": "mcq",
            "question": "What is 1+1?",
            "options": ["1", "2", "3", "4"],
            "correct_answer": "2",
            "difficulty": "easy",
            "points": 1
        }],
        "passing_score": 50
    }' || echo "FAILED")

if [[ "$CREATE_RESPONSE" == "FAILED" ]]; then
    echo "❌ FAIL: Manual quiz creation failed"
    exit 1
fi

MANUAL_QUIZ_ID=$(echo "$CREATE_RESPONSE" | jq -r '.id // empty')
if [[ -n "$MANUAL_QUIZ_ID" ]]; then
    echo "✅ PASS (Quiz ID: ${MANUAL_QUIZ_ID})"
else
    echo "❌ FAIL: No quiz ID returned"
    exit 1
fi

# Test 4: Get quiz
echo -n "Testing quiz retrieval... "
if curl -sf "${API_URL}/api/v1/quiz/${MANUAL_QUIZ_ID}" > /dev/null; then
    echo "✅ PASS"
else
    echo "❌ FAIL: Quiz retrieval failed"
    exit 1
fi

# Test 5: Export quiz as JSON
echo -n "Testing JSON export... "
EXPORT_RESPONSE=$(curl -sf "${API_URL}/api/v1/quiz/${MANUAL_QUIZ_ID}/export?format=json" || echo "FAILED")

if [[ "$EXPORT_RESPONSE" == "FAILED" ]]; then
    echo "❌ FAIL: Export failed"
    exit 1
fi

if echo "$EXPORT_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    echo "✅ PASS"
else
    echo "❌ FAIL: Invalid export format"
    exit 1
fi

# Test 6: CLI availability
echo -n "Testing CLI availability... "
if command -v quiz-generator &> /dev/null; then
    echo "✅ PASS"
else
    echo "❌ FAIL: CLI not found in PATH"
    exit 1
fi

# Test 7: UI availability
echo -n "Testing UI availability... "
if curl -sf "${UI_URL}" | grep -q 'root'; then
    echo "✅ PASS"
else
    echo "❌ FAIL: UI not responding"
    exit 1
fi

echo ""
echo "✅ All smoke tests passed!"
exit 0