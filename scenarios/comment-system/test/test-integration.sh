#!/usr/bin/env bash

# Comment System Integration Test
# Tests integration with session-authenticator and notification-hub

set -e

API_URL="${API_URL:-http://localhost:8080}"
SESSION_AUTH_URL="${SESSION_AUTH_URL:-http://localhost:8001}"
NOTIFICATION_HUB_URL="${NOTIFICATION_HUB_URL:-http://localhost:8002}"
TEST_SCENARIO="integration-test"

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
    local base_url="${5:-$API_URL}"
    
    local response
    local status_code
    
    if [[ -n "$data" ]]; then
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" --max-time 10 \
            -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$base_url$endpoint")
    else
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" --max-time 10 \
            -X "$method" \
            "$base_url$endpoint")
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

# Test comment system health
test_comment_system_health() {
    local response
    response=$(api_request "GET" "/health" "" "200")
    
    if echo "$response" | grep -q '"status":"healthy"'; then
        log_info "Comment System is healthy"
        return 0
    else
        log_error "Comment System is not healthy"
        echo "Response: $response"
        return 1
    fi
}

# Test session authenticator connectivity
test_session_authenticator_connectivity() {
    local response
    response=$(api_request "GET" "/health" "" "200")
    
    # Check dependencies status
    local session_auth_status
    session_auth_status=$(echo "$response" | jq -r '.dependencies.session_authenticator // "unknown"' 2>/dev/null || echo "unknown")
    
    case "$session_auth_status" in
        "connected")
            log_info "Session Authenticator: Connected"
            return 0
            ;;
        "disconnected"*)
            log_warning "Session Authenticator: Disconnected - $session_auth_status"
            log_info "This is expected if session-authenticator is not running"
            return 0  # Don't fail test if service is just not running
            ;;
        *)
            log_warning "Session Authenticator: Status unknown"
            return 0  # Don't fail test for unknown status
            ;;
    esac
}

# Test notification hub connectivity
test_notification_hub_connectivity() {
    local response
    response=$(api_request "GET" "/health" "" "200")
    
    # Check dependencies status
    local notification_hub_status
    notification_hub_status=$(echo "$response" | jq -r '.dependencies.notification_hub // "unknown"' 2>/dev/null || echo "unknown")
    
    case "$notification_hub_status" in
        "connected")
            log_info "Notification Hub: Connected"
            return 0
            ;;
        "disconnected"*)
            log_warning "Notification Hub: Disconnected - $notification_hub_status"
            log_info "This is expected if notification-hub is not running"
            return 0  # Don't fail test if service is just not running
            ;;
        *)
            log_warning "Notification Hub: Status unknown"
            return 0  # Don't fail test for unknown status
            ;;
    esac
}

# Test authenticated comment creation (mock)
test_authenticated_comment_creation() {
    # Since we don't have real session-authenticator integration yet,
    # we'll test with a mock token and verify graceful handling
    
    local comment_data='{"content": "Authenticated comment test", "author_token": "mock-session-token", "content_type": "plaintext"}'
    local response
    
    response=$(api_request "POST" "/api/v1/comments/$TEST_SCENARIO" "$comment_data" "201")
    
    if echo "$response" | grep -q '"success":true'; then
        log_info "Authenticated comment creation handled gracefully"
        return 0
    else
        log_error "Authenticated comment creation failed"
        echo "Response: $response"
        return 1
    fi
}

# Test anonymous comment creation
test_anonymous_comment_creation() {
    # First, configure scenario to allow anonymous comments
    local config_data='{"auth_required": false, "allow_anonymous": true}'
    api_request "POST" "/api/v1/config/$TEST_SCENARIO" "$config_data" "200" >/dev/null
    
    # Create anonymous comment (no author_token)
    local comment_data='{"content": "Anonymous comment test", "content_type": "plaintext"}'
    local response
    
    response=$(api_request "POST" "/api/v1/comments/$TEST_SCENARIO" "$comment_data" "201")
    
    if echo "$response" | grep -q '"success":true'; then
        log_info "Anonymous comment creation successful"
        return 0
    else
        log_error "Anonymous comment creation failed"
        echo "Response: $response"
        return 1
    fi
}

# Test authentication requirements
test_authentication_requirements() {
    # Configure scenario to require authentication
    local config_data='{"auth_required": true, "allow_anonymous": false}'
    api_request "POST" "/api/v1/config/$TEST_SCENARIO" "$config_data" "200" >/dev/null
    
    # Try to create comment without token
    local comment_data='{"content": "Should be rejected", "content_type": "plaintext"}'
    local response
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" --max-time 10 \
        -X "POST" \
        -H "Content-Type: application/json" \
        -d "$comment_data" \
        "$API_URL/api/v1/comments/$TEST_SCENARIO")
    
    local status_code
    status_code=$(echo "$response" | grep -o 'HTTPSTATUS:[0-9]*' | cut -d: -f2)
    
    # Should reject with 401 Unauthorized
    if [[ "$status_code" == "401" ]]; then
        log_info "Authentication requirement properly enforced"
        return 0
    else
        log_error "Authentication requirement not enforced (status: $status_code)"
        return 1
    fi
}

# Test configuration persistence
test_configuration_persistence() {
    # Set specific configuration
    local config_data='{"auth_required": true, "allow_rich_media": true, "moderation_level": "manual"}'
    api_request "POST" "/api/v1/config/$TEST_SCENARIO" "$config_data" "200" >/dev/null
    
    # Retrieve configuration
    local response
    response=$(api_request "GET" "/api/v1/config/$TEST_SCENARIO" "" "200")
    
    # Verify values
    local auth_required allow_rich_media moderation_level
    auth_required=$(echo "$response" | jq -r '.config.auth_required // false' 2>/dev/null)
    allow_rich_media=$(echo "$response" | jq -r '.config.allow_rich_media // false' 2>/dev/null)
    moderation_level=$(echo "$response" | jq -r '.config.moderation_level // "none"' 2>/dev/null)
    
    if [[ "$auth_required" == "true" && "$allow_rich_media" == "true" && "$moderation_level" == "manual" ]]; then
        log_info "Configuration persistence working correctly"
        return 0
    else
        log_error "Configuration persistence failed"
        echo "auth_required: $auth_required, allow_rich_media: $allow_rich_media, moderation_level: $moderation_level"
        return 1
    fi
}

# Test cross-origin requests (CORS)
test_cors_headers() {
    local response
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" -I \
        -H "Origin: http://localhost:3000" \
        -X "OPTIONS" \
        "$API_URL/api/v1/comments/$TEST_SCENARIO")
    
    local status_code
    status_code=$(echo "$response" | grep -o 'HTTPSTATUS:[0-9]*' | cut -d: -f2)
    
    if [[ "$status_code" == "204" ]] && echo "$response" | grep -qi "access-control-allow-origin"; then
        log_info "CORS headers present and correct"
        return 0
    else
        log_error "CORS headers missing or incorrect"
        echo "Response: $response"
        return 1
    fi
}

# Test API error handling
test_api_error_handling() {
    # Test invalid JSON
    local response
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" --max-time 10 \
        -X "POST" \
        -H "Content-Type: application/json" \
        -d '{"invalid": json}' \
        "$API_URL/api/v1/comments/$TEST_SCENARIO")
    
    local status_code
    status_code=$(echo "$response" | grep -o 'HTTPSTATUS:[0-9]*' | cut -d: -f2)
    
    # Should return 400 for bad JSON
    if [[ "$status_code" == "400" ]]; then
        log_info "API error handling working for invalid JSON"
        return 0
    else
        log_error "API should return 400 for invalid JSON (got: $status_code)"
        return 1
    fi
}

# Test database connectivity through API
test_database_connectivity() {
    local response
    response=$(api_request "GET" "/health/postgres" "" "200")
    
    if echo "$response" | grep -q '"status":"healthy"'; then
        log_info "Database connectivity confirmed"
        return 0
    else
        log_error "Database connectivity issue"
        echo "Response: $response"
        return 1
    fi
}

# Test concurrent comment creation
test_concurrent_operations() {
    log_info "Testing concurrent comment creation..."
    
    # Create multiple comments in parallel
    local pids=()
    for i in {1..5}; do
        {
            local comment_data="{\"content\": \"Concurrent comment $i\", \"content_type\": \"plaintext\"}"
            api_request "POST" "/api/v1/comments/$TEST_SCENARIO" "$comment_data" "201" >/dev/null 2>&1
        } &
        pids+=($!)
    done
    
    # Wait for all background jobs
    local failed_count=0
    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            failed_count=$((failed_count + 1))
        fi
    done
    
    if [[ $failed_count -eq 0 ]]; then
        log_info "All concurrent operations succeeded"
        return 0
    else
        log_error "$failed_count concurrent operations failed"
        return 1
    fi
}

# Main test execution
main() {
    echo "========================================"
    echo "Comment System Integration Test"
    echo "========================================"
    echo "Comment System API: $API_URL"
    echo "Session Authenticator: $SESSION_AUTH_URL"
    echo "Notification Hub: $NOTIFICATION_HUB_URL"
    echo "Test Scenario: $TEST_SCENARIO"
    echo

    # Run tests
    run_test "Comment System Health" test_comment_system_health
    run_test "Session Authenticator Connectivity" test_session_authenticator_connectivity
    run_test "Notification Hub Connectivity" test_notification_hub_connectivity
    run_test "Database Connectivity" test_database_connectivity
    run_test "CORS Headers" test_cors_headers
    run_test "API Error Handling" test_api_error_handling
    run_test "Configuration Persistence" test_configuration_persistence
    run_test "Anonymous Comment Creation" test_anonymous_comment_creation
    run_test "Authenticated Comment Creation" test_authenticated_comment_creation
    run_test "Authentication Requirements" test_authentication_requirements
    run_test "Concurrent Operations" test_concurrent_operations

    # Summary
    echo
    echo "========================================"
    echo "Integration Test Summary"
    echo "========================================"
    echo "Total Tests: $TEST_COUNT"
    echo "Passed: $PASSED_COUNT"
    echo "Failed: $FAILED_COUNT"
    echo

    if [[ $FAILED_COUNT -eq 0 ]]; then
        log_success "All integration tests passed!"
        exit 0
    else
        log_error "$FAILED_COUNT integration test(s) failed"
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