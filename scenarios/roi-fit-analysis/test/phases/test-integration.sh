#!/bin/bash
# Integration test phase for roi-fit-analysis
# Tests integration between components

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR/api"

echo "🔗 Running integration tests for roi-fit-analysis..."

# Run integration tests
echo "  ✓ Running integration test suite..."
go test -v -run="TestIntegration" -timeout=60s 2>&1 | tee test-integration.log

# Check results
if grep -q "FAIL" test-integration.log; then
    testing::phase::fail "Integration tests failed"
fi

if grep -q "PASS" test-integration.log; then
    echo "    ✓ Integration tests passed"
else
    testing::phase::fail "No integration tests found or executed"
fi

# Clean up
rm -f test-integration.log

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::end_with_summary "Integration tests completed"
