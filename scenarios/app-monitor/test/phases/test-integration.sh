#!/bin/bash
# Validate core API integrations and health endpoints while the scenario is running.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for integration checks"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)
if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API port for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

# Basic connectivity
if testing::phase::check "API health endpoint" curl -fsS "$API_URL/health"; then
  true
fi

# App inventory endpoint should respond.
testing::phase::check "Apps endpoint responds" curl -fsS "$API_URL/api/v1/apps"

# Docker information surface is critical for monitoring.
testing::phase::check "Docker info endpoint responds" curl -fsS "$API_URL/api/v1/docker/info"

# Resource catalog should enumerate monitored services.
testing::phase::check "Resources endpoint responds" curl -fsS "$API_URL/api/v1/resources"

# System metrics should be available for dashboards.
testing::phase::check "System metrics endpoint responds" curl -fsS "$API_URL/api/v1/system/metrics"

testing::phase::end_with_summary "Integration validation completed"
