#!/bin/bash

# WebSocket test for AI Chatbot Manager
# Tests real-time chat functionality via WebSocket connection

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Dynamic port discovery
get_api_url() {
    local api_port
    api_port=$(vrooli scenario port ai-chatbot-manager API_PORT 2>/dev/null)
    
    if [[ -z "$api_port" ]]; then
        echo "ERROR: ai-chatbot-manager is not running" >&2
        echo "Start it with: vrooli scenario run ai-chatbot-manager" >&2
        exit 1
    fi
    
    echo "http://localhost:${api_port}"
}

# Get API URL
API_BASE_URL="${API_BASE_URL:-$(get_api_url)}"
WS_URL="${API_BASE_URL/http/ws}"

log_info() {
    echo -e "${YELLOW}[WS-TEST]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# Check dependencies
check_dependencies() {
    local missing_deps=()
    
    if ! command -v websocat >/dev/null 2>&1 && ! command -v wscat >/dev/null 2>&1; then
        missing_deps+=("websocat or wscat")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        echo "Install with:"
        echo "  - websocat: cargo install websocat"
        echo "  - wscat: npm install -g wscat"
        echo "  - jq: apt-get install jq (or brew install jq)"
        exit 1
    fi
}

# Create a test chatbot first
create_test_chatbot() {
    log_info "Creating test chatbot for WebSocket testing..."
    
    local response
    response=$(curl -s -X POST "$API_BASE_URL/api/v1/chatbots" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "WebSocket Test Bot",
            "description": "Bot for WebSocket testing",
            "personality": "You are a helpful test assistant. Keep responses brief.",
            "knowledge_base": "This is a test bot for WebSocket functionality."
        }')
    
    local chatbot_id
    chatbot_id=$(echo "$response" | jq -r '.chatbot.id // empty')
    
    if [[ -z "$chatbot_id" ]]; then
        log_error "Failed to create test chatbot"
        exit 1
    fi
    
    log_success "Created test chatbot: $chatbot_id"
    echo "$chatbot_id"
}

# Test WebSocket connection with websocat
test_websocket_websocat() {
    local chatbot_id="$1"
    local ws_endpoint="$WS_URL/api/v1/ws/$chatbot_id"
    
    log_info "Testing WebSocket connection to: $ws_endpoint"
    
    # Create test message
    local test_message='{"message":"Hello from WebSocket test","session_id":"ws-test-123"}'
    
    # Send message and capture response
    local response
    response=$(echo "$test_message" | timeout 5 websocat -t "$ws_endpoint" 2>/dev/null || true)
    
    if [[ -n "$response" ]]; then
        log_success "Received WebSocket response"
        echo "Response: $response"
        
        # Try to parse as JSON
        if echo "$response" | jq . >/dev/null 2>&1; then
            local ai_response
            ai_response=$(echo "$response" | jq -r '.response // empty')
            if [[ -n "$ai_response" ]]; then
                log_success "AI responded: ${ai_response:0:50}..."
                return 0
            fi
        fi
    else
        log_error "No response received from WebSocket"
        return 1
    fi
}

# Test WebSocket connection with wscat (alternative)
test_websocket_wscat() {
    local chatbot_id="$1"
    local ws_endpoint="$WS_URL/api/v1/ws/$chatbot_id"
    
    log_info "Testing WebSocket connection with wscat to: $ws_endpoint"
    
    # Create test message
    local test_message='{"message":"Hello from WebSocket test","session_id":"ws-test-123"}'
    
    # Use wscat with timeout
    local response
    response=$(echo "$test_message" | timeout 5 wscat -c "$ws_endpoint" 2>/dev/null || true)
    
    if [[ -n "$response" ]]; then
        log_success "Received WebSocket response via wscat"
        echo "Response: $response"
        return 0
    else
        log_error "No response received from WebSocket via wscat"
        return 1
    fi
}

# Test multiple messages in sequence
test_websocket_conversation() {
    local chatbot_id="$1"
    local ws_endpoint="$WS_URL/api/v1/ws/$chatbot_id"
    
    log_info "Testing WebSocket conversation flow..."
    
    # Create a temporary file for the conversation
    local tmpfile=$(mktemp)
    
    # Multiple test messages
    cat > "$tmpfile" << EOF
{"message":"Hello, I need help","session_id":"ws-conv-test"}
{"message":"What can you do?","session_id":"ws-conv-test"}
{"message":"Thank you","session_id":"ws-conv-test"}
EOF
    
    # Send messages and capture responses
    local responses
    if command -v websocat >/dev/null 2>&1; then
        responses=$(cat "$tmpfile" | timeout 10 websocat -t "$ws_endpoint" 2>/dev/null || true)
    else
        responses=$(cat "$tmpfile" | timeout 10 wscat -c "$ws_endpoint" 2>/dev/null || true)
    fi
    
    rm -f "$tmpfile"
    
    if [[ -n "$responses" ]]; then
        local response_count
        response_count=$(echo "$responses" | wc -l)
        log_success "Received $response_count responses in conversation"
        return 0
    else
        log_error "Conversation test failed"
        return 1
    fi
}

# Test WebSocket error handling
test_websocket_error_handling() {
    log_info "Testing WebSocket error handling..."
    
    # Test with invalid chatbot ID
    local invalid_id="00000000-0000-0000-0000-000000000000"
    local ws_endpoint="$WS_URL/api/v1/ws/$invalid_id"
    
    local response
    if command -v websocat >/dev/null 2>&1; then
        response=$(echo '{"message":"test"}' | timeout 3 websocat "$ws_endpoint" 2>&1 || true)
    else
        response=$(echo '{"message":"test"}' | timeout 3 wscat -c "$ws_endpoint" 2>&1 || true)
    fi
    
    # We expect either connection failure or error response
    if [[ "$response" =~ "error" ]] || [[ "$response" =~ "Error" ]] || [[ -z "$response" ]]; then
        log_success "Error handling works correctly"
        return 0
    else
        log_error "Unexpected response for invalid chatbot"
        return 1
    fi
}

# Main test execution
main() {
    echo "========================================="
    echo "AI Chatbot Manager WebSocket Tests"
    echo "========================================="
    echo "API URL: $API_BASE_URL"
    echo "WebSocket URL: $WS_URL"
    echo ""
    
    # Check dependencies
    check_dependencies
    
    # Create test chatbot
    CHATBOT_ID=$(create_test_chatbot)
    
    echo ""
    log_info "Starting WebSocket tests..."
    echo ""
    
    # Run tests
    local failed=0
    
    # Test basic WebSocket connection
    if command -v websocat >/dev/null 2>&1; then
        if ! test_websocket_websocat "$CHATBOT_ID"; then
            failed=$((failed + 1))
        fi
    elif command -v wscat >/dev/null 2>&1; then
        if ! test_websocket_wscat "$CHATBOT_ID"; then
            failed=$((failed + 1))
        fi
    fi
    
    echo ""
    
    # Test conversation flow
    if ! test_websocket_conversation "$CHATBOT_ID"; then
        failed=$((failed + 1))
    fi
    
    echo ""
    
    # Test error handling
    if ! test_websocket_error_handling; then
        failed=$((failed + 1))
    fi
    
    echo ""
    echo "========================================="
    
    if [[ $failed -eq 0 ]]; then
        log_success "All WebSocket tests passed!"
        echo ""
        echo "Test chatbot ID: $CHATBOT_ID"
        echo "You can test manually with:"
        echo "  websocat '$WS_URL/api/v1/ws/$CHATBOT_ID'"
        echo "  Then type: {\"message\":\"Hello\",\"session_id\":\"manual\"}"
        exit 0
    else
        log_error "$failed WebSocket tests failed"
        exit 1
    fi
}

# Run main function
main "$@"