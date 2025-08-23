#!/bin/bash

# Test chat interaction functionality

set -euo pipefail

CHAT_BASE_URL="${CHAT_BASE_URL:-http://localhost:8201}"

echo "Testing chat interaction..."

# First, create a test persona if not exists
if [[ -z "${TEST_PERSONA_ID:-}" ]]; then
    echo "Creating test persona for chat test..."
    response=$(curl -sf -X POST "$API_BASE_URL/api/persona/create" \
        -H "Content-Type: application/json" \
        -d "{
            \"name\": \"Chat Test Persona $(date +%s)\",
            \"description\": \"Test persona for chat interaction\"
        }")
    
    TEST_PERSONA_ID=$(echo "$response" | jq -r '.id')
    echo "Created test persona: $TEST_PERSONA_ID"
fi

# Test chat interaction
echo "Testing chat message..."
chat_response=$(curl -sf -X POST "$CHAT_BASE_URL/api/chat" \
    -H "Content-Type: application/json" \
    -d "{
        \"persona_id\": \"$TEST_PERSONA_ID\",
        \"message\": \"Hello, this is a test message from the integration test.\"
    }")

if echo "$chat_response" | jq -e '.response' > /dev/null; then
    session_id=$(echo "$chat_response" | jq -r '.session_id')
    persona_name=$(echo "$chat_response" | jq -r '.persona_name')
    response_text=$(echo "$chat_response" | jq -r '.response')
    
    echo "✓ Chat response received successfully"
    echo "  Persona: $persona_name"
    echo "  Session ID: $session_id"
    echo "  Response: $response_text"
    
    # Test chat history retrieval
    echo "Testing chat history retrieval..."
    history_response=$(curl -sf "$CHAT_BASE_URL/api/chat/history/$session_id?persona_id=$TEST_PERSONA_ID")
    
    if echo "$history_response" | jq -e '.messages | length > 0' > /dev/null; then
        echo "✓ Chat history retrieved successfully"
        message_count=$(echo "$history_response" | jq '.messages | length')
        echo "  Messages in history: $message_count"
    else
        echo "✗ Failed to retrieve chat history"
        exit 1
    fi
    
    # Test another message in the same session
    echo "Testing follow-up message in same session..."
    followup_response=$(curl -sf -X POST "$CHAT_BASE_URL/api/chat" \
        -H "Content-Type: application/json" \
        -d "{
            \"persona_id\": \"$TEST_PERSONA_ID\",
            \"message\": \"Thank you for the response. How are you today?\",
            \"session_id\": \"$session_id\"
        }")
    
    if echo "$followup_response" | jq -e '.response' > /dev/null; then
        echo "✓ Follow-up message processed successfully"
        
        # Check updated history
        updated_history=$(curl -sf "$CHAT_BASE_URL/api/chat/history/$session_id?persona_id=$TEST_PERSONA_ID")
        updated_count=$(echo "$updated_history" | jq '.messages | length')
        echo "  Updated message count: $updated_count"
        
        if [[ $updated_count -gt $message_count ]]; then
            echo "✓ Chat history properly updated"
        else
            echo "✗ Chat history not properly updated"
            exit 1
        fi
    else
        echo "✗ Failed to process follow-up message"
        exit 1
    fi
    
else
    echo "✗ Failed to get chat response"
    echo "Response: $chat_response"
    exit 1
fi

echo "Chat interaction test passed!"