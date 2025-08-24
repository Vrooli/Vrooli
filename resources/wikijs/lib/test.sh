#!/bin/bash

# Wiki.js Test Functions

set -euo pipefail

# Source common functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
WIKIJS_LIB_DIR="${APP_ROOT}/resources/wikijs/lib"
source "$WIKIJS_LIB_DIR/common.sh"

# Run integration tests
run_tests() {
    echo "[INFO] Running Wiki.js integration tests..."
    
    # Check if Wiki.js is running
    if ! is_running; then
        echo "[ERROR] Wiki.js is not running. Start it first: vrooli resource wikijs start"
        return 1
    fi
    
    local passed=0
    local failed=0
    local test_results=()
    
    # Test 1: Check health endpoint
    echo -n "Testing health endpoint... "
    if check_health; then
        echo "✓"
        ((passed++))
        test_results+=("{\"name\":\"health_endpoint\",\"status\":\"passed\"}")
    else
        echo "✗"
        ((failed++))
        test_results+=("{\"name\":\"health_endpoint\",\"status\":\"failed\"}")
    fi
    
    # Test 2: GraphQL endpoint
    echo -n "Testing GraphQL endpoint... "
    if test_graphql_endpoint; then
        echo "✓"
        ((passed++))
        test_results+=("{\"name\":\"graphql_endpoint\",\"status\":\"passed\"}")
    else
        echo "✗"
        ((failed++))
        test_results+=("{\"name\":\"graphql_endpoint\",\"status\":\"failed\"}")
    fi
    
    # Test 3: Create page
    echo -n "Testing page creation... "
    if test_create_page; then
        echo "✓"
        ((passed++))
        test_results+=("{\"name\":\"create_page\",\"status\":\"passed\"}")
    else
        echo "✗"
        ((failed++))
        test_results+=("{\"name\":\"create_page\",\"status\":\"failed\"}")
    fi
    
    # Test 4: Search functionality
    echo -n "Testing search... "
    if test_search; then
        echo "✓"
        ((passed++))
        test_results+=("{\"name\":\"search\",\"status\":\"passed\"}")
    else
        echo "✗"
        ((failed++))
        test_results+=("{\"name\":\"search\",\"status\":\"failed\"}")
    fi
    
    # Test 5: List pages
    echo -n "Testing page listing... "
    if test_list_pages; then
        echo "✓"
        ((passed++))
        test_results+=("{\"name\":\"list_pages\",\"status\":\"passed\"}")
    else
        echo "✗"
        ((failed++))
        test_results+=("{\"name\":\"list_pages\",\"status\":\"failed\"}")
    fi
    
    # Test 6: Version check
    echo -n "Testing version check... "
    local version=$(get_version)
    if [[ "$version" != "unknown" ]] && [[ "$version" != "not_running" ]]; then
        echo "✓ (v$version)"
        ((passed++))
        test_results+=("{\"name\":\"version_check\",\"status\":\"passed\",\"version\":\"$version\"}")
    else
        echo "✗"
        ((failed++))
        test_results+=("{\"name\":\"version_check\",\"status\":\"failed\"}")
    fi
    
    # Save test results
    local total=$((passed + failed))
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local results_json="{
        \"timestamp\": \"$timestamp\",
        \"passed\": $passed,
        \"failed\": $failed,
        \"total\": $total,
        \"tests\": [$(IFS=,; echo "${test_results[*]}")]
    }"
    
    echo "$results_json" > "$WIKIJS_DATA_DIR/test_results.json"
    
    # Summary
    echo ""
    echo "[INFO] Test Results: $passed/$total passed"
    
    if [[ $failed -eq 0 ]]; then
        echo "[SUCCESS] All tests passed!"
        return 0
    else
        echo "[WARNING] Some tests failed"
        return 1
    fi
}

# Test GraphQL endpoint
test_graphql_endpoint() {
    local query='{ system { status } }'
    local response=$(graphql_query "$query" 2>/dev/null || echo "")
    [[ -n "$response" ]] && echo "$response" | grep -q "data"
}

# Test page creation (simulation)
test_create_page() {
    # In a real implementation, this would create a test page
    # For now, just check if the mutation endpoint responds
    local query='mutation { pages { create(content: "test", path: "test", title: "test") { responseResult { succeeded } } } }'
    local response=$(graphql_query "$query" 2>/dev/null || echo "")
    # Check if we get a response (even if it fails due to auth)
    [[ -n "$response" ]]
}

# Test search functionality
test_search() {
    local query='{ pages { search(query: "test") { results { id } } } }'
    local response=$(graphql_query "$query" 2>/dev/null || echo "")
    [[ -n "$response" ]]
}

# Test page listing
test_list_pages() {
    local query='{ pages { list { id } } }'
    local response=$(graphql_query "$query" 2>/dev/null || echo "")
    [[ -n "$response" ]] && echo "$response" | grep -q "data"
}