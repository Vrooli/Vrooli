#!/bin/bash
# Integration test phase - tests interactions between components
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running integration tests..."

# Test API integration with database
log::step "Testing API and database integration"
if cd api && go test -tags=testing -v -run "TestDatabaseService|TestSaaSDetectionService|TestLandingPageService" -timeout 120s; then
    log::success "Database integration tests passed"
else
    testing::phase::add_error "Database integration tests failed"
fi

# Test handler integration
log::step "Testing HTTP handler integration"
if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && go test -tags=testing -v -run "TestScanScenariosHandler|TestGenerateLandingPageHandler|TestDeployLandingPageHandler" -timeout 120s; then
    log::success "Handler integration tests passed"
else
    testing::phase::add_error "Handler integration tests failed"
fi

# Test service integration
log::step "Testing service-to-service integration"
if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && go test -tags=testing -v -run "TestClaudeCodeService" -timeout 120s; then
    log::success "Service integration tests passed"
else
    testing::phase::add_error "Service integration tests failed"
fi

# End with summary
testing::phase::end_with_summary "Integration tests completed"
