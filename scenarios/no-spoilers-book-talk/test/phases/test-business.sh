#!/bin/bash
# Validates spoiler-prevention business logic and CLI workflows
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "300s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"

api_url=$(testing::connectivity::get_api_url "$SCENARIO_NAME" 2>/dev/null || true)
if [ -z "$api_url" ]; then
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Business tests incomplete"
fi

API_PORT="${api_url##*:}"
API_PORT="${API_PORT#/}"
export API_PORT

# Spoiler prevention validation requires curl/jq
missing_tools=()
for cmd in curl jq; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    missing_tools+=("$cmd")
  fi
done
if [ ${#missing_tools[@]} -gt 0 ]; then
  testing::phase::add_warning "Missing tools for spoiler validation: ${missing_tools[*]}"
  testing::phase::add_test skipped
else
  testing::phase::check "Spoiler prevention workflow" bash test/test-spoiler-prevention.sh
fi

# CLI bats suite (optional)
if command -v bats >/dev/null 2>&1 && [ -f "cli/no-spoilers-book-talk.bats" ]; then
  testing::phase::check "CLI workflow bats suite" bash -c 'cd cli && bats no-spoilers-book-talk.bats'
else
  testing::phase::add_warning "Skipping CLI bats tests (bats not installed or file missing)"
  testing::phase::add_test skipped
fi


testing::phase::end_with_summary "Business validation completed"
