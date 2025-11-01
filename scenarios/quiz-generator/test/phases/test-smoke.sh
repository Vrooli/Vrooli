#!/bin/bash
# Smoke tests for quiz-generator scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "90s" --require-runtime

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not installed; skipping smoke tests that require JSON parsing"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Smoke tests skipped"
fi

API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME" || true)
UI_URL=$(testing::connectivity::get_ui_url "$TESTING_PHASE_SCENARIO_NAME" || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to determine API URL"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Smoke tests failed"
fi

QUIZ_ID=""
MANUAL_QUIZ_ID=""

testing::phase::register_cleanup cleanup_smoke_artifacts
cleanup_smoke_artifacts() {
  if [ -n "$MANUAL_QUIZ_ID" ]; then
    curl -s -X DELETE "${API_URL}/api/v1/quiz/${MANUAL_QUIZ_ID}" >/dev/null 2>&1 || true
  fi
  if [ -n "$QUIZ_ID" ]; then
    curl -s -X DELETE "${API_URL}/api/v1/quiz/${QUIZ_ID}" >/dev/null 2>&1 || true
  fi
}

if testing::phase::check "Health endpoint responds" curl -sf "${API_URL}/api/health"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

QUIZ_RESPONSE=$(curl -sSf --max-time 60 -X POST "${API_URL}/api/v1/quiz/generate" \
  -H "Content-Type: application/json" \
  -d '{"content":"The sun is a star.","question_count":2}' 2>/dev/null || true)
if [ -n "$QUIZ_RESPONSE" ] && QUIZ_ID=$(echo "$QUIZ_RESPONSE" | jq -r '.quiz_id // empty'); then
  if [ -n "$QUIZ_ID" ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Quiz generation response missing quiz_id"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_error "Quiz generation request failed"
  testing::phase::add_test failed
fi

MANUAL_RESPONSE=$(curl -sSf -X POST "${API_URL}/api/v1/quiz" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test Quiz","questions":[{"type":"mcq","question":"What is 1+1?","options":["1","2","3","4"],"correct_answer":"2","difficulty":"easy","points":1}],"passing_score":50}' 2>/dev/null || true)
if [ -n "$MANUAL_RESPONSE" ]; then
  MANUAL_QUIZ_ID=$(echo "$MANUAL_RESPONSE" | jq -r '.id // empty')
  if [ -n "$MANUAL_QUIZ_ID" ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Manual quiz creation response missing id"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_error "Manual quiz creation failed"
  testing::phase::add_test failed
fi

if [ -n "$MANUAL_QUIZ_ID" ]; then
  if testing::phase::check "Retrieve created quiz" curl -sf "${API_URL}/api/v1/quiz/${MANUAL_QUIZ_ID}"; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "Skipping retrieval test (no manual quiz id)"
  testing::phase::add_test skipped
fi

if [ -n "$MANUAL_QUIZ_ID" ]; then
  EXPORT_RESPONSE=$(curl -sSf "${API_URL}/api/v1/quiz/${MANUAL_QUIZ_ID}/export?format=json" 2>/dev/null || true)
  if [ -n "$EXPORT_RESPONSE" ] && echo "$EXPORT_RESPONSE" | jq -e '.id' >/dev/null 2>&1; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "JSON export missing id"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "Skipping export test (no manual quiz id)"
  testing::phase::add_test skipped
fi

if command -v quiz-generator >/dev/null 2>&1; then
  if testing::phase::check "CLI status command" quiz-generator status >/dev/null; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "quiz-generator CLI not available"
  testing::phase::add_test skipped
fi

if [ -n "$UI_URL" ]; then
  if testing::phase::check "UI root available" curl -sf "$UI_URL"; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "Unable to determine UI URL"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Smoke tests completed"
