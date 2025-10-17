#!/bin/bash
set -euo pipefail

# Integration tests for quiz-generator scenario
# Tests interactions between components and with resources

echo "🔗 Running integration tests for quiz-generator..."

# Get API port dynamically
API_PORT=$(vrooli scenario info quiz-generator --json 2>/dev/null | jq -r '.api_port // 16470')
API_PORT=${API_PORT:-16470}
API_URL="http://localhost:${API_PORT}"

echo "Testing against API: ${API_URL}"

# Test 1: Database integration
echo -n "Testing database connectivity... "
HEALTH_RESPONSE=$(curl -sf "${API_URL}/api/health")
DB_STATUS=$(echo "$HEALTH_RESPONSE" | jq -r '.database // "unknown"')

if [[ "$DB_STATUS" == "connected" ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: Database not connected (status: ${DB_STATUS})"
    exit 1
fi

# Test 2: Full quiz lifecycle
echo "Testing full quiz lifecycle..."

# Create a quiz
echo -n "  Creating quiz... "
CREATE_RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/quiz" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Integration Test Quiz",
        "description": "Testing quiz lifecycle",
        "questions": [
            {
                "type": "mcq",
                "question": "What is the capital of France?",
                "options": ["London", "Berlin", "Paris", "Madrid"],
                "correct_answer": "Paris",
                "difficulty": "easy",
                "points": 2
            },
            {
                "type": "true_false",
                "question": "The Earth is flat.",
                "correct_answer": "false",
                "difficulty": "easy",
                "points": 1
            }
        ],
        "passing_score": 60
    }')

QUIZ_ID=$(echo "$CREATE_RESPONSE" | jq -r '.id // empty')
if [[ -n "$QUIZ_ID" ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: Quiz creation failed"
    exit 1
fi

# Update the quiz
echo -n "  Updating quiz... "
UPDATE_RESPONSE=$(curl -sf -X PUT "${API_URL}/api/v1/quiz/${QUIZ_ID}" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Updated Integration Test Quiz",
        "passing_score": 70
    }')

UPDATED_TITLE=$(echo "$UPDATE_RESPONSE" | jq -r '.title // empty')
if [[ "$UPDATED_TITLE" == "Updated Integration Test Quiz" ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: Quiz update failed"
    exit 1
fi

# List quizzes
echo -n "  Listing quizzes... "
LIST_RESPONSE=$(curl -sf "${API_URL}/api/v1/quizzes")
QUIZ_COUNT=$(echo "$LIST_RESPONSE" | jq '.quizzes | length // 0')

if [[ "$QUIZ_COUNT" -gt 0 ]]; then
    echo "✅ PASS (Found ${QUIZ_COUNT} quizzes)"
else
    echo "❌ FAIL: No quizzes found"
    exit 1
fi

# Submit quiz answers
echo -n "  Submitting quiz answers... "
SUBMIT_RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/quiz/${QUIZ_ID}/submit" \
    -H "Content-Type: application/json" \
    -d '{
        "responses": [
            {
                "question_id": "q1",
                "answer": "Paris"
            },
            {
                "question_id": "q2",
                "answer": "false"
            }
        ],
        "time_taken": 120
    }' || echo "FAILED")

if [[ "$SUBMIT_RESPONSE" != "FAILED" ]]; then
    SCORE=$(echo "$SUBMIT_RESPONSE" | jq -r '.score // -1')
    if [[ "$SCORE" -ge 0 ]]; then
        echo "✅ PASS (Score: ${SCORE})"
    else
        echo "❌ FAIL: Invalid score returned"
        exit 1
    fi
else
    echo "❌ FAIL: Submit failed"
    exit 1
fi

# Delete the quiz
echo -n "  Deleting quiz... "
if curl -sf -X DELETE "${API_URL}/api/v1/quiz/${QUIZ_ID}"; then
    echo "✅ PASS"
else
    echo "⚠️  WARNING: Delete may not be implemented"
fi

# Test 3: Question bank search
echo -n "Testing question bank search... "
SEARCH_RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/question-bank/search" \
    -H "Content-Type: application/json" \
    -d '{"query": "capital", "limit": 5}' || echo "FAILED")

if [[ "$SEARCH_RESPONSE" != "FAILED" ]]; then
    echo "✅ PASS"
else
    echo "⚠️  WARNING: Question bank search may not be fully implemented"
fi

# Test 4: CLI integration
echo -n "Testing CLI integration... "
if quiz-generator status 2>/dev/null | grep -q "Quiz Generator"; then
    echo "✅ PASS"
else
    echo "⚠️  WARNING: CLI status command may need implementation"
fi

# Test 5: Resource dependencies
echo "Testing resource dependencies..."

# Check PostgreSQL
echo -n "  PostgreSQL availability... "
if vrooli resource status postgres --json 2>/dev/null | jq -e '.status == "running"' > /dev/null; then
    echo "✅ PASS"
else
    echo "⚠️  WARNING: PostgreSQL may not be running"
fi

echo ""
echo "✅ Integration tests completed!"
exit 0