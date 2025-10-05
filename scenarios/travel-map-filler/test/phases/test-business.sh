#!/bin/bash
# Business logic test phase for travel-map-filler
# Tests business rules and domain logic

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "💼 Running business logic tests..."

# Run Go business logic tests
if [ -d "api" ] && [ -f "api/comprehensive_test.go" ]; then
    echo "Running business logic tests..."
    cd api
    go test -v -run "TestBusinessLogic|TestIntegration" \
        -timeout 90s \
        -coverprofile=business_coverage.out 2>&1 | tee business_test.log

    BUSINESS_EXIT_CODE=${PIPESTATUS[0]}

    if [ $BUSINESS_EXIT_CODE -eq 0 ]; then
        echo "✅ Business logic tests passed"
    else
        echo "❌ Business logic tests failed with exit code $BUSINESS_EXIT_CODE"
    fi

    cd ..
else
    echo "⚠️  No business logic tests found"
    BUSINESS_EXIT_CODE=0
fi

testing::phase::end_with_summary "Business logic tests completed"

exit $BUSINESS_EXIT_CODE
