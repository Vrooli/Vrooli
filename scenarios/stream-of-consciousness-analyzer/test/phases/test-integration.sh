#!/bin/bash
# Integration test phase - tests API with database
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running integration tests..."

# Check if API is running
if ! curl -sf "http://localhost:${API_PORT:-18888}/health" &>/dev/null; then
    testing::phase::add_error "API not available for integration testing"
    testing::phase::end_with_summary "Integration tests skipped"
    exit 0
fi

# Test health endpoint
log::info "Testing health endpoint..."
if curl -sf "http://localhost:${API_PORT:-18888}/health" | grep -q "healthy"; then
    log::success "Health endpoint working"
else
    testing::phase::add_error "Health endpoint failed"
fi

# Test campaigns endpoint
log::info "Testing campaigns endpoint..."
if curl -sf "http://localhost:${API_PORT:-18888}/api/campaigns" &>/dev/null; then
    log::success "Campaigns endpoint working"
else
    testing::phase::add_error "Campaigns endpoint failed"
fi

# Test notes endpoint
log::info "Testing notes endpoint..."
if curl -sf "http://localhost:${API_PORT:-18888}/api/notes" &>/dev/null; then
    log::success "Notes endpoint working"
else
    testing::phase::add_error "Notes endpoint failed"
fi

# Test insights endpoint
log::info "Testing insights endpoint..."
if curl -sf "http://localhost:${API_PORT:-18888}/api/insights" &>/dev/null; then
    log::success "Insights endpoint working"
else
    testing::phase::add_error "Insights endpoint failed"
fi

# Test search endpoint (expect 400 without query param)
log::info "Testing search endpoint..."
if curl -sf "http://localhost:${API_PORT:-18888}/api/search" &>/dev/null; then
    log::warn "Search endpoint should require query parameter"
else
    log::success "Search endpoint validation working"
fi

testing::phase::end_with_summary "Integration tests completed"
