#!/bin/bash
# Business phase: validates end-to-end learner workflows across API, UI, and CLI surfaces.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "150s" --require-runtime

API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME")
UI_URL=$(testing::connectivity::get_ui_url "$TESTING_PHASE_SCENARIO_NAME")

if [ -z "$API_URL" ] || [ -z "$UI_URL" ]; then
  testing::phase::add_error "Unable to resolve scenario endpoints"
  testing::phase::end_with_summary "Business workflow checks aborted"
fi

check_cli_output() {
  local expected="$1"
  shift
  local output
  output=$("$@" 2>&1 || true)
  printf '%s' "$output" | grep -qi "$expected"
}

validate_ui_homepage() {
  curl -sf "$UI_URL" | grep -qi "Study Buddy"
}

validate_materials_listing() {
  curl -sf "$API_URL/api/study/materials" | jq -e '.materials' >/dev/null 2>&1
}

validate_due_cards() {
  curl -sf "$API_URL/api/study/due-cards?user_id=business-user&limit=5" | jq -e '.cards' >/dev/null 2>&1
}

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not found; JSON assertions in business phase downgraded"
  validate_materials_listing() { curl -sf "$API_URL/api/study/materials" >/dev/null; }
  validate_due_cards() { curl -sf "$API_URL/api/study/due-cards?user_id=business-user&limit=5" >/dev/null; }
fi

# UI smoke validation keeps the business phase aware of front-end regressions.
testing::phase::check "UI landing page renders" validate_ui_homepage

# API-level business flows that combine multiple services.
testing::phase::check "Study materials catalogue available" validate_materials_listing
testing::phase::check "Due card pipeline returns results" validate_due_cards

# CLI flows remain optional but deliver high-value coverage when the symlink is installed.
if command -v study-buddy >/dev/null 2>&1; then
  testing::phase::check "CLI help text" check_cli_output "Study Buddy" study-buddy help
  testing::phase::check "CLI progress summary" check_cli_output "progress" study-buddy show-progress
else
  testing::phase::add_warning "study-buddy CLI binary not found on PATH; skipping CLI workflow checks"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business workflow validation completed"
