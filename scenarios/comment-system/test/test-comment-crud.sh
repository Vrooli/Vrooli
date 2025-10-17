#!/usr/bin/env bash

# Comment System CRUD Operations Test
# Tests basic comment creation, reading, updating, and deletion

set -e

API_URL="${API_URL:-http://localhost:8080}"
TEST_SCENARIO="crud-test-scenario"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test counter
TEST_COUNT=0
PASSED_COUNT=0
FAILED_COUNT=0

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TEST_COUNT=$((TEST_COUNT + 1))
    echo
    log_info "Test $TEST_COUNT: $test_name"
    
    if eval "$test_command"; then
        log_success "✓ PASSED"
        PASSED_COUNT=$((PASSED_COUNT + 1))
        return 0
    else
        log_error "✗ FAILED"
        FAILED_COUNT=$((FAILED_COUNT + 1))
        return 1
    fi
}

# HTTP request helper
api_request() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    local expected_status="${4:-200}"
    
    local response
    local status_code
    
    if [[ -n "$data" ]]; then
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
            -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$API_URL$endpoint")
    else
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
            -X "$method" \
            "$API_URL$endpoint")
    fi
    
    status_code=$(echo "$response" | grep -o 'HTTPSTATUS:[0-9]*' | cut -d: -f2)
    response_body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
    
    echo "Response: $response_body"
    echo "Status: $status_code"
    
    if [[ "$status_code" != "$expected_status" ]]; then
        log_error "Expected status $expected_status, got $status_code"
        return 1
    fi
    
    return 0
}

# Test API health
test_api_health() {
    api_request "GET" "/health" "" "200"
}

# Test getting empty comments list
test_empty_comments_list() {
    local response
    response=$(api_request "GET" "/api/v1/comments/$TEST_SCENARIO" "" "200")
    
    # Check if comments array is empty
    if echo "$response" | grep -q '"comments":\[\]'; then
        return 0
    else
        log_error "Expected empty comments array"
        return 1
    fi
}

# Test creating a comment
test_create_comment() {
    local comment_data='{"content": "This is a test comment", "content_type": "plaintext"}'
    local response
    
    response=$(api_request "POST" "/api/v1/comments/$TEST_SCENARIO" "$comment_data" "201")
    
    # Extract comment ID for later tests
    COMMENT_ID=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    if [[ -n "$COMMENT_ID" ]]; then
        log_info "Created comment with ID: $COMMENT_ID"
        return 0
    else
        log_error "Failed to extract comment ID from response"
        return 1
    fi
}

# Test creating a markdown comment
test_create_markdown_comment() {
    local comment_data='{"content": "This is a **markdown** comment with `code`", "content_type": "markdown"}'
    local response
    
    response=$(api_request "POST" "/api/v1/comments/$TEST_SCENARIO" "$comment_data" "201")
    
    # Extract comment ID
    MARKDOWN_COMMENT_ID=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    if [[ -n "$MARKDOWN_COMMENT_ID" ]]; then
        log_info "Created markdown comment with ID: $MARKDOWN_COMMENT_ID"
        return 0
    else
        log_error "Failed to create markdown comment"
        return 1
    fi
}

# Test creating a reply comment
test_create_reply_comment() {
    if [[ -z "$COMMENT_ID" ]]; then
        log_error "No parent comment ID available for reply test"
        return 1
    fi
    
    local reply_data="{\"content\": \"This is a reply\", \"parent_id\": \"$COMMENT_ID\", \"content_type\": \"plaintext\"}"
    local response
    
    response=$(api_request "POST" "/api/v1/comments/$TEST_SCENARIO" "$reply_data" "201")
    
    # Extract reply ID
    REPLY_ID=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    if [[ -n "$REPLY_ID" ]]; then
        log_info "Created reply comment with ID: $REPLY_ID"
        return 0
    else
        log_error "Failed to create reply comment"
        return 1
    fi
}

# Test listing comments
test_list_comments() {
    local response
    response=$(api_request "GET" "/api/v1/comments/$TEST_SCENARIO" "" "200")
    
    # Check if we have comments in the response
    if echo "$response" | grep -q '"comments":\['; then
        local comment_count
        comment_count=$(echo "$response" | grep -o '"id":"[^"]*"' | wc -l)
        log_info "Found $comment_count comments"
        
        if [[ "$comment_count" -gt 0 ]]; then
            return 0
        else
            log_error "No comments found in response"
            return 1
        fi
    else
        log_error "Invalid comments response format"
        return 1
    fi
}

# Test pagination
test_pagination() {
    local response
    response=$(api_request "GET" "/api/v1/comments/$TEST_SCENARIO?limit=1" "" "200")
    
    # Check if pagination info is included
    if echo "$response" | grep -q '"total_count"' && echo "$response" | grep -q '"has_more"'; then
        log_info "Pagination information present"
        return 0
    else
        log_error "Missing pagination information"
        return 1
    fi
}

# Test sorting
test_sorting() {
    local response
    response=$(api_request "GET" "/api/v1/comments/$TEST_SCENARIO?sort=oldest" "" "200")
    
    if [[ $? -eq 0 ]]; then
        log_info "Sorting by oldest works"
        return 0
    else
        log_error "Sorting failed"
        return 1
    fi
}

# Test scenario configuration
test_scenario_config() {
    local response
    response=$(api_request "GET" "/api/v1/config/$TEST_SCENARIO" "" "200")
    
    if echo "$response" | grep -q '"config"'; then
        log_info "Scenario configuration retrieved"
        return 0
    else
        log_error "Failed to get scenario configuration"
        return 1
    fi
}

# Test updating scenario configuration
test_update_config() {
    local config_data='{"auth_required": false, "allow_anonymous": true, "moderation_level": "none"}'
    local response
    
    response=$(api_request "POST" "/api/v1/config/$TEST_SCENARIO" "$config_data" "200")
    
    if echo "$response" | grep -q '"success":true'; then
        log_info "Configuration updated successfully"
        return 0
    else
        log_error "Failed to update configuration"
        return 1
    fi
}

# Test content validation
test_content_validation() {
    # Test empty content
    local empty_data='{"content": "", "content_type": "plaintext"}'
    local response
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X "POST" \
        -H "Content-Type: application/json" \
        -d "$empty_data" \
        "$API_URL/api/v1/comments/$TEST_SCENARIO")
    
    local status_code
    status_code=$(echo "$response" | grep -o 'HTTPSTATUS:[0-9]*' | cut -d: -f2)
    
    # Should return 400 for empty content
    if [[ "$status_code" == "400" ]]; then
        log_info "Empty content properly rejected"
        return 0
    else
        log_error "Empty content should be rejected with 400 status"
        return 1
    fi
}

# Test error handling
test_invalid_scenario() {
    local response
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X "GET" \
        "$API_URL/api/v1/comments/nonexistent-scenario-name-that-should-not-exist")
    
    local status_code
    status_code=$(echo "$response" | grep -o 'HTTPSTATUS:[0-9]*' | cut -d: -f2)
    
    # Should handle gracefully (200 with empty results)
    if [[ "$status_code" == "200" ]]; then
        log_info "Invalid scenario handled gracefully"
        return 0
    else
        log_error "Invalid scenario should be handled gracefully"
        return 1
    fi
}

# Main test execution
main() {
    echo "========================================"
    echo "Comment System CRUD Operations Test"
    echo "========================================"
    echo "API URL: $API_URL"
    echo "Test Scenario: $TEST_SCENARIO"
    echo

    # Run tests
    run_test "API Health Check" test_api_health
    run_test "Empty Comments List" test_empty_comments_list
    run_test "Create Comment" test_create_comment
    run_test "Create Markdown Comment" test_create_markdown_comment
    run_test "Create Reply Comment" test_create_reply_comment
    run_test "List Comments" test_list_comments
    run_test "Pagination" test_pagination
    run_test "Sorting" test_sorting
    run_test "Scenario Configuration" test_scenario_config
    run_test "Update Configuration" test_update_config
    run_test "Content Validation" test_content_validation
    run_test "Invalid Scenario Handling" test_invalid_scenario

    # Summary
    echo
    echo "========================================"
    echo "Test Summary"
    echo "========================================"
    echo "Total Tests: $TEST_COUNT"
    echo "Passed: $PASSED_COUNT"
    echo "Failed: $FAILED_COUNT"
    echo

    if [[ $FAILED_COUNT -eq 0 ]]; then
        log_success "All tests passed!"
        exit 0
    else
        log_error "$FAILED_COUNT test(s) failed"
        exit 1
    fi
}

# Check dependencies
check_dependencies() {
    local missing_deps=()

    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        echo "Please install the missing dependencies and try again."
        exit 1
    fi
}

# Entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    check_dependencies
    main "$@"
fi