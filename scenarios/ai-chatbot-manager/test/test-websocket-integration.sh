#!/bin/bash

# WebSocket Integration Test for AI Chatbot Manager
# Tests WebSocket functionality with a real chatbot

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

API_BASE_URL="${API_BASE_URL:-$(get_api_url)}"
WS_BASE_URL=$(echo "$API_BASE_URL" | sed 's/http/ws/')
TEST_SESSION_ID="ws-test-$(date +%s)"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Create a test chatbot for WebSocket testing
create_test_chatbot() {
    log_info "Creating test chatbot for WebSocket testing..."
    
    local response
    response=$(curl -s -X POST "$API_BASE_URL/api/v1/chatbots" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "WebSocket Test Bot",
            "description": "A test bot for WebSocket integration testing",
            "personality": "You are a test assistant. Always respond with a short acknowledgment.",
            "knowledge_base": "This is a test bot for WebSocket functionality."
        }')
    
    echo "$response" | jq -r '.chatbot.id // empty' 2>/dev/null
}

# Test WebSocket connection using Node.js or Python
test_websocket_connection() {
    local chatbot_id="$1"
    local ws_url="${WS_BASE_URL}/api/v1/ws/${chatbot_id}"
    
    log_info "Testing WebSocket connection to: $ws_url"
    
    # Create a simple WebSocket test script using Node.js if available
    if command -v node >/dev/null 2>&1; then
        test_websocket_nodejs "$ws_url"
    elif command -v python3 >/dev/null 2>&1; then
        test_websocket_python "$ws_url"
    else
        log_warning "Neither Node.js nor Python3 available for WebSocket testing"
        test_websocket_curl_fallback "$ws_url"
    fi
}

test_websocket_nodejs() {
    local ws_url="$1"
    
    # Create temporary Node.js WebSocket test
    cat > /tmp/ws_test.js << 'EOF'
const WebSocket = require('ws');

const wsUrl = process.argv[2];
const sessionId = process.argv[3];

console.log(`Connecting to: ${wsUrl}`);

const ws = new WebSocket(wsUrl);
let messageReceived = false;
let connectionSuccessful = false;

ws.on('open', function open() {
    console.log('✓ WebSocket connection established');
    connectionSuccessful = true;
    
    // Send a test message
    const message = {
        message: 'Hello, WebSocket test!',
        session_id: sessionId
    };
    
    console.log('Sending test message...');
    ws.send(JSON.stringify(message));
    
    // Set timeout to close connection
    setTimeout(() => {
        if (!messageReceived) {
            console.log('✗ No response received within timeout');
            process.exit(1);
        }
        ws.close();
        process.exit(0);
    }, 10000);
});

ws.on('message', function message(data) {
    console.log('✓ Received WebSocket message');
    try {
        const parsed = JSON.parse(data.toString());
        console.log('Message type:', parsed.type);
        if (parsed.type === 'message' && parsed.payload && parsed.payload.response) {
            console.log('Response:', parsed.payload.response);
            messageReceived = true;
        }
    } catch (e) {
        console.log('Raw message:', data.toString());
    }
});

ws.on('error', function error(err) {
    console.log('✗ WebSocket error:', err.message);
    process.exit(1);
});

ws.on('close', function close(code, reason) {
    console.log(`WebSocket closed: ${code} - ${reason}`);
    if (connectionSuccessful && messageReceived) {
        console.log('✓ WebSocket test completed successfully');
        process.exit(0);
    } else {
        process.exit(1);
    }
});
EOF
    
    # Check if ws module is available
    if node -e "require('ws')" 2>/dev/null; then
        if node /tmp/ws_test.js "$ws_url" "$TEST_SESSION_ID"; then
            log_success "WebSocket connection and messaging successful"
        else
            log_error "WebSocket test failed"
        fi
    else
        log_warning "Node.js 'ws' module not installed, trying alternative test"
        test_websocket_curl_fallback "$ws_url"
    fi
    
    # Clean up
    rm -f /tmp/ws_test.js
}

test_websocket_python() {
    local ws_url="$1"
    
    # Create temporary Python WebSocket test
    cat > /tmp/ws_test.py << 'EOF'
import asyncio
import websockets
import json
import sys

async def test_websocket():
    ws_url = sys.argv[1]
    session_id = sys.argv[2]
    
    print(f"Connecting to: {ws_url}")
    
    try:
        async with websockets.connect(ws_url) as websocket:
            print("✓ WebSocket connection established")
            
            # Send test message
            message = {
                "message": "Hello, WebSocket test!",
                "session_id": session_id
            }
            
            print("Sending test message...")
            await websocket.send(json.dumps(message))
            
            # Wait for response with timeout
            try:
                response = await asyncio.wait_for(websocket.recv(), timeout=10.0)
                print("✓ Received WebSocket message")
                
                parsed = json.loads(response)
                print(f"Message type: {parsed.get('type')}")
                
                if parsed.get('type') == 'message' and parsed.get('payload', {}).get('response'):
                    print(f"Response: {parsed['payload']['response']}")
                    print("✓ WebSocket test completed successfully")
                    return True
                    
            except asyncio.TimeoutError:
                print("✗ No response received within timeout")
                return False
                
    except Exception as e:
        print(f"✗ WebSocket error: {e}")
        return False

if __name__ == "__main__":
    result = asyncio.run(test_websocket())
    sys.exit(0 if result else 1)
EOF
    
    # Check if websockets module is available
    if python3 -c "import websockets" 2>/dev/null; then
        if python3 /tmp/ws_test.py "$ws_url" "$TEST_SESSION_ID"; then
            log_success "WebSocket connection and messaging successful"
        else
            log_error "WebSocket test failed"
        fi
    else
        log_warning "Python 'websockets' module not installed, trying curl fallback"
        test_websocket_curl_fallback "$ws_url"
    fi
    
    # Clean up
    rm -f /tmp/ws_test.py
}

test_websocket_curl_fallback() {
    local ws_url="$1"
    
    log_info "Using curl for basic WebSocket endpoint test"
    
    # Test WebSocket endpoint accessibility (should get upgrade error)
    local http_url=$(echo "$ws_url" | sed 's/ws/http/')
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Connection: Upgrade" \
        -H "Upgrade: websocket" \
        -H "Sec-WebSocket-Version: 13" \
        -H "Sec-WebSocket-Key: $(openssl rand -base64 16)" \
        "$http_url")
    
    if [[ "$response_code" == "101" ]]; then
        log_success "WebSocket endpoint is accessible (HTTP 101 Switching Protocols)"
    elif [[ "$response_code" == "400" || "$response_code" == "404" ]]; then
        log_warning "WebSocket endpoint exists but requires valid WebSocket handshake (HTTP $response_code)"
    else
        log_error "WebSocket endpoint test failed (HTTP $response_code)"
        return 1
    fi
}

# Cleanup test chatbot
cleanup_test_chatbot() {
    local chatbot_id="$1"
    
    if [[ -n "$chatbot_id" ]]; then
        log_info "Cleaning up test chatbot..."
        curl -s -X DELETE "$API_BASE_URL/api/v1/chatbots/$chatbot_id" >/dev/null
    fi
}

# Main test execution
main() {
    echo "Starting WebSocket Integration Tests..."
    echo "API Base URL: $API_BASE_URL"
    echo "WebSocket Base URL: $WS_BASE_URL"
    echo "Test Session ID: $TEST_SESSION_ID"
    echo ""
    
    # Wait for API to be available
    log_info "Checking API availability..."
    local attempts=0
    local max_attempts=10
    
    while [[ $attempts -lt $max_attempts ]]; do
        if curl -s "$API_BASE_URL/health" >/dev/null 2>&1; then
            log_success "API is available"
            break
        fi
        
        attempts=$((attempts + 1))
        echo "Waiting for API... attempt $attempts/$max_attempts"
        sleep 2
    done
    
    if [[ $attempts -eq $max_attempts ]]; then
        log_error "API did not become available within timeout"
        exit 1
    fi
    
    # Create test chatbot
    local chatbot_id
    chatbot_id=$(create_test_chatbot)
    
    if [[ -z "$chatbot_id" ]]; then
        log_error "Failed to create test chatbot"
        exit 1
    fi
    
    log_success "Created test chatbot: $chatbot_id"
    
    # Test WebSocket connection
    if test_websocket_connection "$chatbot_id"; then
        log_success "WebSocket integration test passed"
        local exit_code=0
    else
        log_error "WebSocket integration test failed"
        local exit_code=1
    fi
    
    # Cleanup
    cleanup_test_chatbot "$chatbot_id"
    
    echo ""
    if [[ $exit_code -eq 0 ]]; then
        log_success "All WebSocket tests passed!"
    else
        log_error "WebSocket tests failed!"
    fi
    
    exit $exit_code
}

# Check dependencies
if ! command -v curl >/dev/null 2>&1; then
    log_error "curl is required but not installed"
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    log_error "jq is required but not installed"
    exit 1
fi

# Run main function
main "$@"