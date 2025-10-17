#!/usr/bin/env bash
################################################################################
# Wiki.js Integration Test - v2.0 Universal Contract Compliant
# 
# End-to-end functionality testing that must complete in <120 seconds
#
################################################################################

set -euo pipefail

# Resolve paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source common functions
source "${RESOURCE_DIR}/lib/common.sh"

# Test timeout
TIMEOUT=120

# ANSI color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test result tracking
TESTS_PASSED=0
TESTS_FAILED=0

# GraphQL test query
test_graphql_query() {
    local port=$(get_wikijs_port)
    local query='{"query":"{info{version title}}"}'
    
    timeout 5 curl -sf \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$query" \
        "http://localhost:${port}/graphql" 2>/dev/null
}

# Test page creation via GraphQL
test_create_page() {
    local port=$(get_wikijs_port)
    local mutation='{
        "query": "mutation { pages { create(content: \"Test content\", description: \"Test page\", editor: \"markdown\", isPublished: true, isPrivate: false, locale: \"en\", path: \"test-page\", tags: [], title: \"Test Page\") { responseResult { succeeded errorCode slug message } page { id } } } }"
    }'
    
    # Note: This will likely fail without authentication, but we're testing the endpoint exists
    timeout 5 curl -sf \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$mutation" \
        "http://localhost:${port}/graphql" 2>/dev/null | grep -q "responseResult"
}

# Test search functionality
test_search() {
    local port=$(get_wikijs_port)
    local query='{"query":"{pages{search(query:\"test\"){results{id title path}totalHits}}}"}'
    
    timeout 5 curl -sf \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$query" \
        "http://localhost:${port}/graphql" 2>/dev/null
}

# Run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "  Testing ${test_name}... "
    
    if bash -c "$test_command" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Main integration test
main() {
    echo "Running Wiki.js integration tests..."
    echo ""
    
    # Check if Wiki.js is running
    if ! is_running; then
        echo -e "${YELLOW}[WARNING]${NC} Wiki.js is not running. Starting it for tests..."
        if ! start_wikijs; then
            echo -e "${RED}[ERROR]${NC} Failed to start Wiki.js"
            return 1
        fi
        # Wait for startup
        sleep 10
    fi
    
    # Test 1: GraphQL info query
    run_test "GraphQL info query" "test_graphql_query" || true
    
    # Test 2: Health check response time
    local port=$(get_wikijs_port)
    run_test "health response time" "timeout 1 curl -sf http://localhost:${port}/ >/dev/null" || true
    
    # Test 3: Static assets
    run_test "static assets" "timeout 5 curl -sf http://localhost:${port}/js/app.js >/dev/null" || true
    
    # Test 4: GraphQL endpoint structure
    run_test "GraphQL structure" "test_create_page" || true
    
    # Test 5: Search endpoint
    run_test "search endpoint" "test_search" || true
    
    # Test 6: Database connectivity
    run_test "database connection" "docker logs wikijs 2>&1 | grep -q 'Database connection successful\\|Connected to database'" || true
    
    # Test 7: Container restart
    echo -n "  Testing container restart... "
    if docker restart wikijs &>/dev/null && sleep 15 && is_running; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
    
    # Test 8: Data persistence
    run_test "data persistence" "[[ -d '$WIKIJS_DATA_DIR/data' ]] && [[ -d '$WIKIJS_DATA_DIR/content' ]]" || true
    
    echo ""
    echo "Integration test summary: ${TESTS_PASSED} passed, ${TESTS_FAILED} failed"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        return 1
    fi
    
    return 0
}

# Execute integration tests
main "$@"