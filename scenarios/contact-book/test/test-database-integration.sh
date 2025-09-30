#!/bin/bash

# Contact Book Database Integration Tests
# Tests the PostgreSQL schema, API endpoints, and CLI functionality

set -e

# Test configuration
# Try to get the actual running port from the API process
API_PORT=$(lsof -i -P | grep contact-b | grep LISTEN | awk '{print $9}' | cut -d: -f2 | head -1)
API_PORT="${API_PORT:-8080}"  # Fallback to 8080 if not found
API_BASE_URL="${CONTACT_BOOK_API_URL:-http://localhost:${API_PORT}}"
TEST_TIMEOUT=30
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test results
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# =============================================================================
# TEST UTILITIES
# =============================================================================

log_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1" >&2
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

# Wait for API to be ready
wait_for_api() {
    local attempts=0
    local max_attempts=10
    
    echo "Waiting for Contact Book API to be ready..."
    while [[ $attempts -lt $max_attempts ]]; do
        if curl -sf "${API_BASE_URL}/health" >/dev/null 2>&1; then
            echo "API is ready!"
            return 0
        fi
        attempts=$((attempts + 1))
        echo "Waiting... (attempt $attempts/$max_attempts)"
        sleep 3
    done
    
    log_fail "API failed to start within $((max_attempts * 3)) seconds"
    return 1
}

# Make API request and check response
api_test() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"
    local expected_status="$5"
    
    log_test "$name"
    
    local curl_args=(-s -w "%{http_code}" -o /tmp/api_response.json)
    [[ "$method" != "GET" ]] && curl_args+=(-X "$method")
    [[ -n "$data" ]] && curl_args+=(-H "Content-Type: application/json" -d "$data")
    
    local status
    status=$(curl "${curl_args[@]}" "${API_BASE_URL}${endpoint}")
    
    if [[ "$status" == "$expected_status" ]]; then
        log_pass "$name (HTTP $status)"
        return 0
    else
        log_fail "$name (Expected HTTP $expected_status, got $status)"
        echo "Response:" >&2
        cat /tmp/api_response.json >&2
        echo "" >&2
        return 1
    fi
}

# =============================================================================
# DATABASE SCHEMA TESTS
# =============================================================================

test_database_schema() {
    echo ""
    echo "=== Database Schema Tests ==="

    # Test database connectivity through the API health check
    log_test "Database connectivity"
    local health_response
    health_response=$(curl -sf "${API_BASE_URL}/health" 2>/dev/null || echo "{}")

    if echo "$health_response" | grep -q '"database":"healthy"'; then
        log_pass "Database is connected and healthy"
    else
        log_fail "Database connection issue"
        return 1
    fi

    # Test that data exists via the API
    log_test "Sample data exists"
    local contacts_response
    contacts_response=$(curl -sf "${API_BASE_URL}/api/v1/contacts" 2>/dev/null || echo "{}")

    local person_count=0
    if command -v jq >/dev/null 2>&1; then
        person_count=$(echo "$contacts_response" | jq -r '.count // 0')
    else
        # Fallback to grep if jq not available
        person_count=$(echo "$contacts_response" | grep -o '"count":[0-9]*' | sed 's/"count"://')
    fi

    if [[ "$person_count" -gt 0 ]]; then
        log_pass "Sample data exists ($person_count persons)"
    else
        log_fail "No sample data found"
    fi
}

# =============================================================================
# API ENDPOINT TESTS
# =============================================================================

test_api_endpoints() {
    echo ""
    echo "=== API Endpoint Tests ==="
    
    # Health check
    api_test "Health check endpoint" "GET" "/health" "" "200"
    
    # List contacts
    api_test "List contacts endpoint" "GET" "/api/v1/contacts" "" "200"
    
    # Search contacts
    api_test "Search contacts endpoint" "POST" "/api/v1/search" '{"query": "sarah", "limit": 10}' "200"
    
    # List relationships
    api_test "List relationships endpoint" "GET" "/api/v1/relationships" "" "200"
    
    # Get analytics
    api_test "Analytics endpoint" "GET" "/api/v1/analytics" "" "200"
    
    # Test creating a contact
    local test_contact='{"full_name": "Test User", "emails": ["test@example.com"], "tags": ["test"]}'
    api_test "Create contact endpoint" "POST" "/api/v1/contacts" "$test_contact" "201"
    
    # Get the created contact ID
    if [[ -f /tmp/api_response.json ]]; then
        local contact_id
        if command -v jq >/dev/null 2>&1; then
            contact_id=$(jq -r '.id' /tmp/api_response.json)
        else
            contact_id=$(grep -o '"id":"[^"]*"' /tmp/api_response.json | cut -d'"' -f4)
        fi
        
        if [[ -n "$contact_id" && "$contact_id" != "null" ]]; then
            # Test getting the specific contact
            api_test "Get specific contact" "GET" "/api/v1/contacts/$contact_id" "" "200"
            
            # Test creating a relationship (need another contact first)
            local test_contact2='{"full_name": "Test User 2", "emails": ["test2@example.com"]}'
            if api_test "Create second contact" "POST" "/api/v1/contacts" "$test_contact2" "201" >/dev/null 2>&1; then
                local contact_id2
                if command -v jq >/dev/null 2>&1; then
                    contact_id2=$(jq -r '.id' /tmp/api_response.json)
                else
                    contact_id2=$(grep -o '"id":"[^"]*"' /tmp/api_response.json | cut -d'"' -f4)
                fi
                
                if [[ -n "$contact_id2" && "$contact_id2" != "null" ]]; then
                    local test_relationship="{\"from_person_id\": \"$contact_id\", \"to_person_id\": \"$contact_id2\", \"relationship_type\": \"test\", \"strength\": 0.5}"
                    api_test "Create relationship" "POST" "/api/v1/relationships" "$test_relationship" "201"
                fi
            fi
        fi
    fi
}

# =============================================================================
# CLI TESTS
# =============================================================================

test_cli_functionality() {
    echo ""
    echo "=== CLI Functionality Tests ==="
    
    # Check if CLI is installed and executable
    log_test "CLI executable exists"
    if command -v contact-book >/dev/null 2>&1; then
        log_pass "CLI is in PATH"
    elif [[ -x "./cli/contact-book" ]]; then
        log_pass "CLI executable exists locally"
        # Use local CLI for tests
        alias contact-book="./cli/contact-book"
    else
        log_fail "CLI not found"
        return 1
    fi
    
    # Test CLI commands
    log_test "CLI version command"
    if contact-book version >/dev/null 2>&1; then
        log_pass "CLI version command works"
    else
        log_fail "CLI version command failed"
    fi
    
    log_test "CLI status command"
    if timeout 10 contact-book status >/dev/null 2>&1; then
        log_pass "CLI status command works"
    else
        log_fail "CLI status command failed or timed out"
    fi
    
    log_test "CLI list command"
    if timeout 10 contact-book list --limit 5 >/dev/null 2>&1; then
        log_pass "CLI list command works"
    else
        log_fail "CLI list command failed or timed out"
    fi
    
    log_test "CLI search command"
    if timeout 10 contact-book search "test" --limit 2 >/dev/null 2>&1; then
        log_pass "CLI search command works"
    else
        log_fail "CLI search command failed or timed out"
    fi
}

# =============================================================================
# INTEGRATION TESTS
# =============================================================================

test_cross_scenario_integration() {
    echo ""
    echo "=== Cross-Scenario Integration Tests ==="
    
    # Test JSON output for programmatic use
    log_test "JSON output format"
    local json_output
    json_output=$(timeout 10 contact-book list --json --limit 1 2>/dev/null || echo "")
    
    if [[ -n "$json_output" ]]; then
        if command -v jq >/dev/null 2>&1; then
            if echo "$json_output" | jq . >/dev/null 2>&1; then
                log_pass "CLI produces valid JSON output"
            else
                log_fail "CLI JSON output is invalid"
            fi
        else
            log_pass "CLI produces JSON output (jq not available for validation)"
        fi
    else
        log_fail "CLI failed to produce JSON output"
    fi
    
    # Test analytics for other scenarios
    log_test "Analytics data availability"
    local analytics_output
    analytics_output=$(timeout 10 contact-book analytics --json 2>/dev/null || echo "")
    
    if [[ -n "$analytics_output" ]]; then
        log_pass "Analytics data accessible via CLI"
    else
        log_fail "Analytics data not accessible"
    fi
}

# =============================================================================
# MAIN TEST EXECUTION
# =============================================================================

run_all_tests() {
    echo "🧪 Contact Book Integration Tests"
    echo "================================="
    
    # Wait for API to be ready
    if ! wait_for_api; then
        echo ""
        echo "❌ Tests cannot proceed - API not available"
        exit 1
    fi
    
    # Run test suites
    test_database_schema
    test_api_endpoints
    test_cli_functionality
    test_cross_scenario_integration
    
    # Summary
    echo ""
    echo "=== Test Summary ==="
    echo "Total Tests: $TESTS_TOTAL"
    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo ""
        echo -e "${GREEN}✅ All tests passed!${NC}"
        echo ""
        echo "🚀 Contact Book is ready for cross-scenario integration!"
        echo "   • API available at: $API_BASE_URL"
        echo "   • CLI command: contact-book"
        echo "   • Example usage: contact-book search 'john' --json"
        exit 0
    else
        echo ""
        echo -e "${RED}❌ Some tests failed!${NC}"
        echo "Please check the errors above and ensure:"
        echo "  1. PostgreSQL is running and schema is initialized"
        echo "  2. Contact Book API is running on port 8080"
        echo "  3. CLI is properly installed"
        exit 1
    fi
}

# Clean up temporary files on exit
cleanup() {
    rm -f /tmp/api_response.json
}
trap cleanup EXIT

# Run tests
run_all_tests