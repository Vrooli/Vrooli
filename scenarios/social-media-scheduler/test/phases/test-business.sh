#!/bin/bash
# Business workflow validations for social-media-scheduler

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for business workflow tests"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business tests incomplete"
fi

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_error "jq is required for JSON parsing"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business tests incomplete"
fi

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to resolve API port"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business tests incomplete"
fi

API_BASE="http://localhost:${API_PORT}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

DEMO_TOKEN=""
REGISTERED_TOKEN=""

record_pass() {
  local message="$1"
  log::success "✅ ${message}"
  testing::phase::add_test passed
}

record_fail() {
  local message="$1"
  log::error "❌ ${message}"
  testing::phase::add_error "$message"
  testing::phase::add_test failed
}

perform_demo_login() {
  local out="$TMP_DIR/demo-login.json"
  if curl -sf -X POST "$API_BASE/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{"email":"demo@vrooli.com","password":"demo123"}' >"$out"; then
    if jq -e '.success == true' "$out" >/dev/null; then
      DEMO_TOKEN=$(jq -r '.data.token' "$out")
      record_pass "Demo login returned token"
    else
      record_fail "Demo login response did not indicate success"
    fi
  else
    record_fail "Demo login request failed"
  fi
}

perform_user_registration() {
  local email="test-$(date +%s)@scheduler.test"
  local out="$TMP_DIR/register.json"
  if curl -sf -X POST "$API_BASE/api/v1/auth/register" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$email\",\"password\":\"testpass123\",\"first_name\":\"Test\",\"last_name\":\"User\"}" >"$out"; then
    if jq -e '.success == true' "$out" >/dev/null; then
      REGISTERED_TOKEN=$(jq -r '.data.token' "$out")
      record_pass "User registration succeeded"
    else
      record_fail "User registration response did not indicate success"
    fi
  else
    record_fail "User registration request failed"
  fi
}

check_protected_endpoint() {
  local token="$1"
  local out="$TMP_DIR/me.json"
  if [ -z "$token" ]; then
    log::warning "⚠️  Auth token unavailable; skipping protected endpoint check"
    testing::phase::add_warning "Auth token unavailable; skipping protected endpoint check"
    testing::phase::add_test skipped
    return
  fi
  if curl -sf -H "Authorization: Bearer $token" "$API_BASE/api/v1/auth/me" >"$out"; then
    if jq -e '.success == true' "$out" >/dev/null; then
      record_pass "Protected endpoint accepted token"
    else
      record_fail "Protected endpoint returned unexpected payload"
    fi
  else
    record_fail "Protected endpoint request failed"
  fi
}

schedule_post() {
  if [ -z "$DEMO_TOKEN" ]; then
    log::warning "⚠️  Demo token unavailable; skipping scheduling test"
    testing::phase::add_warning "Demo token unavailable; skipping scheduling test"
    testing::phase::add_test skipped
    return
  fi
  local publish_at
  publish_at=$(date -u -d '+90 minutes' +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -v+90M -u +"%Y-%m-%dT%H:%M:%SZ")
  local payload="{\"title\":\"Integration Test Post\",\"content\":\"Test content from phased business tests\",\"platforms\":[\"twitter\",\"linkedin\"],\"scheduled_at\":\"$publish_at\",\"timezone\":\"UTC\",\"auto_optimize\":true}"
  if curl -sf -X POST "$API_BASE/api/v1/posts/schedule" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $DEMO_TOKEN" \
    -d "$payload" | jq -e '.success == true' >/dev/null; then
    record_pass "Scheduling endpoint accepted post"
  else
    record_fail "Scheduling endpoint did not accept request"
  fi
}

fetch_calendar() {
  if [ -z "$DEMO_TOKEN" ]; then
    log::warning "⚠️  Demo token unavailable; skipping calendar retrieval"
    testing::phase::add_warning "Demo token unavailable; skipping calendar retrieval"
    testing::phase::add_test skipped
    return
  fi
  local start_date end_date
  start_date=$(date -u -d '-1 month' +"%Y-%m-%d" 2>/dev/null || date -v-1m -u +"%Y-%m-%d")
  end_date=$(date -u -d '+1 month' +"%Y-%m-%d" 2>/dev/null || date -v+1m -u +"%Y-%m-%d")
  if curl -sf -H "Authorization: Bearer $DEMO_TOKEN" \
    "$API_BASE/api/v1/posts/calendar?start_date=$start_date&end_date=$end_date" | jq -e '.success == true' >/dev/null; then
    record_pass "Calendar endpoint returned data"
  else
    record_fail "Calendar endpoint did not return success"
  fi
}

perform_demo_login
perform_user_registration
check_protected_endpoint "$DEMO_TOKEN"
check_protected_endpoint "$REGISTERED_TOKEN"
schedule_post
fetch_calendar

testing::phase::end_with_summary "Business workflow validation completed"
