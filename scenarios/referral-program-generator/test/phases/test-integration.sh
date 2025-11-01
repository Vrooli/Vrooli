#!/bin/bash
# Integration testing phase for referral-program-generator

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "150s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if ! command -v psql >/dev/null 2>&1; then
  testing::phase::add_warning "psql not available; skipping integration suite"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration tests skipped"
fi

if [ -z "${POSTGRES_TEST_URL:-}" ]; then
  testing::phase::add_warning "POSTGRES_TEST_URL not set; skipping integration suite"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration tests skipped"
fi

testing::phase::check "Go integration tests" bash -c 'cd api && go test -v -tags integration -run "TestIntegration" -timeout 2m'

testing::phase::end_with_summary "Integration validation completed"
