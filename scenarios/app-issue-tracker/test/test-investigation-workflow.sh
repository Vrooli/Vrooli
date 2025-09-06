#!/usr/bin/env bash
# Test investigation workflow

set -euo pipefail

API_URL="${ISSUE_TRACKER_API_URL:-http://localhost:8090/api}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

# Test investigation workflow trigger
test_investigation_trigger() {
    local issue_id="$1"
    
    log_info "Testing investigation workflow trigger for issue: $issue_id"
    
    local payload
    payload=$(jq -n \
        --arg id "$issue_id" \
        --arg agent "test-agent" \
        '{
            issue_id: $id,
            agent_id: $agent,
            priority: "normal"
        }')
    
    local response
    response=$(curl -s -X POST "$API_URL/investigate" \
        -H "Content-Type: application/json" \
        -d "$payload")
    
    local success
    success=$(echo "$response" | jq -r '.success')
    
    if [ "$success" = "true" ]; then
        local run_id
        run_id=$(echo "$response" | jq -r '.data.run_id // empty')
        local investigation_id
        investigation_id=$(echo "$response" | jq -r '.data.investigation_id // empty')
        
        if [ -n "$run_id" ] && [ -n "$investigation_id" ]; then
            log_success "Investigation triggered successfully"
            log_info "Run ID: $run_id"
            log_info "Investigation ID: $investigation_id"
            return 0
        else
            log_error "Investigation response missing required fields"
            return 1
        fi
    else
        local message
        message=$(echo "$response" | jq -r '.message // "Unknown error"')
        log_error "Investigation trigger failed: $message"
        return 1
    fi
}

# Test investigation script availability
test_script_availability() {
    log_info "Testing investigation script availability..."
    
    if [ -f "$SCRIPT_DIR/scripts/claude-investigator.sh" ]; then
        log_success "Investigation script is available"
        return 0
    else
        log_error "Investigation script not found at $SCRIPT_DIR/scripts/claude-investigator.sh"
        return 1
    fi
}

# Test API health first
log_info "Testing API health..."
if ! curl -sf "$API_URL/../health" > /dev/null; then
    log_error "API is not available at $API_URL"
    exit 1
fi
log_success "API is healthy"

# Test investigation script
if ! test_script_availability; then
    log_error "Investigation script tests failed"
    exit 1
fi

# Create a test issue first
log_info "Creating test issue for investigation..."
test_issue_payload=$(jq -n '{
    title: "Test Investigation Issue",
    description: "This is a test issue for investigation workflow testing",
    type: "bug",
    priority: "medium",
    error_message: "Test error message",
    tags: ["test", "investigation"]
}')

create_response=$(curl -s -X POST "$API_URL/issues" \
    -H "Content-Type: application/json" \
    -d "$test_issue_payload")

create_success=$(echo "$create_response" | jq -r '.success')

if [ "$create_success" = "true" ]; then
    test_issue_id=$(echo "$create_response" | jq -r '.data.issue_id')
    log_success "Test issue created: $test_issue_id"
    
    # Test investigation trigger
    if test_investigation_trigger "$test_issue_id"; then
        log_success "Investigation workflow tests passed!"
    else
        log_error "Investigation workflow tests failed"
        exit 1
    fi
else
    log_error "Failed to create test issue"
    echo "$create_response" | jq -r '.message // "Unknown error"'
    exit 1
fi

log_success "All investigation workflow tests passed!"