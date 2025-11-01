#!/bin/bash
# Business logic test phase - validates business rules and scenarios
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running business logic tests..."

# Test SaaS detection business logic
log::info "Testing SaaS detection business rules"
if cd api && go test -tags=testing -v -run "TestSaaSDetectionEdgeCases|TestCharacteristicsExtraction" -timeout 90s; then
    log::success "SaaS detection business logic tests passed"
else
    testing::phase::add_error "SaaS detection business logic tests failed"
fi

# Test landing page generation business logic
log::info "Testing landing page generation business rules"
if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && go test -tags=testing -v -run "TestLandingPageServiceComprehensive" -timeout 90s; then
    log::success "Landing page generation business logic tests passed"
else
    testing::phase::add_error "Landing page generation business logic tests failed"
fi

# Test deployment business logic
log::info "Testing deployment business rules"
if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && go test -tags=testing -v -run "TestDeploymentEdgeCases" -timeout 90s; then
    log::success "Deployment business logic tests passed"
else
    testing::phase::add_error "Deployment business logic tests failed"
fi

# Test A/B testing business logic
log::info "Testing A/B testing business rules"
if cd "${TESTING_PHASE_SCENARIO_DIR}/api" && go test -tags=testing -v -run "TestGenerateLandingPage_WithABTesting" -timeout 90s; then
    log::success "A/B testing business logic tests passed"
else
    testing::phase::add_error "A/B testing business logic tests failed"
fi

# End with summary
testing::phase::end_with_summary "Business logic tests completed"
