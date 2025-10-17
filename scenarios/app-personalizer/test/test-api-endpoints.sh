#!/usr/bin/env bash
# Test script for App Personalizer API endpoints
set -euo pipefail

# Configuration
API_BASE="${APP_PERSONALIZER_API_BASE:-http://localhost:${API_PORT:-8300}}"
TEST_APP_PATH="/tmp/test-app"
TEST_APP_ID=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test helper function
test_endpoint() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    local expected_status="${4:-200}"
    local test_name="$5"
    
    log_info "Testing: $test_name"
    
    local curl_args=("-s" "-w" "%{http_code}" "-X" "$method")
    curl_args+=("-H" "Content-Type: application/json")
    
    [[ -n "$data" ]] && curl_args+=("-d" "$data")
    
    local response
    if ! response=$(curl "${curl_args[@]}" "$API_BASE$endpoint" 2>/dev/null); then
        log_error "Failed to connect to API at $API_BASE$endpoint"
        return 1
    fi
    
    local status_code="${response: -3}"
    local body="${response%???}"
    
    if [[ "$status_code" == "$expected_status" ]]; then
        log_info "✅ $test_name - Status: $status_code"
        [[ -n "$body" ]] && echo "Response: $body" | head -c 200
        echo
        return 0
    else
        log_error "❌ $test_name - Expected: $expected_status, Got: $status_code"
        [[ -n "$body" ]] && echo "Response: $body"
        return 1
    fi
}

# Setup test environment
setup_test_env() {
    log_info "Setting up test environment..."
    
    # Create a minimal test app structure
    mkdir -p "$TEST_APP_PATH/src/styles"
    cat > "$TEST_APP_PATH/package.json" << 'EOF'
{
  "name": "test-app",
  "version": "1.0.0",
  "scripts": {
    "build": "echo 'Build successful'",
    "lint": "echo 'Lint successful'"
  }
}
EOF
    
    cat > "$TEST_APP_PATH/src/styles/theme.js" << 'EOF'
export const theme = {
  colors: {
    primary: "#007bff",
    secondary: "#6c757d"
  }
};
EOF
    
    log_info "Test app created at $TEST_APP_PATH"
}

# Cleanup test environment
cleanup_test_env() {
    log_info "Cleaning up test environment..."
    [[ -d "$TEST_APP_PATH" ]] && rm -rf "$TEST_APP_PATH"
}

# Main test suite
run_tests() {
    log_info "Starting App Personalizer API tests against $API_BASE"
    
    # Test 1: Health check
    test_endpoint "GET" "/health" "" "200" "Health check"
    
    # Test 2: List apps (initially empty)
    test_endpoint "GET" "/api/apps" "" "200" "List apps"
    
    # Test 3: Register test app
    local register_data="{\"app_name\": \"test-app\", \"app_path\": \"$TEST_APP_PATH\", \"app_type\": \"generated\", \"framework\": \"react\", \"version\": \"1.0.0\"}"
    local register_response
    register_response=$(test_endpoint "POST" "/api/apps/register" "$register_data" "201" "Register app")
    
    # Extract app_id from response (basic parsing)
    if [[ "$register_response" =~ \"app_id\":\"([^\"]+)\" ]]; then
        TEST_APP_ID="${BASH_REMATCH[1]}"
        log_info "Registered app with ID: $TEST_APP_ID"
    else
        log_warn "Could not extract app_id from registration response"
    fi
    
    # Test 4: Analyze app (only if we have an app_id)
    if [[ -n "$TEST_APP_ID" ]]; then
        local analyze_data="{\"app_id\": \"$TEST_APP_ID\"}"
        test_endpoint "POST" "/api/apps/analyze" "$analyze_data" "200" "Analyze app"
    fi
    
    # Test 5: Create backup
    local backup_data="{\"app_path\": \"$TEST_APP_PATH\", \"backup_type\": \"full\"}"
    test_endpoint "POST" "/api/backup" "$backup_data" "200" "Create backup"
    
    # Test 6: Validate app
    local validate_data="{\"app_path\": \"$TEST_APP_PATH\", \"tests\": [\"build\", \"lint\"]}"
    test_endpoint "POST" "/api/validate" "$validate_data" "200" "Validate app"
    
    # Test 7: Personalize app (only if we have an app_id)
    if [[ -n "$TEST_APP_ID" ]]; then
        local personalize_data="{\"app_id\": \"$TEST_APP_ID\", \"personalization_type\": \"ui_theme\", \"deployment_mode\": \"copy\"}"
        test_endpoint "POST" "/api/personalize" "$personalize_data" "202" "Start personalization"
    fi
    
    log_info "All API tests completed!"
}

# Error handling
trap cleanup_test_env EXIT

# Run the tests
main() {
    setup_test_env
    run_tests
}

# Check if API is available before running tests
if ! curl -sf "$API_BASE/health" >/dev/null 2>&1; then
    log_error "API is not available at $API_BASE"
    log_error "Make sure the App Personalizer API server is running"
    exit 1
fi

main "$@"