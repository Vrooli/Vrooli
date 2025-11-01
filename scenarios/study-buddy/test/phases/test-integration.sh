#!/bin/bash
# Scenario integration tests validating core API workflows that require the runtime.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# connectivity.sh is sourced by phase::init, giving us helper functions for ports/URLs.
testing::phase::init --target-time "180s" --require-runtime

API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME")
if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL for integration tests"
  testing::phase::end_with_summary "Integration tests aborted"
fi

validate_flashcard_creation() {
  local payload card_id
  payload=$(jq -n \
    --arg user "itest-$(date +%s)" \
    '{
      user_id: $user,
      subject_id: "integration-subject",
      front: "What is spaced repetition?",
      back: "A technique that increases intervals between reviews.",
      difficulty: 2
    }')

  card_id=$(curl -sf -X POST "$API_URL/api/flashcards" \
    -H "Content-Type: application/json" \
    -d "$payload" | jq -r '.id // empty' 2>/dev/null)

  [[ -n "$card_id" ]]
}

validate_xp_tracking() {
  local payload response
  payload=$(jq -n '{
      session_id: "itest-session",
      flashcard_id: "itest-card",
      user_response: "good",
      response_time: 4200,
      user_id: "itest-user"
    }')

  response=$(curl -sf -X POST "$API_URL/api/study/answer" \
    -H "Content-Type: application/json" \
    -d "$payload" 2>/dev/null || true)

  echo "$response" | jq -e '.xp_earned' >/dev/null 2>&1
}

validate_study_session_stats() {
  curl -sf "$API_URL/api/sessions/itest-user/stats" >/dev/null
}

validate_subject_creation() {
  local payload subject_id
  payload=$(jq -n '{
      user_id: "itest-user",
      name: "Integration Biology",
      description: "Cell structure and genetics",
      color: "#8B5A96"
    }')

  subject_id=$(curl -sf -X POST "$API_URL/api/subjects" \
    -H "Content-Type: application/json" \
    -d "$payload" | jq -r '.id // empty' 2>/dev/null)

  [[ -n "$subject_id" ]]
}

# jq is required for the JSON assertions above.
if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_error "jq is required for integration assertions"
  testing::phase::end_with_summary "Integration tests cannot run without jq"
fi

testing::phase::check "Create flashcard via API" validate_flashcard_creation
testing::phase::check "XP tracking endpoint" validate_xp_tracking
testing::phase::check "Study session statistics" validate_study_session_stats
testing::phase::check "Subject creation endpoint" validate_subject_creation

testing::phase::end_with_summary "Integration tests completed"
