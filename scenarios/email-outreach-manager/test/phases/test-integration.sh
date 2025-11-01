#!/bin/bash
# Executes integration-focused Go tests that exercise end-to-end flows
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if [ -f "api/integration_test.go" ]; then
  testing::phase::check "Go integration suite" bash -c 'cd api && go test -run TestIntegrationWithMockDB ./...'
else
  testing::phase::add_warning "Integration test file missing; skipping"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration tests completed"
