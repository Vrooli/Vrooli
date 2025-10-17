#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "90s"

echo "🧪 Running business logic tests..."

cd "$TESTING_PHASE_SCENARIO_DIR/api" || exit 1

# Run Go business logic tests with tags
if command -v go >/dev/null 2>&1; then
    log::info "Running Go business logic tests..."

    # Test relationship processor business logic
    if go test -v -tags=testing -run "TestRelationshipProcessor" -timeout 60s 2>&1 | tee /tmp/business-test-output.txt; then
        log::success "✅ Relationship processor tests passed"
    else
        testing::phase::add_error "❌ Relationship processor tests failed"
    fi

    # Test contact lifecycle
    if go test -v -tags=testing -run "TestContactLifecycle" -timeout 60s 2>&1 | tee -a /tmp/business-test-output.txt; then
        log::success "✅ Contact lifecycle tests passed"
    else
        testing::phase::add_error "❌ Contact lifecycle tests failed"
    fi

    # Test birthday reminder flow
    if go test -v -tags=testing -run "TestBirthdayReminderFlow" -timeout 60s 2>&1 | tee -a /tmp/business-test-output.txt; then
        log::success "✅ Birthday reminder flow tests passed"
    else
        testing::phase::add_error "❌ Birthday reminder flow tests failed"
    fi

    # Show test summary
    if grep -q "PASS" /tmp/business-test-output.txt; then
        passed=$(grep -c "PASS:" /tmp/business-test-output.txt || echo 0)
        log::info "Business tests passed: $passed"
    fi

    if grep -q "FAIL" /tmp/business-test-output.txt; then
        failed=$(grep -c "FAIL:" /tmp/business-test-output.txt || echo 0)
        if [ "$failed" -gt 0 ]; then
            testing::phase::add_error "Business tests failed: $failed"
        fi
    fi
else
    testing::phase::add_error "❌ Go toolchain not available for business logic tests"
fi

# End with summary
testing::phase::end_with_summary "Business logic tests completed"
