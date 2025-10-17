#!/bin/bash
# Integration test phase
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 120-second target
testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running integration tests for chore-tracking"

# Check if scenario is running
if ! vrooli scenario status chore-tracking | grep -q "running"; then
    log::warn "Scenario not running, starting it..."
    vrooli scenario start chore-tracking
    sleep 5
fi

# Test health endpoint
log::info "Testing health endpoint..."
if curl -sf http://localhost:${API_PORT:-8080}/health > /dev/null; then
    log::success "Health check passed"
else
    testing::phase::add_error "Health check failed"
fi

# Test API endpoints
log::info "Testing API endpoints..."

# Test GET /api/chores
if curl -sf http://localhost:${API_PORT:-8080}/api/chores > /dev/null; then
    log::success "GET /api/chores passed"
else
    testing::phase::add_error "GET /api/chores failed"
fi

# Test GET /api/users
if curl -sf http://localhost:${API_PORT:-8080}/api/users > /dev/null; then
    log::success "GET /api/users passed"
else
    testing::phase::add_error "GET /api/users failed"
fi

# Test GET /api/achievements
if curl -sf http://localhost:${API_PORT:-8080}/api/achievements > /dev/null; then
    log::success "GET /api/achievements passed"
else
    testing::phase::add_error "GET /api/achievements failed"
fi

# Test GET /api/rewards
if curl -sf http://localhost:${API_PORT:-8080}/api/rewards > /dev/null; then
    log::success "GET /api/rewards passed"
else
    testing::phase::add_error "GET /api/rewards failed"
fi

# End with summary
testing::phase::end_with_summary "Integration tests completed"
