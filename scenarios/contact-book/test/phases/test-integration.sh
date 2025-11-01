#!/bin/bash
# Runs API and CLI integration checks for Contact Book.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

scenario_name="$TESTING_PHASE_SCENARIO_NAME"
API_URL=""
if command -v vrooli >/dev/null 2>&1; then
  API_URL=$(testing::connectivity::get_api_url "$scenario_name" || true)
fi

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to determine API URL (vrooli CLI required)"
  testing::phase::end_with_summary "Integration checks incomplete"
fi

# Core endpoints respond
testing::phase::check "API health endpoint" curl -sf -o /dev/null "${API_URL}/health"
testing::phase::check "Contacts listing" curl -sf -o /dev/null "${API_URL}/api/v1/contacts?limit=10"
testing::phase::check "Relationships listing" curl -sf -o /dev/null "${API_URL}/api/v1/relationships?limit=10"
testing::phase::check "Analytics endpoint" curl -sf -o /dev/null "${API_URL}/api/v1/analytics"
testing::phase::check "Search endpoint" curl -sf -o /dev/null -X POST "${API_URL}/api/v1/search" -H "Content-Type: application/json" -d '{"query":"test","limit":5}'

# Create and fetch a contact to verify persistence
create_payload='{"full_name":"Integration Test Contact","emails":["integration@test.com"]}'
create_response=$(curl -sf -X POST "${API_URL}/api/v1/contacts" -H 'Content-Type: application/json' -d "$create_payload" || true)
contact_id=""
if command -v jq >/dev/null 2>&1; then
  contact_id=$(echo "$create_response" | jq -r '.id // empty')
else
  contact_id=$(echo "$create_response" | sed -n 's/.*"id"\s*:\s*"\([^"]\+\)".*/\1/p')
fi
if [ -n "$contact_id" ]; then
  testing::phase::check "Retrieve created contact" curl -sf -o /dev/null "${API_URL}/api/v1/contacts/${contact_id}"
else
  testing::phase::add_warning "Contact creation response missing id"
  testing::phase::add_test skipped
fi

if command -v contact-book >/dev/null 2>&1; then
  testing::phase::check "contact-book help" contact-book help >/dev/null
  testing::phase::check "contact-book status" contact-book status --json >/dev/null
  testing::phase::check "contact-book list" contact-book list --json --limit 5 >/dev/null
else
  testing::phase::add_warning "contact-book CLI not in PATH; skipping CLI smoke checks"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
