#!/bin/bash
# Validate business workflows and UI smoke paths for the calendar scenario.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "360s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
SCENARIO_DIR="${TESTING_PHASE_SCENARIO_DIR}"
AUTH_HEADER="Authorization: Bearer ${CALENDAR_AUTH_TOKEN:-test}"

require_command() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    testing::phase::add_error "Required command '$name' not available"
    testing::phase::add_test failed
    testing::phase::end_with_summary "Business tests incomplete"
  fi
}

require_command curl
require_command jq

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi
if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to resolve API port"
  testing::phase::end_with_summary "Business tests incomplete"
fi

UI_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" UI_PORT 2>/dev/null || true)
UI_PORT=$(echo "$UI_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$UI_PORT" ]; then
  UI_PORT=$(echo "$UI_PORT_OUTPUT" | tr -d '[:space:]')
fi

API_URL="http://localhost:${API_PORT}"
UI_URL="${UI_PORT:+http://localhost:${UI_PORT}}"
BROWSERLESS_URL="${BROWSERLESS_URL:-http://localhost:3003}"

test_api_health() {
  curl -sf "${API_URL}/health" | jq -e '.status == "healthy"' >/dev/null
}

schedule_optimization() {
  jq -n '{
      optimization_goal: "minimize_gaps",
      start_date: (now | strftime("%Y-%m-%dT00:00:00Z")),
      end_date: (now + 86400 | strftime("%Y-%m-%dT00:00:00Z")),
      constraints: { business_hours_only: true }
    }' \
    | curl -sf -X POST "${API_URL}/api/v1/schedule/optimize" \
        -H "$AUTH_HEADER" \
        -H "Content-Type: application/json" \
        -d @- \
    | jq -e '.optimization_goal == "minimize_gaps" and (.suggestions | length) >= 0' >/dev/null
}

check_timezone_endpoint() {
  curl -sf "${API_URL}/api/v1/events?timezone=America/New_York" \
    -H "$AUTH_HEADER" | jq -e '.timezone == "America/New_York"' >/dev/null
}

check_ical_export() {
  curl -sf "${API_URL}/api/v1/events/export/ical" -H "$AUTH_HEADER" | grep -q "BEGIN:VCALENDAR"
}

check_ical_import() {
  local payload
  payload="BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//EN
BEGIN:VEVENT
UID:test-import-$(date +%s)@vrooli
DTSTART:$(date -u +%Y%m%dT%H%M%SZ)
DTEND:$(date -u -d '+1 hour' +%Y%m%dT%H%M%SZ)
SUMMARY:Test Import Event
DESCRIPTION:Testing iCal import
END:VEVENT
END:VCALENDAR"
  printf '%s' "$payload" \
    | curl -sf -X POST "${API_URL}/api/v1/events/import/ical" \
        -H "$AUTH_HEADER" \
        -H "Content-Type: text/calendar" \
        --data-binary @- \
    | jq -e '.imported_count == 1' >/dev/null
}

ui_homepage_accessible() {
  if [ -z "$UI_URL" ]; then
    return 1
  fi
  curl -sf "$UI_URL" >/dev/null
}

ui_screenshot() {
  if [ -z "$UI_URL" ]; then
    return 1
  fi
  local tmp_file
  tmp_file=$(mktemp "${TMPDIR:-/tmp}/calendar-ui-XXXXXX.png")
  jq -n --arg url "$UI_URL" '{ url: $url, options: { type: "png", fullPage: true } }' \
    | curl -s -X POST "$BROWSERLESS_URL/screenshot" \
        -H "Content-Type: application/json" \
        -d @- \
        -o "$tmp_file"
  local status=$?
  if [ $status -ne 0 ] || [ ! -s "$tmp_file" ]; then
    rm -f "$tmp_file"
    return 1
  fi
  rm -f "$tmp_file"
  return 0
}

api_integration_via_ui() {
  curl -sf "${API_URL}/health" >/dev/null
}

basic_ui_selector_check() {
  if [ -z "$UI_URL" ]; then
    return 1
  fi
  jq -n --arg url "$UI_URL" '{
      url: $url,
      code: "await page.waitForSelector(\"body\", { timeout: 5000 }); return { success: true };"
    }' \
    | curl -sf -X POST "$BROWSERLESS_URL/function" \
        -H "Content-Type: application/json" \
        -d @- \
    | jq -e '.success == true' >/dev/null
}

testing::phase::check "API health" test_api_health

if ! testing::phase::check "Schedule optimization" schedule_optimization; then
  true
fi
if ! testing::phase::check "Timezone parameter support" check_timezone_endpoint; then
  true
fi
if ! testing::phase::check "iCal export" check_ical_export; then
  true
fi
if ! testing::phase::check "iCal import" check_ical_import; then
  true
fi

if [ -n "$UI_URL" ]; then
  if testing::phase::check "UI homepage accessible" ui_homepage_accessible; then
    true
  else
    testing::phase::add_warning "UI homepage unavailable"
  fi
else
  testing::phase::add_warning "UI port not registered; skipping UI smoke"
  testing::phase::add_test skipped
fi

if ! testing::phase::check "Browserless screenshot" ui_screenshot; then
  testing::phase::add_warning "Screenshot capture failed"
fi
if ! testing::phase::check "Browserless selector probe" basic_ui_selector_check; then
  testing::phase::add_warning "UI selector probe failed"
fi
if ! testing::phase::check "API reachable for UI" api_integration_via_ui; then
  true
fi

testing::phase::end_with_summary "Business validation completed"
