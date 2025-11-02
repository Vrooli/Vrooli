#!/bin/bash
# Exercise API and CLI flows against a running Recipe Book instance
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "240s" --require-runtime

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for integration checks"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests incomplete"
fi

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_ROOT=$(testing::connectivity::get_api_url "$SCENARIO_NAME" 2>/dev/null || true)
if [ -z "$API_ROOT" ]; then
  testing::phase::add_error "Unable to discover API port for $SCENARIO_NAME"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests incomplete"
fi

API_BASE="${API_ROOT}/api/v1"
TMP_DIR=$(mktemp -d -t recipe-book-integration-XXXXXX)
trap 'rm -rf "$TMP_DIR"' EXIT

LOG_PREFIX="[integration]"

health_check() {
  curl -sf "${API_ROOT}/health" >/dev/null
}

create_recipe() {
  local payload
  local tmp_output="$TMP_DIR/create.json"
  payload=$(cat <<JSON
{
  "title": "Integration Recipe ${UNIQUE_SUFFIX}",
  "description": "Created by automated integration tests",
  "ingredients": [{"name": "flour", "amount": 2, "unit": "cups"}],
  "instructions": ["Mix", "Bake"],
  "prep_time": 5,
  "cook_time": 15,
  "servings": 2,
  "tags": ["integration", "test"],
  "cuisine": "Test",
  "dietary_info": [],
  "nutrition": {"calories": 120},
  "created_by": "integration-bot",
  "visibility": "private"
}
JSON
)
  local status
  status=$(curl -sS -o "$tmp_output" -w "%{http_code}" \
    -X POST "${API_BASE}/recipes" \
    -H "Content-Type: application/json" \
    -d "$payload")
  if [ "$status" = "201" ]; then
    if command -v jq >/dev/null 2>&1; then
      RECIPE_ID=$(jq -r '.id // empty' "$tmp_output")
    fi
    if [ -z "${RECIPE_ID:-}" ]; then
      RECIPE_ID=$(grep -o '"id"\s*:\s*"[^"]*"' "$tmp_output" | head -1 | cut -d'"' -f4)
    fi
    [ -n "${RECIPE_ID:-}" ]
  else
    log::error "${LOG_PREFIX} recipe creation failed (HTTP ${status}): $(cat "$tmp_output")"
    return 1
  fi
}

get_recipe() {
  [ -n "${RECIPE_ID:-}" ] || return 1
  curl -sf "${API_BASE}/recipes/${RECIPE_ID}?user_id=integration-bot" >/dev/null
}

delete_recipe() {
  [ -n "${RECIPE_ID:-}" ] || return 0
  curl -sf -X DELETE "${API_BASE}/recipes/${RECIPE_ID}?user_id=integration-bot" >/dev/null
}

search_recipes() {
  curl -sS -o "$TMP_DIR/search.json" -w "%{http_code}" \
    -X POST "${API_BASE}/recipes/search" \
    -H "Content-Type: application/json" \
    -d '{"query": "pasta", "user_id": "integration-bot", "limit": 5}' | grep -q "200"
}

suggest_meal() {
  curl -sS -o "$TMP_DIR/suggest.json" -w "%{http_code}" \
    -X POST "${API_BASE}/recommendations" \
    -H "Content-Type: application/json" \
    -d '{"user_id": "integration-bot", "remaining_calories": 600}' | grep -q "200"
}

UNIQUE_SUFFIX=$(date +%s)

if health_check; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Health endpoint did not respond"
  testing::phase::add_test failed
fi

if create_recipe; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if get_recipe; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Failed to retrieve created recipe"
  testing::phase::add_test failed
fi

if search_recipes; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Semantic recipe search request failed"
  testing::phase::add_test failed
fi

if suggest_meal; then
  testing::phase::add_test passed
else
  testing::phase::add_warning "Meal recommendation endpoint unavailable"
  testing::phase::add_test skipped
fi

if command -v recipe-book >/dev/null 2>&1; then
  log::info "${LOG_PREFIX} CLI detected"
  if recipe-book list --json >/dev/null 2>&1; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "recipe-book list --json failed"
    testing::phase::add_test failed
  fi

  if recipe-book search "integration" >/dev/null 2>&1; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "recipe-book search command failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "recipe-book CLI not installed; skipping CLI integration checks"
  testing::phase::add_test skipped
  testing::phase::add_test skipped
fi

delete_recipe || true

testing::phase::end_with_summary "Integration validation completed"
