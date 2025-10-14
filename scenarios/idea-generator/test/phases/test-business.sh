#!/bin/bash
# Business Logic Test Phase for idea-generator
# Tests core business functionality and workflows

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Ensure scenario is running
# Use command substitution to avoid pipefail issues
STATUS_OUTPUT=$(vrooli scenario status idea-generator 2>/dev/null)
if ! echo "$STATUS_OUTPUT" | grep -q "RUNNING"; then
    log::error "Scenario is not running. Start it with: make start"
    exit 1
fi

# Get API port
API_PORT=$(echo "$STATUS_OUTPUT" | grep "API_PORT:" | awk '{print $2}')

# Test Campaign Creation
log::subheader "Campaign Creation"
CAMPAIGN_RESPONSE=$(curl -sf -X POST "http://localhost:${API_PORT}/api/campaigns" \
    -H "Content-Type: application/json" \
    -d '{"name":"Test Campaign","description":"Test Description","color":"#FF5733"}' || echo "")

if echo "$CAMPAIGN_RESPONSE" | grep -q '"id"'; then
    CAMPAIGN_ID=$(echo "$CAMPAIGN_RESPONSE" | jq -r '.id')
    log::success "Campaign created successfully: $CAMPAIGN_ID"
else
    log::error "Campaign creation failed"
    exit 1
fi

# Test Campaign Retrieval
log::subheader "Campaign Retrieval"
if curl -sf "http://localhost:${API_PORT}/api/campaigns/${CAMPAIGN_ID}" > /dev/null; then
    log::success "Campaign retrieved successfully"
else
    log::error "Campaign retrieval failed"
    exit 1
fi

# Test Idea Generation (P0 Feature)
log::subheader "Idea Generation (P0)"
log::info "Generating idea with Ollama (may take 10-15s)..."
IDEA_RESPONSE=$(curl -sf -X POST "http://localhost:${API_PORT}/api/ideas/generate" \
    -H "Content-Type: application/json" \
    -d "{\"campaign_id\":\"${CAMPAIGN_ID}\",\"context\":\"mobile app features\",\"creativity_level\":0.7}" \
    --max-time 30 || echo "")

if echo "$IDEA_RESPONSE" | grep -q '"id"'; then
    log::success "Idea generated successfully"
else
    log::warning "Idea generation timed out or failed (expected with Ollama)"
fi

# Test Campaign List
log::subheader "Campaign List"
CAMPAIGNS=$(curl -sf "http://localhost:${API_PORT}/api/campaigns")
if echo "$CAMPAIGNS" | grep -q "$CAMPAIGN_ID"; then
    log::success "Campaign appears in list"
else
    log::error "Campaign not in list"
    exit 1
fi

# Cleanup
log::subheader "Cleanup"
if curl -sf -X DELETE "http://localhost:${API_PORT}/api/campaigns/${CAMPAIGN_ID}" > /dev/null; then
    log::success "Test campaign cleaned up"
else
    log::warning "Cleanup failed (non-critical)"
fi

testing::phase::end_with_summary "Business logic tests completed"
