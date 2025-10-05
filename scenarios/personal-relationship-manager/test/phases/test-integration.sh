#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "120s"

echo "🧪 Running integration tests..."

cd "$TESTING_PHASE_SCENARIO_DIR/api" || exit 1

# Run Go integration tests
if command -v go >/dev/null 2>&1; then
    log::info "Running Go integration tests..."

    # Run all integration test files
    if go test -v -tags=testing -run "TestContactLifecycle|TestRelationshipProcessorIntegration|TestConcurrentOperations" -timeout 90s 2>&1 | tee /tmp/integration-test-output.txt; then
        log::success "✅ Integration tests passed"
    else
        testing::phase::add_error "❌ Integration tests failed"
    fi

    # Check for specific integration scenarios
    if grep -q "TestContactLifecycle" /tmp/integration-test-output.txt; then
        if grep "TestContactLifecycle.*PASS" /tmp/integration-test-output.txt >/dev/null; then
            log::success "✅ Contact lifecycle integration test passed"
        else
            testing::phase::add_error "❌ Contact lifecycle integration test failed"
        fi
    fi

    if grep -q "TestRelationshipProcessorIntegration" /tmp/integration-test-output.txt; then
        if grep "TestRelationshipProcessorIntegration.*PASS" /tmp/integration-test-output.txt >/dev/null; then
            log::success "✅ Relationship processor integration test passed"
        else
            testing::phase::add_error "❌ Relationship processor integration test failed"
        fi
    fi

    if grep -q "TestConcurrentOperations" /tmp/integration-test-output.txt; then
        if grep "TestConcurrentOperations.*PASS" /tmp/integration-test-output.txt >/dev/null; then
            log::success "✅ Concurrent operations test passed"
        else
            testing::phase::add_error "❌ Concurrent operations test failed"
        fi
    fi

    # Show test summary
    if grep -q "PASS" /tmp/integration-test-output.txt; then
        passed=$(grep -c "PASS:" /tmp/integration-test-output.txt || echo 0)
        log::info "Integration tests passed: $passed"
    fi

    if grep -q "FAIL" /tmp/integration-test-output.txt; then
        failed=$(grep -c "FAIL:" /tmp/integration-test-output.txt || echo 0)
        if [ "$failed" -gt 0 ]; then
            testing::phase::add_error "Integration tests failed: $failed"
        fi
    fi
else
    testing::phase::add_error "❌ Go toolchain not available for integration tests"
fi

# End with summary
testing::phase::end_with_summary "Integration tests completed"
