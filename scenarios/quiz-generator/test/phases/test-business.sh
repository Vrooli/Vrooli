#!/bin/bash
# Business logic testing phase for quiz-generator scenario
# Tests core business requirements and user workflows

set -euo pipefail

# Determine APP_ROOT
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with target time
testing::phase::init --target-time "180s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# Get API port from environment or fallback
API_PORT="${API_PORT:-16470}"

echo "🎓 Testing Quiz Generator Business Logic..."
echo ""

# Test 1: Quiz generation from content
echo "Test 1: Quiz generation from content"
RESPONSE=$(curl -sf http://localhost:${API_PORT}/api/v1/quiz/generate \
  -H "Content-Type: application/json" \
  -d '{
    "content": "The Earth orbits the Sun once every 365.25 days. This period is called a year.",
    "question_count": 3,
    "difficulty": "medium"
  }' || echo "FAILED")

if echo "$RESPONSE" | grep -q "quiz_id"; then
  echo "✅ Quiz generation works"
else
  echo "❌ Quiz generation failed"
  testing::phase::end_with_summary "Business logic test failed"
  exit 1
fi

# Test 2: Quiz retrieval
QUIZ_ID=$(echo "$RESPONSE" | jq -r '.quiz_id' 2>/dev/null || echo "")
if [ -n "$QUIZ_ID" ] && [ "$QUIZ_ID" != "null" ]; then
  echo "Test 2: Quiz retrieval"
  QUIZ=$(curl -sf http://localhost:${API_PORT}/api/v1/quiz/${QUIZ_ID} || echo "FAILED")

  if echo "$QUIZ" | grep -q "questions"; then
    echo "✅ Quiz retrieval works"
  else
    echo "❌ Quiz retrieval failed"
    testing::phase::end_with_summary "Quiz retrieval test failed"
    exit 1
  fi
else
  echo "⚠️  Skipping quiz retrieval test (no quiz ID)"
fi

# Test 3: Question quality validation
echo "Test 3: Question quality validation"
QUESTION_COUNT=$(echo "$QUIZ" | jq '.questions | length' 2>/dev/null || echo "0")
if [ "$QUESTION_COUNT" -ge 1 ]; then
  echo "✅ Generated $QUESTION_COUNT questions"
else
  echo "❌ No questions generated"
  testing::phase::end_with_summary "Question generation quality test failed"
  exit 1
fi

echo ""
echo "✅ All business logic tests passed"

# End phase with summary
testing::phase::end_with_summary "Business logic tests completed successfully"
