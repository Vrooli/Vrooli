#!/bin/bash
# Integration test phase for scalable-app-cookbook
# Tests API endpoints with real database

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::section "Integration Tests"

# Check if scenario is running
if ! vrooli scenario status scalable-app-cookbook | grep -q "running"; then
    testing::phase::error "Scenario must be running for integration tests"
    exit 1
fi

# Get API port from environment or default
API_PORT="${API_PORT:-3300}"
API_URL="http://localhost:${API_PORT}"

testing::phase::info "Testing API at ${API_URL}"

# Test health endpoint
testing::phase::subsection "Health Check"
if curl -sf "${API_URL}/health" | grep -q "healthy"; then
    testing::phase::success "Health check passed"
else
    testing::phase::error "Health check failed"
    exit 1
fi

# Test pattern search endpoint
testing::phase::subsection "Pattern Search API"
if curl -sf "${API_URL}/api/v1/patterns/search" | grep -q "patterns"; then
    testing::phase::success "Pattern search endpoint working"
else
    testing::phase::error "Pattern search endpoint failed"
    exit 1
fi

# Test chapters endpoint
testing::phase::subsection "Chapters API"
if curl -sf "${API_URL}/api/v1/patterns/chapters" >/dev/null; then
    testing::phase::success "Chapters endpoint working"
else
    testing::phase::error "Chapters endpoint failed"
    exit 1
fi

# Test stats endpoint
testing::phase::subsection "Statistics API"
if curl -sf "${API_URL}/api/v1/patterns/stats" | grep -q "statistics"; then
    testing::phase::success "Statistics endpoint working"
else
    testing::phase::error "Statistics endpoint failed"
    exit 1
fi

# Test error handling - invalid pattern ID
testing::phase::subsection "Error Handling"
STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/api/v1/patterns/non-existent-pattern-id")
if [ "$STATUS_CODE" = "404" ]; then
    testing::phase::success "404 error handling working"
else
    testing::phase::warning "Expected 404, got ${STATUS_CODE}"
fi

# Test pagination
testing::phase::subsection "Pagination"
if curl -sf "${API_URL}/api/v1/patterns/search?limit=5&offset=0" | grep -q "limit"; then
    testing::phase::success "Pagination working"
else
    testing::phase::error "Pagination failed"
    exit 1
fi

testing::phase::end_with_summary "Integration tests completed"
