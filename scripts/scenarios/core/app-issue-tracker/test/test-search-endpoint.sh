#!/usr/bin/env bash
# Test semantic search functionality

set -euo pipefail

API_URL="${ISSUE_TRACKER_API_URL:-http://localhost:8090/api}"

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

# Test search endpoint
test_search() {
    local query="$1"
    local expected_count="$2"
    
    log_info "Testing search for: '$query'"
    
    local response
    response=$(curl -s "$API_URL/issues/search?q=$(echo "$query" | jq -sRr @uri)")
    
    local success
    success=$(echo "$response" | jq -r '.success')
    
    if [ "$success" = "true" ]; then
        local count
        count=$(echo "$response" | jq -r '.data.count')
        
        if [ "$count" -ge "$expected_count" ]; then
            log_success "Search returned $count results (expected >= $expected_count)"
            return 0
        else
            log_error "Search returned $count results (expected >= $expected_count)"
            return 1
        fi
    else
        local message
        message=$(echo "$response" | jq -r '.message // "Unknown error"')
        log_error "Search failed: $message"
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

# Run search tests
log_info "Running search endpoint tests..."

# Test basic search
test_search "test" 0

# Test common error keywords
test_search "error" 0
test_search "bug" 0
test_search "authentication" 0

# Test empty query handling
log_info "Testing empty query..."
response=$(curl -s "$API_URL/issues/search")
if echo "$response" | grep -q "Query parameter.*required"; then
    log_success "Empty query properly rejected"
else
    log_error "Empty query should be rejected"
    exit 1
fi

log_success "All search endpoint tests passed!"