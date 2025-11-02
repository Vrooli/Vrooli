#!/bin/bash
# Validates core business workflows by chaining persona creation, data ingestion,
# and chat interaction journeys.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "300s" --require-runtime

missing_tools=()
for tool in curl jq; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    missing_tools+=("$tool")
  fi
done

if [ ${#missing_tools[@]} -gt 0 ]; then
  testing::phase::add_error "Required tooling missing: ${missing_tools[*]}"
  testing::phase::end_with_summary "Business workflows blocked"
fi

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_BASE_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)
if [ -z "$API_BASE_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Business workflows blocked"
fi

# Attempt to resolve dedicated chat port, fall back to API base if unavailable.
CHAT_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" CHAT_PORT 2>/dev/null || true)
CHAT_PORT=$(echo "$CHAT_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$CHAT_PORT" ]; then
  CHAT_BASE_URL="$API_BASE_URL"
else
  CHAT_BASE_URL="http://localhost:${CHAT_PORT}"
fi

export API_BASE_URL CHAT_BASE_URL

run_business_check() {
  local description="$1"
  shift
  log::info "🔸 $description"
  if "$@"; then
    testing::phase::add_test passed
    log::success "✅ $description"
    return 0
  else
    testing::phase::add_error "$description failed"
    testing::phase::add_test failed
    return 1
  fi
}

run_business_check "Persona creation workflow" bash test/test-persona-creation.sh
run_business_check "Data ingestion workflow" bash test/test-data-ingestion.sh
run_business_check "Chat interaction workflow" bash test/test-chat-interaction.sh

# CLI tests are optional depending on bats availability.
if command -v bats >/dev/null 2>&1 && [ -f "cli/personal-digital-twin.bats" ]; then
  run_business_check "CLI command suite" bash -c 'cd cli && bats personal-digital-twin.bats'
else
  testing::phase::add_warning "CLI BATS tests skipped (bats not available or test file missing)"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business workflows validated"
