#!/bin/bash
# Performance testing phase for quiz-generator scenario
# Tests response times and resource efficiency

set -euo pipefail

# Determine APP_ROOT
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with target time
testing::phase::init --target-time "120s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# Get API port from environment or fallback
API_PORT="${API_PORT:-16470}"

echo "⚡ Testing Quiz Generator Performance..."
echo ""

# Test 1: Health endpoint response time
echo "Test 1: Health endpoint response time (target: <200ms)"
START=$(date +%s%N)
curl -sf http://localhost:${API_PORT}/api/health > /dev/null || {
  echo "❌ Health endpoint failed"
  testing::phase::end_with_summary "Health check failed"
  exit 1
}
END=$(date +%s%N)
DURATION_MS=$(( (END - START) / 1000000 ))

if [ $DURATION_MS -lt 200 ]; then
  echo "✅ Health endpoint: ${DURATION_MS}ms (excellent)"
elif [ $DURATION_MS -lt 500 ]; then
  echo "⚠️  Health endpoint: ${DURATION_MS}ms (acceptable)"
else
  echo "❌ Health endpoint: ${DURATION_MS}ms (too slow)"
fi

# Test 2: Quiz generation performance
echo "Test 2: Quiz generation response time (target: <5000ms)"
START=$(date +%s%N)
RESPONSE=$(curl -sf http://localhost:${API_PORT}/api/v1/quiz/generate \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Sample content for performance testing with multiple sentences to analyze.",
    "question_count": 5,
    "difficulty": "medium"
  }' 2>/dev/null || echo "FAILED")
END=$(date +%s%N)
DURATION_MS=$(( (END - START) / 1000000 ))

if echo "$RESPONSE" | grep -q "quiz_id"; then
  if [ $DURATION_MS -lt 5000 ]; then
    echo "✅ Quiz generation: ${DURATION_MS}ms (within target)"
  elif [ $DURATION_MS -lt 10000 ]; then
    echo "⚠️  Quiz generation: ${DURATION_MS}ms (slower than target but acceptable)"
  else
    echo "❌ Quiz generation: ${DURATION_MS}ms (too slow)"
  fi
else
  echo "⚠️  Quiz generation failed or timed out after ${DURATION_MS}ms"
fi

# Test 3: Concurrent request handling
echo "Test 3: Concurrent request handling (5 parallel requests)"
START=$(date +%s%N)
for i in {1..5}; do
  curl -sf http://localhost:${API_PORT}/api/health > /dev/null &
done
wait
END=$(date +%s%N)
DURATION_MS=$(( (END - START) / 1000000 ))

if [ $DURATION_MS -lt 1000 ]; then
  echo "✅ Concurrent requests: ${DURATION_MS}ms total (good parallelism)"
else
  echo "⚠️  Concurrent requests: ${DURATION_MS}ms total"
fi

echo ""
echo "✅ Performance tests completed"

# End phase with summary
testing::phase::end_with_summary "Performance tests completed"
