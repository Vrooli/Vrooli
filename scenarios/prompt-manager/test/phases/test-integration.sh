#!/bin/bash
# Integration test phase - tests API endpoints with real database
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 120-second target
testing::phase::init --target-time "120s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running integration tests for prompt-manager..."

# Test database connection
if ! testing::phase::require_env TEST_POSTGRES_URL "PostgreSQL test database URL"; then
    testing::phase::add_error "TEST_POSTGRES_URL not configured"
    testing::phase::end_with_summary "Integration tests skipped - database not configured"
    exit 1
fi

# Run Go integration tests (use main_test.go which includes integration scenarios)
log::info "Running Go integration tests..."
if cd api && go test -v -tags=integration -run="TestCampaign|TestPrompt|TestSearch|TestExport" -timeout=120s 2>&1 | tee /tmp/integration-test-output.txt; then
    log::success "Integration tests passed"
else
    testing::phase::add_error "Integration tests failed"
    cat /tmp/integration-test-output.txt
fi

# Test API health endpoint
log::info "Testing API health endpoint..."
if curl -sf "http://localhost:${API_PORT:-15280}/health" &> /dev/null; then
    log::success "API health check passed"
else
    log::warn "API not running - skipping live endpoint tests"
fi

# Test campaigns endpoint
if [ -n "${API_PORT}" ]; then
    log::info "Testing campaigns endpoint..."
    if curl -sf "http://localhost:${API_PORT}/api/v1/campaigns" &> /dev/null; then
        log::success "Campaigns endpoint accessible"
    else
        testing::phase::add_error "Campaigns endpoint not accessible"
    fi
fi

# End with summary
testing::phase::end_with_summary "Integration tests completed"
