#!/bin/bash
# Validates core business workflows for Contact Book.

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
  testing::phase::end_with_summary "Business validation incomplete"
fi

parse_value() {
  local json="$1"
  local fallback_key="$2"
  if command -v jq >/dev/null 2>&1; then
    echo "$json" | jq -r "$fallback_key // empty"
  else
    echo "$json" | sed -n 's/.*"id"\s*:\s*"\([^"]\+\)".*/\1/p'
  fi
}

# === P0: Contact CRUD ===
create_resp=$(curl -sf -X POST "${API_URL}/api/v1/contacts" -H 'Content-Type: application/json' -d '{"full_name":"Business Phase Contact","emails":["business@test.com"],"tags":["phase"]}' || true)
contact_id=$(parse_value "$create_resp" '.id')
if [ -n "$contact_id" ]; then
  testing::phase::check "Retrieve created contact" curl -sf -o /dev/null "${API_URL}/api/v1/contacts/${contact_id}"
  testing::phase::check "Update contact" curl -sf -o /dev/null -X PUT "${API_URL}/api/v1/contacts/${contact_id}" -H 'Content-Type: application/json' -d '{"full_name":"Business Phase Contact Updated"}'
else
  testing::phase::add_error "Contact creation failed"
  testing::phase::add_test failed
fi

# === P0: Relationship graph ===
get_contact_id() {
  local offset="$1"
  local response
  response=$(curl -sf "${API_URL}/api/v1/contacts?offset=${offset}&limit=1" || echo '')
  if [ -z "$response" ]; then
    echo ""
    return
  fi
  if command -v jq >/dev/null 2>&1; then
    echo "$response" | jq -r '.persons[0].id // empty'
  else
    echo "$response" | sed -n 's/.*"id"\s*:\s*"\([^"]\+\)".*/\1/p'
  fi
}

contact_a=$(get_contact_id 0)
contact_b=$(get_contact_id 1)
if [ -n "$contact_a" ] && [ -n "$contact_b" ]; then
  testing::phase::check "Create relationship" curl -sf -o /dev/null -X POST "${API_URL}/api/v1/relationships" -H 'Content-Type: application/json' -d "{\"from_person_id\":\"${contact_a}\",\"to_person_id\":\"${contact_b}\",\"relationship_type\":\"phase-test\",\"strength\":0.6}"
else
  testing::phase::add_warning "Insufficient contacts to exercise relationship API"
  testing::phase::add_test skipped
fi

# === P0: Analytics ===
testing::phase::check "Social analytics" curl -sf -o /dev/null "${API_URL}/api/v1/analytics"

# === P1: Communication preferences & attachments (optional) ===
seed_id="b2ff0db4-6e5a-4159-8c81-8f7dadbd5fea"
if curl -sf "${API_URL}/api/v1/contacts/${seed_id}" >/dev/null 2>&1; then
  if curl -sf "${API_URL}/api/v1/contacts/${seed_id}/preferences" >/dev/null 2>&1; then
    log::success "Communication preferences endpoint responsive"
    testing::phase::add_test passed
  else
    testing::phase::add_warning "Communication preferences endpoint unavailable"
    testing::phase::add_test skipped
  fi

  if curl -sf "${API_URL}/api/v1/contacts/${seed_id}/attachments" >/dev/null 2>&1; then
    log::success "Attachments endpoint responsive"
    testing::phase::add_test passed
  else
    testing::phase::add_warning "Attachments endpoint unavailable"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "Seed contact ${seed_id} not present; skipping preference/attachment checks"
  testing::phase::add_test skipped
  testing::phase::add_test skipped
fi

# Cross-scenario readiness via CLI JSON output
if command -v contact-book >/dev/null 2>&1; then
  testing::phase::check "contact-book list JSON" bash -c 'contact-book list --json --limit 3 | grep -q "persons"'
else
  testing::phase::add_warning "contact-book CLI not in PATH; skipping cross-scenario validation"
  testing::phase::add_test skipped
fi


testing::phase::end_with_summary "Business workflow validation completed"
