#!/bin/bash
# Basic LiteLLM Chat Example
# Demonstrates simple chat completion using LiteLLM proxy

set -e

# Configuration
LITELLM_URL="http://localhost:11435"
MODEL="gpt-3.5-turbo"
MESSAGE="Hello! Can you tell me about LiteLLM?"

echo "🤖 LiteLLM Basic Chat Example"
echo "============================="
echo

# Check if LiteLLM is running
echo "Checking LiteLLM status..."
if ! resource-litellm test >/dev/null 2>&1; then
    echo "❌ LiteLLM is not running or not accessible"
    echo "   Please start LiteLLM first: resource-litellm start"
    exit 1
fi
echo "✅ LiteLLM is running"
echo

# Get master key
echo "Getting master key..."
MASTER_KEY=$(grep "LITELLM_MASTER_KEY=" "${var_ROOT_DIR}/data/litellm/config/.env" 2>/dev/null | cut -d'=' -f2 | tr -d '"' || echo "")

if [[ -z "$MASTER_KEY" ]]; then
    echo "❌ Master key not found"
    echo "   Please check your LiteLLM configuration"
    exit 1
fi
echo "✅ Master key obtained"
echo

# Make API request
echo "Sending chat request..."
echo "Model: $MODEL"
echo "Message: $MESSAGE"
echo

response=$(curl -s -X POST "$LITELLM_URL/chat/completions" \
    -H "Authorization: Bearer $MASTER_KEY" \
    -H "Content-Type: application/json" \
    -d "{
        \"model\": \"$MODEL\",
        \"messages\": [
            {\"role\": \"user\", \"content\": \"$MESSAGE\"}
        ],
        \"max_tokens\": 150,
        \"temperature\": 0.7
    }")

# Check for errors
if echo "$response" | grep -q '"error"'; then
    echo "❌ API Error:"
    echo "$response" | jq -r '.error.message // .error' 2>/dev/null || echo "$response"
    exit 1
fi

# Extract and display response
echo "🤖 Response:"
echo "============"
echo "$response" | jq -r '.choices[0].message.content' 2>/dev/null || {
    echo "❌ Failed to parse response"
    echo "Raw response: $response"
    exit 1
}

echo
echo "✅ Chat example completed successfully!"

# Show usage statistics
echo
echo "📊 Usage Information:"
echo "===================="
echo "$response" | jq '{
    model: .model,
    tokens_used: .usage.total_tokens,
    prompt_tokens: .usage.prompt_tokens,
    completion_tokens: .usage.completion_tokens
}' 2>/dev/null || echo "Usage information not available"