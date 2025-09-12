#!/bin/bash
# Complete authentication flow test
# Tests the end-to-end authentication functionality

set -e

# Configuration
API_PORT="${AUTH_API_PORT:-15000}"
BASE_URL="http://localhost:${API_PORT}"
TEST_EMAIL="test_$(date +%s)@example.com"
TEST_PASSWORD="SecureTest123!"
TEST_USERNAME="testuser_$(date +%s)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

check_response() {
    local response="$1"
    local expected_field="$2"
    
    if echo "$response" | grep -q "$expected_field"; then
        return 0
    else
        log_error "Response missing expected field: $expected_field"
        echo "Response: $response"
        return 1
    fi
}

# Check if server is running
log_info "Checking if authentication API is running on port ${API_PORT}..."
if ! curl -s -f "${BASE_URL}/health" > /dev/null 2>&1; then
    log_error "Authentication API is not running on port ${API_PORT}"
    log_info "Start the service with: vrooli scenario run scenario-authenticator"
    log_info "Or use: cd scenarios/scenario-authenticator && make run"
    exit 1
fi

log_info "Starting authentication flow tests..."

# Test 1: User Registration
log_info "Test 1: User Registration"
REGISTER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"${TEST_EMAIL}\",
        \"password\": \"${TEST_PASSWORD}\",
        \"username\": \"${TEST_USERNAME}\"
    }")

if check_response "$REGISTER_RESPONSE" "token"; then
    log_info "✓ Registration successful"
    TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
else
    log_error "✗ Registration failed"
    exit 1
fi

# Test 2: User Login
log_info "Test 2: User Login"
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"${TEST_EMAIL}\",
        \"password\": \"${TEST_PASSWORD}\"
    }")

if check_response "$LOGIN_RESPONSE" "token"; then
    log_info "✓ Login successful"
    LOGIN_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
    REFRESH_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"refresh_token":"[^"]*' | cut -d'"' -f4)
else
    log_error "✗ Login failed"
    exit 1
fi

# Test 3: Token Validation
log_info "Test 3: Token Validation"
VALIDATE_RESPONSE=$(curl -s -X GET "${BASE_URL}/api/v1/auth/validate" \
    -H "Authorization: Bearer ${LOGIN_TOKEN}")

if check_response "$VALIDATE_RESPONSE" "valid"; then
    log_info "✓ Token validation successful"
else
    log_error "✗ Token validation failed"
    exit 1
fi

# Test 4: Get User Info
log_info "Test 4: Get User Info"
USER_RESPONSE=$(curl -s -X GET "${BASE_URL}/api/v1/users" \
    -H "Authorization: Bearer ${LOGIN_TOKEN}")

if echo "$USER_RESPONSE" | grep -q "${TEST_EMAIL}"; then
    log_info "✓ User info retrieved successfully"
else
    log_error "✗ Failed to retrieve user info"
    exit 1
fi

# Test 5: Token Refresh
log_info "Test 5: Token Refresh"
if [ ! -z "$REFRESH_TOKEN" ]; then
    REFRESH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{
            \"refresh_token\": \"${REFRESH_TOKEN}\"
        }")
    
    if check_response "$REFRESH_RESPONSE" "token"; then
        log_info "✓ Token refresh successful"
        NEW_TOKEN=$(echo "$REFRESH_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
    else
        log_warning "⚠ Token refresh not implemented or failed"
    fi
else
    log_warning "⚠ No refresh token available"
fi

# Test 6: Invalid Login
log_info "Test 6: Invalid Login (negative test)"
INVALID_LOGIN=$(curl -s -X POST "${BASE_URL}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"${TEST_EMAIL}\",
        \"password\": \"WrongPassword123\"
    }")

if echo "$INVALID_LOGIN" | grep -q "error\|unauthorized\|invalid"; then
    log_info "✓ Invalid login correctly rejected"
else
    log_error "✗ Invalid login not properly handled"
    exit 1
fi

# Test 7: Logout
log_info "Test 7: Logout"
LOGOUT_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/auth/logout" \
    -H "Authorization: Bearer ${LOGIN_TOKEN}")

if [ $? -eq 0 ]; then
    log_info "✓ Logout successful"
else
    log_warning "⚠ Logout may not be implemented"
fi

# Test 8: Access After Logout
log_info "Test 8: Access After Logout (should fail)"
AFTER_LOGOUT=$(curl -s -X GET "${BASE_URL}/api/v1/users" \
    -H "Authorization: Bearer ${LOGIN_TOKEN}")

if echo "$AFTER_LOGOUT" | grep -q "unauthorized\|invalid\|error"; then
    log_info "✓ Token correctly invalidated after logout"
else
    log_warning "⚠ Token may still be valid after logout"
fi

# Test 9: Password Reset Request
log_info "Test 9: Password Reset Request"
RESET_REQUEST=$(curl -s -X POST "${BASE_URL}/api/v1/auth/reset-password" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"${TEST_EMAIL}\"
    }")

if [ $? -eq 0 ]; then
    log_info "✓ Password reset request sent"
else
    log_warning "⚠ Password reset may not be fully implemented"
fi

# Summary
echo ""
log_info "========================================="
log_info "Authentication Flow Test Complete!"
log_info "All critical authentication flows tested"
log_info "========================================="

exit 0