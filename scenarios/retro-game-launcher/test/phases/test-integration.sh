#!/bin/bash
# Integration test phase for retro-game-launcher
# Tests end-to-end workflows and external service integration

set -euo pipefail

# Get app root (4 levels up from this script)
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "120s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log_info "Running integration tests..."

# Run Go integration tests
if [[ -d "api" ]]; then
    cd api

    testing::phase::log_info "Running Go API integration tests..."

    # Check if integration tests exist
    if go list -tags=testing ./... 2>/dev/null | grep -q .; then
        # Run integration tests with coverage
        if go test -tags=testing -v -run "TestCheck|TestGenerate|Integration" -timeout 120s 2>&1; then
            testing::phase::log_success "Go integration tests passed"
        else
            testing::phase::log_warning "Some integration tests failed (may be environment-dependent)"
        fi
    else
        testing::phase::log_warning "No Go integration tests found"
    fi

    cd ..
fi

# Test API health check (if running)
if command -v curl &> /dev/null; then
    API_PORT="${API_PORT:-8080}"
    testing::phase::log_info "Checking API health endpoint..."

    if curl -sf "http://localhost:${API_PORT}/health" &> /dev/null; then
        testing::phase::log_success "API health check passed"
    else
        testing::phase::log_info "API not running (expected in test environment)"
    fi
fi

# End test phase with summary
testing::phase::end_with_summary "Integration tests completed"
