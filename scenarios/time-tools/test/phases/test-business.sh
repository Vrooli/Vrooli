#!/bin/bash
# Validate business workflows via combined API + CLI smoke tests.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

BUSINESS_SUITE="test/test-basic-functionality.sh"
if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not available; skipping business smoke suite"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Business tests skipped"
fi
if [ ! -x "$BUSINESS_SUITE" ]; then
  if [ -f "$BUSINESS_SUITE" ]; then
    chmod +x "$BUSINESS_SUITE" || true
  else
    testing::phase::add_error "Business smoke suite missing at $BUSINESS_SUITE"
    testing::phase::end_with_summary "Business tests incomplete"
  fi
fi

testing::phase::check "Run business smoke suite" bash "$BUSINESS_SUITE"

testing::phase::end_with_summary "Business workflow validation completed"
