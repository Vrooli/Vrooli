#!/bin/bash
# Test dependencies phase - validates scenario dependencies

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Testing smart-shopping-assistant dependencies"

# Test required resources
testing::phase::add_test "Check PostgreSQL dependency"
if command -v psql &> /dev/null; then
    log::success "PostgreSQL client available"
else
    testing::phase::add_warning "PostgreSQL client not found"
fi

testing::phase::add_test "Check Redis dependency"
if command -v redis-cli &> /dev/null; then
    log::success "Redis client available"
else
    testing::phase::add_warning "Redis client not found"
fi

# Test Go dependencies
testing::phase::add_test "Verify Go module dependencies"
if [[ -f "api/go.mod" ]]; then
    cd api
    if go mod verify; then
        log::success "Go dependencies verified"
    else
        testing::phase::add_error "Go dependency verification failed"
    fi
    cd ..
else
    testing::phase::add_warning "No Go module file found"
fi

# Test Node dependencies if UI exists
testing::phase::add_test "Check Node dependencies"
if [[ -f "ui/package.json" ]]; then
    cd ui
    if [[ -d "node_modules" ]]; then
        log::success "Node modules installed"
    else
        testing::phase::add_warning "Node modules not installed"
    fi
    cd ..
fi

# Check for integration scenarios
testing::phase::add_test "Check scenario-authenticator integration"
if [[ -f "../scenario-authenticator/.vrooli/service.json" ]]; then
    log::success "scenario-authenticator available for integration"
else
    log::info "scenario-authenticator not available (optional)"
fi

testing::phase::end_with_summary "Dependency tests completed"
