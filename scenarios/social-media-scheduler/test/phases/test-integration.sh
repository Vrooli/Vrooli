#!/bin/bash
# HTTP-level integration checks for social-media-scheduler

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for integration checks"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_error "jq is required for JSON assertions"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests incomplete"
fi

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
UI_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" UI_PORT 2>/dev/null || true)

API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
UI_PORT=$(echo "$UI_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')

if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi
if [ -z "$UI_PORT" ]; then
  UI_PORT=$(echo "$UI_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ] || [ -z "$UI_PORT" ]; then
  testing::phase::add_error "Unable to resolve scenario ports"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests incomplete"
fi

API_BASE="http://localhost:${API_PORT}"
UI_BASE="http://localhost:${UI_PORT}"

health_check() {
  testing::phase::check "API health" bash -c "curl -sf ${API_BASE}/health | jq -e '.status == \"healthy\"'"
  testing::phase::check "Queue health" bash -c "curl -sf ${API_BASE}/health/queue | jq -e '.success == true'"
  testing::phase::check "UI health" bash -c "curl -sf ${UI_BASE}/health | jq -e '.status == \"healthy\"'"
}

http_surface() {
  testing::phase::check "UI root serves shell" bash -c "curl -sf ${UI_BASE}/ | grep -qi 'social media scheduler'"
  testing::phase::check "Platform configuration" bash -c "curl -sf ${API_BASE}/api/v1/auth/platforms | jq -e '.success == true and (.data | length) >= 1'"
  testing::phase::check "UI API proxy" bash -c "curl -sf ${UI_BASE}/api/v1/auth/platforms | jq -e '.success == true'"
}

security_surface() {
  testing::phase::check "Protected endpoint rejects anonymous requests" bash -c "curl -sf ${API_BASE}/api/v1/auth/me | jq -e '.success == false'"
}

health_check
http_surface
security_surface

if command -v psql >/dev/null 2>&1; then
  POSTGRES_PORT=$(vrooli scenario port "$SCENARIO_NAME" POSTGRES_PORT 2>/dev/null | awk -F= '/=/{print $2}' | tr -d ' ')
  POSTGRES_PORT=${POSTGRES_PORT:-5432}
  testing::phase::check "Database connectivity" bash -c "psql -h localhost -p ${POSTGRES_PORT} -U postgres -d vrooli_social_media_scheduler -c 'SELECT 1' >/dev/null"
else
  testing::phase::add_warning "psql not available; skipping database connectivity check"
  testing::phase::add_test skipped
fi

if command -v redis-cli >/dev/null 2>&1; then
  REDIS_PORT=$(vrooli scenario port "$SCENARIO_NAME" REDIS_PORT 2>/dev/null | awk -F= '/=/{print $2}' | tr -d ' ')
  REDIS_PORT=${REDIS_PORT:-6379}
  testing::phase::check "Redis connectivity" redis-cli -h localhost -p "$REDIS_PORT" ping
else
  testing::phase::add_warning "redis-cli not available; skipping Redis connectivity check"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration checks completed"
