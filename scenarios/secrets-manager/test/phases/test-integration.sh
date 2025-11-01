#!/bin/bash
# Integration validation for Secrets Manager API and backing services
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_URL=""
UI_URL=""

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_error "jq is required for integration validation"
  testing::phase::end_with_summary "Integration tests failed"
fi

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for integration validation"
  testing::phase::end_with_summary "Integration tests failed"
fi

API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)
UI_URL=$(testing::connectivity::get_ui_url "$SCENARIO_NAME" || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to determine API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests failed"
fi

if [ -z "$UI_URL" ]; then
  testing::phase::add_error "Unable to determine UI URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests failed"
fi

# Smoke test health endpoints
if testing::phase::check "API health" curl -sf "$API_URL/health"; then
  :
fi

if testing::phase::check "UI health" curl -sf "$UI_URL/health"; then
  :
fi

# Validate read endpoints
if testing::phase::check "Secrets scan endpoint (GET)" env API_URL="$API_URL" bash -c 'curl -sf "$API_URL/api/v1/secrets/scan" | jq -e ".discovered_secrets" >/dev/null'; then
  :
else
  log::error "Response: $(curl -s "$API_URL/api/v1/secrets/scan")"
fi

if testing::phase::check "Secrets validation endpoint (GET)" env API_URL="$API_URL" bash -c 'curl -sf "$API_URL/api/v1/secrets/validate" | jq -e ".total_secrets" >/dev/null'; then
  :
fi

# Validate write endpoints and data persistence
if testing::phase::check "Secrets scan endpoint (POST full scan)" env API_URL="$API_URL" bash -c 'curl -sf -H "Content-Type: application/json" -d "{\"scan_type\":\"full\"}" "$API_URL/api/v1/secrets/scan" | jq -e ".scan_duration_ms" >/dev/null'; then
  :
fi

if testing::phase::check "Secrets validation endpoint (POST)" env API_URL="$API_URL" bash -c 'curl -sf -H "Content-Type: application/json" -d "{\"resource\":\"postgres\"}" "$API_URL/api/v1/secrets/validate" | jq -e "has(\"validation_id\")" >/dev/null'; then
  :
fi

if command -v psql >/dev/null 2>&1; then
  testing::phase::check "resource_secrets table populated" \
    env PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" \
      psql -h "${POSTGRES_HOST:-localhost}" -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-vrooli}" \
      -c "SELECT 1 FROM resource_secrets LIMIT 1;" >/dev/null
  testing::phase::check "secret_validations table accessible" \
    env PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" \
      psql -h "${POSTGRES_HOST:-localhost}" -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-vrooli}" \
      -c "SELECT 1 FROM secret_validations LIMIT 1;" >/dev/null
else
  testing::phase::add_warning "psql not available; skipping database integration checks"
  testing::phase::add_test skipped
fi

# Optional n8n workflow health check
if curl -sf "http://localhost:5678/healthz" >/dev/null 2>&1; then
  testing::phase::check "n8n workflow API reachable" curl -sf "http://localhost:5678/api/v1/workflows"
else
  testing::phase::add_warning "n8n workflow engine not reachable; skipping workflow validation"
  testing::phase::add_test skipped
fi

# UI serves primary document
if testing::phase::check "UI landing page renders" env UI_URL="$UI_URL" bash -c 'curl -sf "$UI_URL" | grep -q "Secrets Manager"'; then
  :
fi

testing::phase::end_with_summary "Integration validation completed"
