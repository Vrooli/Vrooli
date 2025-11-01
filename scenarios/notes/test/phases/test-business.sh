#!/usr/bin/env bash
# Business workflow validation for SmartNotes
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

scenario_name="$TESTING_PHASE_SCENARIO_NAME"
API_URL=$(testing::connectivity::get_api_url "$scenario_name" || echo "")

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL"
  testing::phase::end_with_summary "Business tests incomplete"
fi

FOLDERS_CREATED=()
TAGS_CREATED=()
TEMPLATES_CREATED=()

cleanup_business_objects() {
  for folder in "${FOLDERS_CREATED[@]}"; do
    curl -sf -X DELETE "${API_URL}/api/folders/${folder}" >/dev/null 2>&1 || true
  done
  for tag in "${TAGS_CREATED[@]}"; do
    curl -sf -X DELETE "${API_URL}/api/tags/${tag}" >/dev/null 2>&1 || true
  done
  for template in "${TEMPLATES_CREATED[@]}"; do
    curl -sf -X DELETE "${API_URL}/api/templates/${template}" >/dev/null 2>&1 || true
  done
}

testing::phase::register_cleanup cleanup_business_objects

post_json() {
  local endpoint=$1
  local payload=$2
  local list_ref=$3
  local response
  if ! response=$(curl -sf -X POST "${API_URL}${endpoint}" -H "Content-Type: application/json" -d "$payload"); then
    return 1
  fi
  local id
  id=$(echo "$response" | jq -r '.id // empty')
  if [ -z "$id" ]; then
    return 1
  fi
  case "$list_ref" in
    folder) FOLDERS_CREATED+=("$id") ;;
    tag) TAGS_CREATED+=("$id") ;;
    template) TEMPLATES_CREATED+=("$id") ;;
  esac
  printf '%s' "$id"
}

if folder_id=$(post_json "/api/folders" '{"name":"Test Folder","icon":"📂","color":"#6366f1"}' folder); then
  log::success "✅ Folder created (${folder_id})"
  testing::phase::add_test passed
else
  testing::phase::add_error "Failed to create folder"
  testing::phase::add_test failed
fi

test_listing() {
  local endpoint=$1
  local description=$2
  if curl -sf "${API_URL}${endpoint}" >/dev/null; then
    log::success "✅ ${description}"
    testing::phase::add_test passed
  else
    log::error "❌ ${description}"
    testing::phase::add_test failed
  fi
}

test_listing "/api/folders" "List folders"

tag_payload='{"name":"test-suite-tag","color":"#10b981"}'
if tag_id=$(post_json "/api/tags" "$tag_payload" tag); then
  log::success "✅ Tag created (${tag_id})"
  testing::phase::add_test passed
else
  testing::phase::add_error "Failed to create tag"
  testing::phase::add_test failed
fi

test_listing "/api/tags" "List tags"

template_payload='{ "name":"Meeting Notes","description":"Template used during tests","content":"# Meeting Notes","category":"business" }'
if template_id=$(post_json "/api/templates" "$template_payload" template); then
  log::success "✅ Template created (${template_id})"
  testing::phase::add_test passed
else
  testing::phase::add_error "Failed to create template"
  testing::phase::add_test failed
fi

test_listing "/api/templates" "List templates"

text_search_payload='{"query":"meeting","limit":5}'
if curl -sf -X POST "${API_URL}/api/search" -H "Content-Type: application/json" -d "$text_search_payload" >/dev/null; then
  log::success "✅ Text search works"
  testing::phase::add_test passed
else
  log::error "❌ Text search failed"
  testing::phase::add_test failed
fi

if vrooli resource status qdrant >/dev/null 2>&1; then
  semantic_payload='{"query":"collaboration","limit":5}'
  if curl -sf -X POST "${API_URL}/api/search/semantic" -H "Content-Type: application/json" -d "$semantic_payload" >/dev/null; then
    log::success "✅ Semantic search available"
    testing::phase::add_test passed
  else
    log::error "❌ Semantic search failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "Qdrant not running; semantic search skipped"
  testing::phase::add_test skipped
fi

# Verify analytics endpoint if available
if curl -sf "${API_URL}/api/analytics/summary" >/dev/null 2>&1; then
  log::success "✅ Analytics summary reachable"
  testing::phase::add_test passed
else
  testing::phase::add_warning "Analytics summary endpoint unavailable"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business workflow validation completed"
