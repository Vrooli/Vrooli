#!/bin/bash
# Test script for Resume Screening Assistant API search endpoint

set -euo pipefail

# Configuration
API_PORT="${API_PORT:-8090}"
API_BASE_URL="http://localhost:${API_PORT}"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test functions
test_log() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

test_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

test_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    exit 1
}

# Check if API is running
check_api() {
    test_log "Checking if API is running..."
    if curl -s -f "$API_BASE_URL/health" > /dev/null 2>&1; then
        test_pass "API is responding"
    else
        test_fail "API is not responding at $API_BASE_URL"
    fi
}

# Test search endpoint with different parameters
test_search_endpoint() {
    test_log "Testing search endpoint with query 'python'"
    
    local response
    response=$(curl -s "$API_BASE_URL/api/search?query=python&type=both")
    
    # Check if response is valid JSON
    if echo "$response" | jq empty 2>/dev/null; then
        test_pass "Search endpoint returns valid JSON"
    else
        test_fail "Search endpoint did not return valid JSON"
    fi
    
    # Check if response has expected structure
    local success
    success=$(echo "$response" | jq -r '.success // false')
    if [[ "$success" == "true" ]]; then
        test_pass "Search endpoint returns success: true"
    else
        test_fail "Search endpoint did not return success: true"
    fi
    
    # Check if results array exists
    local has_results
    has_results=$(echo "$response" | jq -r 'has("results")')
    if [[ "$has_results" == "true" ]]; then
        test_pass "Search response contains results array"
    else
        test_fail "Search response missing results array"
    fi
    
    # Check count field
    local has_count
    has_count=$(echo "$response" | jq -r 'has("count")')
    if [[ "$has_count" == "true" ]]; then
        test_pass "Search response contains count field"
    else
        test_fail "Search response missing count field"
    fi
    
    test_log "Search endpoint test completed successfully"
}

# Test different search types
test_search_types() {
    test_log "Testing search with type='candidates'"
    local candidates_response
    candidates_response=$(curl -s "$API_BASE_URL/api/search?query=engineer&type=candidates")
    
    if echo "$candidates_response" | jq empty 2>/dev/null; then
        test_pass "Candidates-only search returns valid JSON"
    else
        test_fail "Candidates-only search failed"
    fi
    
    test_log "Testing search with type='jobs'"
    local jobs_response
    jobs_response=$(curl -s "$API_BASE_URL/api/search?query=engineer&type=jobs")
    
    if echo "$jobs_response" | jq empty 2>/dev/null; then
        test_pass "Jobs-only search returns valid JSON"
    else
        test_fail "Jobs-only search failed"
    fi
}

# Test empty query handling
test_empty_query() {
    test_log "Testing search with empty query"
    local empty_response
    empty_response=$(curl -s "$API_BASE_URL/api/search?query=")
    
    if echo "$empty_response" | jq empty 2>/dev/null; then
        local count
        count=$(echo "$empty_response" | jq -r '.count // 0')
        if [[ "$count" == "0" ]]; then
            test_pass "Empty query returns 0 results"
        else
            test_log "Note: Empty query returned $count results (this may be expected behavior)"
        fi
    else
        test_fail "Empty query search failed"
    fi
}

# Main test execution
main() {
    echo "Resume Screening Assistant API - Search Endpoint Tests"
    echo "====================================================="
    
    check_api
    test_search_endpoint
    test_search_types
    test_empty_query
    
    echo
    echo "All search endpoint tests passed! ✅"
}

# Run tests
main "$@"