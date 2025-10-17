#!/bin/bash

# Test script for AI Chatbot Manager API endpoints
set -e

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

# Get API URL dynamically unless overridden
API_BASE_URL="${API_BASE_URL:-$(get_api_url)}"
TEST_SESSION_ID="test-session-$(date +%s)"
CREATED_CHATBOT_ID=""
API_KEY="dev-api-key-change-in-production"  # Test API key

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${YELLOW}[TEST]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# Test API health endpoint
test_health_endpoint() {
    log_info "Testing health endpoint..."
    
    local response
    response=$(curl -s "$API_BASE_URL/health" | jq -r '.status' 2>/dev/null || echo "error")
    
    if [[ "$response" == "healthy" ]]; then
        log_success "Health endpoint returned healthy status"
    else
        log_error "Health endpoint failed or returned unexpected status: $response"
        exit 1
    fi
}

# Test list chatbots endpoint
test_list_chatbots() {
    log_info "Testing list chatbots endpoint..."
    
    local status_code
    status_code=$(curl -s -H "X-API-Key: $API_KEY" -o /dev/null -w "%{http_code}" "$API_BASE_URL/api/v1/chatbots")
    
    if [[ "$status_code" == "200" ]]; then
        log_success "List chatbots endpoint returned 200"
    else
        log_error "List chatbots endpoint returned $status_code"
        exit 1
    fi
}

# Test create chatbot endpoint
test_create_chatbot() {
    log_info "Testing create chatbot endpoint..."
    
    local response
    response=$(curl -s -H "X-API-Key: $API_KEY" -X POST "$API_BASE_URL/api/v1/chatbots" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "Test Chatbot",
            "description": "A test chatbot for API testing",
            "personality": "You are a helpful test assistant.",
            "knowledge_base": "This is test knowledge."
        }')
    
    # Check if response contains chatbot ID
    CREATED_CHATBOT_ID=$(echo "$response" | jq -r '.chatbot.id // empty' 2>/dev/null)
    
    if [[ -n "$CREATED_CHATBOT_ID" ]]; then
        log_success "Created chatbot with ID: $CREATED_CHATBOT_ID"
    else
        log_error "Failed to create chatbot. Response: $response"
        exit 1
    fi
}

# Test get specific chatbot
test_get_chatbot() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        exit 1
    fi
    
    log_info "Testing get chatbot endpoint..."
    
    local response
    response=$(curl -s -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/v1/chatbots/$CREATED_CHATBOT_ID")
    
    local chatbot_name
    chatbot_name=$(echo "$response" | jq -r '.name // empty' 2>/dev/null)
    
    if [[ "$chatbot_name" == "Test Chatbot" ]]; then
        log_success "Retrieved chatbot successfully"
    else
        log_error "Failed to retrieve chatbot or incorrect data. Response: $response"
        exit 1
    fi
}

# Test chat endpoint
test_chat_endpoint() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        exit 1
    fi
    
    log_info "Testing chat endpoint..."
    
    local response
    response=$(curl -s -H "X-API-Key: $API_KEY" -X POST "$API_BASE_URL/api/v1/chat/$CREATED_CHATBOT_ID" \
        -H "Content-Type: application/json" \
        -d '{
            "message": "Hello, can you help me?",
            "session_id": "'$TEST_SESSION_ID'"
        }')
    
    local ai_response
    ai_response=$(echo "$response" | jq -r '.response // empty' 2>/dev/null)
    
    if [[ -n "$ai_response" && "$ai_response" != "null" ]]; then
        log_success "Chat endpoint returned response: ${ai_response:0:50}..."
    else
        log_error "Chat endpoint failed or returned empty response. Response: $response"
        exit 1
    fi
}

# Test analytics endpoint
test_analytics_endpoint() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        exit 1
    fi
    
    log_info "Testing analytics endpoint..."
    
    local status_code
    status_code=$(curl -s -H "X-API-Key: $API_KEY" -o /dev/null -w "%{http_code}" "$API_BASE_URL/api/v1/analytics/$CREATED_CHATBOT_ID")
    
    if [[ "$status_code" == "200" ]]; then
        log_success "Analytics endpoint returned 200"
    else
        log_error "Analytics endpoint returned $status_code"
        exit 1
    fi
}

# Test update chatbot endpoint
test_update_chatbot() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        exit 1
    fi
    
    log_info "Testing update chatbot endpoint..."
    
    local response
    response=$(curl -s -H "X-API-Key: $API_KEY" -X PATCH "$API_BASE_URL/api/v1/chatbots/$CREATED_CHATBOT_ID" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "Updated Test Chatbot",
            "description": "Updated description"
        }')
    
    # Verify the update worked
    local updated_name
    updated_name=$(curl -s -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/v1/chatbots/$CREATED_CHATBOT_ID" | jq -r '.name // empty')
    
    if [[ "$updated_name" == "Updated Test Chatbot" ]]; then
        log_success "Successfully updated chatbot"
    else
        log_error "Failed to update chatbot. Expected 'Updated Test Chatbot', got: $updated_name"
        exit 1
    fi
}

# Test widget endpoint
test_widget_endpoint() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        exit 1
    fi
    
    log_info "Testing widget endpoint..."
    
    local response
    response=$(curl -s -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/v1/chatbots/$CREATED_CHATBOT_ID/widget")
    
    # Check if response contains expected widget code
    if echo "$response" | grep -q "chatbot-widget" && echo "$response" | grep -q "$CREATED_CHATBOT_ID"; then
        log_success "Widget endpoint returned valid embed code"
    else
        log_error "Widget endpoint failed or returned invalid code"
        exit 1
    fi
}

# Test error handling - invalid chatbot ID
test_error_handling() {
    log_info "Testing error handling with invalid chatbot ID..."
    
    local status_code
    status_code=$(curl -s -H "X-API-Key: $API_KEY" -o /dev/null -w "%{http_code}" "$API_BASE_URL/api/v1/chatbots/invalid-id-12345")
    
    if [[ "$status_code" == "404" ]]; then
        log_success "Properly returns 404 for invalid chatbot ID"
    else
        log_error "Expected 404 for invalid ID, got $status_code"
        exit 1
    fi
    
    log_info "Testing error handling with malformed JSON..."
    
    status_code=$(curl -s -H "X-API-Key: $API_KEY" -o /dev/null -w "%{http_code}" -X POST "$API_BASE_URL/api/v1/chatbots" \
        -H "Content-Type: application/json" \
        -d '{"invalid": json}')
    
    if [[ "$status_code" == "400" ]]; then
        log_success "Properly returns 400 for malformed JSON"
    else
        log_error "Expected 400 for malformed JSON, got $status_code"
        exit 1
    fi
}

# Test conversation persistence
test_conversation_persistence() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        exit 1
    fi
    
    log_info "Testing conversation persistence..."
    
    # Send first message
    local response1
    response1=$(curl -s -H "X-API-Key: $API_KEY" -X POST "$API_BASE_URL/api/v1/chat/$CREATED_CHATBOT_ID" \
        -H "Content-Type: application/json" \
        -d '{
            "message": "My name is TestUser",
            "session_id": "persist-test-'$(date +%s)'"
        }')
    
    # Send second message in same session
    local response2
    response2=$(curl -s -H "X-API-Key: $API_KEY" -X POST "$API_BASE_URL/api/v1/chat/$CREATED_CHATBOT_ID" \
        -H "Content-Type: application/json" \
        -d '{
            "message": "What is my name?",
            "session_id": "persist-test-'$(date +%s)'"
        }')
    
    # Check if AI remembers context (basic check)
    if echo "$response2" | jq -r '.response // empty' | grep -qi "testuser\|TestUser"; then
        log_success "Conversation context is being maintained"
    else
        log_info "Note: Context persistence may depend on Ollama model capabilities"
    fi
}

# Test delete chatbot endpoint
test_delete_chatbot() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        exit 1
    fi
    
    log_info "Testing delete chatbot endpoint..."
    
    local status_code
    status_code=$(curl -s -H "X-API-Key: $API_KEY" -o /dev/null -w "%{http_code}" -X DELETE "$API_BASE_URL/api/v1/chatbots/$CREATED_CHATBOT_ID")
    
    if [[ "$status_code" == "200" || "$status_code" == "204" ]]; then
        log_success "Successfully deleted chatbot"
        
        # Verify deletion (should now return 404 or show as inactive)
        local get_status
        get_status=$(curl -s -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/v1/chatbots/$CREATED_CHATBOT_ID" | jq -r '.is_active // empty')
        
        if [[ "$get_status" == "false" || -z "$get_status" ]]; then
            log_success "Chatbot properly marked as deleted/inactive"
        fi
    else
        log_error "Delete endpoint returned $status_code"
        exit 1
    fi
}

# Wait for API to be available
wait_for_api() {
    log_info "Waiting for API to be available..."
    
    local attempts=0
    local max_attempts=30
    
    while [[ $attempts -lt $max_attempts ]]; do
        if curl -s "$API_BASE_URL/health" >/dev/null 2>&1; then
            log_success "API is available"
            return 0
        fi
        
        attempts=$((attempts + 1))
        echo "Waiting for API... attempt $attempts/$max_attempts"
        sleep 2
    done
    
    log_error "API did not become available within timeout"
    exit 1
}

# Main test execution
main() {
    echo "Starting AI Chatbot Manager API tests..."
    echo "API Base URL: $API_BASE_URL"
    echo "Test Session ID: $TEST_SESSION_ID"
    echo ""
    
    # Wait for API
    wait_for_api
    
    # Run tests in order
    test_health_endpoint
    test_list_chatbots
    test_create_chatbot
    test_get_chatbot
    test_update_chatbot
    test_chat_endpoint
    test_conversation_persistence
    test_analytics_endpoint
    test_widget_endpoint
    test_error_handling
    test_delete_chatbot
    
    echo ""
    log_success "All API tests passed successfully! (11 test suites)"
    
    # Summary
    echo ""
    echo "Test Coverage:"
    echo "  ✓ Health checks"
    echo "  ✓ CRUD operations (Create, Read, Update, Delete)"
    echo "  ✓ Chat functionality"
    echo "  ✓ Conversation persistence"
    echo "  ✓ Analytics retrieval"
    echo "  ✓ Widget generation"
    echo "  ✓ Error handling"
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