#!/bin/bash
# Integration test phase for travel-map-filler
# Tests database operations and multi-component interactions

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "📊 Running integration tests..."

# Run Go integration tests
if [ -d "api" ] && [ -f "api/integration_test.go" ]; then
    echo "Running API integration tests..."
    cd api
    go test -v -run "TestDatabaseIntegration|TestConcurrentOperations|TestDatabaseResilience" \
        -timeout 120s \
        -coverprofile=integration_coverage.out 2>&1 | tee integration_test.log

    INTEGRATION_EXIT_CODE=${PIPESTATUS[0]}

    if [ $INTEGRATION_EXIT_CODE -eq 0 ]; then
        echo "✅ Integration tests passed"
    else
        echo "❌ Integration tests failed with exit code $INTEGRATION_EXIT_CODE"
    fi

    cd ..
else
    echo "⚠️  No integration tests found"
    INTEGRATION_EXIT_CODE=0
fi

testing::phase::end_with_summary "Integration tests completed"

exit $INTEGRATION_EXIT_CODE
