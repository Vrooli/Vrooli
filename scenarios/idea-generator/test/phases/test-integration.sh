#!/bin/bash
# Integration Test Phase for idea-generator
# Tests API endpoints, database connections, and resource integrations

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Ensure scenario is running
# Use command substitution to avoid pipefail issues
STATUS_OUTPUT=$(vrooli scenario status idea-generator 2>/dev/null)
if ! echo "$STATUS_OUTPUT" | grep -q "RUNNING"; then
    log::error "Scenario is not running. Start it with: make start"
    exit 1
fi

# Get ports from status
API_PORT=$(echo "$STATUS_OUTPUT" | grep "API_PORT:" | awk '{print $2}')
UI_PORT=$(echo "$STATUS_OUTPUT" | grep "UI_PORT:" | awk '{print $2}')

log::info "Testing API on port ${API_PORT}"
log::info "Testing UI on port ${UI_PORT}"

# API Health Check
log::subheader "API Health Check"
if curl -sf "http://localhost:${API_PORT}/health" > /dev/null; then
    log::success "API health endpoint responsive"
else
    log::error "API health endpoint failed"
    exit 1
fi

# Test Campaigns Endpoint
log::subheader "Campaigns Endpoint"
if curl -sf "http://localhost:${API_PORT}/api/campaigns" > /dev/null; then
    log::success "Campaigns endpoint responsive"
else
    log::error "Campaigns endpoint failed"
    exit 1
fi

# Test Workflows Endpoint
log::subheader "Workflows Endpoint"
if curl -sf "http://localhost:${API_PORT}/api/workflows" > /dev/null; then
    log::success "Workflows endpoint responsive"
else
    log::error "Workflows endpoint failed"
    exit 1
fi

# UI Health Check
log::subheader "UI Health Check"
if curl -sf "http://localhost:${UI_PORT}/health" > /dev/null; then
    log::success "UI health endpoint responsive"
else
    log::error "UI health endpoint failed"
    exit 1
fi

# Test UI Page Load
log::subheader "UI Page Load"
if curl -sf "http://localhost:${UI_PORT}/" > /dev/null; then
    log::success "UI page loads successfully"
else
    log::error "UI page failed to load"
    exit 1
fi

# Test Database Connection
log::subheader "Database Integration"
HEALTH_RESPONSE=$(curl -sf "http://localhost:${API_PORT}/health")
if echo "$HEALTH_RESPONSE" | grep -q '"connected":true'; then
    log::success "Database connection verified"
else
    log::error "Database connection failed"
    exit 1
fi

testing::phase::end_with_summary "Integration tests completed"
