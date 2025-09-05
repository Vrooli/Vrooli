#!/usr/bin/env bash

# Device Sync Hub Integration Tests
# Comprehensive testing of the cross-device sync functionality

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
TEST_DATA_DIR="$SCRIPT_DIR/data"
API_URL="${API_URL:-http://localhost:3300}"
AUTH_URL="${AUTH_URL:-http://localhost:3250}"
UI_URL="${UI_URL:-http://localhost:3301}"

# Test user credentials
TEST_EMAIL="${TEST_EMAIL:-test@example.com}"
TEST_PASSWORD="${TEST_PASSWORD:-TestPassword123!}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

# Test state
TESTS_PASSED=0
TESTS_FAILED=0
AUTH_TOKEN=""
TEST_ITEM_IDS=()

# Setup
mkdir -p "$TEST_DATA_DIR"

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_failure() {
    echo -e "${RED}[FAIL]${NC} $1" >&2
    ((TESTS_FAILED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_bold() {
    echo -e "${BOLD}$1${NC}"
}

# Test helper functions
assert_http_status() {
    local url="$1"
    local expected_status="$2"
    local method="${3:-GET}"
    local data="${4:-}"
    local headers="${5:-}"
    
    local args=(-s -o /dev/null -w "%{http_code}")
    
    if [[ -n "$headers" ]]; then
        args+=(-H "$headers")
    fi
    
    if [[ "$method" != "GET" ]]; then
        args+=(-X "$method")
    fi
    
    if [[ -n "$data" ]]; then
        args+=(-d "$data")
    fi
    
    local status
    status=$(curl "${args[@]}" "$url")
    
    if [[ "$status" == "$expected_status" ]]; then
        return 0
    else
        echo "Expected $expected_status, got $status"
        return 1
    fi
}

api_request() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    local content_type="${4:-application/json}"
    
    local args=(-s)
    
    if [[ -n "$AUTH_TOKEN" ]]; then
        args+=(-H "Authorization: Bearer $AUTH_TOKEN")
    fi
    
    if [[ "$method" != "GET" ]]; then
        args+=(-X "$method")
    fi
    
    if [[ -n "$data" ]]; then
        if [[ "$content_type" == "multipart/form-data" ]]; then
            args+=($data)  # $data should contain -F flags for form data
        else
            args+=(-H "Content-Type: $content_type" -d "$data")
        fi
    fi
    
    curl "${args[@]}" "$API_URL$endpoint"
}

create_test_file() {
    local filename="$1"
    local content="$2"
    local filepath="$TEST_DATA_DIR/$filename"
    
    echo "$content" > "$filepath"
    echo "$filepath"
}

cleanup_test_files() {
    if [[ -d "$TEST_DATA_DIR" ]]; then
        rm -rf "$TEST_DATA_DIR"/*
    fi
    
    # Cleanup any uploaded items
    for item_id in "${TEST_ITEM_IDS[@]}"; do
        api_request "DELETE" "/api/v1/sync/items/$item_id" >/dev/null 2>&1 || true
    done
    TEST_ITEM_IDS=()
}

# Test functions
test_service_health() {
    log_info "Testing service health checks..."
    
    # Test API health
    if assert_http_status "$API_URL/health" "200"; then
        log_success "API health check passes"
    else
        log_failure "API health check failed"
    fi
    
    # Test UI availability
    if assert_http_status "$UI_URL/" "200"; then
        log_success "Web UI is accessible"
    else
        log_failure "Web UI is not accessible"
    fi
    
    # Test auth service (should return 401 without token)
    if assert_http_status "$AUTH_URL/api/v1/auth/validate" "401"; then
        log_success "Auth service is responding"
    else
        log_failure "Auth service is not responding correctly"
    fi
}

test_authentication() {
    log_info "Testing authentication flow..."
    
    # Create test user (may already exist)
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$TEST_EMAIL\", \"password\": \"$TEST_PASSWORD\"}" \
        "$AUTH_URL/api/v1/auth/register" >/dev/null 2>&1 || true
    
    # Login and get token
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$TEST_EMAIL\", \"password\": \"$TEST_PASSWORD\"}" \
        "$AUTH_URL/api/v1/auth/login" || echo "")
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.token' >/dev/null 2>&1; then
        AUTH_TOKEN=$(echo "$response" | jq -r '.token')
        log_success "Authentication successful"
    else
        log_failure "Authentication failed"
        return 1
    fi
    
    # Validate token
    local validation
    validation=$(curl -s -H "Authorization: Bearer $AUTH_TOKEN" \
        "$AUTH_URL/api/v1/auth/validate" || echo "")
    
    if [[ -n "$validation" ]] && echo "$validation" | jq -e '.valid' >/dev/null 2>&1; then
        log_success "Token validation successful"
    else
        log_failure "Token validation failed"
        return 1
    fi
}

test_file_upload() {
    log_info "Testing file upload functionality..."
    
    # Create test file
    local test_file
    test_file=$(create_test_file "test.txt" "This is a test file for Device Sync Hub")
    
    # Upload file
    local response
    response=$(api_request "POST" "/api/v1/sync/upload" \
        "-F file=@$test_file -F content_type=file -F expires_in=1" \
        "multipart/form-data" || echo "")
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        local item_id
        item_id=$(echo "$response" | jq -r '.item_id')
        TEST_ITEM_IDS+=("$item_id")
        log_success "File upload successful (ID: ${item_id:0:8}...)"
    else
        log_failure "File upload failed"
        return 1
    fi
}

test_text_upload() {
    log_info "Testing text upload functionality..."
    
    local test_text="Hello from Device Sync Hub integration test!"
    
    # Upload text
    local response
    response=$(api_request "POST" "/api/v1/sync/upload" \
        "{\"text\": \"$test_text\", \"content_type\": \"text\", \"expires_in\": 1}" || echo "")
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        local item_id
        item_id=$(echo "$response" | jq -r '.item_id')
        TEST_ITEM_IDS+=("$item_id")
        log_success "Text upload successful (ID: ${item_id:0:8}...)"
    else
        log_failure "Text upload failed"
        return 1
    fi
}

test_clipboard_upload() {
    log_info "Testing clipboard upload functionality..."
    
    local clipboard_content="Clipboard content from integration test"
    
    # Upload clipboard content
    local response
    response=$(api_request "POST" "/api/v1/sync/upload" \
        "{\"text\": \"$clipboard_content\", \"content_type\": \"clipboard\", \"expires_in\": 1}" || echo "")
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        local item_id
        item_id=$(echo "$response" | jq -r '.item_id')
        TEST_ITEM_IDS+=("$item_id")
        log_success "Clipboard upload successful (ID: ${item_id:0:8}...)"
    else
        log_failure "Clipboard upload failed"
        return 1
    fi
}

test_list_items() {
    log_info "Testing item listing functionality..."
    
    # List all items
    local response
    response=$(api_request "GET" "/api/v1/sync/items" || echo "")
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.items' >/dev/null 2>&1; then
        local item_count
        item_count=$(echo "$response" | jq -r '.items | length')
        
        if [[ "$item_count" -ge "${#TEST_ITEM_IDS[@]}" ]]; then
            log_success "Item listing successful ($item_count items found)"
        else
            log_failure "Item listing incomplete (expected >= ${#TEST_ITEM_IDS[@]}, got $item_count)"
        fi
    else
        log_failure "Item listing failed"
        return 1
    fi
}

test_item_download() {
    log_info "Testing item download functionality..."
    
    if [[ ${#TEST_ITEM_IDS[@]} -eq 0 ]]; then
        log_warning "No test items to download"
        return
    fi
    
    local item_id="${TEST_ITEM_IDS[0]}"
    local download_path="$TEST_DATA_DIR/downloaded_file"
    
    # Download item
    local http_code
    http_code=$(curl -s -w "%{http_code}" -o "$download_path" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        "$API_URL/api/v1/sync/items/$item_id/download")
    
    if [[ "$http_code" == "200" ]] && [[ -f "$download_path" ]]; then
        log_success "Item download successful"
    else
        log_failure "Item download failed (HTTP $http_code)"
        return 1
    fi
}

test_item_deletion() {
    log_info "Testing item deletion functionality..."
    
    if [[ ${#TEST_ITEM_IDS[@]} -eq 0 ]]; then
        log_warning "No test items to delete"
        return
    fi
    
    local item_id="${TEST_ITEM_IDS[0]}"
    
    # Delete item
    local response
    response=$(api_request "DELETE" "/api/v1/sync/items/$item_id" || echo "")
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        log_success "Item deletion successful"
        # Remove from test items array
        TEST_ITEM_IDS=("${TEST_ITEM_IDS[@]/$item_id}")
    else
        log_failure "Item deletion failed"
        return 1
    fi
}

test_image_upload_thumbnail() {
    log_info "Testing image upload with thumbnail generation..."
    
    # Create a simple test image (1x1 PNG)
    local test_image="$TEST_DATA_DIR/test.png"
    # This is a minimal 1x1 transparent PNG
    echo -n -e '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\x0f\x00\x00\x01\x00\x01\x00\x00\x00\x00\x00\x00\x8a\x1e\xe5\x7f\x00\x00\x00\x00IEND\xaeB`\x82' > "$test_image"
    
    # Upload image
    local response
    response=$(api_request "POST" "/api/v1/sync/upload" \
        "-F file=@$test_image -F content_type=file -F expires_in=1" \
        "multipart/form-data" || echo "")
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.success' >/dev/null 2>&1; then
        local item_id thumbnail_url
        item_id=$(echo "$response" | jq -r '.item_id')
        thumbnail_url=$(echo "$response" | jq -r '.thumbnail_url // ""')
        TEST_ITEM_IDS+=("$item_id")
        
        if [[ -n "$thumbnail_url" ]]; then
            log_success "Image upload with thumbnail successful"
        else
            log_warning "Image upload successful but no thumbnail generated"
        fi
    else
        log_failure "Image upload failed"
        return 1
    fi
}

test_settings_endpoint() {
    log_info "Testing settings endpoint..."
    
    local response
    response=$(api_request "GET" "/api/v1/sync/settings" || echo "")
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.max_file_size' >/dev/null 2>&1; then
        local max_size default_expiry
        max_size=$(echo "$response" | jq -r '.max_file_size')
        default_expiry=$(echo "$response" | jq -r '.default_expiry_hours')
        
        log_success "Settings endpoint working (max_size: $max_size, default_expiry: ${default_expiry}h)"
    else
        log_failure "Settings endpoint failed"
        return 1
    fi
}

test_cli_functionality() {
    log_info "Testing CLI functionality..."
    
    local cli_path="$SCENARIO_DIR/cli/device-sync-hub"
    
    if [[ ! -x "$cli_path" ]]; then
        log_failure "CLI not found or not executable: $cli_path"
        return 1
    fi
    
    # Test version command
    if "$cli_path" version >/dev/null 2>&1; then
        log_success "CLI version command works"
    else
        log_failure "CLI version command failed"
        return 1
    fi
    
    # Test help command
    if "$cli_path" help >/dev/null 2>&1; then
        log_success "CLI help command works"  
    else
        log_failure "CLI help command failed"
        return 1
    fi
}

test_websocket_connection() {
    log_info "Testing WebSocket connectivity..."
    
    # Use a simple Node.js script to test WebSocket connection
    local ws_test_script="$TEST_DATA_DIR/ws_test.js"
    
    cat << 'EOF' > "$ws_test_script"
const WebSocket = require('ws');

const wsUrl = process.env.WS_URL || 'ws://localhost:3300/api/v1/sync/websocket';
const authToken = process.env.AUTH_TOKEN || '';

const ws = new WebSocket(wsUrl);
let connected = false;

ws.on('open', () => {
  connected = true;
  console.log('WebSocket connected');
  
  // Send auth message
  ws.send(JSON.stringify({
    type: 'auth',
    token: authToken,
    device_info: { platform: 'test', browser: 'integration-test' }
  }));
  
  setTimeout(() => {
    ws.close();
    process.exit(0);
  }, 2000);
});

ws.on('message', (data) => {
  const message = JSON.parse(data);
  console.log('Received:', message.type);
  
  if (message.type === 'auth_success') {
    console.log('WebSocket authentication successful');
  }
});

ws.on('error', (error) => {
  console.error('WebSocket error:', error.message);
  process.exit(1);
});

ws.on('close', () => {
  if (connected) {
    console.log('WebSocket closed gracefully');
    process.exit(0);
  } else {
    console.error('WebSocket connection failed');
    process.exit(1);
  }
});

// Timeout after 5 seconds
setTimeout(() => {
  if (!connected) {
    console.error('WebSocket connection timeout');
    process.exit(1);
  }
}, 5000);
EOF
    
    # Run WebSocket test
    if WS_URL="ws://localhost:3300/api/v1/sync/websocket" AUTH_TOKEN="$AUTH_TOKEN" \
       node "$ws_test_script" >/dev/null 2>&1; then
        log_success "WebSocket connection test passed"
    else
        log_failure "WebSocket connection test failed"
        return 1
    fi
}

# Main test execution
main() {
    log_bold "=== Device Sync Hub Integration Tests ==="
    log_info "Starting comprehensive integration tests..."
    echo
    
    # Cleanup from any previous runs
    cleanup_test_files
    
    # Run tests
    test_service_health
    test_authentication
    test_file_upload
    test_text_upload
    test_clipboard_upload
    test_list_items
    test_item_download
    test_settings_endpoint
    test_image_upload_thumbnail
    test_item_deletion
    test_cli_functionality
    test_websocket_connection
    
    # Cleanup
    cleanup_test_files
    
    # Results
    echo
    log_bold "=== Test Results ==="
    log_success "Tests passed: $TESTS_PASSED"
    if [[ $TESTS_FAILED -gt 0 ]]; then
        log_failure "Tests failed: $TESTS_FAILED"
        exit 1
    else
        log_success "All tests passed!"
        exit 0
    fi
}

# Check dependencies
if ! command -v curl >/dev/null 2>&1; then
    log_failure "curl is required but not installed"
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    log_failure "jq is required but not installed"
    exit 1
fi

if ! command -v node >/dev/null 2>&1; then
    log_warning "node.js not found - WebSocket test will be skipped"
fi

# Run main function
main "$@"