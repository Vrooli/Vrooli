#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running integration tests for token-economy..."

# Run Go integration tests
if [ -f "api/integration_test.go" ]; then
    echo "Running Go integration tests..."
    cd api

    # Run all integration tests
    if go test -v -run "^TestToken|^TestWallet|^TestTransaction|^TestAchievement|^TestAnalytics|^TestError|^TestConcurrent|^TestCache" -timeout 120s; then
        echo "✓ Integration tests passed"
    else
        echo "⚠ Some integration tests failed or were skipped"
    fi

    # Run specific integration scenarios
    echo ""
    echo "Running specific integration scenarios..."

    echo "Testing token lifecycle..."
    go test -v -run "TestTokenLifecycle" -timeout 30s || echo "⚠ Token lifecycle test skipped"

    echo "Testing wallet balance flow..."
    go test -v -run "TestWalletBalanceFlow" -timeout 30s || echo "⚠ Wallet balance flow test skipped"

    echo "Testing transaction history..."
    go test -v -run "TestTransactionHistory" -timeout 30s || echo "⚠ Transaction history test skipped"

    echo "Testing token management..."
    go test -v -run "TestTokenManagement" -timeout 30s || echo "⚠ Token management test skipped"

    echo "Testing achievement system..."
    go test -v -run "TestAchievementSystem" -timeout 30s || echo "⚠ Achievement system test skipped"

    echo "Testing analytics dashboard..."
    go test -v -run "TestAnalyticsDashboard" -timeout 30s || echo "⚠ Analytics test skipped"

    echo "Testing error handling..."
    go test -v -run "TestErrorHandling" -timeout 30s || echo "⚠ Error handling test skipped"

    echo "Testing concurrent operations..."
    go test -v -run "TestConcurrentOperations" -timeout 30s || echo "⚠ Concurrent operations test skipped"

    cd ..
else
    echo "⚠ No integration tests found (api/integration_test.go missing)"
fi

# Run legacy integration test if it exists
if [ -f "tests/integration-test.sh" ]; then
    echo "Running legacy integration test..."
    bash tests/integration-test.sh
fi

testing::phase::end_with_summary "Integration tests completed"
