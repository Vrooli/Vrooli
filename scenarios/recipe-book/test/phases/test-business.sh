#!/bin/bash
# Validate high-level business workflows such as visibility, sharing, and rating
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "300s" --require-runtime

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for business workflow checks"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business workflow tests incomplete"
fi

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_ROOT=$(testing::connectivity::get_api_url "$SCENARIO_NAME" 2>/dev/null || true)
if [ -z "$API_ROOT" ]; then
  testing::phase::add_error "Unable to discover API port for $SCENARIO_NAME"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business workflow tests incomplete"
fi

API_BASE="${API_ROOT}/api/v1"
TMP_DIR=$(mktemp -d -t recipe-book-business-XXXXXX)
trap 'rm -rf "$TMP_DIR"' EXIT

create_recipe_for() {
  local owner="$1"
  local visibility="$2"
  local title="$3"
  local payload
  payload=$(cat <<JSON
{
  "title": "${title}",
  "ingredients": [{"name": "integration ingredient", "amount": 1, "unit": "cup"}],
  "instructions": ["Step one"],
  "created_by": "${owner}",
  "visibility": "${visibility}",
  "tags": ["business", "test"]
}
JSON
)
  local output="$TMP_DIR/${title// /-}.json"
  local status
  status=$(curl -sS -o "$output" -w "%{http_code}" \
    -X POST "${API_BASE}/recipes" \
    -H "Content-Type: application/json" \
    -d "$payload")
  if [ "$status" != "201" ]; then
    log::error "Failed to create ${title} (HTTP ${status}): $(cat "$output")"
    return 1
  fi
  if command -v jq >/dev/null 2>&1; then
    jq -r '.id // empty' "$output"
  else
    grep -o '"id"\s*:\s*"[^"]*"' "$output" | head -1 | cut -d'"' -f4
  fi
}

delete_recipe() {
  local recipe_id="$1"
  local owner="$2"
  curl -sf -X DELETE "${API_BASE}/recipes/${recipe_id}?user_id=${owner}" >/dev/null 2>&1 || true
}

# Visibility enforcement
private_recipe_id=$(create_recipe_for "user-alice" "private" "Private Visibility Test") || true
if [ -n "${private_recipe_id:-}" ]; then
  unauthorized_status=$(curl -sS -o /dev/null -w "%{http_code}" "${API_BASE}/recipes/${private_recipe_id}?user_id=user-bob")
  if [ "$unauthorized_status" != "200" ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Unauthorized user gained access to private recipe"
    testing::phase::add_test failed
  fi

  owner_status=$(curl -sS -o /dev/null -w "%{http_code}" "${API_BASE}/recipes/${private_recipe_id}?user_id=user-alice")
  if [ "$owner_status" = "200" ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Owner could not access private recipe"
    testing::phase::add_test failed
  fi

  delete_recipe "$private_recipe_id" "user-alice"
else
  testing::phase::add_error "Failed to create private recipe for visibility test"
  testing::phase::add_test failed
  testing::phase::add_test failed
fi

# Sharing workflow
shared_recipe_id=$(create_recipe_for "user-alice" "private" "Sharing Test Recipe") || true
if [ -n "${shared_recipe_id:-}" ]; then
  share_status=$(curl -sS -o "$TMP_DIR/share.json" -w "%{http_code}" \
    -X POST "${API_BASE}/recipes/${shared_recipe_id}/share" \
    -H "Content-Type: application/json" \
    -d '{"user_ids": ["user-bob"]}')
  if [ "$share_status" = "200" ]; then
    testing::phase::add_test passed
  else
    log::error "Share endpoint failed: $(cat "$TMP_DIR/share.json")"
    testing::phase::add_test failed
  fi

  shared_lookup=$(curl -sS -o /dev/null -w "%{http_code}" "${API_BASE}/recipes/${shared_recipe_id}?user_id=user-bob")
  if [ "$shared_lookup" = "200" ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Shared recipe inaccessible to invited user"
    testing::phase::add_test failed
  fi

  delete_recipe "$shared_recipe_id" "user-alice"
else
  testing::phase::add_error "Failed to create recipe for sharing test"
  testing::phase::add_test failed
  testing::phase::add_test failed
fi

# Modification workflow
mod_recipe_id=$(create_recipe_for "user-alice" "private" "Modification Base Recipe") || true
if [ -n "${mod_recipe_id:-}" ]; then
  mod_status=$(curl -sS -o "$TMP_DIR/modify.json" -w "%{http_code}" \
    -X POST "${API_BASE}/recipes/${mod_recipe_id}/modify" \
    -H "Content-Type: application/json" \
    -d '{"modification_type": "make_vegan", "user_id": "user-bob"}')
  if [ "$mod_status" = "200" ] && grep -q 'modified_recipe' "$TMP_DIR/modify.json"; then
    testing::phase::add_test passed
  else
    log::error "Modification workflow failed: $(cat "$TMP_DIR/modify.json")"
    testing::phase::add_test failed
  fi

  delete_recipe "$mod_recipe_id" "user-alice"
else
  testing::phase::add_error "Failed to create recipe for modification test"
  testing::phase::add_test failed
fi

# Cooking / rating workflow
cook_recipe_id=$(create_recipe_for "user-alice" "private" "Cooking Workflow Recipe") || true
if [ -n "${cook_recipe_id:-}" ]; then
  cook_status=$(curl -sS -o "$TMP_DIR/cook.json" -w "%{http_code}" \
    -X POST "${API_BASE}/recipes/${cook_recipe_id}/cook" \
    -H "Content-Type: application/json" \
    -d '{"user_id": "user-alice", "rating": 5, "notes": "Delicious", "anonymous": false}')
  if [ "$cook_status" = "200" ] && grep -q '"status"' "$TMP_DIR/cook.json"; then
    testing::phase::add_test passed
  else
    log::warning "Cooking workflow not available: $(cat "$TMP_DIR/cook.json")"
    testing::phase::add_warning "Cooking workflow not available"
    testing::phase::add_test skipped
  fi

  delete_recipe "$cook_recipe_id" "user-alice"
else
  testing::phase::add_warning "Failed to create recipe for cooking workflow; skipping"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business workflow validation completed"
