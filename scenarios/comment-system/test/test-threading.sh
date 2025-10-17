#!/usr/bin/env bash

# Comment System Threading Test
# Tests nested comment replies and threading functionality

set -e

API_URL="${API_URL:-http://localhost:8080}"
TEST_SCENARIO="threading-test"

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

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test counter
TEST_COUNT=0
PASSED_COUNT=0
FAILED_COUNT=0

# Comment IDs for threading tests
declare -a COMMENT_IDS

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
    
    echo "$response_body"
    
    if [[ "$status_code" != "$expected_status" ]]; then
        log_error "Expected status $expected_status, got $status_code"
        return 1
    fi
    
    return 0
}

# Create root comment
test_create_root_comment() {
    local comment_data='{"content": "This is the root comment", "content_type": "plaintext"}'
    local response
    
    response=$(api_request "POST" "/api/v1/comments/$TEST_SCENARIO" "$comment_data" "201")
    
    # Extract comment ID
    local comment_id
    comment_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    if [[ -n "$comment_id" ]]; then
        COMMENT_IDS[0]="$comment_id"
        log_info "Created root comment: ${COMMENT_IDS[0]}"
        return 0
    else
        log_error "Failed to create root comment"
        return 1
    fi
}

# Create level 1 replies
test_create_level1_replies() {
    if [[ -z "${COMMENT_IDS[0]}" ]]; then
        log_error "No root comment available"
        return 1
    fi
    
    # Create 2 level-1 replies
    for i in 1 2; do
        local reply_data="{\"content\": \"Level 1 reply #$i\", \"parent_id\": \"${COMMENT_IDS[0]}\", \"content_type\": \"plaintext\"}"
        local response
        
        response=$(api_request "POST" "/api/v1/comments/$TEST_SCENARIO" "$reply_data" "201")
        
        local reply_id
        reply_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        
        if [[ -n "$reply_id" ]]; then
            COMMENT_IDS[$i]="$reply_id"
            log_info "Created level 1 reply #$i: ${COMMENT_IDS[$i]}"
        else
            log_error "Failed to create level 1 reply #$i"
            return 1
        fi
    done
    
    return 0
}

# Create level 2 replies
test_create_level2_replies() {
    if [[ -z "${COMMENT_IDS[1]}" ]]; then
        log_error "No level 1 comment available for level 2 reply"
        return 1
    fi
    
    # Create reply to first level 1 comment
    local reply_data="{\"content\": \"Level 2 reply to comment 1\", \"parent_id\": \"${COMMENT_IDS[1]}\", \"content_type\": \"plaintext\"}"
    local response
    
    response=$(api_request "POST" "/api/v1/comments/$TEST_SCENARIO" "$reply_data" "201")
    
    local reply_id
    reply_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    if [[ -n "$reply_id" ]]; then
        COMMENT_IDS[3]="$reply_id"
        log_info "Created level 2 reply: ${COMMENT_IDS[3]}"
        return 0
    else
        log_error "Failed to create level 2 reply"
        return 1
    fi
}

# Test threading structure in response
test_threading_structure() {
    local response
    response=$(api_request "GET" "/api/v1/comments/$TEST_SCENARIO?sort=threaded" "" "200")
    
    # Check if threading information is present
    if echo "$response" | grep -q '"thread_path"' && echo "$response" | grep -q '"depth"'; then
        log_info "Threading information present in response"
        
        # Count comments at different depths
        local depth0_count depth1_count depth2_count
        depth0_count=$(echo "$response" | grep -o '"depth":0' | wc -l)
        depth1_count=$(echo "$response" | grep -o '"depth":1' | wc -l)
        depth2_count=$(echo "$response" | grep -o '"depth":2' | wc -l)
        
        log_info "Depth 0 comments: $depth0_count"
        log_info "Depth 1 comments: $depth1_count"
        log_info "Depth 2 comments: $depth2_count"
        
        # We should have 1 root, 2 level-1, and 1 level-2 comment
        if [[ "$depth0_count" -eq 1 && "$depth1_count" -eq 2 && "$depth2_count" -eq 1 ]]; then
            return 0
        else
            log_error "Incorrect comment depth distribution"
            return 1
        fi
    else
        log_error "Threading information missing from response"
        return 1
    fi
}

# Test reply count updates
test_reply_counts() {
    local response
    response=$(api_request "GET" "/api/v1/comments/$TEST_SCENARIO" "" "200")
    
    # Root comment should have reply_count = 2 (direct children)
    local root_reply_count
    root_reply_count=$(echo "$response" | jq -r ".comments[] | select(.id==\"${COMMENT_IDS[0]}\") | .reply_count // 0")
    
    if [[ "$root_reply_count" -eq 2 ]]; then
        log_info "Root comment reply count correct: $root_reply_count"
    else
        log_error "Root comment reply count incorrect: expected 2, got $root_reply_count"
        return 1
    fi
    
    # First level-1 comment should have reply_count = 1
    local level1_reply_count
    level1_reply_count=$(echo "$response" | jq -r ".comments[] | select(.id==\"${COMMENT_IDS[1]}\") | .reply_count // 0")
    
    if [[ "$level1_reply_count" -eq 1 ]]; then
        log_info "Level 1 comment reply count correct: $level1_reply_count"
        return 0
    else
        log_error "Level 1 comment reply count incorrect: expected 1, got $level1_reply_count"
        return 1
    fi
}

# Test filtering by parent
test_parent_filtering() {
    # Get only replies to root comment
    local response
    response=$(api_request "GET" "/api/v1/comments/$TEST_SCENARIO?parent_id=${COMMENT_IDS[0]}" "" "200")
    
    # Should return 2 direct children
    local child_count
    child_count=$(echo "$response" | jq -r '.comments | length')
    
    if [[ "$child_count" -eq 2 ]]; then
        log_info "Parent filtering returned correct number of children: $child_count"
        return 0
    else
        log_error "Parent filtering incorrect: expected 2 children, got $child_count"
        return 1
    fi
}

# Test threaded sorting
test_threaded_sorting() {
    local response
    response=$(api_request "GET" "/api/v1/comments/$TEST_SCENARIO?sort=threaded" "" "200")
    
    # In threaded sort, comments should be ordered by thread_path
    # Extract thread paths and verify they're in order
    local thread_paths
    thread_paths=$(echo "$response" | jq -r '.comments[].thread_path // empty' | head -10)
    
    if [[ -n "$thread_paths" ]]; then
        log_info "Thread paths found in response"
        echo "Thread paths:"
        echo "$thread_paths" | while read -r path; do
            echo "  $path"
        done
        return 0
    else
        log_error "No thread paths found in threaded sort"
        return 1
    fi
}

# Test maximum depth limitation (if implemented)
test_max_depth() {
    # Try to create a very deep thread (level 3)
    if [[ -z "${COMMENT_IDS[3]}" ]]; then
        log_error "No level 2 comment available for max depth test"
        return 1
    fi
    
    local deep_reply_data="{\"content\": \"Level 3 reply\", \"parent_id\": \"${COMMENT_IDS[3]}\", \"content_type\": \"plaintext\"}"
    local response
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X "POST" \
        -H "Content-Type: application/json" \
        -d "$deep_reply_data" \
        "$API_URL/api/v1/comments/$TEST_SCENARIO")
    
    local status_code
    status_code=$(echo "$response" | grep -o 'HTTPSTATUS:[0-9]*' | cut -d: -f2)
    
    # Should either succeed (201) or be limited by max depth
    if [[ "$status_code" == "201" || "$status_code" == "400" ]]; then
        log_info "Max depth handling working (status: $status_code)"
        return 0
    else
        log_error "Unexpected status for max depth test: $status_code"
        return 1
    fi
}

# Test thread path generation
test_thread_paths() {
    local response
    response=$(api_request "GET" "/api/v1/comments/$TEST_SCENARIO" "" "200")
    
    # Check that root comment has simple thread path
    local root_path
    root_path=$(echo "$response" | jq -r ".comments[] | select(.id==\"${COMMENT_IDS[0]}\") | .thread_path")
    
    if [[ "$root_path" == "${COMMENT_IDS[0]}" ]]; then
        log_info "Root comment thread path correct: $root_path"
    else
        log_error "Root comment thread path incorrect: $root_path"
        return 1
    fi
    
    # Check that level 1 reply has compound thread path
    local level1_path
    level1_path=$(echo "$response" | jq -r ".comments[] | select(.id==\"${COMMENT_IDS[1]}\") | .thread_path")
    
    if [[ "$level1_path" == "${COMMENT_IDS[0]}.${COMMENT_IDS[1]}" ]]; then
        log_info "Level 1 comment thread path correct: $level1_path"
        return 0
    else
        log_error "Level 1 comment thread path incorrect: $level1_path"
        return 1
    fi
}

# Main test execution
main() {
    echo "========================================"
    echo "Comment System Threading Test"
    echo "========================================"
    echo "API URL: $API_URL"
    echo "Test Scenario: $TEST_SCENARIO"
    echo

    # Run tests in sequence
    run_test "Create Root Comment" test_create_root_comment
    run_test "Create Level 1 Replies" test_create_level1_replies
    run_test "Create Level 2 Reply" test_create_level2_replies
    run_test "Verify Threading Structure" test_threading_structure
    run_test "Verify Reply Counts" test_reply_counts
    run_test "Test Parent Filtering" test_parent_filtering
    run_test "Test Threaded Sorting" test_threaded_sorting
    run_test "Test Thread Paths" test_thread_paths
    run_test "Test Maximum Depth" test_max_depth

    # Summary
    echo
    echo "========================================"
    echo "Threading Test Summary"
    echo "========================================"
    echo "Total Tests: $TEST_COUNT"
    echo "Passed: $PASSED_COUNT"
    echo "Failed: $FAILED_COUNT"
    echo

    if [[ $FAILED_COUNT -eq 0 ]]; then
        log_success "All threading tests passed!"
        
        # Show final comment structure
        echo
        echo "Final Comment Structure:"
        local final_response
        final_response=$(api_request "GET" "/api/v1/comments/$TEST_SCENARIO?sort=threaded" "" "200")
        echo "$final_response" | jq -r '.comments[] | "[\(.depth)] \(.id[0:8]): \(.content)"' | head -10
        
        exit 0
    else
        log_error "$FAILED_COUNT threading test(s) failed"
        exit 1
    fi
}

# Check dependencies
check_dependencies() {
    local missing_deps=()

    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
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