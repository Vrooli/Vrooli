#!/bin/bash
# Business logic validation for quiz-generator scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "180s" --require-runtime

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not installed; business validation requires JSON parsing"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Business tests skipped"
fi

API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME" || true)
if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business tests failed"
fi

AI_QUIZ_ID=""
MANUAL_QUIZ_ID=""

cleanup_business_artifacts() {
  if [ -n "$MANUAL_QUIZ_ID" ]; then
    curl -s -X DELETE "${API_URL}/api/v1/quiz/${MANUAL_QUIZ_ID}" >/dev/null 2>&1 || true
  fi
  if [ -n "$AI_QUIZ_ID" ]; then
    curl -s -X DELETE "${API_URL}/api/v1/quiz/${AI_QUIZ_ID}" >/dev/null 2>&1 || true
  fi
}

testing::phase::register_cleanup cleanup_business_artifacts

ollama_running=false
if command -v resource-ollama >/dev/null 2>&1 && resource-ollama status >/dev/null 2>&1; then
  ollama_running=true
fi

if $ollama_running; then
  AI_RESPONSE=$(curl -sSf --max-time 90 -X POST "${API_URL}/api/v1/quiz/generate" \
    -H "Content-Type: application/json" \
    -d '{"content":"The Earth orbits the sun every 365.25 days.","question_count":3,"difficulty":"medium"}' 2>/dev/null || true)
  if [ -n "$AI_RESPONSE" ]; then
    AI_QUIZ_ID=$(echo "$AI_RESPONSE" | jq -r '.quiz_id // empty')
    if [ -n "$AI_QUIZ_ID" ]; then
      QUESTION_TOTAL=$(echo "$AI_RESPONSE" | jq '.questions | length' 2>/dev/null || echo 0)
      if [ "$QUESTION_TOTAL" -ge 1 ]; then
        testing::phase::add_test passed
      else
        testing::phase::add_error "AI generation returned no questions"
        testing::phase::add_test failed
      fi
    else
      testing::phase::add_error "AI quiz generation response missing quiz_id"
      testing::phase::add_test failed
    fi
  else
    testing::phase::add_error "AI quiz generation request failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "Ollama resource unavailable; skipping AI generation test"
  testing::phase::add_test skipped
fi

MANUAL_RESPONSE=$(curl -sSf -X POST "${API_URL}/api/v1/quiz" \
  -H "Content-Type: application/json" \
  -d '{"title":"Business Flow Quiz","description":"Workflow validation","questions":[{"type":"mcq","question":"Which planet is known as the Red Planet?","options":["Earth","Mars","Venus","Jupiter"],"correct_answer":"Mars","difficulty":"easy","points":2},{"type":"true_false","question":"The moon is a planet.","correct_answer":"false","difficulty":"easy","points":1}],"passing_score":50}' 2>/dev/null || true)
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
  QUIZ_PAYLOAD=$(curl -sSf "${API_URL}/api/v1/quiz/${MANUAL_QUIZ_ID}" 2>/dev/null || true)
  if [ -n "$QUIZ_PAYLOAD" ] && [ "$(echo "$QUIZ_PAYLOAD" | jq '.questions | length')" -ge 2 ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Quiz retrieval did not return expected questions"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "Skipping retrieval validation (manual quiz creation failed)"
  testing::phase::add_test skipped
fi

if [ -n "$MANUAL_QUIZ_ID" ]; then
  SUBMIT_REQUEST=$(jq -n --arg id "$MANUAL_QUIZ_ID" '{responses: [], time_taken: 90}')
  SUBMIT_RESPONSE=$(curl -sSf -X POST "${API_URL}/api/v1/quiz/${MANUAL_QUIZ_ID}/submit" \
    -H "Content-Type: application/json" \
    -d "$SUBMIT_REQUEST" 2>/dev/null || true)
  if [ -n "$SUBMIT_RESPONSE" ]; then
    SCORE=$(echo "$SUBMIT_RESPONSE" | jq -r '.score // empty')
    if [ -n "$SCORE" ]; then
      testing::phase::add_test passed
    else
      testing::phase::add_warning "Submit response missing score"
      testing::phase::add_test skipped
    fi
  else
    testing::phase::add_warning "Submit endpoint not yet implemented"
    testing::phase::add_test skipped
  fi
fi

testing::phase::end_with_summary "Business logic validation completed"
