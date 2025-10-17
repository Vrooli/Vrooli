#!/bin/bash
#
# Integration Test Phase for feature-request-voting
# Tests interaction with database and external systems
#

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::announce "Running integration tests with database"

# Check database availability
if ! command -v psql &> /dev/null; then
    testing::phase::warning "PostgreSQL client not available, skipping integration tests"
    testing::phase::end_with_summary "Integration tests skipped (no database)"
    exit 0
fi

# Run Go integration tests
if [ -d "api" ]; then
    testing::phase::step "Running Go integration tests"
    cd api

    if go test -tags=testing -v -run TestIntegration 2>&1 | tee /tmp/integration-test.log; then
        testing::phase::success "Integration tests passed"
    else
        testing::phase::error "Integration tests failed"
        testing::phase::end_with_summary "Integration tests failed"
        exit 1
    fi

    cd ..
fi

testing::phase::end_with_summary "Integration tests completed"
