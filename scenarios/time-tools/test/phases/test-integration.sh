#!/bin/bash
# Execute API-level integration checks against a running time-tools instance.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for API validation"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not available; skipping API suite"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration tests skipped"
fi

API_SUITE="test/test-api-endpoints.sh"
if [ ! -x "$API_SUITE" ]; then
  if [ -f "$API_SUITE" ]; then
    chmod +x "$API_SUITE" || true
  else
    testing::phase::add_error "API integration suite missing at $API_SUITE"
    testing::phase::end_with_summary "Integration tests incomplete"
  fi
fi

testing::phase::check "Run API endpoint suite" bash "$API_SUITE"

testing::phase::end_with_summary "Integration validation completed"
