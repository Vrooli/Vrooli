#!/bin/bash
# Integration tests for Make It Vegan - Integrated with centralized testing infrastructure

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running integration tests..."

# Run Go integration tests
if [ -d "api" ]; then
    cd api
    echo "Running API integration tests..."
    go test -v -run="^TestIntegration|^TestEndToEnd|^TestConcurrent|^TestAPIContract" -timeout 120s
    TEST_EXIT=$?
    cd ..

    if [ $TEST_EXIT -ne 0 ]; then
        testing::phase::end_with_summary "Integration tests failed" $TEST_EXIT
        exit $TEST_EXIT
    fi
fi

testing::phase::end_with_summary "Integration tests completed successfully"
