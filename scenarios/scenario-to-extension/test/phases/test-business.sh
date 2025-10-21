#!/bin/bash
# Business logic tests for scenario-to-extension

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR/api" || { echo "Failed to cd to api directory"; exit 1; }

testing::phase::info "Starting business logic tests for scenario-to-extension"

# Run Go business tests
testing::phase::step "Running Go business logic tests"
if ! go test -v -run "TestBusiness" -timeout 60s 2>&1 | tee /tmp/business-test-output.log; then
    testing::phase::error "Business logic tests failed"
    testing::phase::end_with_summary "Business tests failed"
    exit 1
fi

# Verify test coverage for business logic
BUSINESS_COVERAGE=$(go test -run "TestBusiness" -cover 2>&1 | grep -oP 'coverage: \K[0-9.]+' || echo "0")
testing::phase::info "Business logic test coverage: ${BUSINESS_COVERAGE}%"

if (( $(echo "$BUSINESS_COVERAGE < 70" | bc -l) )); then
    testing::phase::warn "Business logic coverage is below 70%: ${BUSINESS_COVERAGE}%"
fi

testing::phase::success "Business logic tests passed"
testing::phase::end_with_summary "Business tests completed successfully"
