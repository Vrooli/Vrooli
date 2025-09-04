#!/usr/bin/env bash

# Integration tests for Scenario Authenticator
# Tests the complete authentication flow

set -e

# Configuration
API_URL="${AUTH_API_URL:-http://localhost:3250}"
UI_URL="${AUTH_UI_URL:-http://localhost:3251}"
TEST_EMAIL="integration-test-$(date +%s)@vrooli.test"
TEST_PASSWORD="IntegrationTest123!"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Helper functions
log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# Test helper
test_api() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"
    local expected_status="$5"
    local auth_header="$6"
    
    log_info "Testing: $name"
    
    local curl_cmd="curl -s -w '%{http_code}' -X $method"
    
    if [ -n "$auth_header" ]; then
        curl_cmd="$curl_cmd -H 'Authorization: Bearer $auth_header'"
    fi
    
    if [ -n "$data" ]; then
        curl_cmd="$curl_cmd -H 'Content-Type: application/json' -d '$data'"
    fi
    
    local response
    response=$(eval "$curl_cmd '${API_URL}${endpoint}'")
    local status_code="${response: -3}"
    local body="${response%???}"
    
    if [ "$status_code" = "$expected_status" ]; then
        log_success "$name (HTTP $status_code)"
        echo "$body"
    else
        log_error "$name - Expected HTTP $expected_status, got $status_code"
        echo "Response: $body"
        return 1
    fi
}

# Wait for services to be ready
wait_for_services() {
    log_info "Waiting for services to be ready..."
    
    # Wait for API
    for i in {1..30}; do
        if curl -sf "$API_URL/health" >/dev/null 2>&1; then
            log_success "API is ready"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "API did not start in time"
            return 1
        fi
        sleep 1
    done
    
    # Wait for UI
    for i in {1..30}; do
        if curl -sf "$UI_URL/health" >/dev/null 2>&1; then
            log_success "UI is ready"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "UI did not start in time"
            return 1
        fi
        sleep 1
    done
}

# Test health endpoints
test_health() {
    log_info "Testing health endpoints..."
    
    test_api "API Health Check" GET "/health" "" "200"
    
    local ui_response
    ui_response=$(curl -s "$UI_URL/health")
    if echo "$ui_response" | jq -e '.status == "healthy"' >/dev/null 2>&1; then
        log_success "UI Health Check"
    else
        log_error "UI Health Check failed"
        echo "Response: $ui_response"
        return 1
    fi
}

# Test user registration
test_registration() {
    log_info "Testing user registration..."
    
    local register_data="{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\",\"username\":\"integrationtest\"}"
    local response
    response=$(test_api "User Registration" POST "/api/v1/auth/register" "$register_data" "201")
    
    # Extract token for later use
    if echo "$response" | jq -e '.success == true' >/dev/null 2>&1; then
        TEST_TOKEN=$(echo "$response" | jq -r '.token')
        TEST_REFRESH_TOKEN=$(echo "$response" | jq -r '.refresh_token')
        TEST_USER_ID=$(echo "$response" | jq -r '.user.id')
        log_success "Registration successful, token obtained"
    else
        log_error "Registration failed"
        echo "$response"
        return 1
    fi
}

# Test token validation
test_token_validation() {
    log_info "Testing token validation..."
    
    if [ -z "$TEST_TOKEN" ]; then
        log_error "No token available for validation test"
        return 1
    fi
    
    local response
    response=$(test_api "Token Validation" GET "/api/v1/auth/validate" "" "200" "$TEST_TOKEN")
    
    if echo "$response" | jq -e '.valid == true' >/dev/null 2>&1; then
        log_success "Token validation successful"
        
        # Verify user details in validation response
        local returned_email
        returned_email=$(echo "$response" | jq -r '.email')
        if [ "$returned_email" = "$TEST_EMAIL" ]; then
            log_success "Token contains correct user email"
        else
            log_error "Token validation returned wrong email: $returned_email"
            return 1
        fi
    else
        log_error "Token validation failed"
        echo "$response"
        return 1
    fi
}

# Test login with created user
test_login() {
    log_info "Testing user login..."
    
    local login_data="{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}"
    local response
    response=$(test_api "User Login" POST "/api/v1/auth/login" "$login_data" "200")
    
    if echo "$response" | jq -e '.success == true' >/dev/null 2>&1; then
        # Get new token from login
        LOGIN_TOKEN=$(echo "$response" | jq -r '.token')
        log_success "Login successful, new token obtained"
    else
        log_error "Login failed"
        echo "$response"
        return 1
    fi
}

# Test invalid credentials
test_invalid_login() {
    log_info "Testing invalid login credentials..."
    
    local invalid_data="{\"email\":\"$TEST_EMAIL\",\"password\":\"WrongPassword123!\"}"
    local response
    response=$(test_api "Invalid Login" POST "/api/v1/auth/login" "$invalid_data" "401")
    
    if echo "$response" | jq -e '.success == false' >/dev/null 2>&1; then
        log_success "Invalid login correctly rejected"
    else
        log_error "Invalid login was incorrectly accepted"
        echo "$response"
        return 1
    fi
}

# Test token blacklisting via logout
test_logout() {
    log_info "Testing user logout and token blacklisting..."
    
    if [ -z "$TEST_TOKEN" ]; then
        log_error "No token available for logout test"
        return 1
    fi
    
    # Logout
    local response
    response=$(test_api "User Logout" POST "/api/v1/auth/logout" "" "200" "$TEST_TOKEN")
    
    if echo "$response" | jq -e '.success == true' >/dev/null 2>&1; then
        log_success "Logout successful"
        
        # Try to use the token after logout (should fail)
        response=$(test_api "Token After Logout" GET "/api/v1/auth/validate" "" "200" "$TEST_TOKEN")
        
        if echo "$response" | jq -e '.valid == false' >/dev/null 2>&1; then
            log_success "Token correctly invalidated after logout"
        else
            log_error "Token still valid after logout"
            echo "$response"
            return 1
        fi
    else
        log_error "Logout failed"
        echo "$response"
        return 1
    fi
}

# Test CLI functionality
test_cli() {
    log_info "Testing CLI functionality..."
    
    # Check if CLI is executable
    if [ ! -x "./cli/scenario-authenticator" ]; then
        log_error "CLI not found or not executable"
        return 1
    fi
    
    # Test CLI status
    if ./cli/scenario-authenticator status >/dev/null 2>&1; then
        log_success "CLI status command works"
    else
        log_error "CLI status command failed"
        return 1
    fi
    
    # Test CLI help
    if ./cli/scenario-authenticator help | grep -q "Commands:"; then
        log_success "CLI help command works"
    else
        log_error "CLI help command failed"
        return 1
    fi
    
    # Test CLI token validation (if we have a valid token)
    if [ -n "$LOGIN_TOKEN" ]; then
        if ./cli/scenario-authenticator token validate "$LOGIN_TOKEN" >/dev/null 2>&1; then
            log_success "CLI token validation works"
        else
            log_error "CLI token validation failed"
            return 1
        fi
    fi
}

# Test database connectivity
test_database() {
    log_info "Testing database integration..."
    
    # This is a simple test - just verify the API can connect to DB
    # by checking that health endpoint returns database: true
    local response
    response=$(curl -s "$API_URL/health")
    
    if echo "$response" | jq -e '.database == true' >/dev/null 2>&1; then
        log_success "Database connectivity confirmed"
    else
        log_error "Database connectivity failed"
        echo "Health response: $response"
        return 1
    fi
}

# Test Redis connectivity
test_redis() {
    log_info "Testing Redis integration..."
    
    # Similar to database test - verify via health endpoint
    local response
    response=$(curl -s "$API_URL/health")
    
    if echo "$response" | jq -e '.redis == true' >/dev/null 2>&1; then
        log_success "Redis connectivity confirmed"
    else
        log_error "Redis connectivity failed"
        echo "Health response: $response"
        return 1
    fi
}

# Main test execution
main() {
    log_info "Starting Scenario Authenticator Integration Tests"
    echo "API URL: $API_URL"
    echo "UI URL: $UI_URL"
    echo "Test Email: $TEST_EMAIL"
    echo ""
    
    # Check prerequisites
    if ! command -v curl >/dev/null 2>&1; then
        log_error "curl is required for integration tests"
        exit 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        log_error "jq is required for integration tests"
        exit 1
    fi
    
    # Run tests
    local failed_tests=0
    
    test_health || ((failed_tests++))
    test_database || ((failed_tests++))
    test_redis || ((failed_tests++))
    test_registration || ((failed_tests++))
    test_token_validation || ((failed_tests++))
    test_login || ((failed_tests++))
    test_invalid_login || ((failed_tests++))
    test_logout || ((failed_tests++))
    test_cli || ((failed_tests++))
    
    echo ""
    if [ $failed_tests -eq 0 ]; then
        log_success "All integration tests passed! ✅"
        echo ""
        log_info "Test Summary:"
        echo "  - API Health: ✅"
        echo "  - Database: ✅"
        echo "  - Redis: ✅"
        echo "  - User Registration: ✅"
        echo "  - Token Validation: ✅"
        echo "  - User Login: ✅"
        echo "  - Invalid Login Rejection: ✅"
        echo "  - Logout & Token Blacklisting: ✅"
        echo "  - CLI Functionality: ✅"
        echo ""
        log_success "Authentication service is fully functional!"
        exit 0
    else
        log_error "$failed_tests test(s) failed ❌"
        exit 1
    fi
}

# Run tests
main "$@"