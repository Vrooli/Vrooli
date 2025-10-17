#!/bin/bash

# Improved test script for AI Chatbot Manager API endpoints
# This version continues testing even if individual tests fail

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
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
API_KEY="dev-api-key-change-in-production"

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
    ((PASSED_TESTS++))
    ((TOTAL_TESTS++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED_TESTS++))
    ((TOTAL_TESTS++))
}

# Test API health endpoint
test_health_endpoint() {
    log_info "Testing health endpoint..."
    
    local response
    response=$(curl -s "$API_BASE_URL/health" | jq -r '.status' 2>/dev/null || echo "error")
    
    if [[ "$response" == "healthy" ]]; then
        log_success "Health endpoint returned healthy status"
        return 0
    else
        log_error "Health endpoint failed or returned unexpected status: $response"
        return 1
    fi
}

# Test list chatbots endpoint
test_list_chatbots() {
    log_info "Testing list chatbots endpoint..."
    
    local status_code
    status_code=$(curl -s -H "X-API-Key: $API_KEY" -o /dev/null -w "%{http_code}" "$API_BASE_URL/api/v1/chatbots")
    
    if [[ "$status_code" == "200" ]]; then
        log_success "List chatbots endpoint returned 200"
        return 0
    else
        log_error "List chatbots endpoint returned $status_code"
        return 1
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
        return 0
    else
        log_error "Failed to create chatbot. Response: $response"
        return 1
    fi
}

# Test get specific chatbot
test_get_chatbot() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        return 1
    fi
    
    log_info "Testing get chatbot endpoint..."
    
    local response
    response=$(curl -s -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/v1/chatbots/$CREATED_CHATBOT_ID")
    
    local chatbot_name
    chatbot_name=$(echo "$response" | jq -r '.name // empty' 2>/dev/null)
    
    if [[ "$chatbot_name" == "Test Chatbot" ]]; then
        log_success "Retrieved chatbot successfully"
        return 0
    else
        log_error "Failed to retrieve chatbot or incorrect data. Response: $response"
        return 1
    fi
}

# Test update chatbot endpoint
test_update_chatbot() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        return 1
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
        return 0
    else
        log_error "Failed to update chatbot. Expected 'Updated Test Chatbot', got: $updated_name"
        return 1
    fi
}

# Test chat endpoint
test_chat_endpoint() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        return 1
    fi
    
    log_info "Testing chat endpoint..."
    
    local response
    response=$(curl -s -H "X-API-Key: $API_KEY" -X POST "$API_BASE_URL/api/v1/chat/$CREATED_CHATBOT_ID" \
        -H "Content-Type: application/json" \
        -d '{
            "message": "Hello, can you help me?",
            "session_id": "'$TEST_SESSION_ID'",
            "user_ip": "127.0.0.1"
        }')
    
    local ai_response
    ai_response=$(echo "$response" | jq -r '.response // empty' 2>/dev/null)
    
    if [[ -n "$ai_response" && "$ai_response" != "null" ]]; then
        log_success "Chat endpoint returned response: ${ai_response:0:50}..."
        return 0
    else
        log_error "Chat endpoint failed or returned empty response. Response: $response"
        return 1
    fi
}

# Test analytics endpoint
test_analytics_endpoint() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        return 1
    fi
    
    log_info "Testing analytics endpoint..."
    
    local status_code
    status_code=$(curl -s -H "X-API-Key: $API_KEY" -o /dev/null -w "%{http_code}" "$API_BASE_URL/api/v1/analytics/$CREATED_CHATBOT_ID")
    
    if [[ "$status_code" == "200" ]]; then
        log_success "Analytics endpoint returned 200"
        return 0
    else
        log_error "Analytics endpoint returned $status_code"
        return 1
    fi
}

# Test widget endpoint
test_widget_endpoint() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        return 1
    fi
    
    log_info "Testing widget endpoint..."
    
    local response
    response=$(curl -s -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/v1/chatbots/$CREATED_CHATBOT_ID/widget")
    
    # Check if response contains expected widget code
    if echo "$response" | grep -q "AIChatbotWidget" && echo "$response" | grep -q "$CREATED_CHATBOT_ID"; then
        log_success "Widget endpoint returned valid embed code"
        return 0
    else
        log_error "Widget endpoint failed or returned invalid code"
        return 1
    fi
}

# Test error handling - invalid chatbot ID
test_error_handling() {
    log_info "Testing error handling with invalid chatbot ID format..."
    
    local status_code
    status_code=$(curl -s -H "X-API-Key: $API_KEY" -o /dev/null -w "%{http_code}" "$API_BASE_URL/api/v1/chatbots/invalid-id-12345")
    
    # API returns 400 for invalid UUID format, which is correct behavior
    if [[ "$status_code" == "400" ]]; then
        log_success "Properly returns 400 for invalid UUID format"
    else
        log_error "Expected 400 for invalid UUID format, got $status_code"
    fi
    
    log_info "Testing error handling with non-existent chatbot ID..."
    
    status_code=$(curl -s -H "X-API-Key: $API_KEY" -o /dev/null -w "%{http_code}" "$API_BASE_URL/api/v1/chatbots/00000000-0000-0000-0000-000000000000")
    
    # API should return 404 for valid UUID that doesn't exist
    if [[ "$status_code" == "404" ]]; then
        log_success "Properly returns 404 for non-existent chatbot"
    else
        log_error "Expected 404 for non-existent chatbot, got $status_code"
    fi
    
    log_info "Testing error handling with malformed JSON..."
    
    status_code=$(curl -s -H "X-API-Key: $API_KEY" -o /dev/null -w "%{http_code}" -X POST "$API_BASE_URL/api/v1/chatbots" \
        -H "Content-Type: application/json" \
        -d '{"invalid": json}')
    
    if [[ "$status_code" == "400" ]]; then
        log_success "Properly returns 400 for malformed JSON"
    else
        log_error "Expected 400 for malformed JSON, got $status_code"
    fi
}

# Test conversation persistence
test_conversation_persistence() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_error "No chatbot ID available for testing"
        return 1
    fi
    
    log_info "Testing conversation persistence..."
    
    # Send first message
    local response1
    response1=$(curl -s -H "X-API-Key: $API_KEY" -X POST "$API_BASE_URL/api/v1/chat/$CREATED_CHATBOT_ID" \
        -H "Content-Type: application/json" \
        -d '{
            "message": "Remember: my favorite color is blue",
            "session_id": "'$TEST_SESSION_ID'-persist",
            "user_ip": "127.0.0.1"
        }')
    
    local conv_id
    conv_id=$(echo "$response1" | jq -r '.conversation_id // empty')
    
    if [[ -n "$conv_id" ]]; then
        log_success "First message stored with conversation ID: $conv_id"
    else
        log_error "Failed to get conversation ID from first message"
        return 1
    fi
    
    # Send second message in same conversation
    local response2
    response2=$(curl -s -H "X-API-Key: $API_KEY" -X POST "$API_BASE_URL/api/v1/chat/$CREATED_CHATBOT_ID" \
        -H "Content-Type: application/json" \
        -d '{
            "message": "What is my favorite color?",
            "session_id": "'$TEST_SESSION_ID'-persist",
            "user_ip": "127.0.0.1"
        }')
    
    local ai_response
    ai_response=$(echo "$response2" | jq -r '.response // empty' 2>/dev/null)
    
    if echo "$ai_response" | grep -qi "blue"; then
        log_success "Conversation context maintained across messages"
        return 0
    else
        log_error "Conversation context not maintained. Response: $ai_response"
        return 1
    fi
}

# Test cleanup - delete created chatbot
test_cleanup() {
    if [[ -z "$CREATED_CHATBOT_ID" ]]; then
        log_info "No chatbot to clean up"
        return 0
    fi
    
    log_info "Testing delete chatbot endpoint..."
    
    local status_code
    status_code=$(curl -s -H "X-API-Key: $API_KEY" -o /dev/null -w "%{http_code}" -X DELETE "$API_BASE_URL/api/v1/chatbots/$CREATED_CHATBOT_ID")
    
    if [[ "$status_code" == "200" || "$status_code" == "204" ]]; then
        log_success "Successfully deleted test chatbot"
        return 0
    else
        log_error "Failed to delete chatbot. Status: $status_code"
        return 1
    fi
}

# Main test execution
main() {
    echo "Starting AI Chatbot Manager API tests..."
    echo "API Base URL: $API_BASE_URL"
    echo "Test Session ID: $TEST_SESSION_ID"
    echo ""
    
    # Wait for API to be available
    log_info "Waiting for API to be available..."
    local retries=0
    while ! curl -s "$API_BASE_URL/health" > /dev/null 2>&1; do
        sleep 1
        retries=$((retries + 1))
        if [[ $retries -ge 10 ]]; then
            log_error "API not available after 10 seconds"
            exit 1
        fi
    done
    log_success "API is available"
    echo ""
    
    # Run tests
    test_health_endpoint
    test_list_chatbots
    test_create_chatbot
    test_get_chatbot
    test_update_chatbot
    test_chat_endpoint
    test_analytics_endpoint
    test_widget_endpoint
    test_error_handling
    test_conversation_persistence
    test_cleanup
    
    # Report results
    echo ""
    echo "========================================"
    echo "Test Results Summary"
    echo "========================================"
    echo -e "Total Tests:  $TOTAL_TESTS"
    echo -e "Passed:       ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed:       ${RED}$FAILED_TESTS${NC}"
    
    if [[ $FAILED_TESTS -eq 0 ]]; then
        echo -e "${GREEN}✅ All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some tests failed${NC}"
        exit 1
    fi
}

main