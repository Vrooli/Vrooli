#!/bin/bash
# Scenario Authenticator Integration Tests

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
SCENARIO_NAME="scenario-authenticator"
TEST_EMAIL="test_$(date +%s)@example.com"
TEST_PASSWORD="TestPass123!"

# Helper functions
print_test() {
    echo -e "${YELLOW}Testing: $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}

# Get API URL dynamically
get_api_url() {
    local api_port
    api_port=$(vrooli scenario port "${SCENARIO_NAME}" AUTH_API_PORT 2>/dev/null)
    
    if [[ -z "$api_port" ]]; then
        api_port=$(vrooli scenario port "${SCENARIO_NAME}" API_PORT 2>/dev/null)
    fi
    
    if [[ -z "$api_port" ]]; then
        print_error "Scenario ${SCENARIO_NAME} is not running"
    fi
    
    echo "http://localhost:${api_port}"
}

# Main test execution
main() {
    echo "==================================="
    echo "Scenario Authenticator Integration Tests"
    echo "==================================="
    
    API_URL=$(get_api_url)
    echo "API URL: $API_URL"
    echo ""
    
    # Test 1: Health check
    print_test "Health check endpoint"
    response=$(curl -s "${API_URL}/health")
    status=$(echo "$response" | jq -r '.status')
    if [[ "$status" == "healthy" ]] || [[ "$status" == "degraded" ]]; then
    print_success "Health check passed"
else
    print_error "Health check failed"
fi

# Test 2: Applications endpoint (admin access)
    print_test "List applications as admin"
    admin_login=$(curl -s -X POST "${API_URL}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"email":"admin@vrooli.local","password":"Admin123!"}')

    admin_token=$(echo "$admin_login" | jq -r '.token')
    if [[ -z "$admin_token" || "$admin_token" == "null" ]]; then
        print_error "Failed to authenticate seeded admin user"
    fi

    apps_response=$(curl -s -w "\n%{http_code}" "${API_URL}/api/v1/applications?stats=true" \
        -H "Authorization: Bearer ${admin_token}")

    apps_body=$(echo "$apps_response" | head -n 1)
    apps_status=$(echo "$apps_response" | tail -n 1)

    if [[ "$apps_status" != "200" ]]; then
        print_error "Applications endpoint returned status ${apps_status}: ${apps_body}"
    fi

    total=$(echo "$apps_body" | jq -r '.total // -1')
    if [[ "$total" -lt 0 ]]; then
        print_error "Applications endpoint response malformed: ${apps_body}"
    fi

    print_success "Applications endpoint reachable"

    # Test 3: User registration
    print_test "User registration"
    response=$(curl -s -X POST "${API_URL}/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\"}")
    
    success=$(echo "$response" | jq -r '.success')
    if [[ "$success" == "true" ]]; then
        TOKEN=$(echo "$response" | jq -r '.token')
        USER_ID=$(echo "$response" | jq -r '.user.id')
        print_success "User registered successfully"
    else
        print_error "User registration failed: $response"
    fi
    
    # Test 4: Token validation
    print_test "Token validation"
    response=$(curl -s -X GET "${API_URL}/api/v1/auth/validate" \
        -H "Authorization: Bearer ${TOKEN}")
    
    valid=$(echo "$response" | jq -r '.valid')
    if [[ "$valid" == "true" ]]; then
        print_success "Token validation passed"
    else
        print_error "Token validation failed"
    fi
    
    # Test 5: User login
    print_test "User login"
    response=$(curl -s -X POST "${API_URL}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\"}")
    
    success=$(echo "$response" | jq -r '.success')
    if [[ "$success" == "true" ]]; then
        LOGIN_TOKEN=$(echo "$response" | jq -r '.token')
        REFRESH_TOKEN=$(echo "$response" | jq -r '.refresh_token')
        print_success "User login successful"
    else
        print_error "User login failed"
    fi
    
    # Test 6: Token refresh
    print_test "Token refresh"
    response=$(curl -s -X POST "${API_URL}/api/v1/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{\"refresh_token\":\"${REFRESH_TOKEN}\"}")
    
    success=$(echo "$response" | jq -r '.success')
    if [[ "$success" == "true" ]]; then
        NEW_TOKEN=$(echo "$response" | jq -r '.token')
        print_success "Token refresh successful"
    else
        print_error "Token refresh failed"
    fi
    
    # Test 7: Get user
    print_test "Get user by ID"
    response=$(curl -s -X GET "${API_URL}/api/v1/users/${USER_ID}" \
        -H "Authorization: Bearer ${NEW_TOKEN}")
    
    user_email=$(echo "$response" | jq -r '.email')
    if [[ "$user_email" == "${TEST_EMAIL}" ]]; then
        print_success "Get user successful"
    else
        print_error "Get user failed"
    fi
    
    # Test 8: List sessions
    print_test "List user sessions"
    response=$(curl -s -X GET "${API_URL}/api/v1/sessions?user_id=${USER_ID}" \
        -H "Authorization: Bearer ${NEW_TOKEN}")
    
    total=$(echo "$response" | jq -r '.total')
    if [[ "$total" -ge 0 ]]; then
        print_success "List sessions successful"
    else
        print_error "List sessions failed"
    fi
    
    # Test 9: Logout
    print_test "User logout"
    response=$(curl -s -X POST "${API_URL}/api/v1/auth/logout" \
        -H "Authorization: Bearer ${NEW_TOKEN}")
    
    success=$(echo "$response" | jq -r '.success')
    if [[ "$success" == "true" ]]; then
        print_success "Logout successful"
    else
        print_error "Logout failed"
    fi
    
    # Test 10: Validate logged out token (should fail)
    print_test "Token validation after logout (should fail)"
    response=$(curl -s -X GET "${API_URL}/api/v1/auth/validate" \
        -H "Authorization: Bearer ${NEW_TOKEN}")
    
    valid=$(echo "$response" | jq -r '.valid')
    if [[ "$valid" == "false" ]]; then
        print_success "Token correctly invalidated after logout"
    else
        print_error "Token still valid after logout"
    fi
    
    echo ""
    echo "==================================="
    echo -e "${GREEN}All tests passed successfully!${NC}"
    echo "==================================="
}

# Check dependencies
if ! command -v jq &> /dev/null; then
    echo "jq is required for JSON parsing. Please install it."
    exit 1
fi

if ! command -v vrooli &> /dev/null; then
    echo "vrooli CLI is required. Please install it."
    exit 1
fi

# Run tests
main "$@"
