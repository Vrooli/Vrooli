#!/bin/bash
# Validates core business workflows and CLI interactions for symbol-search

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "180s" --require-runtime

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not available; skipping business workflow validations"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Business workflow checks skipped"
fi

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)
CLI_PATH="${TESTING_PHASE_SCENARIO_DIR}/cli/symbol-search"

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to determine API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Business workflow checks incomplete"
fi

if [ ! -x "$CLI_PATH" ]; then
  testing::phase::add_error "CLI binary not executable at $CLI_PATH"
  testing::phase::end_with_summary "Business workflow checks incomplete"
fi

# Ensure CLI uses discovered API port
export SYMBOL_SEARCH_API_URL="$API_URL"

# CLI smoke tests
testing::phase::check "CLI help command" bash -c '(cd cli && ./symbol-search --help >/dev/null)'

testing::phase::check "CLI search command" bash -c '(cd cli && ./symbol-search search --query heart --limit 3 | grep -qi "heart")'

testing::phase::check "CLI category listing" bash -c '(cd cli && ./symbol-search categories | grep -qi "letters")'

# Business flow: API search + CLI compare
testing::phase::check "API search returns characters" bash -c "curl -s '${API_URL}/api/search?q=LATIN&limit=5' | jq -e '.characters | length > 0' >/dev/null"

testing::phase::check "API character details" bash -c "curl -s '${API_URL}/api/character/U+0041' | jq -e '.character.codepoint == \"U+0041\"' >/dev/null"

# Verify database-backed categories via API
testing::phase::check "API categories include Symbol Other" bash -c "curl -s '${API_URL}/api/categories' | jq -e '.categories | map(select(.code==\"So\")) | length > 0' >/dev/null"

# Smoke test bulk range workflow comparing API to CLI output
run_bulk_range_validation() {
  local api_payload
  api_payload=$(curl -s -X POST "${API_URL}/api/bulk/range" \
    -H 'Content-Type: application/json' \
    -d '{"ranges":[{"start":"U+0030","end":"U+0039"}]}' )

  local cli_payload
  cli_payload=$(cd cli && ./symbol-search bulk-range --start U+0030 --end U+0039)

  local api_count
  api_count=$(echo "$api_payload" | jq '.characters | length // 0')
  local cli_count
  cli_count=$(echo "$cli_payload" | jq '.characters | length // 0')

  if [ "$api_count" -gt 0 ] && [ "$api_count" -eq "$cli_count" ]; then
    return 0
  fi
  echo "Mismatch between API bulk range ($api_count) and CLI bulk range ($cli_count)" >&2
  return 1
}

testing::phase::check "Bulk range parity between API and CLI" run_bulk_range_validation

# Telemetry: ensure CLI version command works
testing::phase::check "CLI version command" bash -c '(cd cli && ./symbol-search --version >/dev/null)'

# Clean environment variables to avoid leaking context
unset SYMBOL_SEARCH_API_URL

# Summarise results
testing::phase::end_with_summary "Business workflow validation completed"
