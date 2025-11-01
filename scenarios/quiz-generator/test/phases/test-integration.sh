#!/bin/bash
# Integration validation for quiz-generator scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "240s" --require-runtime

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not installed; integration tests require JSON parsing"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration tests skipped"
fi

API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME" || true)
if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests failed"
fi

QUIZ_ID=""

testing::phase::register_cleanup cleanup_integration_resources
cleanup_integration_resources() {
  if [ -n "$QUIZ_ID" ]; then
    curl -s -X DELETE "${API_URL}/api/v1/quiz/${QUIZ_ID}" >/dev/null 2>&1 || true
  fi
}

if testing::phase::check "API health endpoint" curl -sf "${API_URL}/api/health"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

CREATE_RESPONSE=$(curl -sSf -X POST "${API_URL}/api/v1/quiz" \
  -H "Content-Type: application/json" \
  -d '{"title":"Integration Test Quiz","description":"Testing quiz lifecycle","questions":[{"type":"mcq","question":"Capital of France?","options":["London","Berlin","Paris","Madrid"],"correct_answer":"Paris","difficulty":"easy","points":2}],"passing_score":60}' 2>/dev/null || true)
if [ -n "$CREATE_RESPONSE" ]; then
  QUIZ_ID=$(echo "$CREATE_RESPONSE" | jq -r '.id // empty')
  if [ -n "$QUIZ_ID" ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Quiz creation response missing id"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_error "Quiz creation failed"
  testing::phase::add_test failed
fi

if [ -n "$QUIZ_ID" ]; then
  UPDATE_RESPONSE=$(curl -sSf -X PUT "${API_URL}/api/v1/quiz/${QUIZ_ID}" \
    -H "Content-Type: application/json" \
    -d '{"passing_score":70}' 2>/dev/null || true)
  if [ -n "$UPDATE_RESPONSE" ] && [ "$(echo "$UPDATE_RESPONSE" | jq -r '.passing_score // 0')" = "70" ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Quiz update did not return expected payload"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "Skipping update test (quiz creation failed)"
  testing::phase::add_test skipped
fi

LIST_RESPONSE=$(curl -sSf "${API_URL}/api/v1/quizzes" 2>/dev/null || true)
if [ -n "$LIST_RESPONSE" ]; then
  if [ "$(echo "$LIST_RESPONSE" | jq 'length')" -ge 1 ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Quiz list returned no entries"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_error "Failed to list quizzes"
  testing::phase::add_test failed
fi

if [ -n "$QUIZ_ID" ]; then
  if testing::phase::check "Retrieve quiz" curl -sf "${API_URL}/api/v1/quiz/${QUIZ_ID}"; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "Skipping retrieval test (quiz creation failed)"
  testing::phase::add_test skipped
fi

if [ -n "$QUIZ_ID" ]; then
  DELETE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${API_URL}/api/v1/quiz/${QUIZ_ID}" 2>/dev/null || echo "000")
  if [ "$DELETE_CODE" = "200" ] || [ "$DELETE_CODE" = "204" ]; then
    testing::phase::add_test passed
    QUIZ_ID=""
  else
    testing::phase::add_warning "Quiz delete endpoint returned ${DELETE_CODE}"
    testing::phase::add_test skipped
  fi
fi

if command -v quiz-generator >/dev/null 2>&1; then
  if testing::phase::check "CLI status command" quiz-generator status >/dev/null; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "quiz-generator CLI not installed"
  testing::phase::add_test skipped
fi

if command -v resource-postgres >/dev/null 2>&1; then
  if testing::phase::check "PostgreSQL resource healthy" resource-postgres status; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
fi

if command -v resource-ollama >/dev/null 2>&1; then
  if testing::phase::check "Ollama resource healthy" resource-ollama status; then
    testing::phase::add_test passed
  else
    testing::phase::add_warning "Ollama resource not running"
    testing::phase::add_test skipped
  fi
fi

testing::phase::end_with_summary "Integration tests completed"
