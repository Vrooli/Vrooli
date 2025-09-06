#!/bin/bash

# Test script for AI Chatbot Manager API endpoints
set -e

API_BASE_URL="${API_BASE_URL:-http://localhost:8090}"
TEST_SESSION_ID="test-session-$(date +%s)"
CREATED_CHATBOT_ID=""

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
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE_URL/api/v1/chatbots")
    
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
    response=$(curl -s -X POST "$API_BASE_URL/api/v1/chatbots" \
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
    response=$(curl -s "$API_BASE_URL/api/v1/chatbots/$CREATED_CHATBOT_ID")
    
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
    response=$(curl -s -X POST "$API_BASE_URL/api/v1/chat/$CREATED_CHATBOT_ID" \
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
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE_URL/api/v1/analytics/$CREATED_CHATBOT_ID")
    
    if [[ "$status_code" == "200" ]]; then
        log_success "Analytics endpoint returned 200"
    else
        log_error "Analytics endpoint returned $status_code"
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
    test_chat_endpoint
    test_analytics_endpoint
    
    echo ""
    log_success "All API tests passed!"
    
    if [[ -n "$CREATED_CHATBOT_ID" ]]; then
        echo "Test chatbot created: $CREATED_CHATBOT_ID"
        echo "You can test it manually with:"
        echo "  curl -X POST $API_BASE_URL/api/v1/chat/$CREATED_CHATBOT_ID \\"
        echo "       -H 'Content-Type: application/json' \\"
        echo "       -d '{\"message\": \"Hello!\", \"session_id\": \"manual-test\"}'"
    fi
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