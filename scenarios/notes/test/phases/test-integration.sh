#!/usr/bin/env bash
# Integration validation for SmartNotes API and UI workflows
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

scenario_name="$TESTING_PHASE_SCENARIO_NAME"
API_URL=$(testing::connectivity::get_api_url "$scenario_name" || echo "")
UI_URL=$(testing::connectivity::get_ui_url "$scenario_name" || echo "")

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL for $scenario_name"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if [ -z "$UI_URL" ]; then
  testing::phase::add_warning "UI port unavailable; UI checks skipped"
fi

NOTES_CREATED=()

cleanup_notes() {
  for note_id in "${NOTES_CREATED[@]}"; do
    curl -sf -X DELETE "${API_URL}/api/notes/${note_id}" >/dev/null 2>&1 || true
  done
}

testing::phase::register_cleanup cleanup_notes

auth_headers=("Content-Type: application/json")

api_health() {
  local response
  if ! response=$(curl -sf "${API_URL}/health"); then
    return 1
  fi
  echo "$response" | jq -e '.status == "healthy"' >/dev/null 2>&1
}

if api_health; then
  log::success "✅ API health endpoint reports healthy"
  testing::phase::add_test passed
else
  testing::phase::add_error "API health endpoint failed"
  testing::phase::add_test failed
fi

if [ -n "$UI_URL" ]; then
  if curl -sf "$UI_URL/" | grep -q "SmartNotes"; then
    log::success "✅ UI homepage renders"
    testing::phase::add_test passed
  else
    testing::phase::add_error "UI homepage not reachable"
    testing::phase::add_test failed
  fi
fi

create_note() {
  local payload=$1
  local response
  if ! response=$(curl -sf -X POST "${API_URL}/api/notes" -H "Content-Type: application/json" -d "$payload"); then
    return 1
  fi
  local note_id
  note_id=$(echo "$response" | jq -r '.id // empty')
  if [ -z "$note_id" ]; then
    return 1
  fi
  NOTES_CREATED+=("$note_id")
  printf '%s' "$note_id"
}

NOTE_ID=""
if NOTE_ID=$(create_note '{"title":"Integration Note","content":"Created during integration tests","content_type":"markdown"}'); then
  log::success "✅ Note created (${NOTE_ID})"
  testing::phase::add_test passed
else
  testing::phase::add_error "Failed to create note"
  testing::phase::add_test failed
fi

test_endpoint() {
  local method=$1
  local endpoint=$2
  local description=$3
  local data=${4:-}
  local cmd=(curl -sf -X "$method" "${API_URL}${endpoint}")
  if [ -n "$data" ]; then
    cmd+=(-H "Content-Type: application/json" -d "$data")
  fi
  if "${cmd[@]}" >/dev/null; then
    log::success "✅ ${description}"
    testing::phase::add_test passed
    return 0
  fi
  log::error "❌ ${description}"
  testing::phase::add_test failed
  return 1
}

if [ -n "$NOTE_ID" ]; then
  test_endpoint GET "/api/notes/${NOTE_ID}" "Fetch created note"
  test_endpoint PUT "/api/notes/${NOTE_ID}" "Update note" '{"title":"Updated Integration Note","content":"Updated content","content_type":"markdown"}'
fi

test_endpoint GET "/api/notes" "List notes"
test_endpoint GET "/api/folders" "List folders"
test_endpoint GET "/api/tags" "List tags"

delete_note() {
  local note_id=$1
  if curl -sf -X DELETE "${API_URL}/api/notes/${note_id}" >/dev/null; then
    log::success "✅ Delete note"
    testing::phase::add_test passed
    return 0
  fi
  log::error "❌ Delete note"
  testing::phase::add_test failed
  return 1
}

if [ -n "$NOTE_ID" ]; then
  delete_note "$NOTE_ID" || true
  NOTES_CREATED=()
fi

search_payload='{"query":"integration","limit":5}'
if curl -sf -X POST "${API_URL}/api/search" -H "Content-Type: application/json" -d "$search_payload" >/dev/null; then
  log::success "✅ Text search responds"
  testing::phase::add_test passed
else
  log::error "❌ Text search endpoint failed"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Integration tests completed"
