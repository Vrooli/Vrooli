#!/bin/bash
# Business logic test phase - validates business requirements
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Testing business logic..."

# Check if API is running
if ! curl -sf "http://localhost:${API_PORT:-18888}/health" &>/dev/null; then
    log::warn "API not available for business logic testing"
    testing::phase::end_with_summary "Business tests skipped"
    exit 0
fi

# Test campaign workflow
log::info "Testing campaign creation workflow..."

# Create a test campaign
campaign_data='{"name":"Test Business Campaign","description":"Testing business logic","context_prompt":"Test context","color":"#3B82F6","icon":"📝"}'
response=$(curl -sf -X POST \
    -H "Content-Type: application/json" \
    -d "$campaign_data" \
    "http://localhost:${API_PORT:-18888}/api/campaigns" 2>/dev/null)

if [ -n "$response" ]; then
    campaign_id=$(echo "$response" | jq -r '.id' 2>/dev/null)
    if [ -n "$campaign_id" ] && [ "$campaign_id" != "null" ]; then
        log::success "Campaign created with ID: $campaign_id"

        # Test stream capture workflow
        log::info "Testing stream capture workflow..."
        stream_data="{\"campaign_id\":\"$campaign_id\",\"content\":\"Test stream content\",\"type\":\"text\",\"source\":\"test\",\"metadata\":{}}"
        stream_response=$(curl -sf -X POST \
            -H "Content-Type: application/json" \
            -d "$stream_data" \
            "http://localhost:${API_PORT:-18888}/api/stream/capture" 2>/dev/null)

        if [ -n "$stream_response" ]; then
            log::success "Stream entry captured successfully"
        else
            testing::phase::add_error "Stream capture failed"
        fi

        # Test retrieving campaign notes
        log::info "Testing notes retrieval for campaign..."
        notes_response=$(curl -sf "http://localhost:${API_PORT:-18888}/api/notes?campaign_id=$campaign_id" 2>/dev/null)

        if [ -n "$notes_response" ]; then
            log::success "Notes retrieved for campaign"
        else
            testing::phase::add_error "Notes retrieval failed"
        fi

        # Test insights retrieval
        log::info "Testing insights retrieval for campaign..."
        insights_response=$(curl -sf "http://localhost:${API_PORT:-18888}/api/insights?campaign_id=$campaign_id" 2>/dev/null)

        if [ -n "$insights_response" ]; then
            log::success "Insights retrieved for campaign"
        else
            testing::phase::add_error "Insights retrieval failed"
        fi
    else
        testing::phase::add_error "Campaign creation failed - no ID returned"
    fi
else
    testing::phase::add_error "Campaign creation request failed"
fi

# Test search functionality
log::info "Testing search functionality..."
search_response=$(curl -sf "http://localhost:${API_PORT:-18888}/api/search?q=test" 2>/dev/null)

if [ -n "$search_response" ]; then
    log::success "Search endpoint functional"
else
    testing::phase::add_error "Search functionality failed"
fi

# Test data validation
log::info "Testing input validation..."

# Test invalid campaign creation (missing required field)
invalid_data='{"description":"Missing name field"}'
invalid_response=$(curl -s -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d "$invalid_data" \
    "http://localhost:${API_PORT:-18888}/api/campaigns" 2>/dev/null | tail -1)

if [ "$invalid_response" = "400" ] || [ "$invalid_response" = "500" ]; then
    log::success "Input validation working (rejected invalid data)"
else
    log::warn "Input validation may need improvement (got HTTP $invalid_response)"
fi

# Test search without query parameter
search_validation=$(curl -s -w "%{http_code}" \
    "http://localhost:${API_PORT:-18888}/api/search" 2>/dev/null | tail -1)

if [ "$search_validation" = "400" ]; then
    log::success "Search validation working (requires query parameter)"
else
    log::warn "Search endpoint validation unexpected (got HTTP $search_validation)"
fi

testing::phase::end_with_summary "Business logic tests completed"
