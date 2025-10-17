#!/bin/bash
# Business logic test phase for roi-fit-analysis
# Validates core business logic and calculations

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR/api"

echo "💼 Running business logic tests for roi-fit-analysis..."

# Run business logic specific tests
echo "  ✓ Running business logic test suite..."

# Run tests matching business logic patterns
go test -v -run="TestBusinessLogic" -timeout=60s 2>&1 | tee test-business.log

# Check for test failures
if grep -q "FAIL" test-business.log; then
    testing::phase::fail "Business logic tests failed"
fi

if grep -q "PASS" test-business.log; then
    echo "    ✓ Business logic tests passed"
else
    testing::phase::fail "No business logic tests found or executed"
fi

# Validate specific business rules
echo "  ✓ Validating business rules..."

# Test recommendation logic
if grep -q "TestBusinessLogic_ROICalculations" test-business.log; then
    echo "    ✓ ROI calculation logic tested"
else
    echo "    ⚠ ROI calculation tests not found"
fi

# Test data extraction logic
if grep -q "TestBusinessLogic_DataExtraction" test-business.log; then
    echo "    ✓ Data extraction logic tested"
else
    echo "    ⚠ Data extraction tests not found"
fi

# Clean up
rm -f test-business.log

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::end_with_summary "Business logic tests completed"
