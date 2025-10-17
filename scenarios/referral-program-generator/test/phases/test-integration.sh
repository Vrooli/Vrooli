#!/bin/bash
# Integration testing phase for referral-program-generator

set -e

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "ℹ️  Integration tests: Testing API endpoints with database"

# Check if PostgreSQL is available
if ! command -v psql &> /dev/null; then
    echo "⚠️  PostgreSQL not available, skipping integration tests"
    testing::phase::end_with_summary "Integration tests skipped (no PostgreSQL)"
    exit 0
fi

# Check if test database URL is configured
if [ -z "$POSTGRES_TEST_URL" ]; then
    echo "⚠️  POSTGRES_TEST_URL not set, skipping integration tests"
    testing::phase::end_with_summary "Integration tests skipped (no test database)"
    exit 0
fi

# Run integration tests (which require database)
cd api
echo "Running Go integration tests with database..."
go test -v -tags integration -run "TestIntegration" -timeout 2m || {
    echo "❌ Integration tests failed"
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
}

testing::phase::end_with_summary "Integration tests completed"
